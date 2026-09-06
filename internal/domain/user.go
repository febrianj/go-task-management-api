package domain

import (
	"time"

	"github.com/google/uuid"
)

type Team struct {
	ID        uuid.UUID
	Name      string
	CreatedAt time.Time
}

type User struct {
	ID           uuid.UUID
	TeamID       uuid.UUID
	Email        string
	Name         string
	PasswordHash string
	CreatedAt    time.Time
}
