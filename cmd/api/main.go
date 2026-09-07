package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/febrianj/go-task-management-api/internal/config"
	"github.com/febrianj/go-task-management-api/internal/platform/hash"
	"github.com/febrianj/go-task-management-api/internal/platform/jwt"
	"github.com/febrianj/go-task-management-api/internal/platform/logger"
	"github.com/febrianj/go-task-management-api/internal/platform/notifier"
	"github.com/febrianj/go-task-management-api/internal/repository/postgres"
	"github.com/febrianj/go-task-management-api/internal/service/auth"
	"github.com/febrianj/go-task-management-api/internal/service/task"
	transport "github.com/febrianj/go-task-management-api/internal/transport/http"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	logg := logger.New(cfg.LogLevel, cfg.IsProduction())
	slog.SetDefault(logg)

	// cancelled on SIGINT and SIGTERM, context is the shutdown trigger
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		logg.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	hasher := hash.NewBcrypt(cfg.BcryptCost)
	issuer := jwt.NewIssuer(cfg.JWTSecret, cfg.JWTTTL)
	userRepo := postgres.NewUserRepo(pool)
	authSvc := auth.NewService(userRepo, hasher, issuer)
	authHandler := transport.NewAuthHandler(authSvc)
	taskRepo := postgres.NewTaskRepo(pool)
	txManager := postgres.NewTxManager(pool)
	idemStore := postgres.NewIdemStore(pool)
	logNotifier := notifier.NewLogNotifier(logg)
	taskSvc := task.NewService(taskRepo, txManager, idemStore,
		userRepo, notifierAdapter{logNotifier})
	taskHandler := transport.NewTaskHandler(taskSvc)

	srv := &http.Server{
		Addr: ":" + cfg.Port,
		Handler: transport.NewRouter(transport.Deps{
			Pool: pool, Log: logg, Auth: authHandler, Issuer: issuer, Task: taskHandler,
		}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Serve in a goroutine, main can wait on ctx.Done()
	errCh := make(chan error, 1)
	go func() {
		logg.Info("server start", "port", cfg.Port, "env", cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		logg.Error("server failed", "error", err)
		os.Exit(1)
	case <-ctx.Done():
		logg.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logg.Error("shutdown failed", "error", err)
	}
	logg.Info("shutddown completed")
}

type notifierAdapter struct{ n notifier.Notifier }

func (a notifierAdapter) Notify(ctx context.Context, m task.Notification) error {
	return a.n.Notify(ctx, notifier.Notification{
		UserID: m.UserID, Subject: m.Subject, Message: m.Message,
	})
}
