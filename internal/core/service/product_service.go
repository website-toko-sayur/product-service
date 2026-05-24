package service

import (
	"context"
	messageproducer "product-service/internal/adapter/message/producer"
	"product-service/internal/adapter/repository"
	"product-service/internal/core/domain/model"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"
)

type productService struct {
	repo                  repository.ProductRepositoryInterface
	repoCat               repository.CategoryRepositoryInterface
	productDeleteProducer *messageproducer.ProductDeleteProducer
}

type ProductServiceInterface interface {
	// GetAll(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error)
	// GetByID(ctx context.Context, productID int64) (*entity.ProductEntity, error)
	// Create(ctx context.Context, req entity.ProductEntity) error
	// Update(ctx context.Context, req entity.ProductEntity) error
	Delete(ctx context.Context, productID int64) error
	// SearchProducts(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error)
}

func NewProductService(repo repository.ProductRepositoryInterface, repoCat repository.CategoryRepositoryInterface,
	productDeleteProducer *messageproducer.ProductDeleteProducer) ProductServiceInterface {
	return &productService{
		repo:                  repo,
		repoCat:               repoCat,
		productDeleteProducer: productDeleteProducer,
	}
}

func (p *productService) Delete(ctx context.Context, productID int64) error {
	err := p.repo.Delete(ctx, productID)
	if err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.core.productService.Delete")
		return err
	}

	if p.productDeleteProducer != nil {
		event := &model.DeleteProductEvent{
			ProductID: productID,
		}

		log.Info().
			Str("source", "internal.core.productService.Delete").
			Msg("Publishing product delete event")

		if err = p.productDeleteProducer.Send(event); err != nil {
			log.Warn().
				Err(err).
				Str("source", "internal.core.productService.Delete").
				Msg("Failed publish product delete event")
			return fiber.ErrInternalServerError
		} else {
			log.Info().
				Str("source", "internal.core.productService.Delete").
				Msg("Kafka producer is disabled, skipping product delete event")
		}
	}

	return nil
}
