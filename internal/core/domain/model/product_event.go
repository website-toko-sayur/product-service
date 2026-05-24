package model

import (
	"product-service/internal/core/domain/entity"
	"strconv"
	"time"
)

type DeleteProductEvent struct {
	ProductID int64 `json:"product_id"`
}

func (u *DeleteProductEvent) GetId() string {
	return strconv.Itoa(int(u.ProductID))
}

type ProductEvent struct {
	ID           int64          `json:"id"`
	CategorySlug string         `json:"category_slug"`
	ParentID     *int64         `json:"parent_id"`
	Name         string         `json:"name"`
	Image        string         `json:"image"`
	Description  string         `json:"description"`
	RegulerPrice float64        `json:"reguler_price"`
	SalePrice    float64        `json:"sale_price"`
	Unit         string         `json:"unit"`
	Weight       int            `json:"weight"`
	Stock        int            `json:"stock"`
	Variant      int            `json:"variant"`
	Status       string         `json:"status"`
	CategoryName string         `json:"category_name"`
	Child        []ProductEvent `json:"child"`
	CreatedAt    time.Time      `json:"created_at"`
}

func MapProductChildren(children []entity.ProductEntity) []ProductEvent {
	result := make([]ProductEvent, 0, len(children))

	for _, child := range children {
		result = append(result, ProductEvent{
			ID:           child.ID,
			CategorySlug: child.CategorySlug,
			ParentID:     child.ParentID,
			Name:         child.Name,
			Image:        child.Image,
			Description:  child.Description,
			RegulerPrice: child.RegulerPrice,
			SalePrice:    child.SalePrice,
			Unit:         child.Unit,
			Weight:       child.Weight,
			Stock:        child.Stock,
			Variant:      child.Variant,
			Status:       child.Status,
			CategoryName: child.CategoryName,
			Child:        []ProductEvent{},
			CreatedAt:    child.CreatedAt,
		})
	}

	return result
}

func (u *ProductEvent) GetId() string {
	return strconv.Itoa(int(u.ID))
}
