package handlers

import (
	"Products_backend/internal/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type FavouriteHandler struct {
	DB *gorm.DB
}

func NewFavouriteHandler(db *gorm.DB) *FavouriteHandler {
	return &FavouriteHandler{DB: db}
}

func (h *FavouriteHandler) AddProduct(c *gin.Context) {
	type addReq struct {
		ProductID int64 `json:"product_id" binding:"required"`
	}
	var req addReq

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	uidVal, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID, ok := uidVal.(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var product models.Product
	if err := h.DB.First(&product, req.ProductID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product_id"})
		return
	}

	var existing models.FavouriteProduct
	if err := h.DB.Where("user_id = ? AND product_id = ?", userID, req.ProductID).First(&existing).Error; err == nil {
		c.JSON(http.StatusOK, existing)
		return
	}

	fav := models.FavouriteProduct{
		UserID:    userID,
		ProductID: req.ProductID,
	}
	if err := h.DB.Create(&fav).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, fav)
}

func (h *FavouriteHandler) ListMyProducts(c *gin.Context) {
	uidVal, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID, ok := uidVal.(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var rows []models.FavouriteProduct
	if err := h.DB.Where("user_id = ?", userID).Preload("Product").Order("id DESC").Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rows)
}

func (h *FavouriteHandler) RemoveProduct(c *gin.Context) {
	uidVal, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID, ok := uidVal.(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	productID, err := strconv.ParseInt(c.Param("productId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}
	if err := h.DB.Where("user_id = ? AND product_id = ?", userID, productID).Delete(&models.FavouriteProduct{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
