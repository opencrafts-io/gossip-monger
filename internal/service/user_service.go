package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/opencrafts-io/gossip-monger/internal/repository"
)

type UserService interface {
	Create(ctx context.Context, user repository.User) error
	Update(ctx context.Context, user repository.User) error
	Delete(ctx context.Context, user repository.User) error
}

type userService struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

func NewUserService(pool *pgxpool.Pool, logger *slog.Logger) UserService {
	return &userService{
		pool:   pool,
		logger: logger,
	}
}

func (s *userService) Create(ctx context.Context, user repository.User) error {
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	repo := repository.New(tx)

	created, err := repo.CreateUser(ctx, repository.CreateUserParams{
		ID:       user.ID,
		Email:    user.Email,
		Name:     user.Name,
		Username: &user.Name,
		Phone:    user.Phone,
	})
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.Info("successfully created user from message queue",
		slog.String("user_id", created.ID.String()),
		slog.String("username", *created.Username),
	)

	return nil
}

func (s *userService) Update(ctx context.Context, user repository.User) error {
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	repo := repository.New(tx)

	updated, err := repo.UpdateUserByID(ctx, repository.UpdateUserByIDParams{
		ID:       user.ID,
		Email:    user.Email,
		Name:     user.Name,
		Username: derefString(user.Username),
		Phone:    derefString(user.Phone),
	})
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.Info("successfully updated user from message queue",
		slog.String("user_id", updated.ID.String()),
		slog.String("username", *updated.Username),
	)

	return nil
}

func (s *userService) Delete(ctx context.Context, user repository.User) error {
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	repo := repository.New(tx)

	if err = repo.DeleteUserByID(ctx, user.ID); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.Info("successfully deleted user from message queue",
		slog.String("user_id", user.ID.String()),
	)

	return nil
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
