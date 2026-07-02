package messageproducer

import (
	"product-service/config"
	"product-service/internal/core/domain/model"

	"github.com/IBM/sarama"
)

type ProductDeleteProducer struct {
	Producer[*model.DeleteProductEvent]
}

func NewProductDeleteProducer(producer sarama.SyncProducer, cfg *config.Config) *ProductDeleteProducer {
	return &ProductDeleteProducer{
		Producer: Producer[*model.DeleteProductEvent]{
			Producer: producer,
			Topic:    cfg.Topic.ProductDelete,
		},
	}
}
