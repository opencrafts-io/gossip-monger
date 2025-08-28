package eventbus

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/OneSignal/onesignal-go-api/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NotificationEventHandler struct {
	pool            *pgxpool.Pool
	logger          *slog.Logger
	onesignalClient *onesignal.APIClient
}

// NewNotificationEventHandler creates a new handler for all notification based events
func NewNotificationEventHandler(
	pool *pgxpool.Pool,
	onesignalClient *onesignal.APIClient,
	logger *slog.Logger,
) *NotificationEventHandler {
	return &NotificationEventHandler{
		pool:            pool,
		logger:          logger,
		onesignalClient: onesignalClient,
	}
}

func (h *NotificationEventHandler) HandlerPushNotificationSendRequested(
	ctx context.Context, event NotificationEvent,
) {
	h.logger.Info("Notification event requested", slog.Any("event", event))

	// Create a notification payload
	notification := *onesignal.NewNotification(
		os.Getenv("ONESIGNAL_APP_ID"),
	)

	info := onesignal.NewLanguageStringMap()
	info.SetEn("Hello from Golang")
	notification.SetContents(
		*info,
	)

	heading := onesignal.NewLanguageStringMap()
	heading.SetEn("Hello")

	notification.SetHeadings(*heading)

	notification.SetIncludeAliases(map[string][]string{
		"external_id": {"f714651a-2b57-4678-8776-9708c92d8dd1"},
	},
	)
	notification.SetTargetChannel("push")
	btn := onesignal.NewButton("hello")
	btn.SetText("Hello")

	btns := []onesignal.Button{*btn}

	notification.SetButtons(btns)

	h.logger.Info("Stack trace", slog.Any("one signal client", h.onesignalClient), (slog.Any("notification", notification)))

	result, code, err := h.onesignalClient.DefaultApi.CreateNotification(ctx).Notification(notification).Execute()
	if err != nil {
		h.logger.Error("Error occurred while sending notification", slog.Any("error", err),
			slog.Any("more", result.GetErrors()),
		)

	}

	if code.StatusCode == http.StatusOK {
		h.logger.Info("Notification sent successfully", slog.Any("result", result))

	}
}
