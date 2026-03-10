package models

type FavouriteDish struct {
	ID     int64 `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID int64 `gorm:"index;not null" json:"user_id"`
	DishID int64 `gorm:"index;not null" json:"dish_id"`
}
