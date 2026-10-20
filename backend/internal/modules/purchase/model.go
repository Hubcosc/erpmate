package purchase

import (
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Statuses, roles and UI actions
// ---------------------------------------------------------------------------

const (
	StatusDraft          = "draft"
	StatusPendingPM      = "pending_pm"
	StatusPendingFinance = "pending_finance"
	StatusApproved       = "approved"
	StatusRejected       = "rejected"
	StatusSuspended      = "suspended"
	StatusReturned       = "returned_for_correction"
)

// AllStatuses is the canonical list of requisition statuses (used for filter
// validation and for the summary endpoint so the UI always gets every key).
var AllStatuses = []string{
	StatusDraft,
	StatusPendingPM,
	StatusPendingFinance,
	StatusApproved,
	StatusRejected,
	StatusSuspended,
	StatusReturned,
}

func IsValidStatus(s string) bool {
	for _, v := range AllStatuses {
		if v == s {
			return true
		}
	}
	return false
}

// IsEditableStatus reports whether the requester may still edit/delete/submit.
func IsEditableStatus(s string) bool {
	return s == StatusDraft || s == StatusReturned
}

const (
	RoleAdmin             = "admin"
	RoleDepartmentHead    = "department_head"
	RoleProductionManager = "production_manager"
	RoleFinance           = "finance"
	RolePurchasing        = "purchasing"
)

// Action names returned in available_actions so the UI can render buttons
// without duplicating the workflow rules.
const (
	ActionEdit           = "edit"
	ActionSubmit         = "submit"
	ActionDelete         = "delete"
	ActionPMApprove      = "pm_approve"
	ActionPMReject       = "pm_reject"
	ActionPMReturn       = "pm_return"
	ActionFinanceApprove = "finance_approve"
	ActionFinanceSuspend = "finance_suspend"
	ActionFinanceReject  = "finance_reject"
)

// Actor is the authenticated user performing a request.
type Actor struct {
	UserID uuid.UUID
	Role   string
}

// ---------------------------------------------------------------------------
// Domain errors (mapped to HTTP status codes by the handler)
// ---------------------------------------------------------------------------

type ErrorKind int

const (
	KindValidation ErrorKind = iota + 1
	KindNotFound
	KindForbidden
	KindConflict
)

// Error is a safe-to-show domain error. Anything that is not an *Error is
// treated as an internal failure and is never leaked to the client.
type Error struct {
	Kind    ErrorKind
	Message string
}

func (e *Error) Error() string { return e.Message }

func Validation(msg string) *Error { return &Error{Kind: KindValidation, Message: msg} }
func NotFound(msg string) *Error   { return &Error{Kind: KindNotFound, Message: msg} }
func Forbidden(msg string) *Error  { return &Error{Kind: KindForbidden, Message: msg} }
func Conflict(msg string) *Error   { return &Error{Kind: KindConflict, Message: msg} }

var ErrNotFound = NotFound("purchase requisition not found")

// ---------------------------------------------------------------------------
// Entities
// ---------------------------------------------------------------------------

type PurchaseRequisition struct {
	ID                         uuid.UUID      `json:"id"`
	PRNumber                   string         `json:"pr_number"`
	DepartmentID               uuid.UUID      `json:"department_id"`
	DepartmentName             string         `json:"department_name"`
	RequestedBy                uuid.UUID      `json:"requested_by"`
	RequesterName              string         `json:"requester_name"`
	RequiredDate               *time.Time     `json:"required_date"`
	Justification              *string        `json:"justification"`
	Status                     string         `json:"status"`
	PMApprovedBy               *uuid.UUID     `json:"pm_approved_by"`
	PMApprovedAt               *time.Time     `json:"pm_approved_at"`
	PMRejectionReason          *string        `json:"pm_rejection_reason"`
	FinanceApprovedBy          *uuid.UUID     `json:"finance_approved_by"`
	FinanceApprovedAt          *time.Time     `json:"finance_approved_at"`
	FinanceRejectionReason     *string        `json:"finance_rejection_reason"`
	SuspensionReason           *string        `json:"suspension_reason"`
	ReturnReason               *string        `json:"return_reason"`
	PreferredSupplierID        *uuid.UUID     `json:"preferred_supplier_id"`
	StockCheck                 []PRStockCheck `json:"stock_check"`
	SuggestMaterialRequisition bool           `json:"suggest_material_requisition"`
	TotalEstimated             float64        `json:"total_estimated"`
	ItemCount                  int            `json:"item_count"`
	AvailableActions           []string       `json:"available_actions"`
	CreatedAt                  time.Time      `json:"created_at"`
	UpdatedAt                  time.Time      `json:"updated_at"`
	Items                      []PRItem       `json:"items"`
}

// PRSummary is the lightweight row returned by the list endpoint.
type PRSummary struct {
	ID                  uuid.UUID  `json:"id"`
	PRNumber            string     `json:"pr_number"`
	DepartmentID        uuid.UUID  `json:"department_id"`
	DepartmentName      string     `json:"department_name"`
	RequestedBy         uuid.UUID  `json:"requested_by"`
	RequesterName       string     `json:"requester_name"`
	RequiredDate        *time.Time `json:"required_date"`
	Justification       *string    `json:"justification"`
	Status              string     `json:"status"`
	PreferredSupplierID *uuid.UUID `json:"preferred_supplier_id"`
	TotalEstimated      float64    `json:"total_estimated"`
	ItemCount           int        `json:"item_count"`
	AvailableActions    []string   `json:"available_actions"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// StatusSummary feeds the dashboard tabs / badges in the UI.
type StatusSummary struct {
	Total          int            `json:"total"`
	Counts         map[string]int `json:"counts"`
	ActionRequired int            `json:"action_required"` // items waiting on the current user's role
}

// StockRequest is one aggregated (item, quantity) pair to check against stock.
type StockRequest struct {
	ItemID   uuid.UUID
	Quantity float64
}

type PRStockCheck struct {
	ItemID     uuid.UUID `json:"item_id"`
	ItemCode   string    `json:"item_code"`
	ItemName   string    `json:"item_name"`
	Requested  float64   `json:"requested_quantity"`
	Available  float64   `json:"available_quantity"`
	Shortfall  float64   `json:"shortfall_quantity"`
	Sufficient bool      `json:"sufficient"`
}

type PRItem struct {
	ID                 uuid.UUID  `json:"id"`
	PRID               uuid.UUID  `json:"pr_id"`
	ItemID             *uuid.UUID `json:"item_id"`
	ItemCode           *string    `json:"item_code,omitempty"`
	ItemName           *string    `json:"item_name,omitempty"`
	Description        *string    `json:"description"`
	Quantity           float64    `json:"quantity"`
	UnitOfMeasure      *string    `json:"unit_of_measure"`
	EstimatedUnitPrice *float64   `json:"estimated_unit_price"`
	EstimatedTotal     *float64   `json:"estimated_total"`
	Notes              *string    `json:"notes"`
}

type StatusHistory struct {
	ID            uuid.UUID  `json:"id"`
	EntityType    string     `json:"entity_type"`
	EntityID      uuid.UUID  `json:"entity_id"`
	OldStatus     *string    `json:"old_status"`
	NewStatus     string     `json:"new_status"`
	ChangedBy     *uuid.UUID `json:"changed_by"`
	ChangedByName string     `json:"changed_by_name"`
	Reason        *string    `json:"reason"`
	CreatedAt     time.Time  `json:"created_at"`
}
