package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/opencrafts-io/gossip-monger/internal/app"
	"github.com/opencrafts-io/gossip-monger/internal/config"
)

func main() {
	var logger *slog.Logger
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{}))

	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Error("Failed to load configuration files for the epic gossip monger.",
			slog.Any("error", err),
		)
		panic(err)
	}

	gossipMonger, err := app.NewGossipMongerApp(logger, cfg)
	if err != nil {
		logger.Error("Failed to create and initialize the epic gossip monger service.",
			slog.Any("error", err),
		)
		panic(err)
	}

	if err = gossipMonger.Start(context.Background()); err != nil {
		logger.Error("Failed to start the epic gossip monger service.", slog.Any("error", err))
		panic(err)
	}
}
