package models

import "time"

// User — сотрудник системы
type User struct {
	ID int64 `gorm:"primaryKey;autoIncrement" json:"id"`
	// Уникальный ID

	Name string `gorm:"type:text;not null" json:"name"`
	// Имя сотрудника

	Role string `gorm:"type:text;not null" json:"role"`
	// Роль (admin, cashier и т.д.)

	HiredAt time.Time `gorm:"type:timestamptz;default:now()" json:"hired_at"`
	// Дата найма
}

type CreateUser struct {
	Name string `json:"name" binding:"required"`
	Role string `json:"role" binding:"required"`
}

type UpdateUser struct {
	Name *string `json:"name"`
	Role *string `json:"role"`
}
