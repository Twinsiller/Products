package models

type Dish struct {
	ID   int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	Name string `gorm:"type:text;not null" json:"name"`
}

type CreateDish struct {
	Name string `json:"name" binding:"required"`
}

type UpdateDish struct {
	Name *string `json:"name"`
}
