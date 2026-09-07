package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/febrianj/go-task-management-api/internal/apperr"
	"github.com/febrianj/go-task-management-api/internal/domain"
	"github.com/febrianj/go-task-management-api/internal/service/task"
	"github.com/google/uuid"
)

type TaskHandler struct {
	svc *task.Service
}

func NewTaskHandler(s *task.Service) *TaskHandler {
	return &TaskHandler{svc: s}
}

func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	user, _ := UserFromContext(r.Context())

	rawKey := r.Header.Get("Idempotency-Key")
	if rawKey == "" {
		WriteError(w, r, apperr.New(http.StatusBadRequest,
			apperr.CodeInvalidIdempotencyKey,
			"Idempotency-Key header is required"))
		return
	}
	idemKey, err := uuid.Parse(rawKey)
	if err != nil {
		WriteError(w, r, apperr.New(http.StatusBadRequest,
			apperr.CodeInvalidIdempotencyKey,
			"Idempotency-Key header must be a valid UUID"))
		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	if err != nil {
		WriteError(w, r, apperr.BadRequest("request body too large or unreadable"))
		return
	}

	var req CreateTaskRequest
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		WriteError(w, r, apperr.BadRequest("malformed JSON body").WithErr(err))
		return
	}
	if details := req.Validate(); len(details) > 0 {
		WriteError(w, r, apperr.Validation(details))
		return
	}

	status := domain.StatusPending
	if req.Status != "" {
		status, _ = domain.ParseStatus(req.Status)
	}

	var assignee *uuid.UUID
	if req.AssigneeID != nil {
		id := uuid.MustParse(*req.AssigneeID)
		assignee = &id
	}

	t := &domain.Task{
		TeamID:      user.TeamID,
		CreatedBy:   user.ID,
		AssigneeID:  assignee,
		Title:       strings.TrimSpace(req.Title),
		Description: req.Description,
		Status:      status,
	}

	render := func(created *domain.Task) ([]byte, error) {
		return json.Marshal(successEnvelope{
			Status: "success",
			Data:   NewTaskResponse(created),
		})
	}

	res, err := h.svc.CreateIdempotent(r.Context(), task.IdemKey{
		UserID:      user.ID,
		Endpoint:    "POST /tasks",
		Key:         idemKey,
		RequestHash: task.HashRequest(body),
	}, t, render)

	if err != nil {
		WriteError(w, r, mapTaskError(err))
		return
	}

	if res.Replayed {
		w.Header().Set("Idempotency-Replayed", "true")
	} else if res.Task != nil {
		w.Header().Set("Location", "/tasks/"+res.Task.ID.String())
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(res.StatusCode)
	_, _ = w.Write(res.Body)
}

func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	user, _ := UserFromContext(r.Context())
	q := r.URL.Query()

	f := task.ListFilter{
		UserID:  user.ID,
		Search:  q.Get("search"),
		Page:    atoiDefault(q.Get("page"), 1),
		Limit:   atoiDefault(q.Get("limit"), 20),
		SortBy:  q.Get("sort_by"),
		SortDir: q.Get("sort_dir"),
	}

	if s := q.Get("status"); s != "" {
		parsed, ok := domain.ParseStatus(s)
		if !ok {
			WriteError(w, r, apperr.BadRequest(
				"status must be one of: pending, in_progress, done, cancelled"))
			return
		}
		f.Status = &parsed
	}

	tasks, total, err := h.svc.List(r.Context(), f)
	if err != nil {
		WriteError(w, r, mapTaskError(err))
		return
	}

	items := make([]TaskResponse, 0, len(tasks))
	for i := range tasks {
		items = append(items, NewTaskResponse(&tasks[i]))
	}

	f.Normalize()
	WriteSuccessMeta(w, http.StatusOK, items, PageMeta{
		Page:       f.Page,
		Limit:      f.Limit,
		Total:      total,
		TotalPages: (total + f.Limit - 1) / f.Limit, // ceiling division
	})
}

func (h *TaskHandler) Get(w http.ResponseWriter, r *http.Request) {
	user, _ := UserFromContext(r.Context())

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		WriteError(w, r, apperr.BadRequest("task id must be a valid UUID"))
		return
	}

	t, err := h.svc.Get(r.Context(), id, user.ID)
	if err != nil {
		WriteError(w, r, mapTaskError(err))
		return
	}
	WriteSuccess(w, http.StatusOK, NewTaskResponse(t))
}

func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	user, _ := UserFromContext(r.Context())

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		WriteError(w, r, apperr.BadRequest("task id must be a valid UUID"))
		return
	}

	var req UpdateTaskRequest
	if err := decodeAndValidate(w, r, &req); err != nil {
		WriteError(w, r, err)
		return
	}

	in := task.UpdateInput{Title: req.Title, Description: req.Description}
	if req.Status != nil {
		s, _ := domain.ParseStatus(*req.Status)
		in.Status = &s
	}

	t, err := h.svc.Update(r.Context(), id, user.ID, in)
	if err != nil {
		WriteError(w, r, mapTaskError(err))
		return
	}
	WriteSuccess(w, http.StatusOK, NewTaskResponse(t))
}

func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user, _ := UserFromContext(r.Context())

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		WriteError(w, r, apperr.BadRequest("task id must be a valid UUID"))
		return
	}

	if err := h.svc.Delete(r.Context(), id, user.ID); err != nil {
		WriteError(w, r, mapTaskError(err))
		return
	}
	w.WriteHeader(http.StatusNoContent) // 204: no body
}

func mapTaskError(err error) error {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return apperr.NotFound(apperr.CodeTaskNotFound, "task not found")
	case errors.Is(err, domain.ErrForbidden):
		return apperr.Forbidden("only the task creator may perform this action")
	default:
		return err
	}
}

func (h *TaskHandler) Assign(w http.ResponseWriter, r *http.Request) {
	user, _ := UserFromContext(r.Context())

	taskID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		WriteError(w, r, apperr.BadRequest("task id must be a valid UUID"))
		return
	}

	var req AssignTaskRequest
	if err := decodeAndValidate(w, r, &req); err != nil {
		WriteError(w, r, err)
		return
	}
	assigneeID := uuid.MustParse(strings.TrimSpace(req.AssigneeID)) // validated above

	t, err := h.svc.Assign(r.Context(), taskID, assigneeID, user.ID)
	if err != nil {
		WriteError(w, r, mapTaskError(err))
		return
	}
	WriteSuccess(w, http.StatusOK, NewTaskResponse(t))
}

func atoiDefault(s string, def int) int {
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return def
}
