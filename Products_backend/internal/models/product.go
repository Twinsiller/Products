package models

import "time"

type Product struct {
	ID int64 `gorm:"primaryKey;autoIncrement" json:"id"`

	Name string `gorm:"type:text;not null" json:"name"`

	CategoryID     *int64 `gorm:"index" json:"category_id"`
	ManufacturerID *int64 `gorm:"index" json:"manufacturer_id"`

	Barcode *string `gorm:"type:text;unique" json:"barcode"`

	DefaultPrice float64 `gorm:"type:numeric(12,2);not null" json:"default_price"`

	// КБЖУ на одну единицу товара (как в каталоге), для расчёта блюд и калорийности.
	CaloriesKcal float64 `gorm:"type:numeric(10,2);default:0" json:"calories_kcal"`
	ProteinG     float64 `gorm:"type:numeric(10,2);default:0" json:"protein_g"`
	FatG         float64 `gorm:"type:numeric(10,2);default:0" json:"fat_g"`
	CarbsG       float64 `gorm:"type:numeric(10,2);default:0" json:"carbs_g"`

	CreatedAt time.Time `gorm:"type:timestamptz;default:now()" json:"created_at"`

	// Связи
	Category     *Category     `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	Manufacturer *Manufacturer `gorm:"foreignKey:ManufacturerID" json:"manufacturer,omitempty"`
	OrderItems   []OrderItem   `gorm:"foreignKey:ProductID" json:"order_items,omitempty"`
	DishProducts []DishProduct `gorm:"foreignKey:ProductID" json:"dish_products,omitempty"`
	Favourites   []FavouriteProduct `gorm:"foreignKey:ProductID" json:"favourites,omitempty"`
}

type CreateProduct struct {
	Name           string  `json:"name" binding:"required"`
	CategoryID     *int64  `json:"category_id"`
	ManufacturerID *int64  `json:"manufacturer_id"`
	Barcode        *string `json:"barcode"`
	DefaultPrice   float64 `json:"default_price" binding:"required"`
	CaloriesKcal   float64 `json:"calories_kcal"`
	ProteinG       float64 `json:"protein_g"`
	FatG           float64 `json:"fat_g"`
	CarbsG         float64 `json:"carbs_g"`
}

type UpdateProduct struct {
	Name           *string  `json:"name"`
	CategoryID     *int64   `json:"category_id"`
	ManufacturerID *int64   `json:"manufacturer_id"`
	Barcode        *string  `json:"barcode"`
	DefaultPrice   *float64 `json:"default_price"`
	CaloriesKcal   *float64 `json:"calories_kcal"`
	ProteinG       *float64 `json:"protein_g"`
	FatG           *float64 `json:"fat_g"`
	CarbsG         *float64 `json:"carbs_g"`
}
