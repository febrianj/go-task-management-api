package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/febrianj/go-task-management-api/internal/service/task"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ task.IdempotencyStore = (*IdemStore)(nil)

type IdemStore struct {
	pool *pgxpool.Pool
}

func NewIdemStore(pool *pgxpool.Pool) *IdemStore {
	return &IdemStore{pool: pool}
}

func (s *IdemStore) Claim(ctx context.Context, k task.IdemKey) (*task.IdemRecord, bool, error) {
	q := conn(ctx, s.pool)

	if _, err := q.Exec(ctx, `
        DELETE FROM idempotency_keys
        WHERE user_id = $1 AND endpoint = $2 AND idem_key = $3
           AND expires_at < now()`,
		k.UserID, k.Endpoint, k.Key); err != nil {
		return nil, false, fmt.Errorf("purge expired key: %w", err)
	}

	var id int64
	err := q.QueryRow(ctx, `
		INSERT INTO idempotency_keys (user_id, endpoint, idem_key, request_hash, state)
		VALUES ($1, $2, $3, $4, 'in_progress')
		ON CONFLICT ON CONSTRAINT idem_unique DO NOTHING
		RETURNING id`,
		k.UserID, k.Endpoint, k.Key, k.RequestHash).Scan(&id)

	if err == nil {
		return &task.IdemRecord{
			ID:          id,
			State:       task.IdemStateInProgress,
			RequestHash: k.RequestHash,
		}, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, false, fmt.Errorf("claim idempotency key: %w", err)
	}

	var rec task.IdemRecord
	var code *int
	var body []byte

	err = q.QueryRow(ctx, `
		SELECT id, state, request_hash, response_code, response_body
		FROM idempotency_keys
		WHERE user_id = $1 AND endpoint = $2 AND idem_key = $3`,
		k.UserID, k.Endpoint, k.Key,
	).Scan(&rec.ID, &rec.State, &rec.RequestHash, &code, &body)

	if err != nil {
		return nil, false, fmt.Errorf("read existing idempotency key: %w", err)
	}
	if code != nil {
		rec.ResponseCode = *code
	}
	rec.ResponseBody = body

	return &rec, false, nil
}

func (s *IdemStore) Complete(ctx context.Context, id int64, code int, body []byte) error {
	_, err := conn(ctx, s.pool).Exec(ctx, `
		UPDATE idempotency_keys
		SET state = 'completed', response_code = $2, response_body = $3
		WHERE id = $1`, id, code, body)
	if err != nil {
		return fmt.Errorf("complete idempotency key: %w", err)
	}
	return nil
}
