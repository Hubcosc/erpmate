package material

import "github.com/google/uuid"

type CreateMRRequest struct {
	DepartmentID uuid.UUID      `json:"department_id"`
	RequiredDate *string        `json:"required_date"` // YYYY-MM-DD
	Purpose      *string        `json:"purpose"`
	Items        []CreateMRItem `json:"items"`
}

type CreateMRItem struct {
	ItemID   uuid.UUID `json:"item_id"`
	Quantity float64   `json:"quantity"`
	Notes    *string   `json:"notes"`
}

type ActionRequest struct {
	Reason *string `json:"reason"`
}

type CreateIssueRequest struct {
	MRID     uuid.UUID           `json:"mr_id"`
	IssuedTo *uuid.UUID          `json:"issued_to"`
	Notes    *string             `json:"notes"`
	Items    []CreateIssueItem   `json:"items"`
}

type CreateIssueItem struct {
	MRItemID uuid.UUID  `json:"mr_item_id"`
	ItemID   uuid.UUID  `json:"item_id"`
	BatchID  *uuid.UUID `json:"batch_id"`
	Quantity float64    `json:"quantity"`
}