package task

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/febrianj/go-task-management-api/internal/domain"
	"github.com/google/uuid"
)

func (s *Service) Assign(ctx context.Context, taskID, assigneeID, actorID uuid.UUID) (*domain.Task, error) {
	t, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if !t.VisibleTo(actorID) {
		return nil, domain.ErrNotFound // do not reveal existence
	}
	if !t.MutableBy(actorID) {
		return nil, domain.ErrForbidden
	}

	assignee, err := s.users.GetByID(ctx, assigneeID)
	if err != nil {
		return nil, err
	}
	if assignee.TeamID != t.TeamID {
		return nil, domain.ErrForbidden // cross-team assignment
	}

	from, _ := json.Marshal(map[string]any{"assignee_id": t.AssigneeID})
	to, _ := json.Marshal(map[string]any{"assignee_id": assigneeID})

	err = s.txm.Do(ctx, func(ctx context.Context) error {
		if err := s.repo.UpdateAssignee(ctx, taskID, assigneeID); err != nil {
			return err
		}

		if err := s.repo.InsertLog(ctx, &domain.TaskLog{
			TaskID:    taskID,
			ActorID:   actorID,
			Action:    domain.ActionAssigned,
			FromValue: from,
			ToValue:   to,
		}); err != nil {
			return err
		}

		if err := s.notifier.Notify(ctx, Notification{
			UserID:  assigneeID,
			Subject: "Task assigned",
			Message: fmt.Sprintf("You have been assigned task %q", t.Title),
		}); err != nil {
			return fmt.Errorf("notify assignee: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	t.AssigneeID = &assigneeID
	return t, nil
}
