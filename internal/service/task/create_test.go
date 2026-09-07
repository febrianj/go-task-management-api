package task

import (
	"bytes"
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/febrianj/go-task-management-api/internal/domain"
	"github.com/google/uuid"
)

func newTestService() (*Service, *fakeRepo, *fakeIdemStore) {
	repo := newFakeRepo()
	idem := newFakeIdemStore()
	svc := NewService(repo, fakeTx{}, idem, &fakeUsers{}, &fakeNotifier{})
	return svc, repo, idem
}

func renderTask(t *domain.Task) ([]byte, error) {
	return json.Marshal(map[string]any{"id": t.ID.String(), "title": t.Title})
}

// TestCreateIdempotent_Sequential: a repeated request with the same key
// creates no second task and returns an identical response.
func TestCreateIdempotent_Sequential(t *testing.T) {
	svc, repo, _ := newTestService()
	ctx := context.Background()

	userID, teamID := uuid.New(), uuid.New()
	key := IdemKey{
		UserID:      userID,
		Endpoint:    "POST /tasks",
		Key:         uuid.New(),
		RequestHash: HashRequest([]byte(`{"title":"first"}`)),
	}

	newTask := func() *domain.Task {
		return &domain.Task{TeamID: teamID, CreatedBy: userID,
			Title: "first", Status: domain.StatusPending}
	}

	first, err := svc.CreateIdempotent(ctx, key, newTask(), renderTask)
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	if first.Replayed {
		t.Error("first request must not be a replay")
	}
	if first.StatusCode != 201 {
		t.Errorf("status: got %d want 201", first.StatusCode)
	}

	second, err := svc.CreateIdempotent(ctx, key, newTask(), renderTask)
	if err != nil {
		t.Fatalf("second create: %v", err)
	}
	if !second.Replayed {
		t.Error("second request must be a replay")
	}
	if !bytes.Equal(first.Body, second.Body) {
		t.Errorf("responses differ:\n first: %s\nsecond: %s", first.Body, second.Body)
	}
	if got := repo.taskCount(); got != 1 {
		t.Errorf("task count: got %d want 1", got)
	}
}

func TestCreateIdempotent_ConcurrentDuplicate(t *testing.T) {
	const n = 50

	svc, repo, _ := newTestService()
	ctx := context.Background()

	userID, teamID := uuid.New(), uuid.New()
	body := []byte(`{"title":"concurrent"}`)
	key := IdemKey{
		UserID:      userID,
		Endpoint:    "POST /tasks",
		Key:         uuid.New(),
		RequestHash: HashRequest(body),
	}

	var start sync.WaitGroup // start gate: maximise real overlap
	var done sync.WaitGroup
	start.Add(1)

	results := make([]*CreateResult, n)
	errs := make([]error, n)

	for i := 0; i < n; i++ {
		done.Add(1)
		go func(i int) {
			defer done.Done()
			start.Wait() // all goroutines block here

			task := &domain.Task{TeamID: teamID, CreatedBy: userID,
				Title: "concurrent", Status: domain.StatusPending}
			results[i], errs[i] = svc.CreateIdempotent(ctx, key, task, renderTask)
		}(i)
	}

	start.Done() // release them all at once
	done.Wait()

	if got := repo.taskCount(); got != 1 {
		t.Fatalf("EXACTLY ONE task must be created: got %d", got)
	}

	created, replayed, conflicts := 0, 0, 0
	for i := 0; i < n; i++ {
		switch {
		case errs[i] != nil:
			conflicts++
		case results[i].Replayed:
			replayed++
		default:
			created++
		}
	}

	if created != 1 {
		t.Errorf("exactly one caller must create: got %d", created)
	}
	if created+replayed+conflicts != n {
		t.Errorf("accounted %d of %d results", created+replayed+conflicts, n)
	}
	t.Logf("created=%d replayed=%d in-progress-conflicts=%d", created, replayed, conflicts)
}

func TestCreateIdempotent_DistinctKeys(t *testing.T) {
	const n = 20

	svc, repo, _ := newTestService()
	ctx := context.Background()
	userID, teamID := uuid.New(), uuid.New()

	var start, done sync.WaitGroup
	start.Add(1)

	for i := 0; i < n; i++ {
		done.Add(1)
		go func() {
			defer done.Done()
			start.Wait()

			key := IdemKey{UserID: userID, Endpoint: "POST /tasks",
				Key: uuid.New(), RequestHash: HashRequest([]byte(`{}`))}
			task := &domain.Task{TeamID: teamID, CreatedBy: userID,
				Title: "distinct", Status: domain.StatusPending}
			_, _ = svc.CreateIdempotent(ctx, key, task, renderTask)
		}()
	}

	start.Done()
	done.Wait()

	if got := repo.taskCount(); got != n {
		t.Errorf("distinct keys must each create: got %d want %d", got, n)
	}
}

// TestCreateIdempotent_SameKeyDifferentBody must be rejected.
func TestCreateIdempotent_SameKeyDifferentBody(t *testing.T) {
	svc, repo, _ := newTestService()
	ctx := context.Background()
	userID, teamID := uuid.New(), uuid.New()
	idemKey := uuid.New()

	mk := func(body string) (IdemKey, *domain.Task) {
		return IdemKey{UserID: userID, Endpoint: "POST /tasks", Key: idemKey,
				RequestHash: HashRequest([]byte(body))},
			&domain.Task{TeamID: teamID, CreatedBy: userID,
				Title: "x", Status: domain.StatusPending}
	}

	k1, t1 := mk(`{"title":"original"}`)
	if _, err := svc.CreateIdempotent(ctx, k1, t1, renderTask); err != nil {
		t.Fatalf("first create: %v", err)
	}

	k2, t2 := mk(`{"title":"different"}`)
	if _, err := svc.CreateIdempotent(ctx, k2, t2, renderTask); err == nil {
		t.Fatal("reusing a key with a different body must be rejected")
	}

	if got := repo.taskCount(); got != 1 {
		t.Errorf("task count: got %d want 1", got)
	}
}
