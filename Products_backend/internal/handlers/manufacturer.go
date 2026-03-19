package handlers

import (
	"net/http"
	"strings"

	"Products_backend/internal/database"
	"Products_backend/internal/models"

	"github.com/gin-gonic/gin"
)

// CreateManufacturer создаёт производителя, если производитель с таким названием ещё не существует.
func CreateManufacturer(c *gin.Context) {
	var req models.CreateManufacturer
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
	if err := database.DbPostgres.Model(&models.Manufacturer{}).Where("name = ?", name).Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "производитель с таким названием уже существует"})
		return
	}
	contactInfo := strings.TrimSpace(req.ContactInfo)
	if contactInfo == "" {
		contactInfo = "{}" // jsonb требует валидный JSON; пустая строка недопустима
	}
	m := models.Manufacturer{
		Name:        name,
		Country:     strings.TrimSpace(req.Country),
		ContactInfo: contactInfo,
	}
	if err := database.DbPostgres.Create(&m).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, m)
}
