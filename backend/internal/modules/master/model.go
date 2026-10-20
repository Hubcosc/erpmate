package master

import (
	"time"

	"github.com/google/uuid"
)

// Allowed enum values. Exposed to the frontend via GET /lookups so dropdowns never drift from the backend.
var (
	ItemTypes      = []string{"raw_material", "finished_good", "consumable", "spare_part"}
	WarehouseTypes = []string{"raw_material", "finished_goods", "quarantine"}
)

type Department struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Code      *string   `json:"code"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ItemCategory struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	ParentID  *uuid.UUID `json:"parent_id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type Item struct {
	ID              uuid.UUID  `json:"id"`
	Code            string     `json:"code"`
	Name            string     `json:"name"`
	Description     *string    `json:"description"`
	CategoryID      *uuid.UUID `json:"category_id"`
	CategoryName    *string    `json:"category_name,omitempty"`
	ItemType        string     `json:"item_type"`
	UnitOfMeasure   string     `json:"unit_of_measure"`
	MinStockLevel   float64    `json:"min_stock_level"`
	MaxStockLevel   *float64   `json:"max_stock_level"`
	ReorderLevel    *float64   `json:"reorder_level"`
	IsBatchTracked  bool       `json:"is_batch_tracked"`
	IsExpiryTracked bool       `json:"is_expiry_tracked"`
	IsActive        bool       `json:"is_active"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type Supplier struct {
	ID            uuid.UUID `json:"id"`
	Code          *string   `json:"code"`
	Name          string    `json:"name"`
	ContactPerson *string   `json:"contact_person"`
	Phone         *string   `json:"phone"`
	Email         *string   `json:"email"`
	Address       *string   `json:"address"`
	TinNumber     *string   `json:"tin_number"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Customer struct {
	ID            uuid.UUID `json:"id"`
	Code          *string   `json:"code"`
	Name          string    `json:"name"`
	CustomerType  string    `json:"customer_type"`
	ContactPerson *string   `json:"contact_person"`
	Phone         *string   `json:"phone"`
	Email         *string   `json:"email"`
	Address       *string   `json:"address"`
	TinNumber     *string   `json:"tin_number"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Warehouse struct {
	ID            uuid.UUID `json:"id"`
	Code          string    `json:"code"`
	Name          string    `json:"name"`
	WarehouseType string    `json:"warehouse_type"`
	Location      *string   `json:"location"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
