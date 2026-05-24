package repository

import (
	"context"
	"errors"
	"product-service/internal/core/domain/entity"
	"product-service/internal/core/domain/model"

	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type productRepository struct {
	db *gorm.DB
}

type ProductRepositoryInterface interface {
	// GetAll(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error)
	// GetByID(ctx context.Context, productID int64) (*entity.ProductEntity, error)
	// Create(ctx context.Context, req entity.ProductEntity) (int64, error)
	Update(ctx context.Context, req entity.ProductEntity) error
	Delete(ctx context.Context, productID int64) error
	// SearchProducts(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error)
}

func NewProductRepository(db *gorm.DB) ProductRepositoryInterface {
	return &productRepository{db: db}
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

	return nil
}

func (p *productRepository) Update(ctx context.Context, req entity.ProductEntity) error {
	tx := p.db.WithContext(ctx).Begin()

	modelProduct := model.Product{}

	err := tx.Where("id = ?", req.ID).First(&modelProduct).Error
	if err != nil {
		tx.Rollback()

		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Error().
				Int64("product_id", req.ID).
				Str("source", "internal.adapter.productRepository.Update").
				Msg("product not found")

			return errors.New("404")
		}

		log.Error().
			Err(err).
			Int64("product_id", req.ID).
			Str("source", "internal.adapter.productRepository.Update").
			Msg("failed get product")

		return err
	}

	modelProduct.CategorySlug = req.CategorySlug
	modelProduct.ParentID = req.ParentID
	modelProduct.Name = req.Name
	modelProduct.Image = req.Image
	modelProduct.Description = req.Description
	modelProduct.RegulerPrice = req.RegulerPrice
	modelProduct.SalePrice = req.SalePrice
	modelProduct.Unit = req.Unit
	modelProduct.Weight = req.Weight
	modelProduct.Stock = req.Stock
	modelProduct.Variant = req.Variant
	modelProduct.Status = req.Status

	err = tx.Save(&modelProduct).Error
	if err != nil {
		tx.Rollback()

		log.Error().
			Err(err).
			Int64("product_id", req.ID).
			Str("source", "internal.adapter.productRepository.Update").
			Msg("failed update product")

		return err
	}

	if len(req.Child) > 0 {
		// hapus seluruh child product lama berdasarkan parent_id
		// supaya data variant/child sebelumnya tidak duplicate
		// dan akan diganti dengan data child terbaru dari request
		err = tx.Where("parent_id = ?", modelProduct.ID).Delete(&model.Product{}).Error
		if err != nil {
			tx.Rollback()

			log.Error().
				Err(err).
				Int64("product_id", req.ID).
				Str("source", "internal.adapter.productRepository.Update").
				Msg("failed delete old child products")

			return err
		}

		// tampung child product baru dari request
		modelProductChild := []model.Product{}

		for _, val := range req.Child {

			// rebuild ulang seluruh child product
			// menggunakan data terbaru dari request update
			modelProductChild = append(modelProductChild, model.Product{
				CategorySlug: req.CategorySlug,
				ParentID:     &modelProduct.ID,
				Name:         req.Name,
				Image:        val.Image,
				Description:  req.Description,
				RegulerPrice: val.RegulerPrice,
				SalePrice:    val.SalePrice,
				Unit:         req.Unit,
				Weight:       val.Weight,
				Stock:        val.Stock,
				Variant:      req.Variant,
				Status:       req.Status,
			})
		}

		// insert ulang seluruh child product baru ke database
		err = tx.Create(&modelProductChild).Error
		if err != nil {
			tx.Rollback()

			log.Error().
				Err(err).
				Int64("product_id", req.ID).
				Str("source", "internal.adapter.productRepository.Update").
				Msg("failed create child products")

			return err
		}
	}

	err = tx.Commit().Error
	if err != nil {
		tx.Rollback()

		log.Error().
			Err(err).
			Int64("product_id", req.ID).
			Str("source", "internal.adapter.productRepository.Update").
			Msg("failed commit update product transaction")

		return err
	}

	log.Info().
		Int64("product_id", req.ID).
		Str("source", "internal.adapter.productRepository.Update").
		Msg("success update product")

	return nil
}

func (p *productRepository) Create(ctx context.Context, req entity.ProductEntity) (int64, error) {

	tx := p.db.WithContext(ctx).Begin()

	modelProduct := model.Product{
		CategorySlug: req.CategorySlug,
		ParentID:     req.ParentID,
		Name:         req.Name,
		Image:        req.Image,
		Description:  req.Description,
		RegulerPrice: req.RegulerPrice,
		SalePrice:    req.SalePrice,
		Unit:         req.Unit,
		Weight:       req.Weight,
		Stock:        req.Stock,
		Variant:      req.Variant,
		Status:       req.Status,
	}

	err := tx.Create(&modelProduct).Error
	if err != nil {
		tx.Rollback()

		log.Error().
			Err(err).
			Str("product_name", req.Name).
			Str("source", "internal.adapter.productRepository.Create").
			Msg("failed create product")

		return 0, err
	}

	if len(req.Child) > 0 {

		// build child products from request
		modelProductChild := []model.Product{}

		for _, val := range req.Child {
			modelProductChild = append(modelProductChild, model.Product{
				CategorySlug: req.CategorySlug,
				ParentID:     &modelProduct.ID,
				Name:         req.Name,
				Image:        val.Image,
				Description:  req.Description,
				RegulerPrice: val.RegulerPrice,
				SalePrice:    val.SalePrice,
				Unit:         req.Unit,
				Weight:       val.Weight,
				Stock:        val.Stock,
				Variant:      req.Variant,
				Status:       req.Status,
			})
		}

		err = tx.Create(&modelProductChild).Error
		if err != nil {
			tx.Rollback()

			log.Error().
				Err(err).
				Int64("product_id", modelProduct.ID).
				Str("source", "internal.adapter.productRepository.Create").
				Msg("failed create child products")

			return 0, err
		}
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()

		log.Error().
			Err(err).
			Str("source", "internal.adapter.productRepository.Create").
			Msg("failed commit create product transaction")

		return 0, err
	}

	log.Info().
		Int64("product_id", modelProduct.ID).
		Str("product_name", modelProduct.Name).
		Str("source", "internal.adapter.productRepository.Create").
		Msg("success create product")

	return modelProduct.ID, nil
}
