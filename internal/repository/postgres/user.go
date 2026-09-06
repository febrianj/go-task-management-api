package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/febrianj/go-task-management-api/internal/domain"
	"github.com/febrianj/go-task-management-api/internal/service/auth"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ auth.Repository = (*UserRepo)(nil)

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo { return &UserRepo{pool: pool} }

// CreateWithTeam joins an existing team or creates it, then insert user
func (r *UserRepo) CreateWithTeam(ctx context.Context, u *domain.User, teamName string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx)

	var teamID string
	err = tx.QueryRow(ctx, `
		INSERT INTO teams(name) VALUES ($1)
		ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
		RETURNING id`, teamName).Scan(&teamID)
	if err != nil {
		return fmt.Errorf("upsert team: %w", err)
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO users(team_id, email, name, password_hash)
		VALUES ($1, $2, $3, $4)
		RETURNING id, team_id, created_at`,
		teamID, u.Email, u.Name, u.PasswordHash,
	).Scan(&u.ID, &u.TeamID, &u.CreatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrEmailTaken
		}
		return fmt.Errorf("insert user: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var u domain.User
	err := r.pool.QueryRow(ctx, `
		SELECT id, team_id, email, password_hash, created_at
		FROM users WHERE email = $1`, email,
	).Scan(&u.ID, &u.TeamID, &u.Email, &u.PasswordHash, &u.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}

	return &u, nil
}
