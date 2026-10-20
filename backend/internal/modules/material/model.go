package material

import (
	"time"

	"github.com/google/uuid"
)

type MaterialRequisition struct {
	ID                uuid.UUID  `json:"id"`
	MRNumber          string     `json:"mr_number"`
	DepartmentID      uuid.UUID  `json:"department_id"`
	DepartmentName    string     `json:"department_name,omitempty"`
	RequestedBy       uuid.UUID  `json:"requested_by"`
	RequesterName     string     `json:"requester_name,omitempty"`
	RequiredDate      *time.Time `json:"required_date"`
	Purpose           *string    `json:"purpose"`
	Status            string     `json:"status"`
	PMApprovedBy      *uuid.UUID `json:"pm_approved_by"`
	PMApprovedAt      *time.Time `json:"pm_approved_at"`
	PMRejectionReason *string    `json:"pm_rejection_reason"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	Items             []MRItem   `json:"items,omitempty"`
}

type MRItem struct {
	ID                         uuid.UUID `json:"id"`
	MRID                       uuid.UUID `json:"mr_id"`
	ItemID                     uuid.UUID `json:"item_id"`
	ItemCode                   string    `json:"item_code,omitempty"`
	ItemName                   string    `json:"item_name,omitempty"`
	Quantity                   float64   `json:"quantity"`
	IssuedQty                  float64   `json:"issued_qty"`
	AvailableQty               float64   `json:"available_qty"`
	ReservedQty                float64   `json:"reserved_qty"`
	StockSufficient            bool      `json:"stock_sufficient"`
	SuggestPurchaseRequisition bool      `json:"suggest_purchase_requisition"`
	Notes                      *string   `json:"notes"`
}

type MaterialIssue struct {
	ID             uuid.UUID           `json:"id"`
	IssueNumber    string              `json:"issue_number"`
	MRID           uuid.UUID           `json:"mr_id"`
	MRNumber       string              `json:"mr_number,omitempty"`
	IssuedBy       uuid.UUID           `json:"issued_by"`
	IssuedByName   string              `json:"issued_by_name,omitempty"`
	IssuedTo       *uuid.UUID          `json:"issued_to"`
	IssuedToName   *string             `json:"issued_to_name,omitempty"`
	IssuedDate     time.Time           `json:"issued_date"`
	Notes          *string             `json:"notes"`
	CreatedAt      time.Time           `json:"created_at"`
	AcknowledgedBy *uuid.UUID          `json:"acknowledged_by"`
	AcknowledgedAt *time.Time          `json:"acknowledged_at"`
	Items          []MaterialIssueItem `json:"items,omitempty"`
}

type MaterialIssueItem struct {
	ID       uuid.UUID  `json:"id"`
	IssueID  uuid.UUID  `json:"issue_id"`
	MRItemID uuid.UUID  `json:"mr_item_id"`
	ItemID   uuid.UUID  `json:"item_id"`
	ItemCode string     `json:"item_code,omitempty"`
	ItemName string     `json:"item_name,omitempty"`
	BatchID  *uuid.UUID `json:"batch_id"`
	Quantity float64    `json:"quantity"`
}

type StatusHistory struct {
	ID            uuid.UUID  `json:"id"`
	EntityType    string     `json:"entity_type"`
	EntityID      uuid.UUID  `json:"entity_id"`
	OldStatus     *string    `json:"old_status"`
	NewStatus     string     `json:"new_status"`
	ChangedBy     *uuid.UUID `json:"changed_by"`
	ChangedByName string     `json:"changed_by_name,omitempty"`
	Reason        *string    `json:"reason"`
	CreatedAt     time.Time  `json:"created_at"`
}
