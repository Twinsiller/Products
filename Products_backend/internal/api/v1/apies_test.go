// Тесты API v1: проверяем публичный маршрут /login и (опционально) CORS.
// Полноценные тесты эндпоинтов /v1/* выполняются в internal/handlers/*_test.go
// с подставной in-memory БД.
package v1

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func Test_login_returns_200_and_json(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(enableCORS())
	r.POST("/login", login)

	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("POST /login status = %d, want %d", w.Code, http.StatusOK)
	}
	if w.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want application/json", w.Header().Get("Content-Type"))
	}
}

func Test_enableCORS_sets_headers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(enableCORS())
	r.GET("/ok", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("CORS Allow-Origin = %q, want *", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func Test_enableCORS_options_returns_204(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(enableCORS())
	r.GET("/ok", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodOptions, "/ok", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("OPTIONS status = %d, want %d", w.Code, http.StatusNoContent)
	}
}
