package service

import (
	"context"
	"errors"
	"product-service/internal/adapter/repository"
	"product-service/internal/core/domain/entity"
	"product-service/utils/conv"

	"github.com/rs/zerolog/log"
)

type categoryService struct {
	repo repository.CategoryRepositoryInterface
}

type CategoryServiceInterface interface {
	GetAll(ctx context.Context, queryString entity.QueryStringEntity) ([]entity.CategoryEntity, int64, int64, error)
	GetByID(ctx context.Context, categoryID int64) (*entity.CategoryEntity, error)
	GetBySlug(ctx context.Context, slug string) (*entity.CategoryEntity, error)
	CreateCategory(ctx context.Context, req entity.CategoryEntity) error
	EditCategory(ctx context.Context, req entity.CategoryEntity) error
	DeleteCategory(ctx context.Context, categoryID int64) error

	GetAllPublished(ctx context.Context) ([]entity.CategoryEntity, error)
}

func NewCategoryService(repo repository.CategoryRepositoryInterface) CategoryServiceInterface {
	return &categoryService{repo: repo}
}

func (c *categoryService) GetAllPublished(ctx context.Context) ([]entity.CategoryEntity, error) {
	return c.repo.GetAllPublished(ctx)
}

func (c *categoryService) CreateCategory(ctx context.Context, req entity.CategoryEntity) error {
	slug := conv.GenerateSlug(req.Name)

	_, err := c.repo.GetBySlug(ctx, slug)
	if err != nil {
		if err.Error() != "404" {
			log.Error().
				Err(err).
				Str("category_name", req.Name).
				Str("category_slug", slug).
				Str("source", "internal.core.service.categoryService.CreateCategory").
				Msg("failed get category by slug")

			return err
		}
	} else {
		log.Info().
			Str("category_name", req.Name).
			Str("category_slug", slug).
			Str("source", "internal.core.service.categoryService.CreateCategory").
			Msg("category already exists")

		return errors.New("409")
	}

	req.Slug = slug

	err = c.repo.CreateCategory(ctx, req)
	if err != nil {
		log.Error().
			Err(err).
			Str("category_name", req.Name).
			Str("category_slug", slug).
			Str("source", "internal.core.service.categoryService.CreateCategory").
			Msg("failed create category")

		return err
	}

	return nil
}

func (c *categoryService) DeleteCategory(ctx context.Context, categoryID int64) error {
	return c.repo.DeleteCategory(ctx, categoryID)
}

func (c *categoryService) EditCategory(ctx context.Context, req entity.CategoryEntity) error {
	slug := conv.GenerateSlug(req.Name)

	category, err := c.repo.GetByID(ctx, req.ID)
	if err != nil {
		log.Error().
			Err(err).
			Int64("category_id", req.ID).
			Str("source", "internal.core.service.categoryService.EditCategory").
			Msg("failed get category by id")

		return err
	}

	if slug != category.Slug {
		_, err := c.repo.GetBySlug(ctx, slug)

		if err == nil {
			log.Info().
				Int64("category_id", req.ID).
				Str("category_slug", slug).
				Str("source", "internal.core.service.categoryService.EditCategory").
				Msg("category slug already exists")

			return errors.New("409")
		}

		if err.Error() != "404" {
			log.Error().
				Err(err).
				Int64("category_id", req.ID).
				Str("category_slug", slug).
				Str("source", "internal.core.service.categoryService.EditCategory").
				Msg("failed get category by slug")

			return err
		}
	}

	req.Slug = slug

	err = c.repo.EditCategory(ctx, req)
	if err != nil {
		log.Error().
			Err(err).
			Int64("category_id", req.ID).
			Str("category_slug", slug).
			Str("source", "internal.core.service.categoryService.EditCategory").
			Msg("failed edit category")

		return err
	}

	return nil
}

func (c *categoryService) GetAll(ctx context.Context, queryString entity.QueryStringEntity) ([]entity.CategoryEntity, int64, int64, error) {
	return c.repo.GetAll(ctx, queryString)
}

func (c *categoryService) GetByID(ctx context.Context, categoryID int64) (*entity.CategoryEntity, error) {
	return c.repo.GetByID(ctx, categoryID)
}

func (c *categoryService) GetBySlug(ctx context.Context, slug string) (*entity.CategoryEntity, error) {
	return c.repo.GetBySlug(ctx, slug)
}
