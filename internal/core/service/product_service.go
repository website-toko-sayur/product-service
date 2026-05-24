package service

import (
	"context"
	"errors"
	messageproducer "product-service/internal/adapter/message/producer"
	"product-service/internal/adapter/repository"
	"product-service/internal/core/domain/entity"
	"product-service/internal/core/domain/model"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"
)

type productService struct {
	repo                   repository.ProductRepositoryInterface
	repoCat                repository.CategoryRepositoryInterface
	productDeleteProducer  *messageproducer.ProductDeleteProducer
	productPublishProducer *messageproducer.ProductPublishProducer
	// productUpdateProducer  *messageproducer.ProductUpdateStockProducer
}

type ProductServiceInterface interface {
	GetAll(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error)
	GetByID(ctx context.Context, productID int64) (*entity.ProductEntity, error)
	Create(ctx context.Context, req entity.ProductEntity) error
	Update(ctx context.Context, req entity.ProductEntity) error
	Delete(ctx context.Context, productID int64) error
	SearchProducts(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error)
}

func NewProductService(repo repository.ProductRepositoryInterface, repoCat repository.CategoryRepositoryInterface,
	productDeleteProducer *messageproducer.ProductDeleteProducer,
	productPublishProducer *messageproducer.ProductPublishProducer) ProductServiceInterface {
	return &productService{
		repo:                   repo,
		repoCat:                repoCat,
		productDeleteProducer:  productDeleteProducer,
		productPublishProducer: productPublishProducer,
	}
}

func (p *productService) SearchProducts(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error) {
	return p.repo.SearchProducts(ctx, query)
}

func (p *productService) Create(ctx context.Context, req entity.ProductEntity) error {
	productID, err := p.repo.Create(ctx, req)
	if err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.core.service.productService.Create").
			Msg("failed create product")
		return err
	}

	getProductByID, err := p.GetByID(ctx, productID)
	if err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.core.service.productService.Create").
			Msg("failed get product after create")
		return err
	}

	if p.productPublishProducer != nil {
		event := &model.ProductEvent{
			ID:           getProductByID.ID,
			CategorySlug: getProductByID.CategorySlug,
			ParentID:     getProductByID.ParentID,
			Name:         getProductByID.Name,
			Image:        getProductByID.Image,
			Description:  getProductByID.Description,
			RegulerPrice: getProductByID.RegulerPrice,
			SalePrice:    getProductByID.SalePrice,
			Unit:         getProductByID.Unit,
			Weight:       getProductByID.Weight,
			Stock:        getProductByID.Stock,
			Variant:      getProductByID.Variant,
			Status:       getProductByID.Status,
			CategoryName: getProductByID.CategoryName,
			Child:        model.MapProductChildren(getProductByID.Child),
			CreatedAt:    getProductByID.CreatedAt,
		}

		log.Info().
			Str("source", "internal.core.productService.Create").
			Msg("Publishing product publish event")

		if err = p.productPublishProducer.Send(event); err != nil {
			log.Warn().
				Err(err).
				Str("source", "internal.core.productService.Create").
				Msg("Failed publish product publish event")
			return fiber.ErrInternalServerError
		} else {
			log.Info().
				Str("source", "internal.core.productService.Create").
				Msg("Kafka producer is disabled, skipping product publish event")
		}
	}

	return nil
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

func (p *productService) GetAll(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error) {
	return p.repo.GetAll(ctx, query)
}

func (p *productService) GetByID(ctx context.Context, productID int64) (*entity.ProductEntity, error) {
	result, err := p.repo.GetByID(ctx, productID)
	if err != nil {
		log.Error().
			Err(err).
			Int64("product_id", productID).
			Str("source", "internal.core.service.productService.GetByID").
			Msg("Failed get product by id")
		return nil, err
	}

	resultCat, err := p.repoCat.GetBySlug(ctx, result.CategorySlug)
	if err != nil {
		log.Error().
			Err(err).
			Int64("product_id", productID).
			Str("category_slug", result.CategorySlug).
			Str("source", "internal.core.service.productService.GetByID").
			Msg("Failed get category by slug")
		return nil, err
	}
	if resultCat == nil {
		log.Error().
			Int64("product_id", productID).
			Str("category_slug", result.CategorySlug).
			Str("source", "internal.core.service.productService.GetByID").
			Msg("Category not found")
		return nil, errors.New("category not found")
	}
	result.CategoryName = resultCat.Name
	return result, nil
}

func (p *productService) Update(ctx context.Context, req entity.ProductEntity) error {
	err := p.repo.Update(ctx, req)
	if err != nil {
		log.Error().
			Err(err).
			Int64("product_id", req.ID).
			Str("source", "internal.core.service.productService.Update").
			Msg("Failed update product")
		return err
	}

	getProductByID, err := p.GetByID(ctx, req.ID)
	if err != nil {
		log.Error().
			Err(err).
			Int64("product_id", req.ID).
			Str("source", "internal.core.service.productService.Update").
			Msg("Failed get product after update")
	}

	if p.productPublishProducer != nil {
		event := &model.ProductEvent{
			ID:           getProductByID.ID,
			CategorySlug: getProductByID.CategorySlug,
			ParentID:     getProductByID.ParentID,
			Name:         getProductByID.Name,
			Image:        getProductByID.Image,
			Description:  getProductByID.Description,
			RegulerPrice: getProductByID.RegulerPrice,
			SalePrice:    getProductByID.SalePrice,
			Unit:         getProductByID.Unit,
			Weight:       getProductByID.Weight,
			Stock:        getProductByID.Stock,
			Variant:      getProductByID.Variant,
			Status:       getProductByID.Status,
			CategoryName: getProductByID.CategoryName,
			Child:        model.MapProductChildren(getProductByID.Child),
			CreatedAt:    getProductByID.CreatedAt,
		}

		log.Info().
			Str("source", "internal.core.productService.Update").
			Msg("Publishing product publish event")

		if err = p.productPublishProducer.Send(event); err != nil {
			log.Warn().
				Err(err).
				Str("source", "internal.core.productService.Update").
				Msg("Failed publish product publihs event")
			return fiber.ErrInternalServerError
		} else {
			log.Info().
				Str("source", "internal.core.productService.Update").
				Msg("Kafka producer is disabled, skipping product publish event")
		}
	}

	return nil
}
