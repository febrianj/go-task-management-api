package http

import (
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Deps struct {
	Pool *pgxpool.Pool
	Log  *slog.Logger
}

func NewRouter(d Deps) http.Handler {
	mux := http.NewServeMux()

	// Liveness, is service up, no dependency checks
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Readiness, check db connection
	mux.HandleFunc("GET /ready", func(w http.ResponseWriter, r *http.Request) {
		if err := d.Pool.Ping(r.Context()); err != nil {
			WriteJSON(w, http.StatusServiceUnavailable, map[string]string{
				"status": "unavailable",
				"reason": "database",
			})
			return
		}
		WriteJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})

	mux.HandleFunc("GET /panic", func(w http.ResponseWriter, r *http.Request) {
		panic("trigger panic")
	})
	return Chain(mux, RequestID, Recovery(d.Log), Logging(d.Log))
}
