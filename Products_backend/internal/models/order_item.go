package models

type OrderItem struct {
	ID int64 `gorm:"primaryKey;autoIncrement" json:"id"`

	OrderID   int64 `gorm:"index;not null" json:"order_id"`
	ProductID int64 `gorm:"index;not null" json:"product_id"`

	Quantity int `gorm:"not null" json:"quantity"`

	PricePerUnit float64 `gorm:"type:numeric(12,2);not null" json:"price_per_unit"`

	Discount int `gorm:"default:0" json:"discount"`

	// Связи
	Order   *Order   `gorm:"foreignKey:OrderID" json:"order,omitempty"`
	Product *Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
}

type CreateOrderItem struct {
	OrderID      int64   `json:"order_id" binding:"required"`
	ProductID    int64   `json:"product_id" binding:"required"`
	Quantity     int     `json:"quantity" binding:"required"`
	PricePerUnit float64 `json:"price_per_unit" binding:"required"`
	Discount     int     `json:"discount"`
}

type UpdateOrderItem struct {
	Quantity     *int     `json:"quantity"`
	PricePerUnit *float64 `json:"price_per_unit"`
	Discount     *int     `json:"discount"`
}