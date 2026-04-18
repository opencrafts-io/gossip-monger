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

type EmailConsumer struct {
	consumer     broker.MessageConsumer
	emailService service.EmailService
	logger       *slog.Logger
}

func NewEmailConsumer(
	conn broker.Connection,
	emailService service.EmailService,
	logger *slog.Logger,
) *EmailConsumer {
	return &EmailConsumer{
		consumer:     broker.NewConsumer(conn, 10, *logger),
		emailService: emailService,
		logger:       logger,
	}
}

func (ec *EmailConsumer) Start(ctx context.Context) error {
	return ec.consumer.Consume(
		ctx,
		"gossip.topic.exchange",
		broker.TopicExchangeType,
		"gossip.emails.queue",
		"gossip.emails.*",
		ec.handleMessage,
	)
}

func (ec *EmailConsumer) handleMessage(
	ctx context.Context,
	message []byte,
) error {
	var emailMsg service.EmailEvent
	if err := json.Unmarshal(message, &emailMsg); err != nil {
		ec.logger.Error(
			"failed to unmarshal email message",
			"error",
			err,
		)
		return err
	}

	if !strings.HasPrefix(emailMsg.Meta.SourceServiceID, "io.opencrafts.") {
		return fmt.Errorf(
			"wrong service id expected service id to be in the io.opencrafts namespace instead got '%s'",
			emailMsg.Meta.SourceServiceID,
		)
	}

	switch emailMsg.Meta.EventType {
	case "email.send":
		return ec.emailService.Send(ctx, emailMsg)
	default:
		ec.logger.Error(
			"got wrong event metadata type",
			slog.String("event_type", emailMsg.Meta.EventType),
			slog.String("source_service", emailMsg.Meta.SourceServiceID),
		)
		return fmt.Errorf(
			"wrong event metadata type: %s, source service %s",
			emailMsg.Meta.EventType,
			emailMsg.Meta.SourceServiceID,
		)
	}
}
