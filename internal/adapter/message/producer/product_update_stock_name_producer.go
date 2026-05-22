package messageproducer

import (
	"product-service/config"
	"product-service/internal/core/domain/model"

	"github.com/IBM/sarama"
)

type ProductUpdateStockNameProducer struct {
	Producer[*model.ProductEvent]
}

func NewProductUpdateStockNameProducer(producer sarama.SyncProducer, cfg *config.Config) *ProductUpdateStockNameProducer {
	return &ProductUpdateStockNameProducer{
		Producer: Producer[*model.ProductEvent]{
			Producer: producer,
			Topic:    cfg.Topic.ProductUpdateStockName,
		},
	}
}
