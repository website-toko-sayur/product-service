package messagingconsumer

import (
	"encoding/json"
	"errors"
	"product-service/config"
	"product-service/internal/core/domain/model"
	"strconv"

	"github.com/IBM/sarama"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/rs/zerolog/log"
)

type ProductDeleteConsumer struct {
	cfg      *config.Config
	esClient *elasticsearch.Client
}

func NewProductDeleteConsumer(cfg *config.Config, esClient *elasticsearch.Client) *ProductDeleteConsumer {
	return &ProductDeleteConsumer{
		cfg:      cfg,
		esClient: esClient,
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

	res, err := c.esClient.Delete(
		"products",
		strconv.Itoa(int(DeleteProductEvent.ProductID)),
		c.esClient.Delete.WithRefresh("true"),
	)

	if err != nil {
		log.Error().
			Err(err).
			Int64("product_id", DeleteProductEvent.ProductID).
			Str("source", "internal.adapter.message.ProductDeleteConsumer.Consume").
			Msg("failed delete product document from elasticsearch")

		return err
	}

	defer res.Body.Close()

	if res.IsError() {
		err = errors.New(res.String())

		log.Error().
			Err(err).
			Int64("product_id", DeleteProductEvent.ProductID).
			Str("source", "internal.adapter.message.ProductDeleteConsumer.Consume").
			Msg("elasticsearch returned delete error")

		return err
	}

	log.Info().
		Int64("product_id", DeleteProductEvent.ProductID).
		Str("source", "internal.adapter.message.ProductDeleteConsumer.Consume").
		Msg("success delete product document from elasticsearch")

	return nil
}
