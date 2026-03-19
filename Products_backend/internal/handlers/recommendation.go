package handlers

import (
	"net/http"
	"strconv"

	"Products_backend/internal/services"

	"github.com/gin-gonic/gin"
)

type RecommendationHandler struct {
	Service *services.RecommendationService
}

// GetProductRecommendations возвращает рекомендованные товары для пользователя.
// GET /v1/users/:id/recommendations/products?limit=10
func (h *RecommendationHandler) GetProductRecommendations(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	limitStr := c.Query("limit")
	limit := 10
	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 {
			limit = v
		}
	}

	recs, err := h.Service.RecommendProducts(userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, recs)
}

// GetDishRecommendations возвращает рекомендованные блюда для пользователя.
// GET /v1/users/:id/recommendations/dishes?limit=10
func (h *RecommendationHandler) GetDishRecommendations(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	limitStr := c.Query("limit")
	limit := 10
	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 {
			limit = v
		}
	}

	recs, err := h.Service.RecommendDishes(userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, recs)
}

