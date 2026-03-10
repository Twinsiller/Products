package models

type Country struct {
	ID   int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	Name string `gorm:"type:text;not null" json:"name"`
}

type CreateCountry struct {
	Name string `json:"name" binding:"required"`
}

type UpdateCountry struct {
	Name *string `json:"name"`
}
