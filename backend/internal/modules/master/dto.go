package master

import "github.com/google/uuid"

// ====================== LIST / PAGINATION ======================

const (
	defaultLimit = 20
	maxLimit     = 100
)

// ListParams are the common query-string filters: ?search=&page=&limit=&is_active=
type ListParams struct {
	Search   string
	Page     int
	Limit    int
	IsActive *bool
}

func (p *ListParams) normalize() {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.Limit < 1 || p.Limit > maxLimit {
		p.Limit = defaultLimit
	}
}

func (p ListParams) offset() int { return (p.Page - 1) * p.Limit }

type ItemListParams struct {
	ListParams
	ItemType   string
	CategoryID *uuid.UUID
}

// ListResult carries the normalized page/limit back so the response meta is always accurate.
type ListResult[T any] struct {
	Items []T
	Page  int
	Limit int
	Total int64
}

// ====================== DEPARTMENT ======================

type CreateDepartmentRequest struct {
	Name string  `json:"name"`
	Code *string `json:"code"`
}

type UpdateDepartmentRequest struct {
	Name     *string          `json:"name"`
	Code     Optional[string] `json:"code"`
	IsActive *bool            `json:"is_active"`
}

// ====================== ITEM CATEGORY ======================

type CreateCategoryRequest struct {
	Name     string     `json:"name"`
	ParentID *uuid.UUID `json:"parent_id"`
}

type UpdateCategoryRequest struct {
	Name     *string             `json:"name"`
	ParentID Optional[uuid.UUID] `json:"parent_id"` // null = move to top level
}

// ====================== ITEM ======================

type CreateItemRequest struct {
	Code            string     `json:"code"`
	Name            string     `json:"name"`
	Description     *string    `json:"description"`
	CategoryID      *uuid.UUID `json:"category_id"`
	ItemType        string     `json:"item_type"`
	UnitOfMeasure   string     `json:"unit_of_measure"`
	MinStockLevel   float64    `json:"min_stock_level"`
	MaxStockLevel   *float64   `json:"max_stock_level"`
	ReorderLevel    *float64   `json:"reorder_level"`
	IsBatchTracked  bool       `json:"is_batch_tracked"`
	IsExpiryTracked bool       `json:"is_expiry_tracked"`
}

type UpdateItemRequest struct {
	Name            *string             `json:"name"`
	Description     Optional[string]    `json:"description"`
	CategoryID      Optional[uuid.UUID] `json:"category_id"`
	UnitOfMeasure   *string             `json:"unit_of_measure"`
	MinStockLevel   *float64            `json:"min_stock_level"`
	MaxStockLevel   Optional[float64]   `json:"max_stock_level"`
	ReorderLevel    Optional[float64]   `json:"reorder_level"`
	IsBatchTracked  *bool               `json:"is_batch_tracked"`
	IsExpiryTracked *bool               `json:"is_expiry_tracked"`
	IsActive        *bool               `json:"is_active"`
}

// ====================== SUPPLIER ======================

type CreateSupplierRequest struct {
	Code          *string `json:"code"`
	Name          string  `json:"name"`
	ContactPerson *string `json:"contact_person"`
	Phone         *string `json:"phone"`
	Email         *string `json:"email"`
	Address       *string `json:"address"`
	TinNumber     *string `json:"tin_number"`
}

type UpdateSupplierRequest struct {
	Code          Optional[string] `json:"code"`
	Name          *string          `json:"name"`
	ContactPerson Optional[string] `json:"contact_person"`
	Phone         Optional[string] `json:"phone"`
	Email         Optional[string] `json:"email"`
	Address       Optional[string] `json:"address"`
	TinNumber     Optional[string] `json:"tin_number"`
	IsActive      *bool            `json:"is_active"`
}

// ====================== CUSTOMER ======================

type CreateCustomerRequest struct {
	Code          *string `json:"code"`
	Name          string  `json:"name"`
	CustomerType  string  `json:"customer_type"`
	ContactPerson *string `json:"contact_person"`
	Phone         *string `json:"phone"`
	Email         *string `json:"email"`
	Address       *string `json:"address"`
	TinNumber     *string `json:"tin_number"`
}

type UpdateCustomerRequest struct {
	Code          Optional[string] `json:"code"`
	Name          *string          `json:"name"`
	CustomerType  *string          `json:"customer_type"`
	ContactPerson Optional[string] `json:"contact_person"`
	Phone         Optional[string] `json:"phone"`
	Email         Optional[string] `json:"email"`
	Address       Optional[string] `json:"address"`
	TinNumber     Optional[string] `json:"tin_number"`
	IsActive      *bool            `json:"is_active"`
}

// ====================== WAREHOUSE ======================

type CreateWarehouseRequest struct {
	Code          string  `json:"code"`
	Name          string  `json:"name"`
	WarehouseType string  `json:"warehouse_type"`
	Location      *string `json:"location"`
}

type UpdateWarehouseRequest struct {
	Name          *string          `json:"name"`
	WarehouseType *string          `json:"warehouse_type"`
	Location      Optional[string] `json:"location"`
	IsActive      *bool            `json:"is_active"`
}
