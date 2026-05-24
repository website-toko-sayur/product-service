package messagingconsumer

import (
	"encoding/json"
	"product-service/config"
	"product-service/internal/core/domain/model"

	"github.com/IBM/sarama"
	"github.com/rs/zerolog/log"
)

type ProductPublishConsumer struct {
	cfg *config.Config
}

func NewProductPublishConsumer(cfg *config.Config) *ProductPublishConsumer {
	return &ProductPublishConsumer{
		cfg: cfg,
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
	return nil
}
