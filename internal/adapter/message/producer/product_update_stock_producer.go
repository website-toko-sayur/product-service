package messageproducer

// import (
// 	"product-service/config"
// 	"product-service/internal/core/domain/model"

// 	"github.com/IBM/sarama"
// )

// type ProductUpdateStockProducer struct {
// 	Producer[*model.ProductEvent]
// }

// func NewProductUpdateStockProducer(producer sarama.SyncProducer, cfg *config.Config) *ProductUpdateStockProducer {
// 	return &ProductUpdateStockProducer{
// 		Producer: Producer[*model.ProductEvent]{
// 			Producer: producer,
// 			Topic:    cfg.Topic.ProductUpdateStock,
// 		},
// 	}
// }
