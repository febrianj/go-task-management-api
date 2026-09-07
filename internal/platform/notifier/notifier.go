package notifier

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
)

type Notification struct {
	UserID  uuid.UUID
	Subject string
	Message string
}

type Notifier interface {
	Notify(ctx context.Context, n Notification) error
}

type LogNotifier struct {
	log *slog.Logger
}

func NewLogNotifier(l *slog.Logger) *LogNotifier { return &LogNotifier{log: l} }

func (n *LogNotifier) Notify(ctx context.Context, m Notification) error {
	n.log.InfoContext(ctx, "notification sent",
		"user_id", m.UserID,
		"subject", m.Subject,
		"message", m.Message,
	)
	return nil
}
