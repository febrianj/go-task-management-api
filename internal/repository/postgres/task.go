package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/febrianj/go-task-management-api/internal/domain"
	"github.com/febrianj/go-task-management-api/internal/service/task"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ task.Repository = (*TaskRepo)(nil)

type TaskRepo struct {
	pool *pgxpool.Pool
}

func NewTaskRepo(pool *pgxpool.Pool) *TaskRepo {
	return &TaskRepo{pool: pool}
}

func (r *TaskRepo) Create(ctx context.Context, t *domain.Task) error {
	err := conn(ctx, r.pool).QueryRow(ctx, `
		INSERT INTO tasks (team_id, created_by, assignee_id, title, description, status)
		VALUES($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`,
		t.TeamID, t.CreatedBy, t.AssigneeID, t.Title, t.Description, t.Status,
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)

	if err != nil {
		return fmt.Errorf("insert task: %w", err)
	}
	return nil
}

func (r *TaskRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Task, error) {
	var t domain.Task
	err := conn(ctx, r.pool).QueryRow(ctx, `
		SELECT id, team_id, created_by, assignee_id, title, description, 
		status, created_at, updated_at 
		FROM tasks WHERE id = $1`, id,
	).Scan(&t.ID, &t.TeamID, &t.CreatedBy, &t.AssigneeID, &t.Title,
		&t.Description, &t.Status, &t.CreatedAt, &t.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get task: %w", err)
	}

	return &t, nil
}

func (r *TaskRepo) Update(ctx context.Context, t *domain.Task) error {
	tag, err := conn(ctx, r.pool).Exec(ctx, `
		UPDATE tasks
		SET title = $1, description = $2, status = $3,
			assignee_id = $4, updated_at = now()
		WHERE id = $5`,
		t.Title, t.Description, t.Status, t.AssigneeID, t.ID)

	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *TaskRepo) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := conn(ctx, r.pool).Exec(ctx, `DELETE FROM tasks WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *TaskRepo) List(ctx context.Context, f task.ListFilter) ([]domain.Task, int, error) {
	query := fmt.Sprintf(`
		SELECT id, team_id, created_by, assignee_id, title, description,
			status, created_at, updated_at,
			COUNT(*) OVER() AS total_count 
		FROM tasks
		WHERE (created_by = $1 OR assignee_id = $1)
			AND ($2::text IS NULL OR status = $2)
			AND ($3::text IS NULL OR title ilike '%%' || $3 || '%%')
		ORDER BY %s %s, id DESC
		LIMIT $4 OFFSET $5`, f.SortBy, f.SortDir)

	var status *string
	if f.Status != nil {
		s := string(*f.Status)
		status = &s
	}

	var search *string
	if f.Search != "" {
		escaped := escapeLike(f.Search)
		search = &escaped
	}

	rows, err := conn(ctx, r.pool).Query(ctx, query,
		f.UserID, status, search, f.Limit, f.Offset())
	if err != nil {
		return nil, 0, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	tasks := make([]domain.Task, 0, f.Limit)
	total := 0

	for rows.Next() {
		var t domain.Task
		if err := rows.Scan(&t.ID, &t.TeamID, &t.CreatedBy, &t.AssigneeID,
			&t.Title, &t.Description, &t.Status,
			&t.CreatedAt, &t.UpdatedAt, &total); err != nil {
			return nil, 0, fmt.Errorf("scan task: %w", err)
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate tasks: %w", err)
	}
	return tasks, total, nil
}

func (r *TaskRepo) UpdateAssignee(ctx context.Context, taskID, assignee uuid.UUID) error {
	tag, err := conn(ctx, r.pool).Exec(ctx, `
         UPDATE tasks SET assignee_id = $2, updated_at = now()
         WHERE id = $1`, taskID, assignee)
	if err != nil {
		return fmt.Errorf("update assignee: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *TaskRepo) InsertLog(ctx context.Context, l *domain.TaskLog) error {
	err := conn(ctx, r.pool).QueryRow(ctx, `
         INSERT INTO task_logs (task_id, actor_id, action, from_value, to_value)
         VALUES ($1, $2, $3, $4, $5)
         RETURNING id, created_at`,
		l.TaskID, l.ActorID, l.Action, l.FromValue, l.ToValue,
	).Scan(&l.ID, &l.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert task log: %w", err)
	}
	return nil
}

// neutralises LIKE wildcards
func escapeLike(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(s)
}
