package messageproducer

import (
	"product-service/config"
	"product-service/internal/core/domain/model"

	"github.com/IBM/sarama"
)

type ProductPublishProducer struct {
	Producer[*model.ProductEvent]
}

func NewProductPublishProducer(producer sarama.SyncProducer, cfg *config.Config) *ProductPublishProducer {
	return &ProductPublishProducer{
		Producer: Producer[*model.ProductEvent]{
			Producer: producer,
			Topic:    cfg.Topic.ProductPublish,
		},
	}
}
