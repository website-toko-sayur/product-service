package messagingconsumer

import (
	"encoding/json"
	"errors"
	"product-service/config"
	"product-service/internal/core/domain/model"
	"strconv"

	"github.com/IBM/sarama"
	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/rs/zerolog/log"
)

type ProductDeleteConsumer struct {
	cfg              *config.Config
	opensearchClient *opensearch.Client
}

func NewProductDeleteConsumer(cfg *config.Config, opensearchClient *opensearch.Client) *ProductDeleteConsumer {
	return &ProductDeleteConsumer{
		cfg:              cfg,
		opensearchClient: opensearchClient,
	}
}

func (c ProductDeleteConsumer) Consume(message *sarama.ConsumerMessage) error {
	DeleteProductEvent := new(model.DeleteProductEvent)
	if err := json.Unmarshal(message.Value, DeleteProductEvent); err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.adapter.message.ProductDeleteConsumer.Consume").
			Msg("Error unmarshalling Delete Product event")
		return err
	}

	// TODO process event
	log.Info().
		Int32("partition", message.Partition).
		Interface("event", DeleteProductEvent).
		Str("topic", c.cfg.Topic.ProductDelete).
		Str("source", "internal.adapter.message.ProductDeleteConsumer.Consume").
		Msg("Received Delete Product event")

	res, err := c.opensearchClient.Delete(
		"products",
		strconv.Itoa(int(DeleteProductEvent.ProductID)),
		c.opensearchClient.Delete.WithRefresh("true"),
	)

	if err != nil {
		log.Error().
			Err(err).
			Int64("product_id", DeleteProductEvent.ProductID).
			Str("source", "internal.adapter.message.ProductDeleteConsumer.Consume").
			Msg("failed delete product document from opensearch")

		return err
	}

	defer res.Body.Close()

	if res.IsError() {
		err = errors.New(res.String())

		log.Error().
			Err(err).
			Int64("product_id", DeleteProductEvent.ProductID).
			Str("source", "internal.adapter.message.ProductDeleteConsumer.Consume").
			Msg("opensearch returned delete error")

		return err
	}

	log.Info().
		Int64("product_id", DeleteProductEvent.ProductID).
		Str("source", "internal.adapter.message.ProductDeleteConsumer.Consume").
		Msg("success delete product document from opensearch")

	return nil
}
