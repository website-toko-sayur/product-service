package repository

import (
	"context"
	"errors"
	"product-service/internal/core/domain/model"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type productRepository struct {
	db       *gorm.DB
	esClient *elasticsearch.Client
}

type ProductRepositoryInterface interface {
	// GetAll(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error)
	// GetByID(ctx context.Context, productID int64) (*entity.ProductEntity, error)
	// Create(ctx context.Context, req entity.ProductEntity) (int64, error)
	// Update(ctx context.Context, req entity.ProductEntity) error
	Delete(ctx context.Context, productID int64) error
	// SearchProducts(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error)
}

func NewProductRepository(db *gorm.DB, esClient *elasticsearch.Client) ProductRepositoryInterface {
	return &productRepository{db: db, esClient: esClient}
}

func (p *productRepository) Delete(ctx context.Context, productID int64) error {
	modelProduct := model.Product{}

	if err := p.db.WithContext(ctx).Preload("Childs").First(&modelProduct, "id = ?", productID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = errors.New("404")
			log.Info().
				Int64("product_id", productID).
				Str("source", "internal.adapter.productRepository.Delete").
				Msg("product not found")
			return err
		}
		log.Error().
			Err(err).
			Int64("product_id", productID).
			Str("source", "internal.adapter.productRepository.Delete").
			Msg("failed to get product")
		return err
	}

	if err := p.db.WithContext(ctx).Select("Childs").Delete(&modelProduct).Error; err != nil {
		log.Error().
			Err(err).
			Int64("product_id", productID).
			Str("source", "internal.adapter.productRepository.Delete").
			Msg("failed to delete product from database")
		return err
	}

	// delete document di elasticsearch
	// res, err := p.esClient.Delete(
	// 	"products",
	// 	strconv.Itoa(int(productID)),
	// 	p.esClient.Delete.WithRefresh("true"),
	// )
	// if err != nil {
	// 	log.Error().
	// 		Err(err).
	// 		Int64("product_id", productID).
	// 		Str("source", "internal.adapter.productRepository.Delete").
	// 		Msg("failed to delete product from elasticsearch")
	// 	return err
	// }

	// defer res.Body.Close()
	// if res.IsError() {
	// 	log.Error().
	// 		Str("status", res.Status()).
	// 		Int64("product_id", productID).
	// 		Str("source", "internal.adapter.productRepository.Delete").
	// 		Msg("elasticsearch returned error response")

	// 	return errors.New("failed delete product from elasticsearch")
	// }

	// log.Info().
	// 	Int64("product_id", productID).
	// 	Str("source", "internal.adapter.productRepository.Delete").
	// 	Msg("success delete product from elasticsearch")

	return nil
}
