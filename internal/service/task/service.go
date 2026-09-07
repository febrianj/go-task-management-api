package task

import (
	"context"

	"github.com/febrianj/go-task-management-api/internal/domain"
	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, t *domain.Task) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Task, error)
	List(ctx context.Context, f ListFilter) ([]domain.Task, int, error)
	Update(ctx context.Context, t *domain.Task) error
	Delete(ctx context.Context, id uuid.UUID) error
	UpdateAssignee(ctx context.Context, taskID uuid.UUID, assignee uuid.UUID) error
	InsertLog(ctx context.Context, l *domain.TaskLog) error
}

type TxManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

type ListFilter struct {
	UserID  uuid.UUID
	Status  *domain.Status
	Search  string
	Page    int
	Limit   int
	SortBy  string
	SortDir string
}

func (f *ListFilter) Normalize() {
	if f.Page < 1 {
		f.Page = 1
	}
	switch {
	case f.Limit < 1:
		f.Limit = 20
	case f.Limit > 100:
		f.Limit = 100
	}

	switch f.SortBy {
	case "created_at", "updated_at", "title":
		// allowed
	default:
		f.SortBy = "created_at"
	}

	if f.SortDir != "asc" {
		f.SortDir = "desc"
	}
}

func (f ListFilter) Offset() int { return (f.Page - 1) * f.Limit }

type Service struct {
	repo     Repository
	txm      TxManager
	idem     IdempotencyStore
	users    UserLookup
	notifier Notifier
}

func NewService(r Repository, txm TxManager, idem IdempotencyStore,
	users UserLookup, n Notifier) *Service {
	return &Service{repo: r, txm: txm, idem: idem, users: users, notifier: n}
}

func (s *Service) List(ctx context.Context, f ListFilter) ([]domain.Task, int, error) {
	f.Normalize()
	return s.repo.List(ctx, f)
}

func (s *Service) Create(ctx context.Context, t *domain.Task) error {
	return s.repo.Create(ctx, t)
}

func (s *Service) Get(ctx context.Context, id, userID uuid.UUID) (*domain.Task, error) {
	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !t.VisibleTo(userID) {
		return nil, domain.ErrNotFound
	}
	return t, nil
}

type UpdateInput struct {
	Title       *string
	Description *string
	Status      *domain.Status
}

func (s *Service) Update(ctx context.Context, id, userID uuid.UUID, in UpdateInput) (*domain.Task, error) {
	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !t.VisibleTo(userID) {
		return nil, domain.ErrNotFound
	}
	if !t.MutableBy(userID) {
		return nil, domain.ErrForbidden
	}

	if in.Title != nil {
		t.Title = *in.Title
	}
	if in.Description != nil {
		t.Description = *in.Description
	}
	if in.Status != nil {
		t.Status = *in.Status
	}

	if err := s.repo.Update(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *Service) Delete(ctx context.Context, id, userID uuid.UUID) error {
	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if !t.VisibleTo(userID) {
		return domain.ErrNotFound
	}
	if !t.MutableBy(userID) {
		return domain.ErrForbidden
	}
	return s.repo.Delete(ctx, id)
}

type IdemKey struct {
	UserID      uuid.UUID
	Endpoint    string
	Key         uuid.UUID
	RequestHash string
}

type IdemRecord struct {
	ID           int64
	State        string // "in_progress" | "completed"
	RequestHash  string
	ResponseCode int
	ResponseBody []byte
}

const (
	IdemStateInProgress = "in_progress"
	IdemStateCompleted  = "completed"
)

type IdempotencyStore interface {
	Claim(ctx context.Context, k IdemKey) (rec *IdemRecord, claimed bool, err error)
	Complete(ctx context.Context, id int64, code int, body []byte) error
}

type UserLookup interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
}

type Notifier interface {
	Notify(ctx context.Context, n Notification) error
}

type Notification struct {
	UserID  uuid.UUID
	Subject string
	Message string
}
