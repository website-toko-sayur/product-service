package handler

import (
	"product-service/internal/adapter/handler/request"
	"product-service/internal/adapter/handler/response"
	"product-service/internal/core/domain/entity"
	"product-service/internal/core/service"
	"product-service/utils/conv"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"
)

type categoryHandler struct {
	categoryService service.CategoryServiceInterface
}

type CategoryHandlerInterface interface {
	GetAllAdmin(c fiber.Ctx) error
	GetByIDAdmin(c fiber.Ctx) error
	GetBySlugAdmin(c fiber.Ctx) error
	Create(c fiber.Ctx) error
	Update(c fiber.Ctx) error
	Delete(c fiber.Ctx) error

	GetAllHome(c fiber.Ctx) error
	GetAllShop(c fiber.Ctx) error
}

func NewCategoryHandler(
	app *fiber.App,
	categoryService service.CategoryServiceInterface
	cfg *config.Config,
	jwtService service.JwtServiceInterface,
	redis *redis.Client,
) CategoryHandlerInterface {
	categoryHandler := &categoryHandler{
		categoryService: categoryService,
	}

	categoryApp := app.Group("/categories")
	categoryApp.Get("/home", category.GetAllHome)
	categoryApp.Get("/shop", category.GetAllShop)

	mid := adapter.NewMiddlewareAdapter(cfg, jwtService, redis)
	adminGroup := app.Group("/admin", mid.CheckToken())

	adminGroup.Get("/categories", category.GetAllAdmin)
	adminGroup.Get("/categories/:id", category.GetByIDAdmin)
	adminGroup.Get("/categories/:slug/slug", category.GetBySlugAdmin)
	adminGroup.Post("/categories", category.Create)
	adminGroup.Put("/categories/:id", category.Update)
	adminGroup.Delete("/categories/:id", category.Delete)

	return categoryHandler
}

func (ch *categoryHandler) GetAllShop(c fiber.Ctx) error {
	ctx := c.Context()

	results, err := ch.categoryService.GetAllPublished(ctx)
	if err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.adapter.categoryHandler.GetAllShop").
			Msg("failed get all published categories")

		if err.Error() == "404" {
			return fiber.NewError(fiber.StatusNotFound, "data not found")
		}

		return err
	}

	respCategories := rekursifCategory(results, nil, 0)

	return c.Status(fiber.StatusOK).JSON(response.DefaultResponse{
		Message: "success",
		Data:    respCategories,
	})
}

// RekursifCategory digunakan untuk menelusuri category parent-child secara recursive
func rekursifCategory(results []entity.CategoryEntity, parentID *int64, level int) []response.CategoryListShopResponse {
	var resps []response.CategoryListShopResponse

	for _, category := range results {
		if category.ParentID == parentID {
			resps = append(resps, response.CategoryListShopResponse{
				Name: category.Name,
				Slug: category.Slug,
			})

			// cari child category berdasarkan parent id category saat ini
			childCategories := rekursifCategory(results, &category.ID, level+1)

			resps = append(resps, childCategories...)
		}
	}

	return resps
}

func (ch *categoryHandler) GetAllHome(c fiber.Ctx) error {
	ctx := c.Context()

	results, err := ch.categoryService.GetAllPublished(ctx)
	if err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.adapter.categoryHandler.GetAllHome").
			Msg("failed get all published categories")

		if err.Error() == "404" {
			return fiber.NewError(fiber.StatusNotFound, "data not found")
		}

		return err
	}

	var respCategories []response.CategoryListHomeResponse

	for _, result := range results {
		if result.ParentID == nil {
			respCategories = append(respCategories, response.CategoryListHomeResponse{
				Name: result.Name,
				Icon: result.Icon,
				Slug: result.Slug,
			})
		}
	}

	return c.Status(fiber.StatusOK).JSON(response.DefaultResponse{
		Message: "success",
		Data:    respCategories,
	})
}

