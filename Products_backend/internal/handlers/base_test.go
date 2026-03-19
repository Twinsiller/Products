// Тесты универсального BaseHandler[T]: проверяем CRUD через HTTP
// (GetAll, GetByID, Create, Update, Delete). Используем in-memory SQLite,
// чтобы не зависеть от реального PostgreSQL.
package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"Products_backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// testDB создаёт in-memory БД и мигрирует модель Category для тестов.
func testDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.Category{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestBaseHandler_Create_and_GetAll(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testDB(t)
	h := BaseHandler[models.Category]{DB: db}
	r := gin.New()
	r.POST("/categories", h.Create)
	r.GET("/categories", h.GetAll)

	// Создаём категорию
	body := []byte(`{"name":"Овощи"}`)
	req := httptest.NewRequest(http.MethodPost, "/categories", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Create status = %d, want %d", w.Code, http.StatusCreated)
	}
	var created models.Category
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal create response: %v", err)
	}
	if created.Name != "Овощи" {
		t.Errorf("Create name = %q, want Овощи", created.Name)
	}

	// GetAll — должна вернуться одна запись
	req2 := httptest.NewRequest(http.MethodGet, "/categories", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Errorf("GetAll status = %d, want %d", w2.Code, http.StatusOK)
	}
	var list []models.Category
	if err := json.Unmarshal(w2.Body.Bytes(), &list); err != nil {
		t.Fatalf("unmarshal list: %v", err)
	}
	if len(list) != 1 || list[0].Name != "Овощи" {
		t.Errorf("GetAll = %+v, want one category Овощи", list)
	}
}

func TestBaseHandler_GetByID_not_found(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testDB(t)
	h := BaseHandler[models.Category]{DB: db}
	r := gin.New()
	r.GET("/categories/:id", h.GetByID)

	req := httptest.NewRequest(http.MethodGet, "/categories/99999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("GetByID(99999) status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestBaseHandler_Create_invalid_json(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testDB(t)
	h := BaseHandler[models.Category]{DB: db}
	r := gin.New()
	r.POST("/categories", h.Create)

	req := httptest.NewRequest(http.MethodPost, "/categories", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Create invalid JSON status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestBaseHandler_Update_and_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testDB(t)
	// Создаём запись напрямую
	cat := models.Category{Name: "Фрукты"}
	if err := db.Create(&cat).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}
	id := cat.ID
	if id == 0 {
		t.Fatal("category ID not set")
	}
	pathID := strconv.FormatInt(id, 10)
	basePath := "/categories/" + pathID

	h := BaseHandler[models.Category]{DB: db}
	r := gin.New()
	r.PUT("/categories/:id", h.Update)
	r.DELETE("/categories/:id", h.Delete)
	r.GET("/categories/:id", h.GetByID)

	// Update
	body := []byte(`{"name":"Ягоды"}`)
	req := httptest.NewRequest(http.MethodPut, basePath, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Update status = %d, want %d", w.Code, http.StatusOK)
	}

	// Delete
	req2 := httptest.NewRequest(http.MethodDelete, basePath, nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Errorf("Delete status = %d, want %d", w2.Code, http.StatusOK)
	}

	// Проверяем: GetByID после удаления должен вернуть 404
	req3 := httptest.NewRequest(http.MethodGet, basePath, nil)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusNotFound {
		t.Errorf("GetByID after Delete status = %d, want 404. Body: %s", w3.Code, w3.Body.String())
	}
}
