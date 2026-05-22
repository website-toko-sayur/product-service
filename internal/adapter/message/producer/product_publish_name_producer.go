package messageproducer

import (
	"product-service/config"
	"product-service/internal/core/domain/model"

	"github.com/IBM/sarama"
)

type ProductPublishNameProducer struct {
	Producer[*model.ProductEvent]
}

func NewProductPublishNameProducer(producer sarama.SyncProducer, cfg *config.Config) *ProductPublishNameProducer {
	return &ProductPublishNameProducer{
		Producer: Producer[*model.ProductEvent]{
			Producer: producer,
			Topic:    cfg.Topic.ProductPublishName,
		},
	}
}
