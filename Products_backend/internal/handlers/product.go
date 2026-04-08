package handlers

import (
	"Products_backend/internal/database"
	"Products_backend/internal/models"
	"Products_backend/internal/services"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type ProductHandler struct {
	Service *services.ProductService
}

func NewProductHandler(s *services.ProductService) *ProductHandler {
	return &ProductHandler{Service: s}
}

func (h *ProductHandler) Create(c *gin.Context) {
	var product models.Product

	if err := c.BindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, err)
		return
	}

	if err := h.Service.Create(&product); err != nil {
		c.JSON(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusCreated, product)
}

type createProductRequest struct {
	Name           string  `json:"name" binding:"required"`
	CategoryID     int64   `json:"category_id" binding:"required"`
	ManufacturerID *int64  `json:"manufacturer_id"`
	Barcode        *string `json:"barcode"`
	DefaultPrice   float64 `json:"default_price" binding:"required"`
	CaloriesKcal   float64 `json:"calories_kcal"`
	ProteinG       float64 `json:"protein_g"`
	FatG           float64 `json:"fat_g"`
	CarbsG         float64 `json:"carbs_g"`
}

type updateProductRequest struct {
	ID             int64   `json:"id"`
	Name           string  `json:"name" binding:"required"`
	CategoryID     int64   `json:"category_id" binding:"required"`
	ManufacturerID *int64  `json:"manufacturer_id"`
	Barcode        *string `json:"barcode"`
	DefaultPrice   float64 `json:"default_price" binding:"required"`
	CaloriesKcal   float64 `json:"calories_kcal"`
	ProteinG       float64 `json:"protein_g"`
	FatG           float64 `json:"fat_g"`
	CarbsG         float64 `json:"carbs_g"`
}

// CreateProduct требует категорию у каждого товара.
func CreateProduct(c *gin.Context) {
	var req createProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	if req.CategoryID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "category_id is required"})
		return
	}

	var category models.Category
	if err := database.DbPostgres.First(&category, req.CategoryID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid category_id"})
		return
	}
	if req.ManufacturerID != nil && *req.ManufacturerID > 0 {
		var manufacturer models.Manufacturer
		if err := database.DbPostgres.First(&manufacturer, *req.ManufacturerID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid manufacturer_id"})
			return
		}
	}

	p := models.Product{
		Name:           strings.TrimSpace(req.Name),
		CategoryID:     &req.CategoryID,
		ManufacturerID: req.ManufacturerID,
		Barcode:        req.Barcode,
		DefaultPrice:   req.DefaultPrice,
		CaloriesKcal:   req.CaloriesKcal,
		ProteinG:       req.ProteinG,
		FatG:           req.FatG,
		CarbsG:         req.CarbsG,
	}
	if err := database.DbPostgres.Create(&p).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, p)
}

// UpdateProduct требует category_id и сохраняет товар целиком.
func UpdateProduct(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var existing models.Product
	if err := database.DbPostgres.First(&existing, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	var req updateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	if req.CategoryID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "category_id is required"})
		return
	}
	var category models.Category
	if err := database.DbPostgres.First(&category, req.CategoryID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid category_id"})
		return
	}
	if req.ManufacturerID != nil && *req.ManufacturerID > 0 {
		var manufacturer models.Manufacturer
		if err := database.DbPostgres.First(&manufacturer, *req.ManufacturerID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid manufacturer_id"})
			return
		}
	}

	existing.Name = strings.TrimSpace(req.Name)
	existing.CategoryID = &req.CategoryID
	existing.ManufacturerID = req.ManufacturerID
	existing.Barcode = req.Barcode
	existing.DefaultPrice = req.DefaultPrice
	existing.CaloriesKcal = req.CaloriesKcal
	existing.ProteinG = req.ProteinG
	existing.FatG = req.FatG
	existing.CarbsG = req.CarbsG

	if err := database.DbPostgres.Save(&existing).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, existing)
}
