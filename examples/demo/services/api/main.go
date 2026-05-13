package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	"github.com/sentiolabs/open-events/examples/demo/services/api/publisher"
	"github.com/sentiolabs/open-events/examples/demo/services/api/server"
)

func main() {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	queueURL := os.Getenv("OPENEVENTS_QUEUE_URL")
	if queueURL == "" {
		logger.Error("OPENEVENTS_QUEUE_URL is required")
		os.Exit(1)
	}
	addr := os.Getenv("OPENEVENTS_API_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		logger.Error("aws config", "err", err)
		os.Exit(1)
	}
	client := sqs.NewFromConfig(cfg)
	pub := &publisher.SQSPublisher{Client: client, QueueURL: queueURL}

	e := server.New(pub, queueURL)
	logger.Info("api listening", "addr", addr, "queue_url", queueURL)
	if err := e.Start(addr); err != nil {
		logger.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
