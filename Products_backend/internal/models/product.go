package models

import "time"

type Product struct {
	ID int64 `gorm:"primaryKey;autoIncrement" json:"id"`

	Name string `gorm:"type:text;not null" json:"name"`

	CategoryID     *int64 `gorm:"index" json:"category_id"`
	ManufacturerID *int64 `gorm:"index" json:"manufacturer_id"`

	Barcode *string `gorm:"type:text;unique" json:"barcode"`

	DefaultPrice float64 `gorm:"type:numeric(12,2);not null" json:"default_price"`

	CreatedAt time.Time `gorm:"type:timestamptz;default:now()" json:"created_at"`
}

type CreateProduct struct {
	Name           string  `json:"name" binding:"required"`
	CategoryID     *int64  `json:"category_id"`
	ManufacturerID *int64  `json:"manufacturer_id"`
	Barcode        *string `json:"barcode"`
	DefaultPrice   float64 `json:"default_price" binding:"required"`
}

type UpdateProduct struct {
	Name           *string  `json:"name"`
	CategoryID     *int64   `json:"category_id"`
	ManufacturerID *int64   `json:"manufacturer_id"`
	Barcode        *string  `json:"barcode"`
	DefaultPrice   *float64 `json:"default_price"`
}
