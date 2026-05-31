package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"product-service/internal/core/domain/entity"
	"product-service/internal/core/domain/model"
	helperSearch "product-service/utils/searchbuilder"
	"strings"

	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type productRepository struct {
	db               *gorm.DB
	opensearchClient *opensearch.Client
}

type ProductRepositoryInterface interface {
	GetAll(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error)
	GetByID(ctx context.Context, productID int64) (*entity.ProductEntity, error)
	Create(ctx context.Context, req entity.ProductEntity) (int64, error)
	Update(ctx context.Context, req entity.ProductEntity) error
	Delete(ctx context.Context, productID int64) error
	SearchProducts(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error)
}

func NewProductRepository(db *gorm.DB, opensearchClient *opensearch.Client) ProductRepositoryInterface {
	return &productRepository{
		db:               db,
		opensearchClient: opensearchClient,
	}
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

	return modelProduct.ID, nil
}

func (p *productRepository) GetByID(ctx context.Context, productID int64) (*entity.ProductEntity, error) {
	modelProduct := model.Product{}

	err := p.db.WithContext(ctx).Preload("Category").First(&modelProduct, "id = ?", productID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Error().
				Int64("product_id", productID).
				Str("source", "internal.adapter.productRepository.GetByID").
				Msg("product not found")

			return nil, errors.New("404")
		}

		log.Error().
			Err(err).
			Int64("product_id", productID).
			Str("source", "internal.adapter.productRepository.GetByID").
			Msg("failed get product")

		return nil, err
	}

	modelParent := []model.Product{}

	err = p.db.WithContext(ctx).Preload("Category").Where("parent_id = ?", modelProduct.ID).Find(&modelParent).Error
	if err != nil {
		log.Error().
			Err(err).
			Int64("product_id", productID).
			Str("source", "internal.adapter.productRepository.GetByID").
			Msg("failed get child products")

		return nil, err
	}

	childEntities := []entity.ProductEntity{}

	for _, val := range modelParent {
		childEntities = append(childEntities, entity.ProductEntity{
			ID:           val.ID,
			CategorySlug: val.CategorySlug,
			ParentID:     val.ParentID,
			Name:         val.Name,
			Image:        val.Image,
			Description:  val.Description,
			RegulerPrice: val.RegulerPrice,
			SalePrice:    val.SalePrice,
			Unit:         val.Unit,
			Weight:       val.Weight,
			Stock:        val.Stock,
			Variant:      val.Variant,
			Status:       val.Status,
			CategoryName: val.Category.Name,
			CreatedAt:    val.CreatedAt,
		})
	}

	return &entity.ProductEntity{
		ID:           modelProduct.ID,
		CategorySlug: modelProduct.CategorySlug,
		ParentID:     modelProduct.ParentID,
		Name:         modelProduct.Name,
		Image:        modelProduct.Image,
		Description:  modelProduct.Description,
		RegulerPrice: modelProduct.RegulerPrice,
		SalePrice:    modelProduct.SalePrice,
		Unit:         modelProduct.Unit,
		Weight:       modelProduct.Weight,
		Stock:        modelProduct.Stock,
		Variant:      modelProduct.Variant,
		Status:       modelProduct.Status,
		CategoryName: modelProduct.Category.Name,
		Child:        childEntities,
		CreatedAt:    modelProduct.CreatedAt,
	}, nil
}

