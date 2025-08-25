package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/opencrafts-io/gossip-monger/internal/config"
	"github.com/opencrafts-io/gossip-monger/internal/middleware"
)

type GossipMonger struct {
	pool   *pgxpool.Pool
	config *config.Config
	logger *slog.Logger
}

// Creates a new gossip-monger application ready to service requests
func NewGossipMongerApp(logger *slog.Logger, cfg *config.Config) (*GossipMonger, error) {

	dbConfig, err := pgxpool.ParseConfig(fmt.Sprintf(
		"postgresql://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.DatabaseConfig.DatabaseUser,
		cfg.DatabaseConfig.DatabasePassword,
		cfg.DatabaseConfig.DatabaseHost,
		cfg.DatabaseConfig.DatabasePort,
		cfg.DatabaseConfig.DatabaseName,
	))
	if err != nil {
		return nil, err
	}

	dbConfig.MaxConns = cfg.DatabaseConfig.DatabasePoolMaxConnections
	dbConfig.MinConns = cfg.DatabaseConfig.DatabasePoolMinConnections
	dbConfig.MaxConnLifetime = time.Hour * time.Duration(cfg.DatabaseConfig.DatabasePoolMaxConnectionLifetime)

	connPool, err := pgxpool.NewWithConfig(context.Background(), dbConfig)
	if err != nil {
		return nil, err
	}

	return &GossipMonger{
		pool:   connPool,
		config: cfg,
		logger: logger,
	}, nil
}

func (gm *GossipMonger) Start(ctx context.Context) error {

	router := LoadRoutes()

	defaultMiddlewares := middleware.CreateStack(
		middleware.Logging(gm.logger),
	)

	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", gm.config.AppConfig.Address, gm.config.AppConfig.Port),
		Handler: defaultMiddlewares(router),
	}

	errCh := make(chan error, 1)

	go func() {
		err := srv.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("failed to listen and serve: %w", err)
		}

		close(errCh)
	}()

	gm.logger.Info("server running",
		slog.String("Address", gm.config.AppConfig.Address),
		slog.Int("port", gm.config.AppConfig.Port),
	)

	select {
	// Wait until we receive SIGINT (ctrl+c on cli)
	case <-ctx.Done():
		break
	case err := <-errCh:
		return err
	}

	sCtx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	srv.Shutdown(sCtx)

	return nil

}
