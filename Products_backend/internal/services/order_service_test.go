// Тесты OrderService: проверяем создание заказа с позициями (CreateOrder)
// в одной транзакции и подсчёт TotalAmount.
package services

import (
	"testing"
	"time"

	"Products_backend/internal/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// orderTestDB создаёт in-memory БД. Таблицы создаём вручную, т.к. в Order
// указано default:now() и timestamptz — SQLite такого не поддерживает.
func orderTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	for _, sql := range []string{
		`CREATE TABLE IF NOT EXISTS orders (id INTEGER PRIMARY KEY AUTOINCREMENT, cashier_id INTEGER NOT NULL, created_at DATETIME, total_amount REAL DEFAULT 0)`,
		`CREATE TABLE IF NOT EXISTS order_items (id INTEGER PRIMARY KEY AUTOINCREMENT, order_id INTEGER NOT NULL, product_id INTEGER NOT NULL, quantity INTEGER NOT NULL, price_per_unit REAL NOT NULL, discount INTEGER DEFAULT 0)`,
	} {
		if err := db.Exec(sql).Error; err != nil {
			t.Fatalf("create table: %v", err)
		}
	}
	return db
}

func TestOrderService_CreateOrder(t *testing.T) {
	db := orderTestDB(t)
	svc := NewOrderService(db)

	order := &models.Order{
		UserID:      1,
		CreatedAt:   time.Now(),
		TotalAmount: 0,
	}
	items := []models.OrderItem{
		{ProductID: 10, Quantity: 2, PricePerUnit: 100.0, Discount: 0},
		{ProductID: 20, Quantity: 1, PricePerUnit: 50.0, Discount: 10},
	}

	err := svc.CreateOrder(order, items)
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}

	// Order должен получить ID после Create
	if order.ID == 0 {
		t.Error("order.ID should be set after CreateOrder")
	}
	// TotalAmount = 2*100 + 1*50 = 250 (discount в модели — число, не вычитается из суммы в коде, проверяем как есть)
	expectedTotal := 2*100.0 + 1*50.0
	if order.TotalAmount != expectedTotal {
		t.Errorf("order.TotalAmount = %v, want %v", order.TotalAmount, expectedTotal)
	}

	// В БД должна быть одна запись Order и две OrderItem
	var orderCount, itemCount int64
	db.Model(&models.Order{}).Count(&orderCount)
	db.Model(&models.OrderItem{}).Count(&itemCount)
	if orderCount != 1 {
		t.Errorf("orders count = %d, want 1", orderCount)
	}
	if itemCount != 2 {
		t.Errorf("order_items count = %d, want 2", itemCount)
	}
}

func TestOrderService_CreateOrder_empty_items(t *testing.T) {
	db := orderTestDB(t)
	svc := NewOrderService(db)

	order := &models.Order{UserID: 1, CreatedAt: time.Now(), TotalAmount: 0}
	items := []models.OrderItem{}

	err := svc.CreateOrder(order, items)
	if err != nil {
		t.Fatalf("CreateOrder empty items: %v", err)
	}
	if order.TotalAmount != 0 {
		t.Errorf("order.TotalAmount = %v, want 0", order.TotalAmount)
	}
}
