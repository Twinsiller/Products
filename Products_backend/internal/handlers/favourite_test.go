// Тесты FavouriteHandler: добавление избранного товара (AddProduct).
// Используем in-memory SQLite и модель FavouriteProduct.
package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"Products_backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func favouriteTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.FavouriteProduct{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestFavouriteHandler_AddProduct(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := favouriteTestDB(t)
	h := NewFavouriteHandler(db)
	r := gin.New()
	r.POST("/favourites/product", h.AddProduct)

	body := []byte(`{"user_id":1,"product_id":5}`)
	req := httptest.NewRequest(http.MethodPost, "/favourites/product", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("AddProduct status = %d, want %d", w.Code, http.StatusCreated)
	}
	var fav models.FavouriteProduct
	if err := json.Unmarshal(w.Body.Bytes(), &fav); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if fav.UserID != 1 || fav.ProductID != 5 {
		t.Errorf("AddProduct response = %+v, want UserID=1 ProductID=5", fav)
	}
}

func TestFavouriteHandler_AddProduct_invalid_json(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := favouriteTestDB(t)
	h := NewFavouriteHandler(db)
	r := gin.New()
	r.POST("/favourites/product", h.AddProduct)

	req := httptest.NewRequest(http.MethodPost, "/favourites/product", bytes.NewReader([]byte("{")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("AddProduct invalid JSON status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
