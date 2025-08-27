package eventbus

import "time"

// NotificationEventMetadata contains crucial information about the event itself.
type NotificationEventMetadata struct {
	EventType       string    `json:"event_type"`
	Timestamp       time.Time `json:"timestamp"`
	SourceServiceID string    `json:"source_service_id"`
	RequestID       string    `json:"request_id"`
}

type NotificationEvent struct {
	Metadata NotificationEventMetadata `json:"meta"`
}
