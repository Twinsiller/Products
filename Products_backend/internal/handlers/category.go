package handlers

import (
	"net/http"
	"strings"

	"Products_backend/internal/database"
	"Products_backend/internal/models"

	"github.com/gin-gonic/gin"
)

// CreateCategory создаёт категорию, если категория с таким названием ещё не существует.
func CreateCategory(c *gin.Context) {
	var req models.CreateCategory
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	var count int64
	if err := database.DbPostgres.Model(&models.Category{}).Where("name = ?", name).Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "категория с таким названием уже существует"})
		return
	}
	cat := models.Category{Name: name}
	if err := database.DbPostgres.Create(&cat).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, cat)
}
