package service

import (
	"time"

	"github.com/opencrafts-io/gossip-monger/internal/repository"
)

type UserEventMetadata struct {
	EventType       string    `json:"event_type"`
	Timestamp       time.Time `json:"timestamp"`
	SourceServiceID string    `json:"source_service_id"`
	RequestID       string    `json:"request_id"`
}

type UserEvent struct {
	User     repository.User   `json:"user"`
	Metadata UserEventMetadata `json:"meta"`
}
