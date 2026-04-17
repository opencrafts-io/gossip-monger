package service

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

type EmailService interface {
	Send(ctx context.Context) error
}

type emailService struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

func NewEmailService(pool *pgxpool.Pool, logger *slog.Logger) EmailService {
	return &emailService{
		pool:   pool,
		logger: logger,
	}
}

func (es *emailService) Send(ctx context.Context) error {
	return nil
}
