package api

import (
	"sort"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/services"
)

// BudgetsApi represents budget api
type BudgetsApi struct {
	budgets    *services.BudgetService
	categories *services.TransactionCategoryService
}

// Initialize a budget api singleton instance
var (
	Budgets = &BudgetsApi{
		budgets:    services.Budgets,
		categories: services.TransactionCategories,
	}
)

// BudgetListHandler returns budget list of current user
func (a *BudgetsApi) BudgetListHandler(c *core.WebContext) (any, *errs.Error) {
	uid := c.GetCurrentUid()
	budgets, err := a.budgets.GetAllBudgetsByUid(c, uid)

	if err != nil {
		log.Errorf(c, "[budgets.BudgetListHandler] failed to get budgets for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	budgetResps := make(models.BudgetInfoResponseSlice, len(budgets))

	for i := 0; i < len(budgets); i++ {
		budgetResps[i] = budgets[i].ToBudgetInfoResponse()
	}

	sort.Sort(budgetResps)
	return budgetResps, nil
}

// BudgetGetHandler returns one specific budget of current user
func (a *BudgetsApi) BudgetGetHandler(c *core.WebContext) (any, *errs.Error) {
	var budgetGetReq models.BudgetGetRequest
	err := c.ShouldBindQuery(&budgetGetReq)

	if err != nil {
		log.Warnf(c, "[budgets.BudgetGetHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	uid := c.GetCurrentUid()
	budget, err := a.budgets.GetBudgetByBudgetId(c, uid, budgetGetReq.Id)

	if err != nil {
		log.Errorf(c, "[budgets.BudgetGetHandler] failed to get budget \"id:%d\" for user \"uid:%d\", because %s", budgetGetReq.Id, uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	return budget.ToBudgetInfoResponse(), nil
}

// BudgetCreateHandler saves a new budget by request parameters for current user
func (a *BudgetsApi) BudgetCreateHandler(c *core.WebContext) (any, *errs.Error) {
	var budgetCreateReq models.BudgetCreateRequest
	err := c.ShouldBindJSON(&budgetCreateReq)

	if err != nil {
		log.Warnf(c, "[budgets.BudgetCreateHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	uid := c.GetCurrentUid()

	if err := a.validateBudgetScope(c, uid, budgetCreateReq.Type, budgetCreateReq.CategoryId); err != nil {
		return nil, err
	}

	if err := a.validateBudgetDateRange(budgetCreateReq.StartYearMonth, budgetCreateReq.EndYearMonth); err != nil {
		return nil, err
	}

	existingBudget, err := a.budgets.GetExistingBudgetByScope(c, uid, budgetCreateReq.Type, budgetCreateReq.CategoryId, 0)

	if err != nil {
		log.Errorf(c, "[budgets.BudgetCreateHandler] failed to check existing budget for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	} else if existingBudget != nil {
		return nil, a.getDuplicateBudgetError(budgetCreateReq.Type)
	}

	maxOrderId, err := a.budgets.GetMaxDisplayOrder(c, uid)

	if err != nil {
		log.Errorf(c, "[budgets.BudgetCreateHandler] failed to get max display order for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	budget := a.createNewBudgetModel(uid, &budgetCreateReq, maxOrderId+1)
	err = a.budgets.CreateBudget(c, budget)

	if err != nil {
		log.Errorf(c, "[budgets.BudgetCreateHandler] failed to create budget \"id:%d\" for user \"uid:%d\", because %s", budget.BudgetId, uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[budgets.BudgetCreateHandler] user \"uid:%d\" has created a new budget \"id:%d\" successfully", uid, budget.BudgetId)
	return budget.ToBudgetInfoResponse(), nil
}

// BudgetModifyHandler saves an existed budget by request parameters for current user
func (a *BudgetsApi) BudgetModifyHandler(c *core.WebContext) (any, *errs.Error) {
	var budgetModifyReq models.BudgetModifyRequest
	err := c.ShouldBindJSON(&budgetModifyReq)

	if err != nil {
		log.Warnf(c, "[budgets.BudgetModifyHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	uid := c.GetCurrentUid()
	budget, err := a.budgets.GetBudgetByBudgetId(c, uid, budgetModifyReq.Id)

	if err != nil {
		log.Errorf(c, "[budgets.BudgetModifyHandler] failed to get budget \"id:%d\" for user \"uid:%d\", because %s", budgetModifyReq.Id, uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	if err := a.validateBudgetScope(c, uid, budgetModifyReq.Type, budgetModifyReq.CategoryId); err != nil {
		return nil, err
	}

	if err := a.validateBudgetDateRange(budgetModifyReq.StartYearMonth, budgetModifyReq.EndYearMonth); err != nil {
		return nil, err
	}

	existingBudget, err := a.budgets.GetExistingBudgetByScope(c, uid, budgetModifyReq.Type, budgetModifyReq.CategoryId, budgetModifyReq.Id)

	if err != nil {
		log.Errorf(c, "[budgets.BudgetModifyHandler] failed to check existing budget for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	} else if existingBudget != nil {
		return nil, a.getDuplicateBudgetError(budgetModifyReq.Type)
	}

	newBudget := &models.Budget{
		BudgetId:       budget.BudgetId,
		Uid:            uid,
		Type:           budgetModifyReq.Type,
		CategoryId:     budgetModifyReq.CategoryId,
		Amount:         budgetModifyReq.Amount,
		StartYearMonth: budgetModifyReq.StartYearMonth,
		EndYearMonth:   budgetModifyReq.EndYearMonth,
	}

	if newBudget.Type == budget.Type &&
		newBudget.CategoryId == budget.CategoryId &&
		newBudget.Amount == budget.Amount &&
		newBudget.StartYearMonth == budget.StartYearMonth &&
		newBudget.EndYearMonth == budget.EndYearMonth {
		return nil, errs.ErrNothingWillBeUpdated
	}

	err = a.budgets.ModifyBudget(c, newBudget)

	if err != nil {
		log.Errorf(c, "[budgets.BudgetModifyHandler] failed to update budget \"id:%d\" for user \"uid:%d\", because %s", budgetModifyReq.Id, uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[budgets.BudgetModifyHandler] user \"uid:%d\" has updated budget \"id:%d\" successfully", uid, budgetModifyReq.Id)

	budget.Type = newBudget.Type
	budget.CategoryId = newBudget.CategoryId
	budget.Amount = newBudget.Amount
	budget.StartYearMonth = newBudget.StartYearMonth
	budget.EndYearMonth = newBudget.EndYearMonth
	return budget.ToBudgetInfoResponse(), nil
}

// BudgetMoveHandler moves display order of existed budgets by request parameters for current user
func (a *BudgetsApi) BudgetMoveHandler(c *core.WebContext) (any, *errs.Error) {
	var budgetMoveReq models.BudgetMoveRequest
	err := c.ShouldBindJSON(&budgetMoveReq)

	if err != nil {
		log.Warnf(c, "[budgets.BudgetMoveHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	uid := c.GetCurrentUid()
	budgets := make([]*models.Budget, len(budgetMoveReq.NewDisplayOrders))

	for i := 0; i < len(budgetMoveReq.NewDisplayOrders); i++ {
		newDisplayOrder := budgetMoveReq.NewDisplayOrders[i]
		budgets[i] = &models.Budget{
			Uid:          uid,
			BudgetId:     newDisplayOrder.Id,
			DisplayOrder: newDisplayOrder.DisplayOrder,
		}
	}

	err = a.budgets.ModifyBudgetDisplayOrders(c, uid, budgets)

	if err != nil {
		log.Errorf(c, "[budgets.BudgetMoveHandler] failed to move budgets for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[budgets.BudgetMoveHandler] user \"uid:%d\" has moved budgets", uid)
	return true, nil
}

// BudgetDeleteHandler deletes an existed budget by request parameters for current user
func (a *BudgetsApi) BudgetDeleteHandler(c *core.WebContext) (any, *errs.Error) {
	var budgetDeleteReq models.BudgetDeleteRequest
	err := c.ShouldBindJSON(&budgetDeleteReq)

	if err != nil {
		log.Warnf(c, "[budgets.BudgetDeleteHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	uid := c.GetCurrentUid()
	err = a.budgets.DeleteBudget(c, uid, budgetDeleteReq.Id)

	if err != nil {
		log.Errorf(c, "[budgets.BudgetDeleteHandler] failed to delete budget \"id:%d\" for user \"uid:%d\", because %s", budgetDeleteReq.Id, uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[budgets.BudgetDeleteHandler] user \"uid:%d\" has deleted budget \"id:%d\"", uid, budgetDeleteReq.Id)
	return true, nil
}

func (a *BudgetsApi) validateBudgetScope(c *core.WebContext, uid int64, budgetType models.BudgetType, categoryId int64) *errs.Error {
	if budgetType == models.BUDGET_TYPE_TOTAL_EXPENSE {
		if categoryId != 0 {
			return errs.ErrBudgetCategoryInvalid
		}

		return nil
	}

	if budgetType != models.BUDGET_TYPE_PRIMARY_CATEGORY && budgetType != models.BUDGET_TYPE_SECONDARY_CATEGORY {
		return errs.ErrBudgetTypeInvalid
	}

	if categoryId <= 0 {
		return errs.ErrBudgetCategoryInvalid
	}

	category, err := a.categories.GetCategoryByCategoryId(c, uid, categoryId)

	if err != nil {
		log.Errorf(c, "[budgets.validateBudgetScope] failed to get category \"id:%d\" for user \"uid:%d\", because %s", categoryId, uid, err.Error())
		return errs.Or(err, errs.ErrOperationFailed)
	}

	if category.Type != models.CATEGORY_TYPE_EXPENSE {
		return errs.ErrBudgetCategoryInvalid
	}

	if budgetType == models.BUDGET_TYPE_PRIMARY_CATEGORY && category.ParentCategoryId != models.LevelOneTransactionCategoryParentId {
		return errs.ErrBudgetCategoryInvalid
	}

	if budgetType == models.BUDGET_TYPE_SECONDARY_CATEGORY && category.ParentCategoryId == models.LevelOneTransactionCategoryParentId {
		return errs.ErrBudgetCategoryInvalid
	}

	return nil
}

func (a *BudgetsApi) validateBudgetDateRange(startYearMonth int32, endYearMonth int32) *errs.Error {
	if !a.isValidYearMonth(startYearMonth) || !a.isValidYearMonth(endYearMonth) {
		return errs.ErrBudgetDateRangeInvalid
	}

	if startYearMonth > 0 && endYearMonth > 0 && startYearMonth > endYearMonth {
		return errs.ErrBudgetDateRangeInvalid
	}

	return nil
}

func (a *BudgetsApi) isValidYearMonth(yearMonth int32) bool {
	if yearMonth == 0 {
		return true
	}

	month := yearMonth % 100
	return yearMonth >= 100001 && month >= 1 && month <= 12
}

func (a *BudgetsApi) getDuplicateBudgetError(budgetType models.BudgetType) *errs.Error {
	if budgetType == models.BUDGET_TYPE_TOTAL_EXPENSE {
		return errs.ErrBudgetTotalAlreadyExists
	}

	return errs.ErrBudgetCategoryInUse
}

func (a *BudgetsApi) createNewBudgetModel(uid int64, budgetCreateReq *models.BudgetCreateRequest, order int32) *models.Budget {
	return &models.Budget{
		Uid:            uid,
		Type:           budgetCreateReq.Type,
		CategoryId:     budgetCreateReq.CategoryId,
		PeriodType:     models.BUDGET_PERIOD_TYPE_MONTHLY,
		Amount:         budgetCreateReq.Amount,
		StartYearMonth: budgetCreateReq.StartYearMonth,
		EndYearMonth:   budgetCreateReq.EndYearMonth,
		DisplayOrder:   order,
	}
}
