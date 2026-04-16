package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/OneSignal/onesignal-go-api/v5"
	"github.com/opencrafts-io/gossip-monger/internal/repository"
)

type PushNotificationEventMetaData struct {
	EventType       string    `json:"event_type"`
	Timestamp       time.Time `json:"timestamp"`
	SourceServiceID string    `json:"source_service_id"`
	RequestID       string    `json:"request_id"`
}

type PushNotificationEvent struct {
	Notification repository.Notification       `json:"notification"`
	Metadata     PushNotificationEventMetaData `json:"metadata"`
}

type PushNotificationService interface {
	Send(context.Context, repository.Notification) error
}

type pushNotificationService struct {
	repo            repository.Querier
	logger          *slog.Logger
	onesignalClient *onesignal.APIClient
}

func NewPushNotificationService(
	repo repository.Querier,
	logger *slog.Logger,
	onesignalClient *onesignal.APIClient,
) PushNotificationService {
	return &pushNotificationService{
		repo:            repo,
		onesignalClient: onesignalClient,
		logger:          logger,
	}
}

func (pns *pushNotificationService) Send(
	ctx context.Context,
	push repository.Notification,
) error {
	payload, err := pns.preparePushPayload(push)
	if err != nil {
		return err
	}

	_, code, err := pns.onesignalClient.DefaultApi.
		CreateNotification(ctx).Notification(*payload).Execute()

	var body []byte
	if code != nil && code.Body != nil {
		defer code.Body.Close()
		body, _ = io.ReadAll(code.Body)
	}

	// Parse response
	notificationID, rawResponse, parseErr := pns.parseOnesignalResponse(
		body,
		code.StatusCode,
	)

	// Enrich notification with response data
	if err != nil {
		enrichNotificationFromResponse(&push, "", rawResponse, err)
		return err
	}

	if parseErr != nil {
		enrichNotificationFromResponse(&push, "", rawResponse, parseErr)
		return parseErr
	}

	// Success path
	enrichNotificationFromResponse(&push, notificationID, rawResponse, nil)

	_, err = pns.repo.CreateNotification(
		ctx,
		pns.notificationToCreateParams(&push),
	)
	if err != nil {
		return fmt.Errorf("failed to persist notification %w", err)
	}

	return nil
}

type notificationButton struct {
	ID   string `json:"id"`   // Unique ID for the button
	Text string `json:"text"` // What the user sees
	Icon string `json:"icon"` // Optional icon URL/resource
}

func (pns *pushNotificationService) parseOnesignalResponse(
	body []byte,
	statusCode int,
) (notificationID string, rawResponse json.RawMessage, err error) {
	rawResponse = json.RawMessage(body)

	if statusCode != http.StatusOK {
		return "", rawResponse, fmt.Errorf(
			"unexpected status code: %d",
			statusCode,
		)
	}

	if len(body) == 0 {
		return "", rawResponse, errors.New("empty response body from OneSignal")
	}

	var responseMap map[string]any
	if err := json.Unmarshal(body, &responseMap); err != nil {
		return "", rawResponse, fmt.Errorf(
			"failed to unmarshal OneSignal response: %w",
			err,
		)
	}

	id, ok := responseMap["id"].(string)
	if !ok {
		return "", rawResponse, errors.New(
			"id field missing or not a string in OneSignal response",
		)
	}

	return id, rawResponse, nil
}

func enrichNotificationFromResponse(
	push *repository.Notification,
	notificationID string,
	rawResponse json.RawMessage,
	apiErr error,
) {
	push.OnesignalResponse = rawResponse

	if apiErr != nil {
		errorStr := apiErr.Error()
		push.OnesignalError = &errorStr
		return
	}

	if notificationID != "" {
		push.OnesignalNotificationID = &notificationID
	}
}

