package errs

import "net/http"

// Error codes related to budget
var (
	ErrBudgetIdInvalid          = NewNormalError(NormalSubcategoryBudget, 0, http.StatusBadRequest, "budget id is invalid")
	ErrBudgetNotFound           = NewNormalError(NormalSubcategoryBudget, 1, http.StatusBadRequest, "budget not found")
	ErrBudgetTypeInvalid        = NewNormalError(NormalSubcategoryBudget, 2, http.StatusBadRequest, "budget type is invalid")
	ErrBudgetCategoryInvalid    = NewNormalError(NormalSubcategoryBudget, 3, http.StatusBadRequest, "budget category is invalid")
	ErrBudgetPeriodInvalid      = NewNormalError(NormalSubcategoryBudget, 4, http.StatusBadRequest, "budget period is invalid")
	ErrBudgetAmountInvalid      = NewNormalError(NormalSubcategoryBudget, 5, http.StatusBadRequest, "budget amount is invalid")
	ErrBudgetDateRangeInvalid   = NewNormalError(NormalSubcategoryBudget, 6, http.StatusBadRequest, "budget date range is invalid")
	ErrBudgetCategoryInUse      = NewNormalError(NormalSubcategoryBudget, 7, http.StatusBadRequest, "a budget for this category already exists")
	ErrBudgetTotalAlreadyExists = NewNormalError(NormalSubcategoryBudget, 8, http.StatusBadRequest, "a total spend budget already exists")
)
