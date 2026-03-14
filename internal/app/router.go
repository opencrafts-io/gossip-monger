package app

import (
	"net/http"

	"github.com/opencrafts-io/gossip-monger/internal/handlers"
)

func LoadRoutes(gm *GossipMonger) http.Handler {
	router := http.NewServeMux()

	ph := handlers.PingHandler{}

	router.HandleFunc("GET /ping", ph.Ping)
	return router
}
