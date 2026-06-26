package consumers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/opencrafts-io/gossip-monger/internal/broker"
	"github.com/opencrafts-io/gossip-monger/internal/service"
)

type PushNotificationConsumer struct {
	consumer            broker.MessageConsumer
	notificationService service.PushNotificationService
	logger              *slog.Logger
}

func NewPushNotificationConsumer(
	conn broker.Connection,
	notificationService service.PushNotificationService,
	logger *slog.Logger,
) *PushNotificationConsumer {
	return &PushNotificationConsumer{
		consumer:            broker.NewConsumer(conn, 10, *logger),
		notificationService: notificationService,
		logger:              logger,
	}
}

func (pnc *PushNotificationConsumer) Start(ctx context.Context) error {
	return pnc.consumer.Consume(
		ctx,
		"gossip.topic.exchange",
		broker.TopicExchangeType,
		"gossip.notification.queue",
		"gossip.push.*",
		pnc.handleMessage,
	)
}

func (pnc *PushNotificationConsumer) handleMessage(
	ctx context.Context,
	message []byte,
) error {
	var notifMsg service.PushNotificationEvent
	if err := json.Unmarshal(message, &notifMsg); err != nil {
		pnc.logger.Error(
			"failed to unmarshal notification message",
			"error",
			err,
		)
		return err
	}

	if !strings.Contains(notifMsg.Metadata.SourceServiceID, "io.opencrafts.") {
		return fmt.Errorf(
			"wrong service id expected service id to be in the io.opencrafts namespace instead got '%s'",
			notifMsg.Metadata.SourceServiceID,
		)
	}

	switch notifMsg.Metadata.EventType {
	case "push.send":
		return pnc.notificationService.Send(ctx, notifMsg.Notification)
	default:
		pnc.logger.Error(
			"Got wrong event metadata type",
			slog.String("event_type", notifMsg.Metadata.EventType),
			slog.String("source_service", notifMsg.Metadata.SourceServiceID),
		)
		return fmt.Errorf(
			"wrong event metadata type: %s, source service %s",
			notifMsg.Metadata.EventType,
			notifMsg.Metadata.SourceServiceID,
		)
	}
}
