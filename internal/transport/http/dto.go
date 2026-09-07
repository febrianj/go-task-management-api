package http

import (
	"net/mail"
	"strings"
	"time"

	"github.com/febrianj/go-task-management-api/internal/apperr"
	"github.com/febrianj/go-task-management-api/internal/domain"
	"github.com/google/uuid"
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

type CreateTaskRequest struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Status      string  `json:"status"`
	AssigneeID  *string `json:"assignee_id"`
}

func (r CreateTaskRequest) Validate() []apperr.FieldError {
	var errs []apperr.FieldError

	title := strings.TrimSpace(r.Title)
	if title == "" {
		errs = append(errs, apperr.FieldError{
			Field: "title", Message: "must not be empty"})
	}
	if len(title) > 100 {
		errs = append(errs, apperr.FieldError{
			Field: "title", Message: "must be at most 100 characters"})
	}
	if r.Status != "" {
		if _, ok := domain.ParseStatus(r.Status); !ok {
			errs = append(errs, apperr.FieldError{
				Field:   "status",
				Message: "must be one of: pending, in_progress, done, cancelled"})
		}
	}
	if r.AssigneeID != nil {
		if _, err := uuid.Parse(*r.AssigneeID); err != nil {
			errs = append(errs, apperr.FieldError{
				Field: "assignee_id", Message: "must be a valid UUID"})
		}
	}
	return errs
}

type UpdateTaskRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
}

func (r UpdateTaskRequest) Validate() []apperr.FieldError {
	var errs []apperr.FieldError

	if r.Title != nil {
		title := strings.TrimSpace(*r.Title)
		if title == "" {
			errs = append(errs, apperr.FieldError{
				Field: "title", Message: "must not be empty"})
		}
		if len(title) > 100 {
			errs = append(errs, apperr.FieldError{
				Field: "title", Message: "must be at most 100 characters"})
		}
	}
	if r.Status != nil {
		if _, ok := domain.ParseStatus(*r.Status); !ok {
			errs = append(errs, apperr.FieldError{
				Field:   "status",
				Message: "must be one of: pending, in_progress, done, cancelled"})
		}
	}
	return errs
}

type TaskResponse struct {
	ID          string    `json:"id"`
	TeamID      string    `json:"team_id"`
	CreatedBy   string    `json:"created_by"`
	AssigneeID  *string   `json:"assignee_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func NewTaskResponse(t *domain.Task) TaskResponse {
	var assignee *string
	if t.AssigneeID != nil {
		s := t.AssigneeID.String()
		assignee = &s
	}
	return TaskResponse{
		ID:          t.ID.String(),
		TeamID:      t.TeamID.String(),
		CreatedBy:   t.CreatedBy.String(),
		AssigneeID:  assignee,
		Title:       t.Title,
		Description: t.Description,
		Status:      string(t.Status),
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}

type AssignTaskRequest struct {
	AssigneeID string `json:"assignee_id"`
}

func (r AssignTaskRequest) Validate() []apperr.FieldError {
	var errs []apperr.FieldError
	if _, err := uuid.Parse(strings.TrimSpace(r.AssigneeID)); err != nil {
		errs = append(errs, apperr.FieldError{
			Field: "assignee_id", Message: "must be a valid UUID"})
	}
	return errs
}

type PageMeta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}
