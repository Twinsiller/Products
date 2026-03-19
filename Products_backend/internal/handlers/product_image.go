package handlers

import (
	"io"
	"net/http"
	"strconv"

	"Products_backend/internal/repository"

	"github.com/gin-gonic/gin"
)

// GetProductImage отдаёт изображение товара из MongoDB (GET /v1/products/:id/image).
func GetProductImage(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}
	doc, err := repository.GetProductImage(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if doc == nil {
		c.Status(http.StatusNotFound)
		return
	}
	contentType := doc.ContentType
	if contentType == "" {
		contentType = "image/jpeg"
	}
	c.Data(http.StatusOK, contentType, doc.Data)
}

// UploadProductImage загружает изображение товара в MongoDB (POST /v1/products/:id/image).
func UploadProductImage(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}
	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "image file required (form field 'image')"})
		return
	}
	const maxSize = 5 << 20 // 5 MiB
	if file.Size > maxSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "image too large (max 5 MB)"})
		return
	}
	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
		return
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
		return
	}
	if int64(len(data)) > maxSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "image too large (max 5 MB)"})
		return
	}
	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg"
	}
	if err := repository.SaveProductImage(c.Request.Context(), id, contentType, data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "image uploaded", "product_id": id})
}
