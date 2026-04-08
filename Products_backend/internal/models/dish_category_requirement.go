package models

type DishCategoryRequirement struct {
	ID int64 `gorm:"primaryKey;autoIncrement" json:"id"`

	DishID int64 `gorm:"index;not null" json:"dish_id"`

	// CategoryID опционален: можно привязать ингредиент к категории, но это не обязательно.
	CategoryID *int64 `gorm:"index" json:"category_id,omitempty"`

	// IngredientName — "сущность" ингредиента, не конкретная упаковка/SKU.
	IngredientName string `gorm:"type:text;not null;default:''" json:"ingredient_name"`

	Quantity int `gorm:"not null" json:"quantity"`

	Note string `gorm:"type:text;default:''" json:"note"`

	// Связи
	Dish     *Dish     `gorm:"foreignKey:DishID" json:"dish,omitempty"`
	Category *Category `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
}

type CreateDishCategoryRequirement struct {
	CategoryID     *int64 `json:"category_id"`
	IngredientName string `json:"ingredient_name" binding:"required"`
	Quantity       int    `json:"quantity" binding:"required,min=1"`
	Note           string `json:"note"`
}

type UpdateDishCategoryRequirement struct {
	CategoryID     *int64  `json:"category_id"`
	IngredientName *string `json:"ingredient_name"`
	Quantity       *int    `json:"quantity"`
	Note           *string `json:"note"`
}
