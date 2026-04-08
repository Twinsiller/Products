package models

type Dish struct {
	ID   int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	Name string `gorm:"type:text;not null" json:"name"`

	// Связи
	Products             []DishProduct             `gorm:"foreignKey:DishID" json:"products,omitempty"`
	CategoryRequirements []DishCategoryRequirement `gorm:"foreignKey:DishID" json:"category_requirements,omitempty"`
	Favourites           []FavouriteDish           `gorm:"foreignKey:DishID" json:"favourites,omitempty"`
}

type CreateDish struct {
	Name string `json:"name" binding:"required"`
}

type UpdateDish struct {
	Name *string `json:"name"`
}
