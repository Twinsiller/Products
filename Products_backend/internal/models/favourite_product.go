package models

type FavouriteProduct struct {
	ID        int64 `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    int64 `gorm:"index;not null" json:"user_id"`
	ProductID int64 `gorm:"index;not null" json:"product_id"`
}
