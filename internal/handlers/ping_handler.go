package handlers

import (
	"encoding/json"
	"net/http"
)

type PingHandler struct {
}

// A health check handler that returns an abitrary message
func (ph *PingHandler) Ping(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]any{
		"message": "In the beginning was the word",
	})
}
