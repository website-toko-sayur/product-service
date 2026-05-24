package app

import (
	"context"
	"os"
	"os/signal"
	"product-service/config"
	messagingconsumer "product-service/internal/adapter/message/consumer"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
)

func RunWorker() {
	cfg := config.NewConfig()

	config.NewLogger(cfg.App.AppEnv, cfg.App.LogLevel, cfg.App.AppName)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go RunProductDeleteConsumer(cfg, ctx)
	go RunProductPublishConsumer(cfg, ctx)

	terminateSignals := make(chan os.Signal, 1)
	signal.Notify(terminateSignals, syscall.SIGINT, syscall.SIGTERM)

	stop := false
	for !stop {
		select {
		case s := <-terminateSignals:
			log.Info().
				Str("signal", s.String()).
				Str("source", "internal.app.RunWorker").
				Msg("got stop signal, shutting down worker gracefully")

			cancel()
			stop = true

		case <-ctx.Done():
			log.Info().
				Str("source", "internal.app.RunWorker").
				Msg("context cancelled")

			stop = true
		}
	}

	time.Sleep(5 * time.Second) // wait for all consumers to finish processing
}

func RunProductDeleteConsumer(cfg *config.Config, ctx context.Context) {
	log.Info().
		Str("source", "internal.app.RunWorker").
		Msg("setup product delete consumer")

	productDeleteConsumerGroup := cfg.NewKafkaConsumerGroup()
	productDeleteHandler := messagingconsumer.NewProductDeleteConsumer(cfg)
	messagingconsumer.ConsumeTopic(ctx, productDeleteConsumerGroup, cfg.Topic.ProductDelete, productDeleteHandler.Consume)
}

func RunProductPublishConsumer(cfg *config.Config, ctx context.Context) {
	log.Info().
		Str("source", "internal.app.RunWorker").
		Msg("setup product publish consumer")

	productPublishConsumerGroup := cfg.NewKafkaConsumerGroup()
	productPublishHandler := messagingconsumer.NewProductPublishConsumer(cfg)
	messagingconsumer.ConsumeTopic(ctx, productPublishConsumerGroup, cfg.Topic.ProductDelete, productPublishHandler.Consume)
}
