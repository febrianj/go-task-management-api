package http

import (
	"context"
	"net/http"
	"strings"

	"github.com/febrianj/go-task-management-api/internal/apperr"
	"github.com/febrianj/go-task-management-api/internal/platform/jwt"

	"github.com/google/uuid"
)

const ctxKeyUser ctxKey = 1

type AuthUser struct {
	ID     uuid.UUID
	TeamID uuid.UUID
}

func UserFromContext(ctx context.Context) (AuthUser, bool) {
	u, ok := ctx.Value(ctxKeyUser).(AuthUser)
	return u, ok
}

func Authenticate(issuer *jwt.Issuer) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			token, ok := strings.CutPrefix(header, "Bearer ")
			if !ok || strings.TrimSpace(token) == "" {
				WriteError(w, r, apperr.Unauthorized(
					apperr.CodeUnauthorized, "missing or malformed authentication header"),
				)
				return
			}

			claims, err := issuer.Verify(strings.TrimSpace(token))
			if err != nil {
				WriteError(w, r, apperr.Unauthorized(
					apperr.CodeUnauthorized, "invalid or expired token"))
				return
			}

			ctx := context.WithValue(r.Context(),
				ctxKeyUser, AuthUser{ID: claims.UserID, TeamID: claims.TeamID})

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
