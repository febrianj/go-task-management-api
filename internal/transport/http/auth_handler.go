package http

import (
	"errors"
	"net/http"

	"github.com/febrianj/go-task-management-api/internal/apperr"
	"github.com/febrianj/go-task-management-api/internal/domain"
	"github.com/febrianj/go-task-management-api/internal/service/auth"
)

type AuthHandler struct {
	svc *auth.Service
}

func NewAuthHandler(s *auth.Service) *AuthHandler { return &AuthHandler{svc: s} }

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := decodeAndValidate(w, r, &req); err != nil {
		WriteError(w, r, err)
		return
	}

	res, err := h.svc.Register(r.Context(), auth.RegisterInput{
		Email:    req.Email,
		Name:     req.Name,
		Password: req.Password,
		TeamName: req.TeamName,
	})
	if err != nil {
		WriteError(w, r, mapAuthError(err))
		return
	}

	WriteSuccess(w, http.StatusCreated, AuthResponse{
		Token:     res.Token,
		ExpiresAt: res.ExpiresAt,
		User:      NewUserResponse(res.User),
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := decodeAndValidate(w, r, &req); err != nil {
		WriteError(w, r, err)
		return
	}

	res, err := h.svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		WriteError(w, r, err)
		return
	}

	WriteSuccess(w, http.StatusOK, AuthResponse{
		Token:     res.Token,
		ExpiresAt: res.ExpiresAt,
		User:      NewUserResponse(res.User),
	})
}

func mapAuthError(err error) error {
	switch {
	case errors.Is(err, domain.ErrEmailTaken):
		return apperr.Conflict(apperr.CodeEmailAlreadyExists, "email already registered")
	case errors.Is(err, domain.ErrInvalidCredentials):
		return apperr.Unauthorized(apperr.CodeInvalidCredentials, "invalid email or password")
	default:
		return err
	}
}
