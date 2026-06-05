package handler

import (
	"product-service/config"
	"product-service/internal/adapter"
	"product-service/internal/adapter/handler/request"
	"product-service/internal/adapter/handler/response"
	"product-service/internal/core/domain/entity"
	"product-service/internal/core/service"
	middleware "product-service/internal/middleware"
	"product-service/utils/conv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

type productHandler struct {
	productService service.ProductServiceInterface
}

type ProductHandlerInterface interface {
	GetAllAdmin(c fiber.Ctx) error
	GetByIDAdmin(c fiber.Ctx) error
	CreateAdmin(c fiber.Ctx) error
	EditAdmin(c fiber.Ctx) error
	DeleteAdmin(c fiber.Ctx) error

	GetAllHome(c fiber.Ctx) error
	GetAllShop(c fiber.Ctx) error
	GetDetailHome(c fiber.Ctx) error
}

func NewProductHandler(
	app *fiber.App,
	productService service.ProductServiceInterface,
	cfg *config.Config,
	jwtService service.JwtServiceInterface,
	redis *redis.Client,
) ProductHandlerInterface {
	productHandler := &productHandler{
		productService: productService,
	}

	mid := adapter.NewMiddlewareAdapter(cfg, jwtService, redis)
	midGateway := middleware.GatewayValidationMiddleware(cfg)
	midInternal := middleware.InternalValidationMiddleware(cfg)

	// products route via gateway
	homeProduct := app.Group("/products", midGateway)
	homeProduct.Get("/home", productHandler.GetAllHome)
	homeProduct.Get("/shop", productHandler.GetAllShop)
	homeProduct.Get("/home/:id", productHandler.GetDetailHome)

	// admin route via gateway + jwt
	adminGroup := app.Group("/admin", midGateway, mid.CheckToken())
	adminGroup.Get("/products", productHandler.GetAllAdmin)
	adminGroup.Post("/products", productHandler.CreateAdmin)
	adminGroup.Get("/products/:id", productHandler.GetByIDAdmin)
	adminGroup.Put("/products/:id", productHandler.EditAdmin)
	adminGroup.Delete("/products/:id", productHandler.DeleteAdmin)

	// internal route
	internalGroup := app.Group("/internal", midInternal)
	internalGroup.Get("/products/:id", productHandler.GetByID)

	return productHandler
}

func (p *productHandler) GetDetailHome(c fiber.Ctx) error {
	ctx := c.Context()

	idStr := c.Params("id")
	if idStr == "" {
		log.Error().
			Str("source", "internal.adapter.productHandler.GetDetailHome").
			Msg("id is required")

		return fiber.NewError(fiber.StatusBadRequest, "id is required")
	}

	id, err := conv.StringToInt64(idStr)
	if err != nil {
		log.Error().
			Err(err).
			Str("id", idStr).
			Str("source", "internal.adapter.productHandler.GetDetailHome").
			Msg("failed convert id")

		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	result, err := p.productService.GetByID(ctx, id)
	if err != nil {
		log.Error().
			Err(err).
			Int64("product_id", id).
			Str("source", "internal.adapter.productHandler.GetDetailHome").
			Msg("failed get product by id")

		if err.Error() == "404" {
			return fiber.NewError(fiber.StatusNotFound, "data not found")
		}

		return err
	}

	respDetail := response.ProductHomeDetailResponse{
		ID:           result.ID,
		ProductName:  result.Name,
		CategoryName: result.CategoryName,
		Description:  result.Description,
		Unit:         result.Unit,
		Weight:       result.Weight,
		Stock:        result.Stock,
		RegulerPrice: int64(result.RegulerPrice),
		SalePrice:    int64(result.SalePrice),
		ProductImage: result.Image,
	}

	for _, child := range result.Child {
		respDetail.Child = append(respDetail.Child, response.ProductChildHomeResponse{
			ID:           child.ID,
			Weight:       child.Weight,
			Stock:        child.Stock,
			RegulerPrice: int64(child.RegulerPrice),
			SalePrice:    int64(child.SalePrice),
			Image:        child.Image,
		})
	}

	return c.Status(fiber.StatusOK).JSON(response.DefaultResponse{
		Message: "success",
		Data:    respDetail,
	})
}

func (p *productHandler) GetAllShop(c fiber.Ctx) error {
	ctx := c.Context()

	orderBy := "created_at"
	orderType := "desc"

	switch c.Query("orderBy") {
	case "price_asc":
		orderBy = "reguler_price"
		orderType = "asc"

	case "price_desc":
		orderBy = "reguler_price"
		orderType = "desc"

	case "newest":
		orderBy = "id"
		orderType = "desc"
	}

	page, err := conv.StringToInt64(c.Query("page", "1"))
	if err != nil || page <= 0 {
		page = 1
	}

	perPage, err := conv.StringToInt64(c.Query("limit", "10"))
	if err != nil || perPage <= 0 {
		perPage = 10
	}

	var startPrice int64
	var endPrice int64

	price := c.Query("price")
	if price != "" {
		priceRange := strings.Split(price, " - ")

		if len(priceRange) == 2 {
			startPrice, _ = conv.StringToInt64(priceRange[0])
			endPrice, _ = conv.StringToInt64(priceRange[1])
		}
	}

	reqEntity := entity.QueryStringProduct{
		Search:       c.Query("search"),
		CategorySlug: c.Query("category"),
		OrderBy:      orderBy,
		OrderType:    orderType,
		Page:         int(page),
		Limit:        int(perPage),
		StartPrice:   startPrice,
		EndPrice:     endPrice,
	}

	results, totalData, totalPage, err := p.productService.SearchProducts(ctx, reqEntity)
	if err != nil {
		log.Error().
			Err(err).
			Interface("query", reqEntity).
			Str("source", "internal.adapter.productHandler.GetAllShop").
			Msg("failed search products")

		if err.Error() == "404" {
			return fiber.NewError(fiber.StatusNotFound, "data not found")
		}

		return err
	}

	var respLists []response.ProductHomeListResponse

	for _, result := range results {
		respLists = append(respLists, response.ProductHomeListResponse{
			ID:           result.ID,
			ProductName:  result.Name,
			ProductImage: result.Image,
			SalePrice:    int64(result.SalePrice),
			RegulerPrice: int64(result.RegulerPrice),
			CategoryName: result.CategoryName,
		})
	}

	return c.Status(fiber.StatusOK).JSON(response.DefaultResponseWithPaginations{
		Message: "success",
		Data:    respLists,
		Pagination: &response.Pagination{
			Page:       page,
			TotalPage:  totalPage,
			TotalCount: totalData,
			PerPage:    perPage,
		},
	})
}

func (p *productHandler) GetAllHome(c fiber.Ctx) error {
	ctx := c.Context()

	reqEntity := entity.QueryStringProduct{
		OrderBy:   "created_at",
		OrderType: "desc",
		Page:      1,
		Limit:     5,
	}

	results, _, _, err := p.productService.GetAll(ctx, reqEntity)
	if err != nil {
		log.Error().
			Err(err).
			Interface("query", reqEntity).
			Str("source", "internal.adapter.productHandler.GetAllHome").
			Msg("failed get all products")

		if err.Error() == "404" {
			return fiber.NewError(fiber.StatusNotFound, "data not found")
		}

		return err
	}

	var respLists []response.ProductHomeListResponse

	for _, result := range results {
		respLists = append(respLists, response.ProductHomeListResponse{
			ID:           result.ID,
			ProductName:  result.Name,
			ProductImage: result.Image,
			SalePrice:    int64(result.SalePrice),
			RegulerPrice: int64(result.RegulerPrice),
			CategoryName: result.CategoryName,
		})
	}

	return c.Status(fiber.StatusOK).JSON(response.DefaultResponse{
		Message: "success",
		Data:    respLists,
	})
}

func (p *productHandler) DeleteAdmin(c fiber.Ctx) error {
	ctx := c.Context()

	user, ok := c.Locals("user").(string)
	if !ok || user == "" {
		log.Error().
			Str("source", "internal.adapter.productHandler.DeleteAdmin").
			Msg("data token not found")

		return fiber.NewError(fiber.StatusUnauthorized, "data token not found")
	}

	idStr := c.Params("id")
	if idStr == "" {
		log.Error().
			Str("source", "internal.adapter.productHandler.DeleteAdmin").
			Msg("id is required")

		return fiber.NewError(fiber.StatusBadRequest, "id is required")
	}

	id, err := conv.StringToInt64(idStr)
	if err != nil {
		log.Error().
			Err(err).
			Str("id", idStr).
			Str("source", "internal.adapter.productHandler.DeleteAdmin").
			Msg("failed convert id")

		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	if err := p.productService.Delete(ctx, id); err != nil {
		log.Error().
			Err(err).
			Int64("product_id", id).
			Str("source", "internal.adapter.productHandler.DeleteAdmin").
			Msg("failed delete product")

		if err.Error() == "404" {
			return fiber.NewError(fiber.StatusNotFound, "data not found")
		}

		return err
	}

	return c.Status(fiber.StatusOK).JSON(response.DefaultResponse{
		Message: "success",
		Data:    nil,
	})
}

func (p *productHandler) EditAdmin(c fiber.Ctx) error {
	var req request.ProductRequest

	ctx := c.Context()

	user, ok := c.Locals("user").(string)
	if !ok || user == "" {
		log.Error().
			Str("source", "internal.adapter.productHandler.EditAdmin").
			Msg("data token not found")

		return fiber.NewError(fiber.StatusUnauthorized, "data token not found")
	}

	idStr := c.Params("id")
	if idStr == "" {
		log.Error().
			Str("source", "internal.adapter.productHandler.EditAdmin").
			Msg("id is required")

		return fiber.NewError(fiber.StatusBadRequest, "id is required")
	}

	id, err := conv.StringToInt64(idStr)
	if err != nil {
		log.Error().
			Err(err).
			Str("id", idStr).
			Str("source", "internal.adapter.productHandler.EditAdmin").
			Msg("failed convert id")

		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	if err := c.Bind().Body(&req); err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.adapter.productHandler.EditAdmin").
			Msg("failed bind or validate request")

		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	reqEntity := entity.ProductEntity{
		ID:           id,
		CategorySlug: req.CategorySlug,
		ParentID:     nil,
		Name:         req.ProductName,
		Image:        req.VariantDetail[0].ProductImage,
		Description:  req.ProductDescription,
		RegulerPrice: float64(req.VariantDetail[0].RegulerPrice),
		SalePrice:    float64(req.VariantDetail[0].SalePrice),
		Unit:         req.Unit,
		Weight:       req.VariantDetail[0].Weight,
		Stock:        req.VariantDetail[0].Stock,
		Variant:      req.Variant,
		Status:       req.Status,
	}

	if len(req.VariantDetail) > 1 {
		var productChilds []entity.ProductEntity

		for i := 1; i < len(req.VariantDetail); i++ {
			productChilds = append(productChilds, entity.ProductEntity{
				Image:        req.VariantDetail[i].ProductImage,
				RegulerPrice: float64(req.VariantDetail[i].RegulerPrice),
				SalePrice:    float64(req.VariantDetail[i].SalePrice),
				Weight:       req.VariantDetail[i].Weight,
				Stock:        req.VariantDetail[i].Stock,
			})
		}

		reqEntity.Child = productChilds
	}

	if err := p.productService.Update(ctx, reqEntity); err != nil {
		log.Error().
			Err(err).
			Int64("product_id", id).
			Str("source", "internal.adapter.productHandler.EditAdmin").
			Msg("failed update product")

		return err
	}

	return c.Status(fiber.StatusOK).JSON(response.DefaultResponse{
		Message: "success",
		Data:    nil,
	})
}

func (p *productHandler) CreateAdmin(c fiber.Ctx) error {
	var req request.ProductRequest

	ctx := c.Context()

	user, ok := c.Locals("user").(string)
	if !ok || user == "" {
		log.Error().
			Str("source", "internal.adapter.productHandler.CreateAdmin").
			Msg("data token not found")

		return fiber.NewError(fiber.StatusUnauthorized, "data token not found")
	}

	if err := c.Bind().Body(&req); err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.adapter.productHandler.CreateAdmin").
			Msg("failed bind or validate request")

		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	reqEntity := entity.ProductEntity{
		CategorySlug: req.CategorySlug,
		ParentID:     nil,
		Name:         req.ProductName,
		Image:        req.VariantDetail[0].ProductImage,
		Description:  req.ProductDescription,
		RegulerPrice: float64(req.VariantDetail[0].RegulerPrice),
		SalePrice:    float64(req.VariantDetail[0].SalePrice),
		Unit:         req.Unit,
		Weight:       req.VariantDetail[0].Weight,
		Stock:        req.VariantDetail[0].Stock,
		Variant:      req.Variant,
		Status:       req.Status,
	}

	if len(req.VariantDetail) > 1 {
		var productChilds []entity.ProductEntity

		for i := 1; i < len(req.VariantDetail); i++ {
			productChilds = append(productChilds, entity.ProductEntity{
				Image:        req.VariantDetail[i].ProductImage,
				RegulerPrice: float64(req.VariantDetail[i].RegulerPrice),
				SalePrice:    float64(req.VariantDetail[i].SalePrice),
				Weight:       req.VariantDetail[i].Weight,
				Stock:        req.VariantDetail[i].Stock,
			})
		}

		reqEntity.Child = productChilds
	}

	if err := p.productService.Create(ctx, reqEntity); err != nil {
		log.Error().
			Err(err).
			Str("product_name", req.ProductName).
			Str("source", "internal.adapter.productHandler.CreateAdmin").
			Msg("failed create product")

		return err
	}

	return c.Status(fiber.StatusCreated).JSON(response.DefaultResponse{
		Message: "success",
		Data:    nil,
	})
}

func (p *productHandler) GetByIDAdmin(c fiber.Ctx) error {
	ctx := c.Context()

	user, ok := c.Locals("user").(string)
	if !ok || user == "" {
		log.Error().
			Str("source", "internal.adapter.productHandler.GetByIDAdmin").
			Msg("data token not found")

		return fiber.NewError(fiber.StatusUnauthorized, "data token not found")
	}

	idStr := c.Params("id")
	if idStr == "" {
		log.Error().
			Str("source", "internal.adapter.productHandler.GetByIDAdmin").
			Msg("id is required")

		return fiber.NewError(fiber.StatusBadRequest, "id is required")
	}

	id, err := conv.StringToInt64(idStr)
	if err != nil {
		log.Error().
			Err(err).
			Str("id", idStr).
			Str("source", "internal.adapter.productHandler.GetByIDAdmin").
			Msg("failed convert id")

		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	result, err := p.productService.GetByID(ctx, id)
	if err != nil {
		log.Error().
			Err(err).
			Int64("product_id", id).
			Str("source", "internal.adapter.productHandler.GetByIDAdmin").
			Msg("failed get product by id")

		if err.Error() == "404" {
			return fiber.NewError(fiber.StatusNotFound, "data not found")
		}

		return err
	}

	var responseChilds []response.ProductChildResponse

	if len(result.Child) > 0 {
		for _, child := range result.Child {
			responseChilds = append(responseChilds, response.ProductChildResponse{
				ID:           child.ID,
				SalePrice:    int64(child.SalePrice),
				RegulerPrice: int64(child.RegulerPrice),
				Weight:       child.Weight,
				Stock:        child.Stock,
				ProductImage: child.Image,
			})
		}
	}

	respProduct := response.ProductDetailResponse{
		ID:                 result.ID,
		ProductName:        result.Name,
		ParentID:           conv.Int64PointerToInt64(result.ParentID),
		ProductImage:       result.Image,
		CategorySlug:       result.CategorySlug,
		CategoryName:       result.CategoryName,
		ProductStatus:      result.Status,
		ProductDescription: result.Description,
		SalePrice:          int64(result.SalePrice),
		RegulerPrice:       int64(result.RegulerPrice),
		Unit:               result.Unit,
		Weight:             result.Weight,
		Stock:              result.Stock,
		CreatedAt:          result.CreatedAt,
		Child:              responseChilds,
	}

	return c.Status(fiber.StatusOK).JSON(response.DefaultResponse{
		Message: "success",
		Data:    respProduct,
	})
}

func (p *productHandler) GetAllAdmin(c fiber.Ctx) error {
	ctx := c.Context()

	search := c.Query("search")

	orderBy := c.Query("orderBy", "created_at")
	orderType := c.Query("orderType", "desc")

	page, err := conv.StringToInt64(c.Query("page", "1"))
	if err != nil || page <= 0 {
		page = 1
	}

	perPage, err := conv.StringToInt64(c.Query("limit", "10"))
	if err != nil || perPage <= 0 {
		perPage = 10
	}

	startPrice, err := conv.StringToInt64(c.Query("startPrice", "0"))
	if err != nil {
		startPrice = 0
	}

	endPrice, err := conv.StringToInt64(c.Query("endPrice", "0"))
	if err != nil {
		endPrice = 0
	}

	reqEntity := entity.QueryStringProduct{
		Search:       search,
		OrderBy:      orderBy,
		OrderType:    orderType,
		Page:         int(page),
		Limit:        int(perPage),
		CategorySlug: c.Query("categorySlug"),
		StartPrice:   startPrice,
		EndPrice:     endPrice,
		Status:       c.Query("status"),
	}

	results, totalData, totalPage, err := p.productService.GetAll(ctx, reqEntity)
	if err != nil {
		log.Error().
			Err(err).
			Interface("query", reqEntity).
			Str("source", "internal.adapter.productHandler.GetAllAdmin").
			Msg("failed get all products")

		if err.Error() == "404" {
			return fiber.NewError(fiber.StatusNotFound, "data not found")
		}

		return err
	}

	var respProducts []response.ProductListResponse

	for _, product := range results {
		respProducts = append(respProducts, response.ProductListResponse{
			ID:            product.ID,
			ProductName:   product.Name,
			ParentID:      conv.Int64PointerToInt64(product.ParentID),
			ProductImage:  product.Image,
			CategoryName:  product.CategoryName,
			ProductStatus: product.Status,
			SalePrice:     int64(product.SalePrice),
			RegulerPrice:  int64(product.RegulerPrice),
			CreatedAt:     product.CreatedAt,
		})
	}

	return c.Status(fiber.StatusOK).JSON(response.DefaultResponseWithPaginations{
		Message: "success",
		Data:    respProducts,
		Pagination: &response.Pagination{
			Page:       page,
			TotalCount: totalData,
			TotalPage:  totalPage,
			PerPage:    perPage,
		},
	})
}

func (p *productHandler) GetByID(c fiber.Ctx) error {
	ctx := c.Context()

	idStr := c.Params("id")
	if idStr == "" {
		log.Error().
			Str("source", "internal.adapter.productHandler.GetByID").
			Msg("id is required")

		return fiber.NewError(fiber.StatusBadRequest, "id is required")
	}

	id, err := conv.StringToInt64(idStr)
	if err != nil {
		log.Error().
			Err(err).
			Str("id", idStr).
			Str("source", "internal.adapter.productHandler.GetByID").
			Msg("failed convert id")

		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	result, err := p.productService.GetByID(ctx, id)
	if err != nil {
		log.Error().
			Err(err).
			Int64("product_id", id).
			Str("source", "internal.adapter.productHandler.GetByID").
			Msg("failed get product by id")

		if err.Error() == "404" {
			return fiber.NewError(fiber.StatusNotFound, "data not found")
		}

		return err
	}

	var responseChilds []response.ProductChildResponse

	if len(result.Child) > 0 {
		for _, child := range result.Child {
			responseChilds = append(responseChilds, response.ProductChildResponse{
				ID:           child.ID,
				SalePrice:    int64(child.SalePrice),
				RegulerPrice: int64(child.RegulerPrice),
				Weight:       child.Weight,
				Stock:        child.Stock,
			})
		}
	}

	respProduct := response.ProductDetailResponse{
		ID:                 result.ID,
		ProductName:        result.Name,
		ParentID:           conv.Int64PointerToInt64(result.ParentID),
		ProductImage:       result.Image,
		CategorySlug:       result.CategorySlug,
		CategoryName:       result.CategoryName,
		ProductStatus:      result.Status,
		ProductDescription: result.Description,
		SalePrice:          int64(result.SalePrice),
		RegulerPrice:       int64(result.RegulerPrice),
		Unit:               result.Unit,
		Weight:             result.Weight,
		Stock:              result.Stock,
		CreatedAt:          result.CreatedAt,
		Child:              responseChilds,
	}

	return c.Status(fiber.StatusOK).JSON(response.DefaultResponse{
		Message: "success",
		Data:    respProduct,
	})
}
