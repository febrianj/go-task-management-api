package http

import (
	"net/mail"
	"strings"
	"time"

	"github.com/febrianj/go-task-management-api/internal/apperr"
	"github.com/febrianj/go-task-management-api/internal/domain"
)

type RegisterRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
	TeamName string `json:"team_name"`
}

func (r RegisterRequest) Validate() []apperr.FieldError {
	var errs []apperr.FieldError

	if _, err := mail.ParseAddress(strings.TrimSpace(r.Email)); err != nil {
		errs = append(errs, apperr.FieldError{
			Field: "email", Message: "must be a valid email address"})
	}
	if strings.TrimSpace(r.Name) == "" {
		errs = append(errs, apperr.FieldError{
			Field: "name", Message: "must be filled",
		})
	}
	if len(r.Password) < 8 {
		errs = append(errs, apperr.FieldError{
			Field: "password", Message: "must be at least 8 characters",
		})
	}
	if len(r.Password) > 72 {
		errs = append(errs, apperr.FieldError{
			Field: "password", Message: "must be at most 72 characters",
		})
	}

	return errs
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (r LoginRequest) Validate() []apperr.FieldError {
	var errs []apperr.FieldError
	if strings.TrimSpace(r.Email) == "" {
		errs = append(errs, apperr.FieldError{
			Field: "email", Message: "must be filled",
		})
	}
	if r.Password == "" {
		errs = append(errs, apperr.FieldError{
			Field: "password", Message: "must be filled",
		})
	}

	return errs
}

type UserResponse struct {
	ID        string    `json:"id"`
	TeamID    string    `json:"team_id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

func NewUserResponse(u *domain.User) UserResponse {
	return UserResponse{
		ID:        u.ID.String(),
		TeamID:    u.TeamID.String(),
		Email:     u.Email,
		Name:      u.Name,
		CreatedAt: u.CreatedAt,
	}
}

type AuthResponse struct {
	Token     string       `json:"token"`
	ExpiresAt time.Time    `json:"expires_at"`
	User      UserResponse `json:"user"`
}
