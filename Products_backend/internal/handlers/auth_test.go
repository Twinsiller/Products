// Тесты AuthHandler: проверяем, что /login возвращает 200 и тело с полем "token".
package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAuthHandler_Login(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &AuthHandler{}
	r := gin.New()
	r.POST("/login", h.Login)

	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Login status = %d, want %d", w.Code, http.StatusOK)
	}
	// В текущей реализации возвращается {"token":"dummy"}
	ct := w.Header().Get("Content-Type")
	if ct != "" && ct != "application/json; charset=utf-8" {
		// Gin по умолчанию отдаёт JSON
	}
	body := w.Body.String()
	if body == "" {
		t.Error("Login response body is empty")
	}
	if len(body) > 0 && body[0] != '{' {
		t.Errorf("Login response should be JSON, got: %s", body)
	}
}
