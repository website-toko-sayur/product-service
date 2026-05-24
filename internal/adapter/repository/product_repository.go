package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"product-service/internal/core/domain/entity"
	"product-service/internal/core/domain/model"
	"strings"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type productRepository struct {
	db       *gorm.DB
	esClient *elasticsearch.Client
}

type ProductRepositoryInterface interface {
	GetAll(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error)
	GetByID(ctx context.Context, productID int64) (*entity.ProductEntity, error)
	Create(ctx context.Context, req entity.ProductEntity) (int64, error)
	Update(ctx context.Context, req entity.ProductEntity) error
	Delete(ctx context.Context, productID int64) error
	SearchProducts(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error)
}

func NewProductRepository(db *gorm.DB, esClient *elasticsearch.Client) ProductRepositoryInterface {
	return &productRepository{
		db:       db,
		esClient: esClient,
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

	log.Info().
		Int64("product_id", productID).
		Str("source", "internal.adapter.productRepository.GetByID").
		Msg("success get product")

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

## mainquery nya

	{
	  "from": 0,
	  "size": 10,
	  "query": {
	    "bool": {
	      "must": [
	        {
	          "multi_match": {
	            "query": "iphone",
	            "fields": [
	              "name",
	              "description",
	              "category_name"
	            ]
	          }
	        }
	      ],
	      "filter": [
	        {
	          "term": {
	            "category_slug.keyword": "electronics"
	          }
	        },
	        {
	          "range": {
	            "reguler_price": {
	              "gte": 100000,
	              "lte": 5000000
	            }
	          }
	        }
	      ]
	    }
	  },
	  "sort": [
	    {
	      "created_at": "desc"
	    }
	  ]
	}
*/
func (p *productRepository) SearchProducts(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error) {

	var mainQueries []string
	var filterQueries []string

	// hitung offset pagination elasticsearch
	from := (query.Page - 1) * query.Limit

	// validate sortable fields supaya user tidak bisa inject field random
	allowedSortFields := map[string]bool{
		"id":            true,
		"name":          true,
		"created_at":    true,
		"reguler_price": true,
		"sale_price":    true,
		"stock":         true,
	}

	// gunakan whitelist field untuk sorting
	// fallback ke "id" jika field tidak valid
	sortField := "id"

	if allowedSortFields[query.OrderBy] {
		sortField = query.OrderBy
	}

	// determine sort order
	sortOrder := "asc"

	if strings.ToLower(query.OrderType) == "desc" {
		sortOrder = "desc"
	}

	sortQuery := fmt.Sprintf(`{ "%s": "%s" }`, sortField, sortOrder)

	if query.CategorySlug != "" {
		filterQueries = append(
			filterQueries,
			fmt.Sprintf(
				`{ "term": { "category_slug.keyword": "%s" } }`,
				query.CategorySlug,
			),
		)
	}

	if query.StartPrice > 0 && query.EndPrice > 0 {
		filterQueries = append(
			filterQueries,
			fmt.Sprintf(
				`{ "range": { "reguler_price": { "gte": %d, "lte": %d } } }`,
				query.StartPrice,
				query.EndPrice,
			),
		)
	}

	if query.Search != "" {
		filterQueries = append(
			mainQueries,
			fmt.Sprintf(
				`{
					"multi_match": {
						"query": %q,
						"fields": ["name", "description", "category_name"]
					}
				}`,
				query.Search,
			),
		)
	}

	mainQuery := fmt.Sprintf(`{
		"from": %d,
		"size": %d,
		"query": {
			"bool": {
				"must": [
					%s
				],
				"filter": [
					%s
				]
			}
		},
		"sort": [
			%s
		]
	}`,
		from,
		query.Limit,
		strings.Join(mainQueries, ","),
		strings.Join(filterQueries, ","),
		sortQuery,
	)

	res, err := p.esClient.Search(
		p.esClient.Search.WithContext(ctx),
		p.esClient.Search.WithIndex("products"),
		p.esClient.Search.WithBody(strings.NewReader(mainQuery)),
	)

	if err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.adapter.productRepository.SearchProducts").
			Msg("failed search products from elasticsearch")

		return nil, 0, 0, err
	}

	defer res.Body.Close()

	if res.IsError() {
		err = errors.New(res.String())

		log.Error().
			Err(err).
			Str("source", "internal.adapter.productRepository.SearchProducts").
			Msg("elasticsearch returned search error")

		return nil, 0, 0, err
	}

	var result map[string]interface{}

	err = json.NewDecoder(res.Body).Decode(&result)
	if err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.adapter.productRepository.SearchProducts").
			Msg("failed decode elasticsearch response")

		return nil, 0, 0, err
	}

	hitsRoot, ok := result["hits"].(map[string]interface{})
	if !ok {
		log.Error().
			Str("source", "internal.adapter.productRepository.SearchProducts").
			Msg("invalid hits response from elasticsearch")

		return nil, 0, 0, errors.New("invalid elasticsearch response")
	}

	totalData := 0

	totalMap, ok := hitsRoot["total"].(map[string]interface{})
	if ok {
		value, ok := totalMap["value"].(float64)
		if ok {
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
		log.Error().
			Str("source", "internal.adapter.productRepository.SearchProducts").
			Msg("invalid hits data from elasticsearch")

		return nil, 0, 0, errors.New("invalid elasticsearch hits")
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
			log.Error().
				Err(err).
				Str("source", "internal.adapter.productRepository.SearchProducts").
				Msg("failed marshal product search source")

			continue
		}

		var product entity.ProductEntity

		err = json.Unmarshal(data, &product)
		if err != nil {
			log.Error().
				Err(err).
				Str("source", "internal.adapter.productRepository.SearchProducts").
				Msg("failed unmarshal product search result")

			continue
		}

		products = append(products, product)
	}

	log.Info().
		Int("total_data", totalData).
		Int("total_page", totalPage).
		Int("result_count", len(products)).
		Str("source", "internal.adapter.productRepository.SearchProducts").
		Msg("success search products")

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

	log.Info().
		Int64("total_data", countData).
		Int("total_page", totalPage).
		Int("result_count", len(respProducts)).
		Str("source", "internal.adapter.productRepository.GetAll").
		Msg("success get products")

	return respProducts, countData, int64(totalPage), nil
}
