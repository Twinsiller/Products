package handlers

import (
	"Products_backend/internal/models"
	"Products_backend/internal/services"
	"net/http"

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
