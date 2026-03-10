package services

import (
	"Products_backend/internal/models"
	"Products_backend/internal/repositories"

	"gorm.io/gorm"
)

type OrderService struct {
	DB   *gorm.DB
	Repo *repositories.OrderRepository
}

func NewOrderService(db *gorm.DB) *OrderService {
	return &OrderService{
		DB:   db,
		Repo: repositories.NewOrderRepository(db),
	}
}

func (s *OrderService) CreateOrder(order *models.Order, items []models.OrderItem) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {

		if err := tx.Create(order).Error; err != nil {
			return err
		}

		var total float64

		for i := range items {
			items[i].OrderID = order.ID
			total += float64(items[i].Quantity) * items[i].PricePerUnit

			if err := tx.Create(&items[i]).Error; err != nil {
				return err
			}
		}

		order.TotalAmount = total

		if err := tx.Save(order).Error; err != nil {
			return err
		}

		return nil
	})
}