func (pns *pushNotificationService) notificationToCreateParams(
	n *repository.Notification,
) repository.CreateNotificationParams {
	return repository.CreateNotificationParams{
		AppID:                   n.AppID,
		IncludedSegments:        n.IncludedSegments,
		ExcludedSegments:        n.ExcludedSegments,
		IncludePlayerIds:        n.IncludePlayerIds,
		IncludeExternalUserIds:  n.IncludeExternalUserIds,
		IncludeEmailTokens:      n.IncludeEmailTokens,
		IncludePhoneNumbers:     n.IncludePhoneNumbers,
		IncludeIosTokens:        n.IncludeIosTokens,
		IncludeWpWnsUris:        n.IncludeWpWnsUris,
		IncludeAmazonRegIds:     n.IncludeAmazonRegIds,
		IncludeChromeRegIds:     n.IncludeChromeRegIds,
		IncludeChromeWebRegIds:  n.IncludeChromeWebRegIds,
		IncludeAndroidRegIds:    n.IncludeAndroidRegIds,
		Contents:                n.Contents,
		Headings:                n.Headings,
		Subtitle:                n.Subtitle,
		BigPicture:              n.BigPicture,
		LargeIcon:               n.LargeIcon,
		SmallIcon:               n.SmallIcon,
		IosAttachments:          n.IosAttachments,
		AndroidChannelID:        n.AndroidChannelID,
		AndroidAccentColor:      n.AndroidAccentColor,
		AndroidLedColor:         n.AndroidLedColor,
		AndroidGroup:            n.AndroidGroup,
		AndroidGroupMessage:     n.AndroidGroupMessage,
		AndroidSound:            n.AndroidSound,
		IosSound:                n.IosSound,
		WpWnsSound:              n.WpWnsSound,
		AdmSound:                n.AdmSound,
		ChromeWebImage:          n.ChromeWebImage,
		ChromeWebIcon:           n.ChromeWebIcon,
		ChromeWebBadge:          n.ChromeWebBadge,
		ChromeWebColor:          n.ChromeWebColor,
		ChromeWebSound:          n.ChromeWebSound,
		Url:                     n.Url,
		WebUrl:                  n.WebUrl,
		AppUrl:                  n.AppUrl,
		Data:                    n.Data,
		Filters:                 n.Filters,
		Tags:                    n.Tags,
		SendAfter:               n.SendAfter,
		DelayedOption:           n.DelayedOption,
		DeliveryTimeOfDay:       n.DeliveryTimeOfDay,
		Ttl:                     n.Ttl,
		Priority:                n.Priority,
		TargetUserID:            n.TargetUserID,
		SourceServiceID:         n.SourceServiceID,
		SourceUserID:            n.SourceUserID,
		NotificationType:        n.NotificationType,
		OnesignalNotificationID: n.OnesignalNotificationID,
		OnesignalStatus:         n.OnesignalStatus,
		OnesignalResponse:       n.OnesignalResponse,
		OnesignalError:          n.OnesignalError,
	}
}

