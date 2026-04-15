package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/OneSignal/onesignal-go-api/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/opencrafts-io/gossip-monger/database"
	"github.com/opencrafts-io/gossip-monger/internal/broker"
	"github.com/opencrafts-io/gossip-monger/internal/config"
	"github.com/opencrafts-io/gossip-monger/internal/eventbus"
	"github.com/opencrafts-io/gossip-monger/internal/middleware"
	"github.com/resend/resend-go/v2"
)

type GossipMonger struct {
	rabbitMQConn         broker.Connection
	pool                 *pgxpool.Pool
	config               *config.Config
	logger               *slog.Logger
	userEventBus         *eventbus.UserEventBus
	notificationEventBus *eventbus.NotificationEventBus
	emailEventBus        *eventbus.EmailEventBus
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

	bus, err := eventbus.NewRabbitMQEventBus(
		rabbitMQConnString,
		"verisafe.exchange",
		eventbus.FanoutExchangeType, logger,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to connect to rabbit mq  event bus %w",
			err,
		)
	}

	nbus, err := eventbus.NewRabbitMQEventBus(
		rabbitMQConnString,
		"gossip-monger.exchange",
		eventbus.DirectExchangeType,
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to connect to rabbit mq  event bus %w",
			err,
		)
	}

	userEventBus := eventbus.NewUserEventBus(bus, connPool, logger)
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
	notificationEventBus := eventbus.NewNotificationEventBus(
		nbus,
		connPool,
		oneSignalService,
		logger,
	)

	client := resend.NewClient(os.Getenv("RESEND_API_KEY"))

	emailEventBus := eventbus.NewEmailEventBus(nbus, connPool, client, logger)

	return &GossipMonger{
		rabbitMQConn:         rabbitMQConn,
		pool:                 connPool,
		config:               cfg,
		logger:               logger,
		userEventBus:         userEventBus,
		notificationEventBus: notificationEventBus,
		emailEventBus:        emailEventBus,
	}, nil
}

func (gm *GossipMonger) Start(ctx context.Context) error {
	database.RunGooseMigrations(gm.logger, gm.pool)

	// Setup verisafe event subscriptions
	gm.userEventBus.SetupEventSubscriptions()
	gm.notificationEventBus.SetupEventSubscriptions()
	gm.emailEventBus.SetupEventSubscriptions()

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
	if err := gm.rabbitMQConn.Close(); err != nil {
		gm.logger.Error("Failed to close rabbitmq connection")
	}
	gm.userEventBus.Close()
	gm.notificationEventBus.Close()
	gm.emailEventBus.Close()

	return nil
}
