package service

import (
	"time"
)

type EmailEventMetadata struct {
	EventType       string    `json:"event_type"`
	Timestamp       time.Time `json:"timestamp"`
	SourceServiceID string    `json:"source_service_id"`
	RequestID       string    `json:"request_id"`
}

type EmailEvent struct {
	Meta EmailEventMetadata `json:"metadata"`
}