func (pns *pushNotificationService) preparePushPayload(
	pushNotification repository.Notification,
) (*onesignal.Notification, error) {
	notification := *onesignal.NewNotification(os.Getenv("ONESIGNAL_APP_ID"))
	// set default to push
	notification.SetTargetChannel("push")

	if !pns.hasTargeting(pushNotification) {
		return nil, errors.New(
			"at least one targeting mechanism must be specified (segments, player IDs, external user IDs, email tokens, or phone numbers)",
		)
	}

	// set the heading
	if pushNotification.Headings == nil {
		return nil, errors.New(
			"at least one heading (\"en\") should be specified",
		)
	}
	var rawHeadings map[string]string
	if err := json.Unmarshal(pushNotification.Headings, &rawHeadings); err != nil {
		return nil, err
	}

	if _, exists := rawHeadings["en"]; !exists {
		return nil, fmt.Errorf(
			"Atleast the English heading must be present for a push notification",
		)
	}
	heading := onesignal.NewLanguageStringMap()
	for lang, content := range rawHeadings {
		switch lang {
		case "en":
			heading.SetEn(content)
		}
	}
	notification.SetHeadings(*heading)

	// set the subtitle
	var rawSubtitles map[string]string
	if err := json.Unmarshal(pushNotification.Subtitle, &rawSubtitles); err != nil {
		return nil, err
	}
	subtitle := onesignal.NewLanguageStringMap()
	for lang, content := range rawSubtitles {
		switch lang {
		case "en":
			subtitle.SetEn(content)
		}
	}
	notification.SetSubtitle(*subtitle)

	if pushNotification.Contents == nil {
		return nil, errors.New("contents are required for push notifictions.")
	}
	// set the contents
	var rawContents map[string]string
	if err := json.Unmarshal(pushNotification.Contents, &rawContents); err != nil {
		return nil, err
	}
	contents := onesignal.NewLanguageStringMap()
	for lang, content := range rawContents {
		switch lang {
		case "en":
			contents.SetEn(content)
		}
	}
	notification.SetContents(*contents)
	// set the notification payload
	if pushNotification.Data != nil {
		var rawPayload map[string]any
		if err := json.Unmarshal(pushNotification.Data, &rawPayload); err != nil {
			return nil, err
		}
		notification.SetData(rawPayload)
	}

	if len(pushNotification.IncludedSegments) > 0 {
		notification.SetIncludedSegments(pushNotification.IncludedSegments)
	}
	if len(pushNotification.ExcludedSegments) > 0 {
		notification.SetExcludedSegments(pushNotification.ExcludedSegments)
	}
	if len(pushNotification.IncludeEmailTokens) > 0 {
		notification.SetIncludeEmailTokens(pushNotification.IncludeEmailTokens)
	}
	if len(pushNotification.IncludePhoneNumbers) > 0 {
		notification.SetIncludePhoneNumbers(
			pushNotification.IncludePhoneNumbers,
		)
	}
	if len(pushNotification.IncludeIosTokens) > 0 {
		notification.SetIncludeIosTokens(pushNotification.IncludeIosTokens)
	}
	if len(pushNotification.IncludeWpWnsUris) > 0 {
		notification.SetIncludeWpWnsUris(pushNotification.IncludeWpWnsUris)
	}
	if len(pushNotification.IncludeAmazonRegIds) > 0 {
		notification.SetIncludeAmazonRegIds(
			pushNotification.IncludeAmazonRegIds,
		)
	}
	if len(pushNotification.IncludeChromeRegIds) > 0 {
		notification.SetIncludeChromeRegIds(
			pushNotification.IncludeChromeRegIds,
		)
	}
	if len(pushNotification.IncludeChromeWebRegIds) > 0 {
		notification.SetIncludeChromeWebRegIds(
			pushNotification.IncludeChromeWebRegIds,
		)
	}
	if len(pushNotification.IncludeAndroidRegIds) > 0 {
		notification.SetIncludeAndroidRegIds(
			pushNotification.IncludeAndroidRegIds,
		)
	}

	var targetUsers []string
	if pushNotification.TargetUserID.Valid {
		targetUsers = append(
			targetUsers,
			pushNotification.TargetUserID.String(),
		)
	}

	if pushNotification.IncludeExternalUserIds != nil {
		targetUsers = append(
			targetUsers,
			pushNotification.IncludeExternalUserIds...)
	}

	if len(targetUsers) > 0 {
		notification.SetIncludeAliases(
			map[string][]string{"external_id": targetUsers},
		)
	}

	if pushNotification.AndroidChannelID != nil {
		notification.SetAndroidChannelId(*pushNotification.AndroidChannelID)
	}

	if pushNotification.AndroidSound != nil {
		notification.SetAndroidSound(*pushNotification.AndroidSound)
	}
	if pushNotification.LargeIcon != nil {
		notification.SetLargeIcon(*pushNotification.LargeIcon)
	}
	if pushNotification.SmallIcon != nil {
		notification.SetSmallIcon(*pushNotification.SmallIcon)
	}
	if pushNotification.BigPicture != nil {
		notification.SetBigPicture(*pushNotification.BigPicture)
	}
	if pushNotification.AndroidLedColor != nil {
		notification.SetAndroidLedColor(*pushNotification.AndroidLedColor)
	}

	if pushNotification.IosSound != nil {
		notification.SetIosSound(*pushNotification.IosSound)
	}

	// Urls
	if pushNotification.Url != nil {
		notification.SetUrl(*pushNotification.Url)
	}
	if pushNotification.WebUrl != nil {
		notification.SetWebUrl(*pushNotification.WebUrl)
	}
	if pushNotification.AppUrl != nil {
		notification.SetAppUrl(*pushNotification.AppUrl)
	}

	// set the send at and ttl
	if pushNotification.Ttl != nil {
		if err := pns.validateTTL(*pushNotification.Ttl); err != nil {
			return nil, err
		}
		notification.SetTtl(*pushNotification.Ttl)
	}

	// Set SendAfter if provided and validate it's a future date
	if pushNotification.SendAfter.Valid {
		if err := pns.validateSendAfter(pushNotification.SendAfter.Time); err != nil {
			return nil, err
		}
		notification.SetSendAfter(
			pushNotification.SendAfter.Time,
		)
	}

	if pushNotification.DelayedOption != nil {
		notification.SetDelayedOption(*pushNotification.DelayedOption)
	}

	if pushNotification.Priority != nil {
		notification.SetPriority(*pushNotification.Priority)
	}

	if pushNotification.Buttons != nil { // Assuming you have a Buttons []byte or JSON field
		var rawButtons []notificationButton
		if err := json.Unmarshal(pushNotification.Buttons, &rawButtons); err != nil {
			return nil, fmt.Errorf("failed to unmarshal buttons: %w", err)
		}

		var osButtons []onesignal.Button
		for _, b := range rawButtons {
			btn := *onesignal.NewButton(b.ID)
			btn.SetText(b.Text)
			if b.Icon != "" {
				btn.SetIcon(b.Icon)
			}
			osButtons = append(osButtons, btn)
		}
		notification.SetButtons(osButtons)
	}

	return &notification, nil
}

