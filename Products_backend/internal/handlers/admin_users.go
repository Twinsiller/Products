package handlers

import (
	"net/http"
	"strconv"

	"Products_backend/internal/database"
	"Products_backend/internal/models"

	"github.com/gin-gonic/gin"
)

// ListUsersDetailed возвращает пользователей со связанными данными (только для админа).
func ListUsersDetailed(c *gin.Context) {
	var users []models.User
	err := database.DbPostgres.
		Preload("Orders").
		Preload("Orders.Items").
		Preload("Orders.Items.Product").
		Preload("FavouriteProducts").
		Preload("FavouriteProducts.Product").
		Preload("FavouriteDishes").
		Preload("FavouriteDishes.Dish").
		Find(&users).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, users)
}

// GetUserDetailed возвращает одного пользователя со связанными данными (только для админа).
func GetUserDetailed(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var user models.User
	err = database.DbPostgres.
		Preload("Orders").
		Preload("Orders.Items").
		Preload("Orders.Items.Product").
		Preload("FavouriteProducts").
		Preload("FavouriteProducts.Product").
		Preload("FavouriteDishes").
		Preload("FavouriteDishes.Dish").
		First(&user, id).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}
