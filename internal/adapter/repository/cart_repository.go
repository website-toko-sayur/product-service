package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"product-service/internal/core/domain/entity"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

type cartRepository struct {
	redis *redis.Client
}

type CartRepositoryInterface interface {
	AddToCart(ctx context.Context, userID string, items []entity.CartItem) error
	GetCart(ctx context.Context, userID string) ([]entity.CartItem, error)
	RemoveFromCart(ctx context.Context, userID int64, productID int64) error
	RemoveAllCart(ctx context.Context, userID int64) error
}

func NewCartRepository(redis *redis.Client) CartRepositoryInterface {
	return &cartRepository{
		redis: redis,
	}
}

func (c *cartRepository) RemoveAllCart(ctx context.Context, userID int64) error {
	return c.redis.Del(ctx, fmt.Sprintf("cart:cart:%d", userID)).Err()
}

func (c *cartRepository) AddToCart(ctx context.Context, userID string, items []entity.CartItem) error {
	data, err := json.Marshal(items)
	if err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.adapter.cartRedisRepository.AddToCart").
			Msg("failed to add cart data to redis")
		return err
	}
	return c.redis.Set(ctx, fmt.Sprintf("cart:%s", userID), data, 0).Err()
}

func (c *cartRepository) GetCart(ctx context.Context, userID string) ([]entity.CartItem, error) {
	val, err := c.redis.Get(ctx, fmt.Sprintf("cart:%s", userID)).Result()
	if err == redis.Nil {
		log.Info().
			Str("user_id", userID).
			Str("source", "internal.adapter.cartRepository.GetCart").
			Msg("cart not found")
		return nil, nil
	}
	if err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Str("source", "internal.adapter.cartRepository.GetCart").
			Msg("failed to get cart from redis")
		return nil, err
	}
	var items []entity.CartItem
	err = json.Unmarshal([]byte(val), &items)
	if err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Str("source", "internal.adapter.cartRepository.GetCart").
			Msg("failed to unmarshal cart data")
		return nil, err
	}
	return items, nil
}

func (c *cartRepository) RemoveFromCart(ctx context.Context, userID int64, productID int64) error {
	cart, err := c.GetCart(ctx, fmt.Sprintf("cart:%d", userID))
	if err != nil {
		log.Error().
			Err(err).
			Int64("user_id", userID).
			Int64("product_id", productID).
			Str("source", "internal.adapter.cartRepository.RemoveFromCart").
			Msg("failed to get cart")
		return err
	}

	newCart := []entity.CartItem{}
	for _, item := range cart {
		if item.ProductID != productID {
			newCart = append(newCart, item)
		}
	}

	err = c.redis.Del(ctx, fmt.Sprintf("cart:cart:%d", userID)).Err()
	if err != nil {
		log.Error().
			Err(err).
			Int64("user_id", userID).
			Int64("product_id", productID).
			Str("source", "internal.adapter.cartRepository.RemoveFromCart").
			Msg("failed to delete cart from redis")
		return err
	}

	err = c.AddToCart(ctx, fmt.Sprintf("%d", userID), newCart)
	if err != nil {
		log.Error().
			Err(err).
			Int64("user_id", userID).
			Int64("product_id", productID).
			Str("source", "internal.adapter.cartRepository.RemoveFromCart").
			Msg("failed to save updated cart")
		return err
	}

	return nil
}
