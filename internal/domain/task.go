package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusPending    Status = "pending"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
	StatusCancelled  Status = "cancelled"
)

func ParseStatus(s string) (Status, bool) {
	switch Status(strings.ToLower(strings.TrimSpace(s))) {
	case StatusPending:
		return StatusPending, true
	case StatusInProgress:
		return StatusInProgress, true
	case StatusDone:
		return StatusDone, true
	case StatusCancelled:
		return StatusCancelled, true
	default:
		return "", false
	}
}

type Task struct {
	ID          uuid.UUID
	TeamID      uuid.UUID
	CreatedBy   uuid.UUID
	AssigneeID  *uuid.UUID
	Title       string
	Description string
	Status      Status
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (t *Task) VisibleTo(userID uuid.UUID) bool {
	if t.CreatedBy == userID {
		return true
	}
	return t.AssigneeID != nil && *t.AssigneeID == userID
}

func (t *Task) MutableBy(userID uuid.UUID) bool {
	return t.CreatedBy == userID
}
