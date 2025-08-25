package eventbus

import (
	"github.com/opencrafts-io/gossip-monger/internal/repository"
)

// UserEvent defines the payload for user-related events.
type UserEvent struct {
	User     repository.User `json:"user"`
	Metadata map[string]any  `json:"meta"`
}
