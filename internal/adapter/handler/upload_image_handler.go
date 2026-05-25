package handler

import (
	"product-service/config"
	"product-service/internal/adapter"
	"product-service/internal/adapter/handler/response"
	"product-service/internal/adapter/storage"
	"product-service/internal/core/service"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

type uploadImage struct {
	storageHandler storage.MinioStorageInterface
}

type UploadImageInterface interface {
	UploadImage(c fiber.Ctx) error
}

func NewUploadImage(
	app *fiber.App,
	cfg *config.Config,
	storageHandler storage.MinioStorageInterface,
	jwtService service.JwtServiceInterface,
	redis *redis.Client,
) UploadImageInterface {
	res := &uploadImage{
		storageHandler: storageHandler,
	}

	mid := adapter.NewMiddlewareAdapter(cfg, jwtService, redis)
	adminGroup := app.Group("/admin", mid.CheckToken())
	adminGroup.Post("/image-upload", res.UploadImage)

	return res
}

func (u *uploadImage) UploadImage(c fiber.Ctx) error {
	fileHeader, err := c.FormFile("image")
	if err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.adapter.uploadImage.UploadImage").
			Msg("failed get uploaded file")

		return fiber.NewError(
			fiber.StatusUnprocessableEntity,
			err.Error(),
		)
	}

	ctx := c.Context()

	url, err := u.storageHandler.ProcessAndUploadImage(
		ctx,
		fileHeader,
	)
	if err != nil {
		log.Error().
			Err(err).
			Str("filename", fileHeader.Filename).
			Str("source", "internal.adapter.uploadImage.UploadImage").
			Msg("failed upload image")

		return fiber.NewError(
			fiber.StatusInternalServerError,
			err.Error(),
		)
	}

	return c.Status(fiber.StatusOK).JSON(response.DefaultResponse{
		Message: "success",
		Data: map[string]string{
			"image_url": url,
		},
	})
}
