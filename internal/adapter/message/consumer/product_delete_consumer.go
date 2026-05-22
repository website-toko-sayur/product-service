package messagingconsumer

import (
	"encoding/json"
	"product-service/config"
	"product-service/internal/core/domain/model"

	"github.com/IBM/sarama"
	"github.com/rs/zerolog/log"
)

type ProductDeleteConsumer struct {
	cfg *config.Config
}

func NewProductDeleteConsumer(cfg *config.Config) *ProductDeleteConsumer {
	return &ProductDeleteConsumer{
		cfg: cfg,
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

	log.Info().
		Int32("partition", message.Partition).
		Interface("event", DeleteProductEvent).
		Str("topic", c.cfg.Topic.ProductDelete).
		Str("source", "internal.adapter.message.ProductDeleteConsumer.Consume").
		Msg("Received Delete Product event")
	return nil
}
