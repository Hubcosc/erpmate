package purchase

import (
	"time"

	"github.com/google/uuid"
)

type CreatePRRequest struct {
	DepartmentID        uuid.UUID      `json:"department_id"`
	PreferredSupplierID *uuid.UUID     `json:"preferred_supplier_id"`
	RequiredDate        *string        `json:"required_date"` // format: "2006-01-02", e.g. "2026-10-20"
	Justification       *string        `json:"justification"`
	Items               []CreatePRItem `json:"items"`
}

type CreatePRItem struct {
	ItemID             *uuid.UUID `json:"item_id"`
	Description        *string    `json:"description"`
	Quantity           float64    `json:"quantity"`
	UnitOfMeasure      *string    `json:"unit_of_measure"`
	EstimatedUnitPrice *float64   `json:"estimated_unit_price"`
	Notes              *string    `json:"notes"`
}

// UpdatePRRequest has PUT semantics: it replaces the editable fields. Omitting
// preferred_supplier_id / required_date / justification clears them; omitting
// department_id keeps the current department.
type UpdatePRRequest struct {
	DepartmentID        uuid.UUID      `json:"department_id"`
	PreferredSupplierID *uuid.UUID     `json:"preferred_supplier_id"`
	RequiredDate        *string        `json:"required_date"`
	Justification       *string        `json:"justification"`
	Items               []CreatePRItem `json:"items"`
}

// ActionRequest is the optional body of workflow actions. A reason is required
// for reject / return / suspend and optional (acts as a comment) otherwise.
type ActionRequest struct {
	Reason *string `json:"reason"`
}

// ListFilter is parsed from the query string of GET /.
//
//	status        comma separated, e.g. status=pending_pm,pending_finance
//	q             search in PR number, justification, requester, department
//	department_id uuid
//	from, to      created date range, YYYY-MM-DD (inclusive)
//	mine          true = only requisitions I created
//	sort_by       created_at | required_date | total_estimated | pr_number | status
//	order         asc | desc
//	page, limit   pagination (limit max 100)
type ListFilter struct {
	Statuses     []string
	Search       string
	DepartmentID *uuid.UUID
	DateFrom     *time.Time
	DateTo       *time.Time
	MineOnly     bool
	SortBy       string
	SortDir      string
	Page         int
	Limit        int
}
