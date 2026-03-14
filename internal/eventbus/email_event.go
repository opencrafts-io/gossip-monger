package eventbus

import (
	"time"

	"github.com/google/uuid"
)

// EmailEventMetadata contains crucial information about the event itself.
type EmailEventMetadata struct {
	EventType       string    `json:"event_type"`
	Timestamp       time.Time `json:"timestamp"`
	SourceServiceID string    `json:"source_service_id"`
	RequestID       string    `json:"request_id"`
}

type EmailEvent struct {
	Subject         *string                   `json:"subject"`
	Body            *string                   `json:"body"`
	ReplyTo         *string                   `json:"reply_to"`
	SourceServiceID *string                   `json:"source_service_id"`
	SourceUserID    uuid.UUID                 `json:"source_user_id"`
	To              []string                  `json:"to"`
	Cc              []string                  `json:"cc"`
	Bcc             []string                  `json:"bcc"`
	Metadata        NotificationEventMetadata `json:"meta"`
}