func (ch *categoryHandler) Create(c fiber.Ctx) error {
	var request request.CreateCategoryRequest

	ctx := c.Context()

	if err := c.Bind().Body(&request); err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.adapter.categoryHandler.Create").
			Msg("failed bind or validate request")

		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	reqEntity := entity.CategoryEntity{
		Name:        request.Name,
		Icon:        request.Icon,
		Description: request.Description,
		Status:      request.Status,
		ParentID:    request.ParentID,
	}

	if err := ch.categoryService.CreateCategory(ctx, reqEntity); err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.adapter.categoryHandler.Create").
			Msg("failed create category")

		return err
	}

	return c.Status(fiber.StatusCreated).JSON(response.DefaultResponse{
		Message: "success",
		Data:    nil,
	})
}

func (ch *categoryHandler) Delete(c fiber.Ctx) error {
	ctx := c.Context()

	idStr := c.Params("id")
	if idStr == "" {
		log.Error().
			Str("source", "internal.adapter.categoryHandler.Delete").
			Msg("id is required")

		return fiber.NewError(fiber.StatusBadRequest, "id is required")
	}

	id, err := conv.StringToInt64(idStr)
	if err != nil {
		log.Error().
			Err(err).
			Str("id", idStr).
			Str("source", "internal.adapter.categoryHandler.Delete").
			Msg("failed convert id")

		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	if err := ch.categoryService.DeleteCategory(ctx, id); err != nil {
		log.Error().
			Err(err).
			Int64("category_id", id).
			Str("source", "internal.adapter.categoryHandler.Delete").
			Msg("failed delete category")

		if err.Error() == "404" {
			return fiber.NewError(fiber.StatusNotFound, "category not found")
		}

		return err
	}

	return c.Status(fiber.StatusOK).JSON(response.DefaultResponse{
		Message: "success",
		Data:    nil,
	})
}

func (ch *categoryHandler) GetByIDAdmin(c fiber.Ctx) error {
	ctx := c.Context()

	idStr := c.Params("id")
	if idStr == "" {
		log.Error().
			Str("source", "internal.adapter.categoryHandler.GetByIDAdmin").
			Msg("id is required")

		return fiber.NewError(fiber.StatusBadRequest, "id is required")
	}

	id, err := conv.StringToInt64(idStr)
	if err != nil {
		log.Error().
			Err(err).
			Str("id", idStr).
			Str("source", "internal.adapter.categoryHandler.GetByIDAdmin").
			Msg("failed convert id")

		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	result, err := ch.categoryService.GetByID(ctx, id)
	if err != nil {
		log.Error().
			Err(err).
			Int64("category_id", id).
			Str("source", "internal.adapter.categoryHandler.GetByIDAdmin").
			Msg("failed get category by id")

		if err.Error() == "404" {
			return fiber.NewError(fiber.StatusNotFound, "data not found")
		}

		return err
	}

	respCategory := response.CategoryDetailResponse{
		ID:          result.ID,
		Name:        result.Name,
		Slug:        result.Slug,
		Icon:        result.Icon,
		Status:      result.Status,
		Description: result.Description,
	}

	return c.Status(fiber.StatusOK).JSON(response.DefaultResponse{
		Message: "success",
		Data:    respCategory,
	})
}

func (ch *categoryHandler) GetBySlugAdmin(c fiber.Ctx) error {
	ctx := c.Context()

	slug := c.Params("slug")
	if slug == "" {
		log.Error().
			Str("source", "internal.adapter.categoryHandler.GetBySlugAdmin").
			Msg("slug is required")

		return fiber.NewError(fiber.StatusBadRequest, "slug is required")
	}

	result, err := ch.categoryService.GetBySlug(ctx, slug)
	if err != nil {
		log.Error().
			Err(err).
			Str("slug", slug).
			Str("source", "internal.adapter.categoryHandler.GetBySlugAdmin").
			Msg("failed get category by slug")

		if err.Error() == "404" {
			return fiber.NewError(fiber.StatusNotFound, "data not found")
		}

		return err
	}

	respCategory := response.CategoryDetailResponse{
		ID:          result.ID,
		Name:        result.Name,
		Slug:        result.Slug,
		Icon:        result.Icon,
		Status:      result.Status,
		Description: result.Description,
	}

	return c.Status(fiber.StatusOK).JSON(response.DefaultResponse{
		Message: "success",
		Data:    respCategory,
	})
}

func (ch *categoryHandler) Update(c fiber.Ctx) error {
	var request request.CreateCategoryRequest

	ctx := c.Context()

	idStr := c.Params("id")
	if idStr == "" {
		log.Error().
			Str("source", "internal.adapter.categoryHandler.Update").
			Msg("id is required")

		return fiber.NewError(fiber.StatusBadRequest, "id is required")
	}

	id, err := conv.StringToInt64(idStr)
	if err != nil {
		log.Error().
			Err(err).
			Str("id", idStr).
			Str("source", "internal.adapter.categoryHandler.Update").
			Msg("failed convert id")

		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	if err := c.Bind().Body(&request); err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.adapter.categoryHandler.Update").
			Msg("failed bind or validate request")

		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	reqEntity := entity.CategoryEntity{
		ID:          id,
		Name:        request.Name,
		Icon:        request.Icon,
		Description: request.Description,
		Status:      request.Status,
		ParentID:    request.ParentID,
	}

	if err := ch.categoryService.EditCategory(ctx, reqEntity); err != nil {
		log.Error().
			Err(err).
			Int64("category_id", id).
			Str("source", "internal.adapter.categoryHandler.Update").
			Msg("failed update category")

		if err.Error() == "404" {
			return fiber.NewError(fiber.StatusNotFound, "category not found")
		}

		return err
	}

	return c.Status(fiber.StatusOK).JSON(response.DefaultResponse{
		Message: "success",
		Data:    nil,
	})
}

func (ch *categoryHandler) GetAllAdmin(c fiber.Ctx) error {
	ctx := c.Context()

	search := c.Query("search")

	orderBy := c.Query("orderBy", "created_at")
	orderType := c.Query("orderType", "desc")

	page, err := conv.StringToInt64(c.Query("page", "1"))
	if err != nil || page <= 0 {
		page = 1
	}

	perPage, err := conv.StringToInt64(c.Query("perPage", "10"))
	if err != nil || perPage <= 0 {
		perPage = 10
	}

	reqEntity := entity.QueryStringEntity{
		Search:    search,
		OrderBy:   orderBy,
		OrderType: orderType,
		Page:      int(page),
		Limit:     int(perPage),
	}

	results, totalData, totalPage, err := ch.categoryService.GetAll(ctx, reqEntity)
	if err != nil {
		log.Error().
			Err(err).
			Interface("query", reqEntity).
			Str("source", "internal.adapter.categoryHandler.GetAllAdmin").
			Msg("failed get all categories")

		if err.Error() == "404" {
			return fiber.NewError(fiber.StatusNotFound, "data not found")
		}

		return err
	}

	var respCategories []response.CategoryListAdminResponse

	for _, result := range results {
		respCategories = append(respCategories, response.CategoryListAdminResponse{
			ID:           result.ID,
			Name:         result.Name,
			Icon:         result.Icon,
			Slug:         result.Slug,
			Status:       result.Status,
			TotalProduct: len(result.Products),
		})
	}

	pagination := response.Pagination{
		Page:       page,
		TotalCount: totalData,
		PerPage:    perPage,
		TotalPage:  totalPage,
	}

	return c.Status(fiber.StatusOK).JSON(response.DefaultResponseWithPaginations{
		Message:    "success",
		Data:       respCategories,
		Pagination: &pagination,
	})
}
