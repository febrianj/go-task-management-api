package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/febrianj/go-task-management-api/internal/apperr"
)

type successEnvelope struct {
	Status string `json:"status"`
	Data   any    `json:"data"`
}

type errorEnvelope struct {
	Status    string              `json:"status"`
	Code      string              `json:"code"`
	Message   string              `json:"message"`
	Timestamp string              `json:"timestamp"`
	RequestID string              `json:"request_id,omitempty"`
	Details   []apperr.FieldError `json:"details,omitempty"`
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

func WriteSuccess(w http.ResponseWriter, status int, data any) {
	WriteJSON(w, status, successEnvelope{Status: "success", Data: data})
}

func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	var appErr *apperr.AppError
	if !errors.As(err, &appErr) {
		appErr = apperr.Internal(err)
	}

	WriteJSON(w, appErr.HTTPStatus, errorEnvelope{
		Status:    "error",
		Code:      appErr.Code,
		Message:   appErr.Message,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		RequestID: RequestIDFromContext(r.Context()),
		Details:   appErr.Details,
	})
}