/*
## contoh request

	query := entity.QueryStringProduct{
		Page:         1,
		Limit:        10,
		OrderBy:      "created_at",
		OrderType:    "desc",
		CategorySlug: "electronics",
		StartPrice:   100000,
		EndPrice:     5000000,
		Search:       "iphone",
	}
*/
func (p *productRepository) SearchProducts(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error) {
	from := (query.Page - 1) * query.Limit

	allowedSortFields := map[string]bool{
		"id":            true,
		"name":          true,
		"created_at":    true,
		"reguler_price": true,
		"sale_price":    true,
		"stock":         true,
	}

	sortField := "id"

	if allowedSortFields[query.OrderBy] {
		sortField = query.OrderBy
	}

	sortOrder := "asc"

	if strings.ToLower(query.OrderType) == "desc" {
		sortOrder = "desc"
	}

	mustQueries := []map[string]interface{}{}
	filterQueries := []map[string]interface{}{}

	// fulltext search
	if query.Search != "" {
		mustQueries = append(
			mustQueries,
			helperSearch.MultiMatchQuery(
				query.Search,
				[]string{
					"name",
					"description",
					"category_name",
				},
			),
		)
	}

	// category filter
	if query.CategorySlug != "" {
		filterQueries = append(
			filterQueries,
			helperSearch.TermFilter(
				"category_slug.keyword",
				query.CategorySlug,
			),
		)
	}

	// price filter
	if query.StartPrice > 0 || query.EndPrice > 0 {

		var gte interface{}
		var lte interface{}

		if query.StartPrice > 0 {
			gte = query.StartPrice
		}

		if query.EndPrice > 0 {
			lte = query.EndPrice
		}

		filterQueries = append(
			filterQueries,
			helperSearch.RangeFilter(
				"reguler_price",
				gte,
				lte,
			),
		)
	}

	searchQuery := map[string]interface{}{
		"from": from,
		"size": query.Limit,
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must":   mustQueries,
				"filter": filterQueries,
			},
		},
		"sort": helperSearch.SortQuery(
			sortField,
			sortOrder,
		),
	}

	body, err := json.Marshal(searchQuery)
	if err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.adapter.productRepository.SearchProducts").
			Msg("failed marshal search query")

		return nil, 0, 0, err
	}

	res, err := p.opensearchClient.Search(
		p.opensearchClient.Search.WithContext(ctx),
		p.opensearchClient.Search.WithIndex("products"),
		p.opensearchClient.Search.WithBody(bytes.NewReader(body)),
	)

	if err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.adapter.productRepository.SearchProducts").
			Msg("failed search products")

		return nil, 0, 0, err
	}

	defer res.Body.Close()

	if err := helperSearch.ParseOpenSearchError(res); err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.adapter.productRepository.SearchProducts").
			Msg("opensearch returned error")

		return nil, 0, 0, err
	}

	var result map[string]interface{}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.adapter.productRepository.SearchProducts").
			Msg("failed decode response")

		return nil, 0, 0, err
	}

	hitsRoot, ok := result["hits"].(map[string]interface{})
	if !ok {
		return nil, 0, 0, errors.New("invalid hits response")
	}

	totalData := 0

	if totalMap, ok := hitsRoot["total"].(map[string]interface{}); ok {
		if value, ok := totalMap["value"].(float64); ok {
			totalData = int(value)
		}
	}

	totalPage := 0

	if query.Limit > 0 {
		totalPage = int(math.Ceil(
			float64(totalData) / float64(query.Limit),
		))
	}

	hits, ok := hitsRoot["hits"].([]interface{})
	if !ok {
		return nil, 0, 0, errors.New("invalid hits data")
	}

	products := []entity.ProductEntity{}

	for _, hit := range hits {

		hitMap, ok := hit.(map[string]interface{})
		if !ok {
			continue
		}

		source, ok := hitMap["_source"]
		if !ok {
			continue
		}

		data, err := json.Marshal(source)
		if err != nil {
			continue
		}

		var product entity.ProductEntity

		if err := json.Unmarshal(data, &product); err != nil {
			continue
		}

		products = append(products, product)
	}

	return products, int64(totalData), int64(totalPage), nil
}

func (p *productRepository) GetAll(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error) {
	modelProducts := []model.Product{}
	var countData int64

	order := fmt.Sprintf("%s %s", query.OrderBy, query.OrderType)
	offset := (query.Page - 1) * query.Limit
	defaultStatus := "ACTIVE"
	if query.Status != "" {
		defaultStatus = query.Status
	}
	sqlMain := p.db.Preload("Category").
		Where("parent_id IS NULL AND status = ?", defaultStatus).
		Where("name ILIKE ? OR description ILIKE ? OR category_slug ILIKE ?", "%"+query.Search+"%", "%"+query.Search+"%", "%"+query.Search+"%")
	if query.CategorySlug != "" {
		sqlMain = sqlMain.Where("category_slug = ?", query.CategorySlug)
	}

	if query.StartPrice > 0 {
		sqlMain = sqlMain.Where("sale_price >= ?", query.StartPrice)
	}

	if query.EndPrice > 0 {
		sqlMain = sqlMain.Where("sale_price <= ?", query.EndPrice)
	}

	if err := sqlMain.Model(&modelProducts).Count(&countData).Error; err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.adapter.productRepository.GetAll").
			Msg("failed count products")
		return nil, 0, 0, err
	}

	totalPage := int(math.Ceil(float64(countData) / float64(query.Limit)))
	if err := sqlMain.Order(order).Limit(int(query.Limit)).Offset(int(offset)).Find(&modelProducts).Error; err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.adapter.productRepository.GetAll").
			Msg("failed get products")
		return nil, 0, 0, err
	}

	if len(modelProducts) == 0 {
		log.Warn().
			Str("source", "internal.adapter.productRepository.GetAll").
			Msg("products not found")
		return nil, 0, 0, errors.New("404")
	}

	respProducts := []entity.ProductEntity{}
	for _, val := range modelProducts {
		respProducts = append(respProducts, entity.ProductEntity{
			ID:           val.ID,
			CategorySlug: val.CategorySlug,
			ParentID:     val.ParentID,
			Name:         val.Name,
			Image:        val.Image,
			Description:  val.Description,
			RegulerPrice: val.RegulerPrice,
			SalePrice:    val.SalePrice,
			Unit:         val.Unit,
			Weight:       val.Weight,
			Stock:        val.Stock,
			Variant:      val.Variant,
			Status:       val.Status,
			CategoryName: val.Category.Name,
			CreatedAt:    val.CreatedAt,
		})
	}

	return respProducts, countData, int64(totalPage), nil
}
