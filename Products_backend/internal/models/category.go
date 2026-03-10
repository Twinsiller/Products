package models

type Category struct {
	ID   int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	Name string `gorm:"type:text;not null" json:"name"`
}

type CreateCategory struct {
	Name string `json:"name" binding:"required"`
}

type UpdateCategory struct {
	Name *string `json:"name"`
}
