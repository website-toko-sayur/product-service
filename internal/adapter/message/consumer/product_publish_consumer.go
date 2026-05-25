package messagingconsumer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"product-service/config"
	"product-service/internal/core/domain/model"

	"github.com/IBM/sarama"
	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/rs/zerolog/log"
)

type ProductPublishConsumer struct {
	cfg              *config.Config
	opensearchClient *opensearch.Client
}

func NewProductPublishConsumer(cfg *config.Config, opensearchClient *opensearch.Client) *ProductPublishConsumer {
	return &ProductPublishConsumer{
		cfg:              cfg,
		opensearchClient: opensearchClient,
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

	// insert/update document ke opensearch
	body, err := json.Marshal(ProductEvent)
	if err != nil {
		log.Error().
			Err(err).
			Int64("product_id", ProductEvent.ID).
			Str("source", "internal.adapter.message.ProductPublishConsumer.Consume").
			Msg("Error marshal product event")
		return err
	}

	res, err := c.opensearchClient.Index(
		"products",
		bytes.NewReader(body),
		c.opensearchClient.Index.WithContext(context.Background()),
		c.opensearchClient.Index.WithDocumentID(fmt.Sprintf("%d", ProductEvent.ID)),
	)

	if err != nil {
		log.Error().
			Err(err).
			Int64("product_id", ProductEvent.ID).
			Str("source", "internal.adapter.message.ProductPublishConsumer.Consume").
			Msg("Error indexing product to opensearch")
		return err
	}

	defer res.Body.Close()

	if res.IsError() {
		log.Error().
			Str("status", res.Status()).
			Int64("product_id", ProductEvent.ID).
			Str("source", "internal.adapter.message.ProductPublishConsumer.Consume").
			Msg("Opensearch indexing failed")
		return fmt.Errorf("opensearch indexing failed")
	}

	log.Info().
		Int32("partition", message.Partition).
		Int64("product_id", ProductEvent.ID).
		Str("topic", c.cfg.Topic.ProductPublish).
		Str("source", "internal.adapter.message.ProductPublishConsumer.Consume").
		Msg("Success indexing product to opensearch")

	return nil
}
