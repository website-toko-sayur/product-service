package model

import (
	"time"

	"gorm.io/gorm"
)

type Product struct {
	ID           int64          `gorm:"primaryKey"`
	ParentID     *int64         `gorm:"column:parent_id"`
	CategorySlug string         `gorm:"column:category_slug;not null"`
	Name         string         `gorm:"column:name;not null"`
	Image        string         `gorm:"column:image;not null"`
	Description  string         `gorm:"column:description"`
	RegulerPrice float64        `gorm:"column:reguler_price;default:0"`
	SalePrice    float64        `gorm:"column:sale_price;default:0"`
	Unit         string         `gorm:"column:unit;default:'gram'"`
	Weight       int            `gorm:"column:weight;default:0"`
	Stock        int            `gorm:"column:stock;default:0"`
	Variant      int            `gorm:"column:variant;default:1"`
	Status       string         `gorm:"column:status;default:'DRAFT';size:20"`
	CreatedAt    time.Time      `gorm:"column:created_at;default:CURRENT_TIMESTAMP"`
	UpdatedAt    *time.Time     `gorm:"column:updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"column:deleted_at;index"`
	Childs       []Product      `gorm:"foreignKey:ParentID;references:ID"`
	Category     Category       `gorm:"foreignKey:CategorySlug;references:Slug"`
}
