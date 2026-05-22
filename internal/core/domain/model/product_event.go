package model

import (
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

func (u *ProductEvent) GetId() string {
	return strconv.Itoa(int(u.ID))
}
