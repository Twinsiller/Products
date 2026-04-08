package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"Products_backend/internal/database"
	"Products_backend/internal/models"

	"github.com/gin-gonic/gin"
)

// ListDishes возвращает все блюда с составом (товары).
func ListDishes(c *gin.Context) {
	var dishes []models.Dish
	if err := database.DbPostgres.
		Preload("Products").
		Preload("Products.Product").
		Preload("CategoryRequirements").
		Preload("CategoryRequirements.Category").
		Find(&dishes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, dishes)
}

// GetDishByID возвращает блюдо с составом.
func GetDishByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var d models.Dish
	if err := database.DbPostgres.
		Preload("Products").
		Preload("Products.Product").
		Preload("CategoryRequirements").
		Preload("CategoryRequirements.Category").
		First(&d, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, d)
}

func CreateDish(c *gin.Context) {
	var req models.CreateDish
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	d := models.Dish{Name: name}
	if err := database.DbPostgres.Create(&d).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, d)
}

func UpdateDish(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var d models.Dish
	if err := database.DbPostgres.First(&d, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	var req models.UpdateDish
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name cannot be empty"})
			return
		}
		d.Name = name
	}
	if err := database.DbPostgres.Save(&d).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, d)
}

func DeleteDish(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := database.DbPostgres.Delete(&models.Dish{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

type addDishProductRequest struct {
	ProductID int64 `json:"product_id" binding:"required"`
	Quantity  int   `json:"quantity" binding:"required,min=1"`
}

func AddDishProduct(c *gin.Context) {
	dishID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid dish id"})
		return
	}
	var dish models.Dish
	if err := database.DbPostgres.First(&dish, dishID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "dish not found"})
		return
	}

	var req addDishProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	var product models.Product
	if err := database.DbPostgres.First(&product, req.ProductID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product_id"})
		return
	}

	var count int64
	if err := database.DbPostgres.Model(&models.DishProduct{}).
		Where("dish_id = ? AND product_id = ?", dishID, req.ProductID).
		Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "product already added to this dish"})
		return
	}

	row := models.DishProduct{
		DishID:       dishID,
		ProductID:    req.ProductID,
		Quantity:     req.Quantity,
		PricePerUnit: product.DefaultPrice,
		Discount:     0,
	}
	if err := database.DbPostgres.Create(&row).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = database.DbPostgres.Preload("Product").First(&row, row.ID).Error
	c.JSON(http.StatusCreated, row)
}

func DeleteDishProduct(c *gin.Context) {
	dishID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid dish id"})
		return
	}
	dishProductID, err := strconv.ParseInt(c.Param("dishProductId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid dish product id"})
		return
	}
	if err := database.DbPostgres.
		Where("dish_id = ?", dishID).
		Delete(&models.DishProduct{}, dishProductID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func AddDishCategoryRequirement(c *gin.Context) {
	dishID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid dish id"})
		return
	}
	var dish models.Dish
	if err := database.DbPostgres.First(&dish, dishID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "dish not found"})
		return
	}
	var req models.CreateDishCategoryRequirement
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	ingredientName := strings.TrimSpace(req.IngredientName)
	if ingredientName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ingredient_name is required"})
		return
	}
	if req.CategoryID != nil {
		var category models.Category
		if err := database.DbPostgres.First(&category, *req.CategoryID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid category_id"})
			return
		}
	}
	var count int64
	if err := database.DbPostgres.Model(&models.DishCategoryRequirement{}).
		Where("dish_id = ? AND lower(ingredient_name) = lower(?)", dishID, ingredientName).
		Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "ingredient already required for this dish"})
		return
	}
	row := models.DishCategoryRequirement{
		DishID:         dishID,
		CategoryID:     req.CategoryID,
		IngredientName: ingredientName,
		Quantity:       req.Quantity,
		Note:           strings.TrimSpace(req.Note),
	}
	if err := database.DbPostgres.Create(&row).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := database.DbPostgres.Preload("Category").First(&row, row.ID).Error; err != nil {
		c.JSON(http.StatusCreated, row)
		return
	}
	c.JSON(http.StatusCreated, row)
}

func UpdateDishCategoryRequirement(c *gin.Context) {
	dishID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid dish id"})
		return
	}
	reqID, err := strconv.ParseInt(c.Param("reqId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid requirement id"})
		return
	}
	var row models.DishCategoryRequirement
	if err := database.DbPostgres.Where("dish_id = ?", dishID).First(&row, reqID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "requirement not found"})
		return
	}
	var req models.UpdateDishCategoryRequirement
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	if req.CategoryID != nil {
		var category models.Category
		if err := database.DbPostgres.First(&category, *req.CategoryID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid category_id"})
			return
		}
		catID := *req.CategoryID
		row.CategoryID = &catID
	}
	if req.IngredientName != nil {
		ingredientName := strings.TrimSpace(*req.IngredientName)
		if ingredientName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ingredient_name cannot be empty"})
			return
		}
		var count int64
		if err := database.DbPostgres.Model(&models.DishCategoryRequirement{}).
			Where("dish_id = ? AND id <> ? AND lower(ingredient_name) = lower(?)", dishID, row.ID, ingredientName).
			Count(&count).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if count > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "ingredient already required for this dish"})
			return
		}
		row.IngredientName = ingredientName
	}
	if req.Quantity != nil {
		if *req.Quantity <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "quantity must be > 0"})
			return
		}
		row.Quantity = *req.Quantity
	}
	if req.Note != nil {
		row.Note = strings.TrimSpace(*req.Note)
	}
	if err := database.DbPostgres.Save(&row).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = database.DbPostgres.Preload("Category").First(&row, row.ID).Error
	c.JSON(http.StatusOK, row)
}

func DeleteDishCategoryRequirement(c *gin.Context) {
	dishID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid dish id"})
		return
	}
	reqID, err := strconv.ParseInt(c.Param("reqId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid requirement id"})
		return
	}
	if err := database.DbPostgres.
		Where("dish_id = ?", dishID).
		Delete(&models.DishCategoryRequirement{}, reqID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
