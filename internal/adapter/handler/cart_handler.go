package handler

import (
	"encoding/json"
	"product-service/config"
	"product-service/internal/adapter"
	"product-service/internal/adapter/handler/request"
	"product-service/internal/adapter/handler/response"
	"product-service/internal/core/domain/entity"
	"product-service/internal/core/service"
	middleware "product-service/internal/middleware"
	"product-service/utils/conv"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

type cartHandler struct {
	cartService    service.CartServiceInterface
	productService service.ProductServiceInterface
}

type CartHandlerInterface interface {
	AddToCart(c fiber.Ctx) error
	GetCart(c fiber.Ctx) error
	RemoveFromCart(c fiber.Ctx) error
	RemoveAllCart(c fiber.Ctx) error
}

func NewCartHandler(
	app *fiber.App,
	cartService service.CartServiceInterface,
	productService service.ProductServiceInterface,
	cfg *config.Config,
	jwtService service.JwtServiceInterface,
	redis *redis.Client,
) CartHandlerInterface {
	cartHandler := &cartHandler{
		cartService:    cartService,
		productService: productService,
	}

	mid := adapter.NewMiddlewareAdapter(cfg, jwtService, redis)
	midGateway := middleware.GatewayValidationMiddleware(cfg)

	authGroup := app.Group("/auth", midGateway, mid.CheckToken())

	authGroup.Post("/cart", cartHandler.AddToCart)
	authGroup.Get("/cart", cartHandler.GetCart)
	authGroup.Delete("/cart", cartHandler.RemoveFromCart)
	authGroup.Delete("/cart/all", cartHandler.RemoveAllCart)

	return cartHandler
}

func (ch *cartHandler) RemoveAllCart(c fiber.Ctx) error {
	var jwtUserData entity.JwtUserData

	ctx := c.Context()

	user, ok := c.Locals("user").(string)
	if !ok || user == "" {
		log.Error().
			Str("source", "internal.adapter.cartHandler.RemoveAllCart").
			Msg("data token not valid")

		return fiber.NewError(fiber.StatusUnauthorized, "data token not valid")
	}

	if err := json.Unmarshal([]byte(user), &jwtUserData); err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.adapter.cartHandler.RemoveAllCart").
			Msg("failed parse jwt user data")

		return fiber.NewError(fiber.StatusBadRequest, "invalid token data")
	}

	if err := ch.cartService.RemoveAllCart(ctx, jwtUserData.UserID); err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.adapter.cartHandler.RemoveAllCart").
			Msg("failed remove all cart")

		return err
	}

	return c.Status(fiber.StatusOK).JSON(response.DefaultResponse{
		Message: "success",
		Data:    nil,
	})
}

func (ch *cartHandler) AddToCart(c fiber.Ctx) error {
	var (
		request     request.CartRequest
		jwtUserData entity.JwtUserData
	)

	ctx := c.Context()

	if err := c.Bind().Body(&request); err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.adapter.cartHandler.AddToCart").
			Msg("failed bind/validate request")

		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	user, ok := c.Locals("user").(string)
	if !ok || user == "" {
		log.Error().
			Str("source", "internal.adapter.cartHandler.AddToCart").
			Msg("data token not valid")

		return fiber.NewError(fiber.StatusUnauthorized, "data token not valid")
	}

	if err := json.Unmarshal([]byte(user), &jwtUserData); err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.adapter.cartHandler.AddToCart").
			Msg("failed parse jwt user data")

		return fiber.NewError(fiber.StatusBadRequest, "invalid token data")
	}

	reqEntity := entity.CartItem{
		ProductID: request.ProductID,
		Quantity:  request.Quantity,
	}

	if err := ch.cartService.AddToCart(ctx, jwtUserData.UserID, reqEntity); err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.adapter.cartHandler.AddToCart").
			Msg("failed add to cart")

		return err
	}

	return c.Status(fiber.StatusCreated).JSON(response.DefaultResponse{
		Message: "success",
		Data:    nil,
	})
}

func (ch *cartHandler) GetCart(c fiber.Ctx) error {
	var (
		jwtUserData entity.JwtUserData
		respList    []response.CartResponse
	)

	ctx := c.Context()

	user, ok := c.Locals("user").(string)
	if !ok || user == "" {
		log.Error().
			Str("source", "internal.adapter.cartHandler.GetCart").
			Msg("data token not found")

		return fiber.NewError(fiber.StatusUnauthorized, "data token not found")
	}

	if err := json.Unmarshal([]byte(user), &jwtUserData); err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.adapter.cartHandler.GetCart").
			Msg("failed parse jwt user data")

		return fiber.NewError(fiber.StatusBadRequest, "invalid token data")
	}

	items, err := ch.cartService.GetCartByUserID(ctx, jwtUserData.UserID)
	if err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.adapter.cartHandler.GetCart").
			Msg("failed get cart by user id")

		return err
	}

	for _, item := range items {
		product, err := ch.productService.GetByID(ctx, item.ProductID)
		if err != nil {
			log.Error().
				Err(err).
				Int64("product_id", item.ProductID).
				Str("source", "internal.adapter.cartHandler.GetCart").
				Msg("failed get product by id")

			return err
		}

		respList = append(respList, response.CartResponse{
			ID:            item.ProductID,
			ProductName:   product.Name,
			ProductImage:  product.Image,
			ProductStatus: product.Status,
			SalePrice:     int64(product.SalePrice),
			Quantity:      item.Quantity,
			Unit:          product.Unit,
			Weight:        int64(product.Weight),
		})
	}

	return c.Status(fiber.StatusOK).JSON(response.DefaultResponse{
		Message: "success",
		Data:    respList,
	})
}

func (ch *cartHandler) RemoveFromCart(c fiber.Ctx) error {
	var jwtUserData entity.JwtUserData

	ctx := c.Context()

	user, ok := c.Locals("user").(string)
	if !ok || user == "" {
		log.Error().
			Str("source", "internal.adapter.cartHandler.RemoveFromCart").
			Msg("data token not found")

		return fiber.NewError(fiber.StatusUnauthorized, "data token not found")
	}

	if err := json.Unmarshal([]byte(user), &jwtUserData); err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.adapter.cartHandler.RemoveFromCart").
			Msg("failed parse jwt user data")

		return fiber.NewError(fiber.StatusBadRequest, "invalid token data")
	}

	productID := c.Query("product_id")
	if productID == "" {
		log.Error().
			Str("source", "internal.adapter.cartHandler.RemoveFromCart").
			Msg("product_id is required")

		return fiber.NewError(fiber.StatusBadRequest, "product_id is required")
	}

	prodID, err := conv.StringToInt64(productID)
	if err != nil {
		log.Error().
			Err(err).
			Str("product_id", productID).
			Str("source", "internal.adapter.cartHandler.RemoveFromCart").
			Msg("failed convert product_id")

		return fiber.NewError(fiber.StatusBadRequest, "invalid product_id")
	}

	if err := ch.cartService.RemoveFromCart(ctx, jwtUserData.UserID, prodID); err != nil {
		log.Error().
			Err(err).
			Int64("product_id", prodID).
			Str("source", "internal.adapter.cartHandler.RemoveFromCart").
			Msg("failed remove from cart")

		return err
	}

	return c.Status(fiber.StatusOK).JSON(response.DefaultResponse{
		Message: "success",
		Data:    nil,
	})
}
