package notification

import (
	"time"

	"github.com/google/uuid"
)

type Notification struct {
	ID         uuid.UUID  `json:"id"`
	Title      string     `json:"title"`
	Message    string     `json:"message"`
	EntityType string     `json:"entity_type"`
	EntityID   uuid.UUID  `json:"entity_id"`
	ReadAt     *time.Time `json:"read_at"`
	CreatedAt  time.Time  `json:"created_at"`
}
