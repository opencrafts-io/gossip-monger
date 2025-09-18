package eventbus

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"

	"github.com/OneSignal/onesignal-go-api/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/opencrafts-io/gossip-monger/internal/repository"
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
	notification, err := h.convertToOneSignalNotification(&event.Notification)
	if err != nil {
		h.logger.Error("Error occurred while attempting to parse notification", slog.Any("error", err))
		return
	}

	result, code, err := h.onesignalClient.DefaultApi.CreateNotification(ctx).Notification(*notification).Execute()
	if err != nil {
		h.logger.Error("Error occurred while sending notification", slog.Any("error", err),
			slog.Any("more", result.GetErrors()),
		)
		h.logger.Error("Result", slog.Any("result", result))

		return

	}

	if code.StatusCode == http.StatusOK {
		h.logger.Info("Notification sent successfully", slog.Any("result", result))
		return
	}
}

func (h *NotificationEventHandler) convertToOneSignalNotification(
	eventNotification *repository.Notification,
) (*onesignal.Notification, error) {
	if eventNotification.Headings == nil {
		return nil, errors.New("at least one heading should be specified")
	}

	// Initialize OneSignal notification
	notification := *onesignal.NewNotification(os.Getenv("ONESIGNAL_APP_ID"))

	// set default to push
	notification.SetTargetChannel("push")

	// Parse the notification heading

	// Notification Recipients
	notification.SetIncludeAliases(map[string][]string{"external_id": {
		eventNotification.TargetUserID.String(),
	},
	})

	// Notification title
	var eventNotificationTitleRawContents map[string]string
	if err := json.Unmarshal(eventNotification.Headings, &eventNotificationTitleRawContents); err != nil {
		return nil, err
	}

	// Support for more languages later
	for lang, content := range eventNotificationTitleRawContents {
		heading := onesignal.NewLanguageStringMap()
		switch lang {
		case "en":
			heading.SetEn(content)
			break
		}

		if heading.GetEn() == "" {
			return nil, errors.New("The english heading must be atleast provided")
		}
		notification.SetHeadings(*heading)
	}

	var eventNotificationSubtitleRawContents map[string]string
	if err := json.Unmarshal(eventNotification.Subtitle,
		&eventNotificationSubtitleRawContents); err != nil {
		return nil, err
	}

	// Support for more languages later
	for lang, content := range eventNotificationSubtitleRawContents {
		subtitle := onesignal.NewLanguageStringMap()
		switch lang {
		case "en":
			subtitle.SetEn(content)
			break
		}
		notification.SetSubtitle(*subtitle)
	}

	// Notification body

	var eventNotificationBodyRawContents map[string]string
	if err := json.Unmarshal(eventNotification.Contents,
		&eventNotificationBodyRawContents); err != nil {
		return nil, err
	}

	// Support for more languages later
	for lang, content := range eventNotificationBodyRawContents {
		subtitle := onesignal.NewLanguageStringMap()
		switch lang {
		case "en":
			subtitle.SetEn(content)
			break
		}
		notification.SetContents(*subtitle)
	}

	// Notification sound
	if eventNotification.AndroidSound != nil {
		notification.SetAndroidSound(*eventNotification.AndroidSound)
	}
	if eventNotification.IosSound != nil {
		notification.SetIosSound(*eventNotification.IosSound)
	}

	// Notification channel
	if eventNotification.AndroidChannelID != nil {
		notification.SetAndroidChannelId(*eventNotification.AndroidChannelID)
	}

	// Android led color
	if eventNotification.AndroidAccentColor != nil {
		notification.SetAndroidAccentColor(*eventNotification.AndroidAccentColor)
	}

	// Notification url
	if eventNotification.Url != nil {
		notification.SetUrl(*eventNotification.Url)
	}

	// Notification pictures
	if eventNotification.BigPicture != nil {
		notification.SetBigPicture(*eventNotification.BigPicture)
	}

	// Icons
	if eventNotification.LargeIcon != nil {
		notification.SetLargeIcon(*eventNotification.LargeIcon)
	}
	if eventNotification.SmallIcon != nil {
		notification.SetSmallIcon(*eventNotification.SmallIcon)
	}

	// Notification body
	// Set the notification payload
	if eventNotification.Data != nil {
		var data map[string]any
		if err := json.Unmarshal(eventNotification.Data, &data); err != nil {
			return nil, errors.New("Failed to decode notification payload")
		}
		notification.SetData(data)
	}

	// Notification buttons
	if eventNotification.Buttons != nil {

		var eventNotificationButtonRawContents []map[string]string
		if err := json.Unmarshal(eventNotification.Buttons,
			&eventNotificationButtonRawContents); err != nil {
			return nil, err
		}
		var btns []onesignal.Button

		for _, rawButton := range eventNotificationButtonRawContents {
			btn := onesignal.NewButton(rawButton["id"])
			btn.SetText(rawButton["text"])
			btn.SetText(rawButton["icon"])
			btns = append(btns, *btn)
		}
		notification.SetButtons(btns)
	}
	return &notification, nil
}
