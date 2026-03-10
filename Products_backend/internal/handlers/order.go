package handlers

import (
	"Products_backend/internal/models"
	"Products_backend/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	Service *services.OrderService
}

func NewOrderHandler(service *services.OrderService) *OrderHandler {
	return &OrderHandler{Service: service}
}

func (h *OrderHandler) Create(c *gin.Context) {
	var req struct {
		Order models.Order       `json:"order"`
		Items []models.OrderItem `json:"items"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, err)
		return
	}

	if err := h.Service.CreateOrder(&req.Order, req.Items); err != nil {
		c.JSON(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusCreated, req.Order)
}
