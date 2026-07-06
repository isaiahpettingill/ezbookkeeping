package services

import (
	"time"

	"xorm.io/xorm"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/datastore"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/uuid"
)

// BudgetService represents budget service
type BudgetService struct {
	ServiceUsingDB
	ServiceUsingUuid
}

// Initialize a budget service singleton instance
var (
	Budgets = &BudgetService{
		ServiceUsingDB: ServiceUsingDB{
			container: datastore.Container,
		},
		ServiceUsingUuid: ServiceUsingUuid{
			container: uuid.Container,
		},
	}
)

// GetTotalBudgetCountByUid returns total budget count of user
func (s *BudgetService) GetTotalBudgetCountByUid(c core.Context, uid int64) (int64, error) {
	if uid <= 0 {
		return 0, errs.ErrUserIdInvalid
	}

	count, err := s.UserDataDB(uid).NewSession(c).Where("uid=? AND deleted=?", uid, false).Count(&models.Budget{})
	return count, err
}

// GetAllBudgetsByUid returns all budgets of user
func (s *BudgetService) GetAllBudgetsByUid(c core.Context, uid int64) ([]*models.Budget, error) {
	if uid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	var budgets []*models.Budget
	err := s.UserDataDB(uid).NewSession(c).Where("uid=? AND deleted=?", uid, false).OrderBy("display_order asc").Find(&budgets)
	return budgets, err
}

// GetBudgetByBudgetId returns a budget model according to budget id
func (s *BudgetService) GetBudgetByBudgetId(c core.Context, uid int64, budgetId int64) (*models.Budget, error) {
	if uid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	if budgetId <= 0 {
		return nil, errs.ErrBudgetIdInvalid
	}

	budget := &models.Budget{}
	has, err := s.UserDataDB(uid).NewSession(c).ID(budgetId).Where("uid=? AND deleted=?", uid, false).Get(budget)

	if err != nil {
		return nil, err
	} else if !has {
		return nil, errs.ErrBudgetNotFound
	}

	return budget, nil
}

// GetExistingBudgetByScope returns an existing budget with the same scope
func (s *BudgetService) GetExistingBudgetByScope(c core.Context, uid int64, budgetType models.BudgetType, categoryId int64, excludeBudgetId int64) (*models.Budget, error) {
	if uid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	budget := &models.Budget{}
	sess := s.UserDataDB(uid).NewSession(c).Where("uid=? AND deleted=? AND type=? AND category_id=?", uid, false, budgetType, categoryId)

	if excludeBudgetId > 0 {
		sess = sess.And("budget_id<>?", excludeBudgetId)
	}

	has, err := sess.Limit(1).Get(budget)

	if err != nil {
		return nil, err
	} else if !has {
		return nil, nil
	}

	return budget, nil
}

// GetMaxDisplayOrder returns the max display order
func (s *BudgetService) GetMaxDisplayOrder(c core.Context, uid int64) (int32, error) {
	if uid <= 0 {
		return 0, errs.ErrUserIdInvalid
	}

	budget := &models.Budget{}
	has, err := s.UserDataDB(uid).NewSession(c).Cols("uid", "deleted", "display_order").Where("uid=? AND deleted=?", uid, false).OrderBy("display_order desc").Limit(1).Get(budget)

	if err != nil {
		return 0, err
	}

	if has {
		return budget.DisplayOrder, nil
	}

	return 0, nil
}

// CreateBudget saves a new budget model to database
func (s *BudgetService) CreateBudget(c core.Context, budget *models.Budget) error {
	if budget.Uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	budget.BudgetId = s.GenerateUuid(uuid.UUID_TYPE_BUDGET)

	if budget.BudgetId < 1 {
		return errs.ErrSystemIsBusy
	}

	budget.Deleted = false
	budget.PeriodType = models.BUDGET_PERIOD_TYPE_MONTHLY
	budget.CreatedUnixTime = time.Now().Unix()
	budget.UpdatedUnixTime = time.Now().Unix()

	return s.UserDataDB(budget.Uid).DoTransaction(c, func(sess *xorm.Session) error {
		_, err := sess.Insert(budget)
		return err
	})
}

// ModifyBudget saves an existed budget model to database
func (s *BudgetService) ModifyBudget(c core.Context, budget *models.Budget) error {
	if budget.Uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	budget.PeriodType = models.BUDGET_PERIOD_TYPE_MONTHLY
	budget.UpdatedUnixTime = time.Now().Unix()

	return s.UserDataDB(budget.Uid).DoTransaction(c, func(sess *xorm.Session) error {
		updatedRows, err := sess.ID(budget.BudgetId).Cols("type", "category_id", "period_type", "amount", "start_year_month", "end_year_month", "updated_unix_time").Where("uid=? AND deleted=?", budget.Uid, false).Update(budget)

		if err != nil {
			return err
		} else if updatedRows < 1 {
			return errs.ErrBudgetNotFound
		}

		return nil
	})
}

// ModifyBudgetDisplayOrders updates display order of given budgets
func (s *BudgetService) ModifyBudgetDisplayOrders(c core.Context, uid int64, budgets []*models.Budget) error {
	if uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	for i := 0; i < len(budgets); i++ {
		budgets[i].UpdatedUnixTime = time.Now().Unix()
	}

	return s.UserDataDB(uid).DoTransaction(c, func(sess *xorm.Session) error {
		for i := 0; i < len(budgets); i++ {
			budget := budgets[i]
			updatedRows, err := sess.ID(budget.BudgetId).Cols("display_order", "updated_unix_time").Where("uid=? AND deleted=?", uid, false).Update(budget)

			if err != nil {
				return err
			} else if updatedRows < 1 {
				return errs.ErrBudgetNotFound
			}
		}

		return nil
	})
}

// DeleteBudget deletes an existed budget from database
func (s *BudgetService) DeleteBudget(c core.Context, uid int64, budgetId int64) error {
	if uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	now := time.Now().Unix()
	updateModel := &models.Budget{
		Deleted:         true,
		DeletedUnixTime: now,
	}

	return s.UserDataDB(uid).DoTransaction(c, func(sess *xorm.Session) error {
		deletedRows, err := sess.ID(budgetId).Cols("deleted", "deleted_unix_time").Where("uid=? AND deleted=?", uid, false).Update(updateModel)

		if err != nil {
			return err
		} else if deletedRows < 1 {
			return errs.ErrBudgetNotFound
		}

		return nil
	})
}

// DeleteAllBudgets deletes all existed budgets
func (s *BudgetService) DeleteAllBudgets(c core.Context, uid int64) error {
	if uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	now := time.Now().Unix()
	updateModel := &models.Budget{
		Deleted:         true,
		DeletedUnixTime: now,
	}

	return s.UserDataDB(uid).DoTransaction(c, func(sess *xorm.Session) error {
		_, err := sess.Cols("deleted", "deleted_unix_time").Where("uid=? AND deleted=?", uid, false).Update(updateModel)
		return err
	})
}
