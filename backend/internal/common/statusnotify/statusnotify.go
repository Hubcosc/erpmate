package statusnotify

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Create records an in-app status notification for the requester and the next role in the workflow.
// It must be called in the same transaction as the status change.
func Create(ctx context.Context, tx pgx.Tx, entityType string, entityID, requesterID, actorID uuid.UUID, status string) error {
	title := "Request status updated"
	message := fmt.Sprintf("Your %s status changed to %s.", entityType, status)

	// Notification for the requester.
	if _, err := tx.Exec(ctx, `
		INSERT INTO notifications (user_id, title, message, entity_type, entity_id, created_by)
		VALUES ($1::uuid, $2::text, $3::text, $4::text, $5::uuid, $6::uuid)
	`, requesterID, title, message, entityType, entityID, actorID); err != nil {
		return err
	}

	// Work out which role must act next.
	var nextRole string
	switch status {
	case "pending_pm":
		nextRole = "production_manager"
	case "pending_finance":
		nextRole = "finance"
	case "approved":
		switch entityType {
		case "purchase_requisition":
			nextRole = "purchasing"
		case "material_requisition":
			nextRole = "store_keeper"
		}
	}
	if nextRole == "" {
		return nil
	}

	// Notification for everyone holding that role (except the actor).
	// Every parameter is cast explicitly: $6 is used twice, and without a cast
	// Postgres deduces two different types for it (SQLSTATE 42P08).
	_, err := tx.Exec(ctx, `
		INSERT INTO notifications (user_id, title, message, entity_type, entity_id, created_by)
		SELECT DISTINCT u.id, $1::text, $2::text, $3::text, $4::uuid, $6::uuid
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE r.name = $5::text
		  AND u.is_active = TRUE
		  AND u.deleted_at IS NULL
		  AND u.id <> $6::uuid
	`, title, fmt.Sprintf("%s %s is waiting for your action.", entityType, status), entityType, entityID, nextRole, actorID)
	return err
}