// Helper: Check if at least one targeting mechanism is specified
func (pns *pushNotificationService) hasTargeting(
	n repository.Notification,
) bool {
	return len(n.IncludedSegments) > 0 ||
		len(n.ExcludedSegments) > 0 ||
		len(n.IncludePlayerIds) > 0 ||
		len(n.IncludeExternalUserIds) > 0 ||
		len(n.IncludeEmailTokens) > 0 ||
		len(n.IncludePhoneNumbers) > 0 ||
		len(n.IncludeIosTokens) > 0 ||
		len(n.IncludeWpWnsUris) > 0 ||
		len(n.IncludeAmazonRegIds) > 0 ||
		len(n.IncludeChromeRegIds) > 0 ||
		len(n.IncludeChromeWebRegIds) > 0 ||
		len(n.IncludeAndroidRegIds) > 0
}

// Helper: Validate TTL is a positive integer
func (pns *pushNotificationService) validateTTL(ttl int32) error {
	if ttl <= 0 {
		return fmt.Errorf("TTL must be positive, got: %d", ttl)
	}
	// Optionally add a reasonable upper bound (e.g., 2592000 seconds = 30 days)
	maxTTL := int32(2592000)
	if ttl > maxTTL {
		return fmt.Errorf(
			"TTL exceeds maximum of %d seconds, got: %d",
			maxTTL,
			ttl,
		)
	}
	return nil
}

// Helper: Validate SendAfter is a future date
func (pns *pushNotificationService) validateSendAfter(
	sendAfter time.Time,
) error {
	if sendAfter.Before(time.Now()) {
		return fmt.Errorf(
			"send_after must be a future date, got: %s",
			sendAfter.Format(time.RFC3339),
		)
	}
	return nil
}
