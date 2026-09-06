package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/febrianj/go-task-management-api/internal/domain"
	"github.com/google/uuid"
)

type Repository interface {
	CreateWithTeam(ctx context.Context, u *domain.User, teamName string) error
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
}

type Hasher interface {
	Hash(plain string) (string, error)
	Compare(hashed, plain string) error
}

type TokenIssuer interface {
	Issue(userID, teamID uuid.UUID) (string, time.Time, error)
}

type Service struct {
	repo   Repository
	hasher Hasher
	token  TokenIssuer
}

func NewService(r Repository, h Hasher, t TokenIssuer) *Service {
	return &Service{repo: r, hasher: h, token: t}
}

type RegisterInput struct {
	Email    string
	Name     string
	Password string
	TeamName string
}

type Result struct {
	User      *domain.User
	Token     string
	ExpiresAt time.Time
}

func (s *Service) Register(ctx context.Context, in RegisterInput) (*Result, error) {
	email := normalizeEmail(in.Email)

	hashed, err := s.hasher.Hash(in.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &domain.User{
		Name:         in.Name,
		Email:        email,
		PasswordHash: hashed,
	}

	teamName := strings.TrimSpace(in.TeamName)
	if teamName == "" {
		teamName = email + "'s Team"
	}

	if err := s.repo.CreateWithTeam(ctx, user, teamName); err != nil {
		return nil, err
	}

	return s.issue(user)
}

func (s *Service) Login(ctx context.Context, email, password string) (*Result, error) {
	user, err := s.repo.GetByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			//
			_ = s.hasher.Compare(dummyHash, password)
			return nil, domain.ErrInvalidCredentials
		}
	}

	if err := s.hasher.Compare(user.PasswordHash, password); err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	return s.issue(user)
}

func (s *Service) issue(u *domain.User) (*Result, error) {
	token, expiresAt, err := s.token.Issue(u.ID, u.TeamID)
	if err != nil {
		return nil, fmt.Errorf("issue token: %w", err)
	}

	return &Result{User: u, Token: token, ExpiresAt: expiresAt}, nil
}

func normalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// valid bcrypt value, used only to equalise timing on the unknown-email path.
const dummyHash = "$2a$12$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"
