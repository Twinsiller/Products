package services

import (
	"Products_backend/internal/models"

	"gorm.io/gorm"
)

type ProductService struct {
	DB *gorm.DB
}

func NewProductService(db *gorm.DB) *ProductService {
	return &ProductService{DB: db}
}

func (s *ProductService) Create(product *models.Product) error {
	return s.DB.Create(product).Error
}

func (s *ProductService) Update(product *models.Product) error {
	return s.DB.Save(product).Error
}
