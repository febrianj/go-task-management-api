package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	ActionCreated  = "created"
	ActionUpdated  = "updated"
	ActionAssigned = "assigned"
	ActionDeleted  = "deleted"
)

type TaskLog struct {
	ID        int64
	TaskID    uuid.UUID
	ActorID   uuid.UUID
	Action    string
	FromValue []byte // JSON
	ToValue   []byte // JSON
	CreatedAt time.Time
}
