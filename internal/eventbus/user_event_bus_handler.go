package eventbus

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/opencrafts-io/gossip-monger/internal/repository"
	"log/slog"
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
	conn, err := h.pool.Acquire(ctx)
	if err != nil {
		h.logger.Error("Failed to acquire connection from pool", slog.String("error", err.Error()))
		return
	}

	tx, _ := conn.Begin(ctx)
	defer tx.Rollback(ctx)
	repo := repository.New(tx)

	created, err := repo.CreateUser(ctx, repository.CreateUserParams{
		ID:       event.User.ID,
		Email:    event.User.Email,
		Name:     event.User.Name,
		Username: &event.User.Name,
		Phone:    event.User.Phone,
	})

	if err != nil {
		h.logger.Error("Failed to create user from the mesage queue", slog.String("error", err.Error()))
		return
	}

	if err = tx.Commit(ctx); err != nil {
		h.logger.Error("Failed to commit transaction for creating user",
			slog.String("error", err.Error()),
			slog.String("user_id", created.ID.String()),
			slog.String("username", *created.Username),
		)
		return
	}

	h.logger.Info("Successfully created user requested from the message queue",
		slog.String("user_id", created.ID.String()),
		slog.String("username", *created.Username),
	)

}

// HandleUserUpdated processes user updated events
func (h *UserEventHandler) HandleUserUpdated(ctx context.Context, event UserEvent) {
	conn, err := h.pool.Acquire(ctx)
	if err != nil {
		h.logger.Error("Failed to acquire connection from pool", slog.String("error", err.Error()))
		return
	}

	tx, _ := conn.Begin(ctx)
	defer tx.Rollback(ctx)
	repo := repository.New(tx)

	created, err := repo.UpdateUserByID(ctx, repository.UpdateUserByIDParams{
		ID:       event.User.ID,
		Email:    event.User.Email,
		Name:     event.User.Name,
		Username: event.User.Name,
		Phone:    *event.User.Phone,
	})

	if err != nil {
		h.logger.Error("Failed to update user requested by the verisafe mesage queue",
			slog.String("error", err.Error()))
		return
	}

	if err = tx.Commit(ctx); err != nil {
		h.logger.Error("Failed to commit transaction for updating user",
			slog.String("error", err.Error()),
			slog.String("user_id", created.ID.String()),
			slog.String("username", *created.Username),
		)
		return
	}

	h.logger.Info("Successfully updated user requested by the verisafe message queue",
		slog.String("user_id", created.ID.String()),
		slog.String("username", *created.Username),
	)
}

// HandleUserDeleted processes user deleted events
func (h *UserEventHandler) HandleUserDeleted(ctx context.Context, event UserEvent) {
	conn, err := h.pool.Acquire(ctx)
	if err != nil {
		h.logger.Error("Failed to acquire connection from pool", slog.String("error", err.Error()))
		return
	}

	tx, _ := conn.Begin(ctx)
	defer tx.Rollback(ctx)
	repo := repository.New(tx)

	err = repo.DeleteUserByID(ctx, event.User.ID)

	if err != nil {
		h.logger.Error("Failed to delete user requested by the verisafe mesage queue",
			slog.String("error", err.Error()))
		return
	}

	if err = tx.Commit(ctx); err != nil {
		h.logger.Error("Failed to commit transaction for updating user",
			slog.String("error", err.Error()),
			slog.String("user_id", event.User.ID.String()),
			slog.String("username", *event.User.Username),
		)
		return
	}

	h.logger.Info("Successfully deleted user requested from the verisafe message queue",
		slog.String("user_id", event.User.ID.String()),
		slog.String("username", *event.User.Username),
	)

}
