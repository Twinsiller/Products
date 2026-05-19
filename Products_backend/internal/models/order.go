package models

import "time"

type Order struct {
	ID int64 `gorm:"primaryKey;autoIncrement" json:"id"`

	UserID int64 `gorm:"index;not null" json:"user_id"`

	CreatedAt time.Time `gorm:"type:timestamptz;default:now()" json:"created_at"`

	TotalAmount float64 `gorm:"type:numeric(14,2);default:0" json:"total_amount"`

	// Связи
	User  *User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Items []OrderItem `gorm:"foreignKey:OrderID" json:"items,omitempty"`
}

type CreateOrder struct {
	UserID int64 `json:"user_id" binding:"required"`
}

type UpdateOrder struct {
	TotalAmount *float64 `json:"total_amount"`
}
