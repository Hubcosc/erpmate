package notification

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func (r *Repository) List(ctx context.Context, userID uuid.UUID, unreadOnly bool, page, limit int) ([]Notification, int64, error) {
	where := "WHERE user_id=$1 AND deleted_at IS NULL"
	if unreadOnly {
		where += " AND read_at IS NULL"
	}
	var total int64
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM notifications "+where, userID).Scan(&total); err != nil {
		return nil, 0, err
	}
	query := fmt.Sprintf("SELECT id,title,message,entity_type,entity_id,read_at,created_at FROM notifications %s ORDER BY created_at DESC LIMIT $2 OFFSET $3", where)
	rows, err := r.db.Query(ctx, query, userID, limit, (page-1)*limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]Notification, 0)
	for rows.Next() {
		var item Notification
		if err := rows.Scan(&item.ID, &item.Title, &item.Message, &item.EntityType, &item.EntityID, &item.ReadAt, &item.CreatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *Repository) MarkRead(ctx context.Context, userID, id uuid.UUID) (bool, error) {
	result, err := r.db.Exec(ctx, `UPDATE notifications SET read_at=$1, updated_at=$1, updated_by=$2 WHERE id=$3 AND user_id=$2 AND read_at IS NULL AND deleted_at IS NULL`, time.Now().UTC(), userID, id)
	return result.RowsAffected() > 0, err
}

func (r *Repository) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE notifications SET read_at=$1, updated_at=$1, updated_by=$2 WHERE user_id=$2 AND read_at IS NULL AND deleted_at IS NULL`, time.Now().UTC(), userID)
	return err
}
