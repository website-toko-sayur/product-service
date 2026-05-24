package messagingconsumer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"product-service/config"
	"product-service/internal/core/domain/model"

	"github.com/IBM/sarama"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/rs/zerolog/log"
)

type ProductPublishConsumer struct {
	cfg      *config.Config
	esClient *elasticsearch.Client
}

func NewProductPublishConsumer(cfg *config.Config, esClient *elasticsearch.Client) *ProductPublishConsumer {
	return &ProductPublishConsumer{
		cfg:      cfg,
		esClient: esClient,
	}
}

func (c ProductPublishConsumer) Consume(message *sarama.ConsumerMessage) error {
	ProductEvent := new(model.ProductEvent)
	if err := json.Unmarshal(message.Value, ProductEvent); err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.adapter.message.ProductPublishConsumer.Consume").
			Msg("Error unmarshalling Product event")
		return err
	}

	// TODO process event
	log.Info().
		Int32("partition", message.Partition).
		Interface("event", ProductEvent).
		Str("topic", c.cfg.Topic.ProductPublish).
		Str("source", "internal.adapter.message.ProductPublishConsumer.Consume").
		Msg("Received Product event")

	// insert/update document ke elasticsearch
	body, err := json.Marshal(ProductEvent)
	if err != nil {
		log.Error().
			Err(err).
			Int64("product_id", ProductEvent.ID).
			Str("source", "internal.adapter.message.ProductPublishConsumer.Consume").
			Msg("Error marshal product event")
		return err
	}

	res, err := c.esClient.Index(
		"products",
		bytes.NewReader(body),
		c.esClient.Index.WithContext(context.Background()),
		c.esClient.Index.WithDocumentID(fmt.Sprintf("%d", ProductEvent.ID)),
	)

	if err != nil {
		log.Error().
			Err(err).
			Int64("product_id", ProductEvent.ID).
			Str("source", "internal.adapter.message.ProductPublishConsumer.Consume").
			Msg("Error indexing product to elasticsearch")
		return err
	}

	defer res.Body.Close()

	if res.IsError() {
		log.Error().
			Str("status", res.Status()).
			Int64("product_id", ProductEvent.ID).
			Str("source", "internal.adapter.message.ProductPublishConsumer.Consume").
			Msg("Elasticsearch indexing failed")
		return fmt.Errorf("elasticsearch indexing failed")
	}

	log.Info().
		Int32("partition", message.Partition).
		Int64("product_id", ProductEvent.ID).
		Str("topic", c.cfg.Topic.ProductPublish).
		Str("source", "internal.adapter.message.ProductPublishConsumer.Consume").
		Msg("Success indexing product to elasticsearch")

	return nil
}
