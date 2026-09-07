package task

import (
	"context"
	"errors"
	"testing"

	"github.com/febrianj/go-task-management-api/internal/domain"
	"github.com/google/uuid"
)

type assignFixture struct {
	svc     *Service
	repo    *fakeRepo
	notif   *fakeNotifier
	users   *fakeUsers
	task    *domain.Task
	creator *domain.User
	mate    *domain.User
}

func setupAssign(t *testing.T) assignFixture {
	t.Helper()

	teamID := uuid.New()
	creator := &domain.User{ID: uuid.New(), TeamID: teamID, Email: "a@x.com"}
	mate := &domain.User{ID: uuid.New(), TeamID: teamID, Email: "b@x.com"}

	repo := newFakeRepo()
	task := &domain.Task{ID: uuid.New(), TeamID: teamID,
		CreatedBy: creator.ID, Title: "t", Status: domain.StatusPending}
	repo.tasks[task.ID] = task

	users := &fakeUsers{users: map[uuid.UUID]*domain.User{
		creator.ID: creator, mate.ID: mate}}
	notif := &fakeNotifier{}

	svc := NewService(repo, snapshotTx{repo: repo}, newFakeIdemStore(), users, notif)

	return assignFixture{
		svc: svc, repo: repo, notif: notif, users: users,
		task: task, creator: creator, mate: mate,
	}
}

func TestAssign_Success(t *testing.T) {
	f := setupAssign(t)

	if _, err := f.svc.Assign(context.Background(), f.task.ID, f.mate.ID, f.creator.ID); err != nil {
		t.Fatalf("assign: %v", err)
	}
	if f.repo.logCount() != 1 {
		t.Errorf("audit log rows: got %d want 1", f.repo.logCount())
	}
	if len(f.notif.sent) != 1 {
		t.Errorf("notifications: got %d want 1", len(f.notif.sent))
	}
}

// if any step fails, nothing persists.
func TestAssign_NotifierFailureRollsBack(t *testing.T) {
	f := setupAssign(t)
	f.notif.err = errors.New("notification service unavailable")

	_, err := f.svc.Assign(context.Background(), f.task.ID, f.mate.ID, f.creator.ID)
	if err == nil {
		t.Fatal("assign must fail when the notifier fails")
	}
	if f.repo.logCount() != 0 {
		t.Errorf("audit log must be rolled back: got %d rows", f.repo.logCount())
	}
	after, _ := f.repo.GetByID(context.Background(), f.task.ID)
	if after.AssigneeID != nil {
		t.Errorf("assignee must be rolled back: got %v", *after.AssigneeID)
	}
}

func TestAssign_DifferentTeamForbidden(t *testing.T) {
	f := setupAssign(t)

	outsider := &domain.User{ID: uuid.New(), TeamID: uuid.New()}
	f.users.users[outsider.ID] = outsider

	_, err := f.svc.Assign(context.Background(), f.task.ID, outsider.ID, f.creator.ID)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("cross-team assign must be forbidden: got %v", err)
	}
}

func TestAssign_InvisibleTaskNotFound(t *testing.T) {
	f := setupAssign(t)

	_, err := f.svc.Assign(context.Background(), f.task.ID, f.mate.ID, f.mate.ID)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("invisible task must return not found: got %v", err)
	}
}

// A user who CAN see the task (as assignee) but did not create it gets 403.
func TestAssign_AssigneeCannotReassign(t *testing.T) {
	f := setupAssign(t)

	// make mate the current assignee → now visible to them
	f.repo.tasks[f.task.ID].AssigneeID = &f.mate.ID

	_, err := f.svc.Assign(context.Background(), f.task.ID, f.creator.ID, f.mate.ID)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("non-creator must not reassign: got %v", err)
	}
}
