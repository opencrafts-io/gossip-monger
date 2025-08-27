package eventbus

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

// UserEventHandler handles user events with database access
type UserEventHandler struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewUserEventHandler creates a new UserEventHandler instance
func NewUserEventHandler(pool *pgxpool.Pool, logger *slog.Logger) *UserEventHandler {
	return &UserEventHandler{
		pool:   pool,
		logger: logger,
	}
}

// HandleUserCreated processes user created events
func (h *UserEventHandler) HandleUserCreated(ctx context.Context, event UserEvent) {
	h.logger.Info("Received user created event",
		slog.String("user_id", event.User.ID.String()),
		slog.String("email", event.User.Email),
	)
}

// HandleUserUpdated processes user updated events
func (h *UserEventHandler) HandleUserUpdated(ctx context.Context, event UserEvent) {
	h.logger.Info("Received user updated event",
		slog.String("user_id", event.User.ID.String()),
		slog.String("email", event.User.Email),
	)

}

// HandleUserDeleted processes user deleted events
func (h *UserEventHandler) HandleUserDeleted(ctx context.Context, event UserEvent) {
	h.logger.Info("Received user deleted event",
		slog.String("user_id", event.User.ID.String()),
		slog.String("email", event.User.Email),
	)

}
