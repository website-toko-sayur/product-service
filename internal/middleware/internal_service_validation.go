package middleware

import (
	"product-service/config"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"
)

func InternalValidationMiddleware(cfg *config.Config) fiber.Handler {
	expectedSecret := cfg.App.InternalSecretKey

	return func(c fiber.Ctx) error {
		if c.Path() == "/health" {
			return c.Next()
		}

		internalHeader := c.Get("X-Internal-Service")
		if internalHeader != "true" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"status":  "error",
				"message": "Access denied. Internal service only.",
				"code":    "INTERNAL_ONLY",
			})
		}

		if expectedSecret != "" {
			receivedSecret := c.Get("X-Internal-Secret")

			if receivedSecret != expectedSecret {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"status":  "error",
					"message": "Invalid internal secret key.",
					"code":    "INVALID_INTERNAL_SECRET",
				})
			}
		}

		fromService := c.Get("X-From-Service")
		if fromService == "" {
			fromService = "unknown"
		}

		log.Info().
			Str("from_service", fromService).
			Str("request_id", c.Get("X-Request-ID")).
			Msg("Internal service request validated")

		return c.Next()
	}
}
