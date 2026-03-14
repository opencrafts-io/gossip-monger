package eventbus

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/resend/resend-go/v2"
)

type EmailEventHandler struct {
	pool         *pgxpool.Pool
	logger       *slog.Logger
	resendClient *resend.Client
}

// Returns a new instance of an EmailEventHandler
func NewEmailEventHandler(
	pool *pgxpool.Pool,
	resendClient *resend.Client,
	logger *slog.Logger,
) *EmailEventHandler {
	return &EmailEventHandler{
		pool:         pool,
		resendClient: resendClient,
		logger:       logger,
	}
}

func (h *EmailEventHandler) HandleEmailSendRequested(context context.Context, event EmailEvent) {
	emailParam := &resend.SendEmailRequest{
		From: "Academia <onboarding@notifications.opencrafts.io>",
		To: event.To,
		Bcc: event.Bcc,
		Cc: event.Cc,
		Subject: *event.Subject,
		ReplyTo: *event.ReplyTo,
		Html: *event.Body,
	}
	h.resendClient.Emails.Send(emailParam)	
}
