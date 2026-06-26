package consumers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/opencrafts-io/gossip-monger/internal/broker"
	"github.com/opencrafts-io/gossip-monger/internal/service"
)

type UserConsumer struct {
	consumer    broker.MessageConsumer
	userService service.UserService
	logger      *slog.Logger
}

func NewUserConsumer(
	conn broker.Connection,
	userService service.UserService,
	logger *slog.Logger,
) *UserConsumer {
	return &UserConsumer{
		consumer:    broker.NewConsumer(conn, 10, *logger),
		userService: userService,
		logger:      logger,
	}
}

func (uc *UserConsumer) Start(ctx context.Context) error {
	return uc.consumer.Consume(
		ctx,
		"verisafe.exchange",
		broker.FanoutExchangeType,
		"verisafe.user.queue",
		"verisafe.user.*",
		uc.handleMessage,
	)
}

func (uc *UserConsumer) handleMessage(
	ctx context.Context,
	message []byte,
) error {
	var event service.UserEvent
	if err := json.Unmarshal(message, &event); err != nil {
		uc.logger.Error("failed to unmarshal user event", "error", err)
		return err
	}

	if event.Metadata.SourceServiceID != "io.opencrafts.verisafe" {
		return fmt.Errorf(
			"unexpected source service id: expected 'io.opencrafts.verisafe', got '%s'",
			event.Metadata.SourceServiceID,
		)
	}

	switch event.Metadata.EventType {
	case "user.created":
		return uc.userService.Create(ctx, event.User)
	case "user.updated":
		return uc.userService.Update(ctx, event.User)
	case "user.deleted":
		return uc.userService.Delete(ctx, event.User)
	default:
		uc.logger.Error("unrecognised user event type",
			slog.String("event_type", event.Metadata.EventType),
			slog.String("source_service", event.Metadata.SourceServiceID),
		)
		return fmt.Errorf(
			"unrecognised user event type: %s, source service: %s",
			event.Metadata.EventType,
			event.Metadata.SourceServiceID,
		)
	}
}
