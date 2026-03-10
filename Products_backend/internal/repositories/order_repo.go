package repositories

import (
	"Products_backend/internal/models"

	"gorm.io/gorm"
)

type OrderRepository struct {
	DB *gorm.DB
}

func NewOrderRepository(db *gorm.DB) *OrderRepository {
	return &OrderRepository{DB: db}
}

func (r *OrderRepository) Create(order *models.Order) error {
	return r.DB.Create(order).Error
}

func (r *OrderRepository) GetByID(id int64) (*models.Order, error) {
	var order models.Order
	err := r.DB.Preload("Items").First(&order, id).Error
	return &order, err
}

func (r *OrderRepository) Update(order *models.Order) error {
	return r.DB.Save(order).Error
}

func (r *OrderRepository) Delete(id int64) error {
	return r.DB.Delete(&models.Order{}, id).Error
}
