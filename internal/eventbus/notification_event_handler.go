package eventbus

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

type NotificationEventHandler struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewNotificationEventHandler creates a new handler for all notification based events
func NewNotificationEventHandler(pool *pgxpool.Pool, logger *slog.Logger) *NotificationEventHandler {
	return &NotificationEventHandler{
		pool:   pool,
		logger: logger,
	}
}

func (h *NotificationEventHandler) HandlerPushNotificationSendRequested(
	ctx context.Context, event NotificationEvent,
) {
	h.logger.Info("Notification event requested", slog.Any("event", event))
}
