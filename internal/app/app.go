package app

import (
	"context"
	"os"
	"os/signal"
	"product-service/config"
	"product-service/internal/adapter/handler"
	messageproducer "product-service/internal/adapter/message/producer"
	"product-service/internal/adapter/repository"
	"product-service/internal/adapter/storage"
	"product-service/internal/core/service"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	fiberCors "github.com/gofiber/fiber/v3/middleware/cors"
	fiberRecover "github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/rs/zerolog/log"
)

func RunServer() {
	cfg := config.NewConfig()

	config.NewLogger(cfg.App.AppEnv, cfg.App.LogLevel, cfg.App.AppName)

	db, err := cfg.ConnectionPostgres()
	if err != nil {
		log.Fatal().
			Err(err).
			Str("source", "internal.app.RunServer").
			Msg("failed connect postgres")
	}

	sqlDB, err := db.DB.DB()
	if err != nil {
		log.Fatal().
			Err(err).
			Str("source", "internal.app.RunServer").
			Msg("failed get sql db instance")
	}
	defer sqlDB.Close()

	minio, err := cfg.NewMinio()
	if err != nil {
		log.Fatal().
			Err(err).
			Str("source", "internal.app.RunServer").
			Msg("failed connect to minio")
	}

	redis, err := cfg.NewRedisClient()
	if err != nil {
		log.Fatal().
			Err(err).
			Str("source", "internal.app.RunServer").
			Msg("failed connect to redis")
	}
	defer redis.Close()

	opensearch, err := cfg.NewOpenSearch()
	if err != nil {
		log.Fatal().
			Err(err).
			Str("source", "internal.app.RunServer").
			Msg("failed connect to opensearch")
	}

	producer := cfg.NewKafkaProducer()

	var (
		productDeleteProducer  *messageproducer.ProductDeleteProducer
		productPublishProducer *messageproducer.ProductPublishProducer
	)

	if producer != nil {
		productDeleteProducer = messageproducer.NewProductDeleteProducer(producer, cfg)
		productPublishProducer = messageproducer.NewProductPublishProducer(producer, cfg)
	}

	storageHandler := storage.NewMinioStorage(cfg, minio)

	cartRepo := repository.NewCartRepository(redis)
	categoryRepo := repository.NewCategoryRepository(db.DB)
	productRepo := repository.NewProductRepository(db.DB, opensearch)

	jwtService := service.NewJwtService(cfg)
	cartService := service.NewCartService(cartRepo)
	categoryService := service.NewCategoryService(categoryRepo)
	productService := service.NewProductService(
		productRepo,
		categoryRepo,
		productDeleteProducer,
		productPublishProducer,
	)

	app := cfg.NewFiber()

	app.Use(fiberRecover.New())
	app.Use(fiberCors.New())

	app.Get("/api/check", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})

	handler.NewUploadImage(app, cfg, storageHandler, jwtService, redis)
	handler.NewCartHandler(app, cartService, productService, cfg, jwtService, redis)
	handler.NewCategoryHandler(app, categoryService, cfg, jwtService, redis)
	handler.NewProductHandler(app, productService, cfg, jwtService, redis)

	go func() {
		if cfg.App.AppPort == "" {
			cfg.App.AppPort = os.Getenv("APP_PORT")
		}

		port := ":" + cfg.App.AppPort

		log.Info().
			Str("port", port).
			Str("source", "internal.app.RunServer").
			Msg("server started")

		err = app.Listen(
			port,
			fiber.ListenConfig{
				EnablePrefork: cfg.App.WebPrefork,
			},
		)

		if err != nil {
			log.Fatal().
				Err(err).
				Str("source", "internal.app.RunServer").
				Msg("failed start server")
		}
	}()

	terminateSignals := make(chan os.Signal, 1)

	signal.Notify(
		terminateSignals,
		os.Interrupt,
		syscall.SIGTERM,
	)

	<-terminateSignals

	log.Info().
		Str("source", "internal.app.RunServer").
		Msg("shutting down server in 5 seconds")

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.app.RunServer").
			Msg("failed shutdown server")
	}

	log.Info().
		Str("source", "internal.app.RunServer").
		Msg("server stopped gracefully")

}
