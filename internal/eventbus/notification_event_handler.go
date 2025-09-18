package eventbus

import (
	"context"
	"encoding/json"
	"errors"
	"io"
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

	result, code, err := h.onesignalClient.DefaultApi.
		CreateNotification(ctx).Notification(*notification).Execute()

	var body []byte
	if code != nil && code.Body != nil {
		defer code.Body.Close()
		body, _ = io.ReadAll(code.Body)
	}

	event.Notification.OnesignalResponse = body

	if err != nil {
		error := string(body)
		event.Notification.OnesignalError = &error
		h.logger.Error("Error occurred while sending notification",
			slog.Any("response", string(body)),
			slog.Any("error", err),
			slog.Any("more", result.GetErrors()),
		)
		h.logger.Error("Result", slog.Any("result", result))

		return

	}
	if code.StatusCode == http.StatusOK {
		var bodyMap map[string]any
		if len(body) > 0 {
			if err := json.Unmarshal(body, &bodyMap); err != nil {
				h.logger.Error("Failed to unmarshal response body", slog.Any("error", err))
			}
		}
		if id, ok := bodyMap["id"].(string); ok {
			event.Notification.OnesignalNotificationID = &id
		} else {
			h.logger.Warn("Response did not contain an 'id' field as string", slog.Any("body", bodyMap))
		}

	}

	// Write to the db
	conn, err := h.pool.Acquire(ctx)
	if err != nil {
		h.logger.Error("Failed to acquire connection from pool", slog.Any("notification", notification))
		return
	}
	repo := repository.New(conn)
	if _, err = repo.CreateNotification(ctx, repository.CreateNotificationParams{
		AppID:                   event.Notification.AppID,
		IncludedSegments:        event.Notification.IncludedSegments,
		ExcludedSegments:        event.Notification.ExcludedSegments,
		IncludePlayerIds:        event.Notification.IncludePlayerIds,
		IncludeExternalUserIds:  event.Notification.IncludeExternalUserIds,
		IncludeEmailTokens:      event.Notification.IncludeEmailTokens,
		IncludePhoneNumbers:     event.Notification.IncludePhoneNumbers,
		IncludeIosTokens:        event.Notification.IncludeIosTokens,
		IncludeWpWnsUris:        event.Notification.IncludeWpWnsUris,
		IncludeAmazonRegIds:     event.Notification.IncludeAmazonRegIds,
		IncludeChromeRegIds:     event.Notification.IncludeChromeRegIds,
		IncludeChromeWebRegIds:  event.Notification.IncludeChromeWebRegIds,
		IncludeAndroidRegIds:    event.Notification.IncludeAndroidRegIds,
		Contents:                event.Notification.Contents,
		Headings:                event.Notification.Headings,
		Subtitle:                event.Notification.Subtitle,
		BigPicture:              event.Notification.BigPicture,
		LargeIcon:               event.Notification.LargeIcon,
		SmallIcon:               event.Notification.SmallIcon,
		IosAttachments:          event.Notification.IosAttachments,
		AndroidChannelID:        event.Notification.AndroidChannelID,
		AndroidAccentColor:      event.Notification.AndroidAccentColor,
		AndroidLedColor:         event.Notification.AndroidLedColor,
		AndroidGroup:            event.Notification.AndroidGroup,
		AndroidGroupMessage:     event.Notification.AndroidGroupMessage,
		AndroidSound:            event.Notification.AndroidSound,
		IosSound:                event.Notification.IosSound,
		WpWnsSound:              event.Notification.WpWnsSound,
		AdmSound:                event.Notification.AdmSound,
		ChromeWebImage:          event.Notification.ChromeWebImage,
		ChromeWebIcon:           event.Notification.ChromeWebIcon,
		ChromeWebBadge:          event.Notification.ChromeWebBadge,
		ChromeWebColor:          event.Notification.ChromeWebColor,
		ChromeWebSound:          event.Notification.ChromeWebSound,
		Url:                     event.Notification.Url,
		WebUrl:                  event.Notification.WebUrl,
		AppUrl:                  event.Notification.AppUrl,
		Data:                    event.Notification.Data,
		Filters:                 event.Notification.Filters,
		Tags:                    event.Notification.Tags,
		SendAfter:               event.Notification.SendAfter,
		DelayedOption:           event.Notification.DelayedOption,
		DeliveryTimeOfDay:       event.Notification.DeliveryTimeOfDay,
		Ttl:                     event.Notification.Ttl,
		Priority:                event.Notification.Priority,
		TargetUserID:            event.Notification.TargetUserID,
		SourceServiceID:         event.Notification.SourceServiceID,
		SourceUserID:            event.Notification.SourceUserID,
		NotificationType:        event.Notification.NotificationType,
		OnesignalNotificationID: event.Notification.OnesignalNotificationID,
		OnesignalStatus:         event.Notification.OnesignalStatus,
		OnesignalResponse:       event.Notification.OnesignalResponse,
		OnesignalError:          event.Notification.OnesignalError,
	}); err != nil {
		h.logger.Error("Failed to write notification to db",
			slog.Any("error", err),
			slog.Any("Notification", event.Notification),
		)
		return
	}

	h.logger.Info("Notification successfully created and  sent")

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

	var targetUsers []string
	targetUsers = append(targetUsers, eventNotification.TargetUserID.String())

	if eventNotification.IncludeExternalUserIds != nil {
		targetUsers = append(targetUsers, eventNotification.IncludeExternalUserIds...)
	}

	// Notification Recipients
	notification.SetIncludeAliases(map[string][]string{"external_id": targetUsers})

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
			btn.SetIcon(rawButton["icon"])
			btns = append(btns, *btn)
		}
		notification.SetButtons(btns)
	}
	return &notification, nil
}
