package handlers

import (
	"net/http"
	"strconv"

	"Products_backend/internal/database"
	"Products_backend/internal/models"
	"Products_backend/utils"

	"github.com/gin-gonic/gin"
)

type orderItemView struct {
	ID           int64           `json:"id"`
	OrderID      int64           `json:"order_id"`
	ProductID    int64           `json:"product_id"`
	Quantity     int             `json:"quantity"`
	PricePerUnit float64         `json:"price_per_unit"`
	Discount     int             `json:"discount"`
	Product      *models.Product `json:"product,omitempty"`
}

func isAdminRequest(c *gin.Context) bool {
	roleVal, ok := c.Get("user_role")
	if !ok {
		return false
	}
	role, ok := roleVal.(string)
	return ok && role == "admin"
}

func currentUserID(c *gin.Context) (int64, bool) {
	v, ok := c.Get("user_id")
	if !ok {
		return 0, false
	}
	id, ok := v.(int64)
	return id, ok
}

// ListOrders: admin sees all, user sees only own orders.
func ListOrders(c *gin.Context) {
	var orders []models.Order
	q := database.DbPostgres.Model(&models.Order{})

	if isAdminRequest(c) {
		if userIDStr := c.Query("user_id"); userIDStr != "" {
			userID, err := strconv.ParseInt(userIDStr, 10, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
				return
			}
			q = q.Where("user_id = ?", userID)
		}
	} else {
		uid, ok := currentUserID(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		q = q.Where("user_id = ?", uid)
	}

	if err := q.Order("created_at DESC").Find(&orders).Error; err != nil {
		utils.Logger.Errorf("ListOrders failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось загрузить заказы"})
		return
	}
	c.JSON(http.StatusOK, orders)
}

// GetOrderByID: admin can read any, user can read only own order.
func GetOrderByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var order models.Order
	if err := database.DbPostgres.First(&order, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	if !isAdminRequest(c) {
		uid, ok := currentUserID(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		if order.UserID != uid {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}
	}

	c.JSON(http.StatusOK, order)
}

// ListOrderItemsByOrder returns order items for one order with access control.
func ListOrderItemsByOrder(c *gin.Context) {
	orderID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}

	var order models.Order
	if err := database.DbPostgres.First(&order, orderID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}

	if !isAdminRequest(c) {
		uid, ok := currentUserID(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		if order.UserID != uid {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}
	}

	var items []models.OrderItem
	if err := database.DbPostgres.
		Where("order_id = ?", orderID).
		Preload("Product").
		Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := make([]orderItemView, 0, len(items))
	for _, it := range items {
		resp = append(resp, orderItemView{
			ID:           it.ID,
			OrderID:      it.OrderID,
			ProductID:    it.ProductID,
			Quantity:     it.Quantity,
			PricePerUnit: it.PricePerUnit,
			Discount:     it.Discount,
			Product:      it.Product,
		})
	}
	c.JSON(http.StatusOK, resp)
}

// CreateOrder: user can create only for self, admin can specify user_id.
func CreateOrder(c *gin.Context) {
	var req models.Order
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Logger.Warnf("CreateOrder: invalid JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Не удалось разобрать данные заказа"})
		return
	}

	uid, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Сессия истекла, войдите заново"})
		return
	}

	if isAdminRequest(c) {
		if req.UserID == 0 {
			req.UserID = uid
		}
	} else {
		req.UserID = uid
	}

	// Гарантируем, что Items не утекут в Create через GORM auto-create.
	req.Items = nil

	if err := database.DbPostgres.Create(&req).Error; err != nil {
		utils.Logger.Errorf("CreateOrder failed (user_id=%d, total=%.2f): %v", req.UserID, req.TotalAmount, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось сохранить заказ"})
		return
	}
	c.JSON(http.StatusCreated, req)
}

// UpdateOrder: admin can update any, user can update only own order amount.
func UpdateOrder(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var order models.Order
	if err := database.DbPostgres.First(&order, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	if !isAdminRequest(c) {
		uid, ok := currentUserID(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		if order.UserID != uid {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}
	}

	var req models.UpdateOrder
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if req.TotalAmount != nil {
		order.TotalAmount = *req.TotalAmount
	}

	if err := database.DbPostgres.Save(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, order)
}

// DeleteOrder: admin can delete any, user can delete only own.
func DeleteOrder(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var order models.Order
	if err := database.DbPostgres.First(&order, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	if !isAdminRequest(c) {
		uid, ok := currentUserID(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		if order.UserID != uid {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}
	}

	if err := database.DbPostgres.Delete(&models.Order{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
