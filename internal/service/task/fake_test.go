package task

import (
	"context"
	"errors"
	"sync"

	"github.com/febrianj/go-task-management-api/internal/domain"
	"github.com/google/uuid"
)

type fakeRepo struct {
	mu        sync.Mutex
	tasks     map[uuid.UUID]*domain.Task
	logs      []*domain.TaskLog
	createErr error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{tasks: make(map[uuid.UUID]*domain.Task)}
}

func (f *fakeRepo) Create(ctx context.Context, t *domain.Task) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createErr != nil {
		return f.createErr
	}
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	cp := *t
	f.tasks[t.ID] = &cp
	return nil
}

func (f *fakeRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Task, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.tasks[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *t
	return &cp, nil
}

func (f *fakeRepo) List(ctx context.Context, _ ListFilter) ([]domain.Task, int, error) {
	return nil, 0, nil
}

func (f *fakeRepo) Update(ctx context.Context, t *domain.Task) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.tasks[t.ID]; !ok {
		return domain.ErrNotFound
	}
	cp := *t
	f.tasks[t.ID] = &cp
	return nil
}

func (f *fakeRepo) Delete(ctx context.Context, id uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.tasks, id)
	return nil
}

func (f *fakeRepo) UpdateAssignee(ctx context.Context, taskID, assignee uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.tasks[taskID]
	if !ok {
		return domain.ErrNotFound
	}
	t.AssigneeID = &assignee
	return nil
}

func (f *fakeRepo) InsertLog(ctx context.Context, l *domain.TaskLog) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := *l
	f.logs = append(f.logs, &cp)
	return nil
}

func (f *fakeRepo) taskCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.tasks)
}

func (f *fakeRepo) logCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.logs)
}

type fakeIdemStore struct {
	mu      sync.Mutex
	records map[string]*IdemRecord
	nextID  int64
}

func newFakeIdemStore() *fakeIdemStore {
	return &fakeIdemStore{records: make(map[string]*IdemRecord)}
}

func idemMapKey(k IdemKey) string {
	return k.UserID.String() + "|" + k.Endpoint + "|" + k.Key.String()
}
func (f *fakeIdemStore) Claim(ctx context.Context, k IdemKey) (*IdemRecord, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	mk := idemMapKey(k)
	if existing, ok := f.records[mk]; ok {
		cp := *existing
		return &cp, false, nil
	}

	f.nextID++
	rec := &IdemRecord{
		ID:          f.nextID,
		State:       IdemStateInProgress,
		RequestHash: k.RequestHash,
	}
	f.records[mk] = rec
	cp := *rec
	return &cp, true, nil
}

func (f *fakeIdemStore) Complete(ctx context.Context, id int64, code int, body []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, r := range f.records {
		if r.ID == id {
			r.State = IdemStateCompleted
			r.ResponseCode = code
			r.ResponseBody = append([]byte(nil), body...)
			return nil
		}
	}
	return errors.New("idempotency record not found")
}

// runs fn directly for concurrency test
type fakeTx struct{}

func (fakeTx) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

// simulates rollback for SINGLE-GOROUTINE tests by restoring the repo if fn fails.
type snapshotTx struct{ repo *fakeRepo }

func (s snapshotTx) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	s.repo.mu.Lock()
	logsBefore := len(s.repo.logs)
	tasksBefore := make(map[uuid.UUID]domain.Task, len(s.repo.tasks))
	for k, v := range s.repo.tasks {
		tasksBefore[k] = *v
	}
	s.repo.mu.Unlock()

	if err := fn(ctx); err != nil {
		s.repo.mu.Lock()
		s.repo.logs = s.repo.logs[:logsBefore]
		s.repo.tasks = make(map[uuid.UUID]*domain.Task, len(tasksBefore))
		for k, v := range tasksBefore {
			cp := v
			s.repo.tasks[k] = &cp
		}
		s.repo.mu.Unlock()
		return err
	}
	return nil
}

type fakeUsers struct {
	users map[uuid.UUID]*domain.User
}

func (f *fakeUsers) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	u, ok := f.users[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return u, nil
}

type fakeNotifier struct {
	mu   sync.Mutex
	sent []Notification
	err  error
}

func (f *fakeNotifier) Notify(ctx context.Context, n Notification) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	f.sent = append(f.sent, n)
	return nil
}
