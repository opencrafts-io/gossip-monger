package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/OneSignal/onesignal-go-api/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/opencrafts-io/gossip-monger/database"
	"github.com/opencrafts-io/gossip-monger/internal/broker"
	"github.com/opencrafts-io/gossip-monger/internal/broker/consumers"
	"github.com/opencrafts-io/gossip-monger/internal/config"
	"github.com/opencrafts-io/gossip-monger/internal/middleware"
	"github.com/opencrafts-io/gossip-monger/internal/repository"
	"github.com/opencrafts-io/gossip-monger/internal/service"
)

type GossipMonger struct {
	rabbitMQConn broker.Connection
	pool         *pgxpool.Pool
	config       *config.Config
	logger       *slog.Logger
	// Track all running consumers for graceful shutdown
	consumerWg      sync.WaitGroup
	cancelConsumers context.CancelFunc

	// Services
	pushNotificationSvc service.PushNotificationService
	userService         service.UserService
}

// Creates a new gossip-monger application ready to service requests
func NewGossipMongerApp(
	logger *slog.Logger,
	cfg *config.Config,
) (*GossipMonger, error) {
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
	dbConfig.MaxConnLifetime = time.Hour * time.Duration(
		cfg.DatabaseConfig.DatabasePoolMaxConnectionLifetime,
	)

	connPool, err := pgxpool.NewWithConfig(context.Background(), dbConfig)
	if err != nil {
		return nil, err
	}

	rabbitMQConnString := fmt.Sprintf("amqp://%s:%s@%s:%d/",
		cfg.RabbitMQConfig.RabbitMQUser,
		cfg.RabbitMQConfig.RabbitMQPass,
		cfg.RabbitMQConfig.RabbitMQAddress,
		cfg.RabbitMQConfig.RabbitMQPort,
	)

	rabbitMQConn, err := broker.NewRabbitMQConnection(
		context.Background(),
		rabbitMQConnString,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to connect to rabbit mq  event bus %w",
			err,
		)
	}
	onesignalConfig := onesignal.NewConfiguration()
	onesignalConfig.AddDefaultHeader("Authorization",
		fmt.Sprintf("Basic %s", cfg.OneSignalConfig.RestAPIKey),
	)
	oneSignalService := onesignal.NewAPIClient(onesignalConfig)

	if oneSignalService == nil {
		logger.Error(
			"Failed to initialize onesignal api client",
			slog.Any("oneSignalService", oneSignalService),
		)
		panic("Failed to initialize one signal service")

	}
	querier := repository.New(connPool)
	pnsvc := service.NewPushNotificationService(
		querier,
		logger,
		oneSignalService,
	)

	userService := service.NewUserService(connPool, logger)

	return &GossipMonger{
		rabbitMQConn:        rabbitMQConn,
		pool:                connPool,
		config:              cfg,
		logger:              logger,
		pushNotificationSvc: pnsvc,
		userService:         userService,
	}, nil
}

func (gm *GossipMonger) Start(ctx context.Context) error {
	database.RunGooseMigrations(gm.logger, gm.pool)

	gm.startConsumers(ctx)

	router := LoadRoutes(gm)

	defaultMiddlewares := middleware.CreateStack(
		middleware.Logging(gm.logger),
	)

	srv := &http.Server{
		Addr: fmt.Sprintf("%s:%d",
			gm.config.AppConfig.Address,
			gm.config.AppConfig.Port,
		),
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
	gm.shutDown()

	return nil
}

func (gm *GossipMonger) startConsumers(ctx context.Context) {
	pushNotificationConsumer := consumers.NewPushNotificationConsumer(
		gm.rabbitMQConn,
		gm.pushNotificationSvc,
		gm.logger,
	)

	userConsumer := consumers.NewUserConsumer(
		gm.rabbitMQConn,
		gm.userService,
		gm.logger,
	)

	gm.consumerWg.Add(1)
	go func() {
		defer gm.consumerWg.Done()
		if err := pushNotificationConsumer.Start(ctx); err != nil {
			gm.logger.Error(
				"Push notification consumer stopped",
				slog.Any("error", err),
			)
		}
	}()

	gm.consumerWg.Add(1)
	go func() {
		defer gm.consumerWg.Done()
		if err := userConsumer.Start(ctx); err != nil {
			gm.logger.Error(
				"User events consumer stopped",
				slog.Any("error", err),
			)
		}
	}()
}

func (gm *GossipMonger) shutDown() {
	// Close the consumers
	gm.logger.Info("Shutting down consumers...")
	if gm.cancelConsumers != nil {
		gm.cancelConsumers()
	}
	gm.consumerWg.Wait()

	// Close the rabbitmq connection
	if err := gm.rabbitMQConn.Close(); err != nil {
		panic(err)
	}
}
