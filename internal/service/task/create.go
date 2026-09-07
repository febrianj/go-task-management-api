package task

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/febrianj/go-task-management-api/internal/apperr"
	"github.com/febrianj/go-task-management-api/internal/domain"
)

type CreateResult struct {
	Task       *domain.Task
	StatusCode int
	Body       []byte
	Replayed   bool
}

func HashRequest(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func (s *Service) CreateIdempotent(
	ctx context.Context,
	key IdemKey,
	t *domain.Task,
	render func(*domain.Task) ([]byte, error),
) (*CreateResult, error) {

	var result *CreateResult

	err := s.txm.Do(ctx, func(ctx context.Context) error {
		rec, claimed, err := s.idem.Claim(ctx, key)
		if err != nil {
			return err
		}

		if !claimed {
			if rec.RequestHash != key.RequestHash {
				return apperr.Unprocessable(
					apperr.CodeIdempotencyKeyReused,
					"idempotency key was already used with a different request body")
			}

			if rec.State == IdemStateCompleted {
				result = &CreateResult{
					StatusCode: rec.ResponseCode,
					Body:       rec.ResponseBody,
					Replayed:   true,
				}
				return nil
			}
			return apperr.Conflict(
				apperr.CodeConcurrentRequest,
				"a request with this idempotency key is currently in progress")
		}
		if err := s.repo.Create(ctx, t); err != nil {
			return err
		}

		body, err := render(t)
		if err != nil {
			return fmt.Errorf("render response: %w", err)
		}

		if err := s.idem.Complete(ctx, rec.ID, 201, body); err != nil {
			return err
		}

		result = &CreateResult{
			Task:       t,
			StatusCode: 201,
			Body:       body,
			Replayed:   false,
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

func MarshalJSON(v any) ([]byte, error) {
	return json.Marshal(v)
}
