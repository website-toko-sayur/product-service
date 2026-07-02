package service

import (
	"context"
	"fmt"
	"product-service/internal/adapter/repository"
	"product-service/internal/core/domain/entity"

	"github.com/rs/zerolog/log"
)

type cartService struct {
	repo repository.CartRepositoryInterface
}

type CartServiceInterface interface {
	AddToCart(ctx context.Context, userID int64, req entity.CartItem) error
	GetCartByUserID(ctx context.Context, userID int64) ([]entity.CartItem, error)
	RemoveFromCart(ctx context.Context, userID int64, productID int64) error
	RemoveAllCart(ctx context.Context, userID int64) error
}

func NewCartService(repo repository.CartRepositoryInterface) CartServiceInterface {
	return &cartService{repo: repo}
}

func (c *cartService) RemoveAllCart(ctx context.Context, userID int64) error {
	return c.repo.RemoveAllCart(ctx, userID)
}

func (c *cartService) RemoveFromCart(ctx context.Context, userID int64, productID int64) error {
	return c.repo.RemoveFromCart(ctx, userID, productID)
}

func (c *cartService) AddToCart(ctx context.Context, userID int64, req entity.CartItem) error {
	cartKey := fmt.Sprintf("cart:%d", userID)

	cart, err := c.repo.GetCart(ctx, cartKey)
	if err != nil {
		log.Error().
			Err(err).
			Int64("user_id", userID).
			Str("cart_key", cartKey).
			Str("source", "internal.core.service.cartService.AddToCart").
			Msg("failed get cart")
		return err
	}

	found := false
	for i, item := range cart {
		if item.ProductID == req.ProductID {
			cart[i].Quantity += req.Quantity
			found = true
			break
		}
	}

	if !found {
		cart = append(cart, req)
	}

	err = c.repo.AddToCart(ctx, cartKey, cart)
	if err != nil {
		log.Error().
			Err(err).
			Int64("user_id", userID).
			Str("cart_key", cartKey).
			Str("source", "internal.core.service.cartService.AddToCart").
			Msg("failed save cart")

		return err
	}

	return nil
}

func (c *cartService) GetCartByUserID(ctx context.Context, userID int64) ([]entity.CartItem, error) {
	cartKey := fmt.Sprintf("cart:%d", userID)

	cart, err := c.repo.GetCart(ctx, cartKey)
	if err != nil {
		log.Error().
			Err(err).
			Int64("user_id", userID).
			Str("cart_key", cartKey).
			Str("source", "internal.core.service.cartService.GetCartByUserID").
			Msg("failed get cart")

		return nil, err
	}

	return cart, nil
}
