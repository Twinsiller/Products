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

	PasswordHash string `gorm:"type:text;not null" json:"-"`

	HiredAt time.Time `gorm:"type:timestamptz;default:now()" json:"hired_at"`
	// Дата найма

	// Связи
	Orders            []Order            `gorm:"foreignKey:CashierID" json:"orders,omitempty"`
	FavouriteProducts []FavouriteProduct `gorm:"foreignKey:UserID" json:"favourite_products,omitempty"`
	FavouriteDishes   []FavouriteDish    `gorm:"foreignKey:UserID" json:"favourite_dishes,omitempty"`
}

type CreateUser struct {
	Name string `json:"name" binding:"required"`
	Role string `json:"role" binding:"required"`
}

// RegisterUser используется для публичной регистрации нового пользователя.
type RegisterUser struct {
	Name     string `json:"name" binding:"required"`
	Password string `json:"password" binding:"required,min=4"`
}

type UpdateUser struct {
	Name *string `json:"name"`
	Role *string `json:"role"`
}
