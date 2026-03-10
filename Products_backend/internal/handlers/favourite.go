package handlers

import (
	"Products_backend/internal/models"
	"net/http"

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
	var fav models.FavouriteProduct

	if err := c.BindJSON(&fav); err != nil {
		c.JSON(http.StatusBadRequest, err)
		return
	}

	if err := h.DB.Create(&fav).Error; err != nil {
		c.JSON(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusCreated, fav)
}
