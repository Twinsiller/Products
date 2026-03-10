package models

import "time"

type Order struct {
	ID int64 `gorm:"primaryKey;autoIncrement" json:"id"`

	CashierID int64 `gorm:"index;not null" json:"cashier_id"`

	CreatedAt time.Time `gorm:"type:timestamptz;default:now()" json:"created_at"`

	TotalAmount float64 `gorm:"type:numeric(14,2);default:0" json:"total_amount"`
}

type CreateOrder struct {
	CashierID int64 `json:"cashier_id" binding:"required"`
}

type UpdateOrder struct {
	TotalAmount *float64 `json:"total_amount"`
}
