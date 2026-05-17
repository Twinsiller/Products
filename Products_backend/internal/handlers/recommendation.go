package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"Products_backend/internal/services"
	"Products_backend/utils"

	"github.com/gin-gonic/gin"
)

type RecommendationHandler struct {
	Service *services.RecommendationService
}

type cartMealItemRequest struct {
	ProductID int64 `json:"product_id" binding:"required"`
	Quantity  int   `json:"quantity" binding:"required,min=1"`
}

type cartMealsRequest struct {
	Items []cartMealItemRequest `json:"items" binding:"required"`
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
		// В лог — техническая причина, наружу — пустой список (UI покажет «пока нет рекомендаций»).
		utils.Logger.Warnf("recommendations/products unavailable for user %d: %v", userID, err)
		c.JSON(http.StatusOK, []interface{}{})
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
		utils.Logger.Warnf("recommendations/dishes unavailable for user %d: %v", userID, err)
		c.JSON(http.StatusOK, []interface{}{})
		return
	}

	c.JSON(http.StatusOK, recs)
}

// GetMealsFromOrder — блюда по товарам из заказа, ранжирование по КБЖУ (GET .../users/:id/meals/from-order/:orderId).
func (h *RecommendationHandler) GetMealsFromOrder(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	uidVal, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	jwtUID, ok := uidVal.(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if jwtUID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	orderIDStr := c.Param("orderId")
	orderID, err := strconv.ParseInt(orderIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}

	limitStr := c.Query("limit")
	limit := 20
	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 {
			limit = v
		}
	}

	recs, err := h.Service.DishesFromOrder(userID, orderID, limit)
	if err != nil {
		if errors.Is(err, services.ErrOrderAccess) {
			c.JSON(http.StatusForbidden, gin.H{"error": "order not found or access denied"})
			return
		}
		utils.Logger.Errorf("meals/from-order failed for user %d order %d: %v", userID, orderID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось подобрать блюда по заказу"})
		return
	}

	c.JSON(http.StatusOK, recs)
}

// GetMealsFromCart — блюда по товарам из корзины (POST .../users/:id/meals/from-cart).
func (h *RecommendationHandler) GetMealsFromCart(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	uidVal, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	jwtUID, ok := uidVal.(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if jwtUID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	limitStr := c.Query("limit")
	limit := 20
	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 {
			limit = v
		}
	}

	var req cartMealsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	if len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cart is empty"})
		return
	}

	avail := make(map[int64]int)
	for _, it := range req.Items {
		avail[it.ProductID] += it.Quantity
	}

	recs, err := h.Service.DishesFromCart(userID, avail, limit)
	if err != nil {
		utils.Logger.Errorf("meals/from-cart failed for user %d: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось подобрать блюда по корзине"})
		return
	}

	c.JSON(http.StatusOK, recs)
}

// GetFinalRecommendations — финальные рекомендации из Python CF/CB модуля.
// GET /v1/users/:id/recommendations/final?limit=5
func (h *RecommendationHandler) GetFinalRecommendations(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	uidVal, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	jwtUID, ok := uidVal.(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if jwtUID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	limitStr := c.Query("limit")
	limit := 5
	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 {
			limit = v
		}
	}

	out, err := h.Service.RunFinalRecipeRecommender(userID, limit)
	if err != nil {
		// Полную причину отправляем в логи, пользователю отдаём пустой,
		// корректно сформированный ответ — UI покажет «пока нет рекомендаций».
		utils.Logger.Warnf("recommendations/final unavailable for user %d: %v", userID, err)
		c.JSON(http.StatusOK, gin.H{
			"recommendations": []interface{}{},
			"precision_at_5":  0,
			"available":       false,
		})
		return
	}
	if out == nil {
		out = map[string]interface{}{}
	}
	if _, ok := out["recommendations"]; !ok {
		out["recommendations"] = []interface{}{}
	}
	out["available"] = true
	c.JSON(http.StatusOK, out)
}
