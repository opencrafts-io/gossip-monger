package consumers

import (
	"context"
	"encoding/json"
	"log/slog"

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

	return pnc.notificationService.Send(
		ctx,
		notifMsg.Notification,
	)
}
