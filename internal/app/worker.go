package app

import (
	"context"
	"os"
	"os/signal"
	"product-service/config"
	messagingconsumer "product-service/internal/adapter/message/consumer"
	"sync"
	"syscall"

	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/rs/zerolog/log"
)

func RunWorker() {
	cfg := config.NewConfig()

	config.NewLogger(cfg.App.AppEnv, cfg.App.LogLevel, cfg.App.AppName)

	opensearchClient, err := cfg.NewOpenSearch()
	if err != nil {
		log.Fatal().
			Err(err).
			Str("source", "internal.app.RunWorker").
			Msg("failed initialize opensearch client")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(2)

	go RunProductDeleteConsumer(cfg, ctx, &wg, opensearchClient)
	go RunProductPublishConsumer(cfg, ctx, &wg, opensearchClient)

	terminateSignals := make(chan os.Signal, 1)

	signal.Notify(
		terminateSignals,
		syscall.SIGINT,
		syscall.SIGTERM,
	)

	select {
	case s := <-terminateSignals:
		log.Info().
			Str("signal", s.String()).
			Str("source", "internal.app.RunWorker").
			Msg("got stop signal, shutting down worker gracefully")

		cancel()

	case <-ctx.Done():
		log.Info().
			Str("source", "internal.app.RunWorker").
			Msg("worker context cancelled")
	}

	log.Info().
		Str("source", "internal.app.RunWorker").
		Msg("waiting all consumers to stop")

	wg.Wait()

	log.Info().
		Str("source", "internal.app.RunWorker").
		Msg("all consumers stopped gracefully")
}

func RunProductDeleteConsumer(cfg *config.Config, ctx context.Context, wg *sync.WaitGroup, opensearchClient *opensearch.Client) {
	defer wg.Done()

	log.Info().
		Str("source", "internal.app.RunWorker").
		Msg("setup product delete consumer")

	productDeleteConsumerGroup := cfg.NewKafkaConsumerGroup()
	defer productDeleteConsumerGroup.Close()

	productDeleteHandler := messagingconsumer.NewProductDeleteConsumer(
		cfg,
		opensearchClient,
	)

	messagingconsumer.ConsumeTopic(
		ctx,
		productDeleteConsumerGroup,
		cfg.Topic.ProductDelete,
		productDeleteHandler.Consume,
	)
}

func RunProductPublishConsumer(cfg *config.Config, ctx context.Context, wg *sync.WaitGroup, opensearchClient *opensearch.Client) {
	defer wg.Done()

	log.Info().
		Str("source", "internal.app.RunWorker").
		Msg("setup product publish consumer")

	productPublishConsumerGroup := cfg.NewKafkaConsumerGroup()
	defer productPublishConsumerGroup.Close()

	productPublishHandler := messagingconsumer.NewProductPublishConsumer(
		cfg,
		opensearchClient,
	)

	messagingconsumer.ConsumeTopic(
		ctx,
		productPublishConsumerGroup,
		cfg.Topic.ProductPublish,
		productPublishHandler.Consume,
	)
}
