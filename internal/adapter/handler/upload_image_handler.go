package handler

import (
	"product-service/internal/adapter/storage"

	"github.com/gofiber/fiber/v3"
)

type uploadImage struct {
	storageHandler storage.MinioStorageInterface
}

type UploadImageInterface interface {
	UploadImage(c fiber.Ctx) error
}
