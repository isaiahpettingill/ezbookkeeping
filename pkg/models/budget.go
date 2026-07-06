package models

// BudgetType represents the scope that a budget applies to
type BudgetType byte

// Budget types
const (
	BUDGET_TYPE_TOTAL_EXPENSE      BudgetType = 1 // total expense across all categories for the month
	BUDGET_TYPE_PRIMARY_CATEGORY   BudgetType = 2 // total expense for a primary (broad) category for the month
	BUDGET_TYPE_SECONDARY_CATEGORY BudgetType = 3 // total expense for a sub-category for the month
)

// BUDGET_PERIOD_TYPE_MONTHLY represents the monthly budget period type
const BUDGET_PERIOD_TYPE_MONTHLY int32 = 1

// Budget represents budget data stored in database
type Budget struct {
	BudgetId        int64      `xorm:"PK"`
	Uid             int64      `xorm:"INDEX(IDX_budget_uid_deleted_type_category_id_order) NOT NULL"`
	Deleted         bool       `xorm:"INDEX(IDX_budget_uid_deleted_type_category_id_order) NOT NULL"`
	Type            BudgetType `xorm:"INDEX(IDX_budget_uid_deleted_type_category_id_order) NOT NULL"`
	CategoryId      int64      `xorm:"INDEX(IDX_budget_uid_deleted_type_category_id_order) NOT NULL"` // 0 for total expense budget
	PeriodType      int32      `xorm:"NOT NULL"`                                                      // currently always 1 (monthly)
	Amount          int64      `xorm:"NOT NULL"`                                                      // monthly budget amount in minor units
	StartYearMonth  int32      `xorm:"NOT NULL"`                                                      // YYYYMM, 0 means active from any time
	EndYearMonth    int32      `xorm:"NOT NULL"`                                                      // YYYYMM, 0 means no end
	DisplayOrder    int32      `xorm:"INDEX(IDX_budget_uid_deleted_type_category_id_order) NOT NULL"`
	CreatedUnixTime int64
	UpdatedUnixTime int64
	DeletedUnixTime int64
}

// BudgetCreateRequest represents all parameters of budget creation request
type BudgetCreateRequest struct {
	Type            BudgetType `json:"type" binding:"required"`
	CategoryId      int64      `json:"categoryId,string" binding:"min=0"`
	Amount          int64      `json:"amount" binding:"required,min=1,max=99999999999"`
	StartYearMonth  int32      `json:"startYearMonth" binding:"min=0"`
	EndYearMonth    int32      `json:"endYearMonth" binding:"min=0"`
	ClientSessionId string     `json:"clientSessionId"`
}

// BudgetModifyRequest represents all parameters of budget modification request
type BudgetModifyRequest struct {
	Id             int64      `json:"id,string" binding:"required,min=1"`
	Type           BudgetType `json:"type" binding:"required"`
	CategoryId     int64      `json:"categoryId,string" binding:"min=0"`
	Amount         int64      `json:"amount" binding:"required,min=1,max=99999999999"`
	StartYearMonth int32      `json:"startYearMonth" binding:"min=0"`
	EndYearMonth   int32      `json:"endYearMonth" binding:"min=0"`
}

// BudgetGetRequest represents all parameters of budget getting request
type BudgetGetRequest struct {
	Id int64 `form:"id,string" binding:"required,min=1"`
}

// BudgetMoveRequest represents all parameters of budget moving request
type BudgetMoveRequest struct {
	NewDisplayOrders []*BudgetNewDisplayOrderRequest `json:"newDisplayOrders" binding:"required,min=1"`
}

// BudgetNewDisplayOrderRequest represents a data pair of id and display order
type BudgetNewDisplayOrderRequest struct {
	Id           int64 `json:"id,string" binding:"required,min=1"`
	DisplayOrder int32 `json:"displayOrder"`
}

// BudgetDeleteRequest represents all parameters of budget deleting request
type BudgetDeleteRequest struct {
	Id int64 `json:"id,string" binding:"required,min=1"`
}

// BudgetInfoResponse represents a view-object of budget
type BudgetInfoResponse struct {
	Id             int64      `json:"id,string"`
	Type           BudgetType `json:"type"`
	CategoryId     int64      `json:"categoryId,string"`
	Amount         int64      `json:"amount"`
	StartYearMonth int32      `json:"startYearMonth"`
	EndYearMonth   int32      `json:"endYearMonth"`
	DisplayOrder   int32      `json:"displayOrder"`
}

// ToBudgetInfoResponse returns a view-object according to database model
func (b *Budget) ToBudgetInfoResponse() *BudgetInfoResponse {
	return &BudgetInfoResponse{
		Id:             b.BudgetId,
		Type:           b.Type,
		CategoryId:     b.CategoryId,
		Amount:         b.Amount,
		StartYearMonth: b.StartYearMonth,
		EndYearMonth:   b.EndYearMonth,
		DisplayOrder:   b.DisplayOrder,
	}
}

// BudgetInfoResponseSlice represents the slice data structure of BudgetInfoResponse
type BudgetInfoResponseSlice []*BudgetInfoResponse

// Len returns the count of items
func (s BudgetInfoResponseSlice) Len() int {
	return len(s)
}

// Swap swaps two items
func (s BudgetInfoResponseSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Less reports whether the first item is less than the second one
func (s BudgetInfoResponseSlice) Less(i, j int) bool {
	return s[i].DisplayOrder < s[j].DisplayOrder
}
