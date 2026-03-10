package v1

import (
	"Products_backend/internal/database"
	"Products_backend/internal/handlers"
	"Products_backend/internal/models"
	"Products_backend/utils"
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

// enableCORS включает простую CORS‑политику для разработки.
func enableCORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// authMiddleware — заглушка авторизации: сейчас пропускает всех.
func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Здесь можно добавить проверку JWT/токена.
		c.Next()
	}
}

func login(c *gin.Context) {
	h := &handlers.AuthHandler{}
	h.Login(c)
}

func Apies() {
	router := gin.Default()

	router.Use(enableCORS())

	// 🔓 Публичные маршруты
	router.POST("/login", login)

	routerv1 := router.Group("/v1")
	routerv1.Use(authMiddleware())

	userHandler := handlers.BaseHandler[models.User]{DB: database.DbPostgres}
	productHandler := handlers.BaseHandler[models.Product]{DB: database.DbPostgres}
	categoryHandler := handlers.BaseHandler[models.Category]{DB: database.DbPostgres}
	manufacturerHandler := handlers.BaseHandler[models.Manufacturer]{DB: database.DbPostgres}
	orderHandler := handlers.BaseHandler[models.Order]{DB: database.DbPostgres}
	orderItemHandler := handlers.BaseHandler[models.OrderItem]{DB: database.DbPostgres}
	favouriteHandler := handlers.NewFavouriteHandler(database.DbPostgres)

	// 👤 USERS
	users := routerv1.Group("/users")
	{
		users.GET("", userHandler.GetAll)
		users.GET("/:id", userHandler.GetByID)
		users.POST("", userHandler.Create)
		users.PUT("/:id", userHandler.Update)
		users.DELETE("/:id", userHandler.Delete)
	}

	// 📂 CATEGORIES
	categories := routerv1.Group("/categories")
	{
		categories.GET("", categoryHandler.GetAll)
		categories.GET("/:id", categoryHandler.GetByID)
		categories.POST("", categoryHandler.Create)
		categories.PUT("/:id", categoryHandler.Update)
		categories.DELETE("/:id", categoryHandler.Delete)
	}

	// 🏭 MANUFACTURERS
	manufacturers := routerv1.Group("/manufacturers")
	{
		manufacturers.GET("", manufacturerHandler.GetAll)
		manufacturers.GET("/:id", manufacturerHandler.GetByID)
		manufacturers.POST("", manufacturerHandler.Create)
		manufacturers.PUT("/:id", manufacturerHandler.Update)
		manufacturers.DELETE("/:id", manufacturerHandler.Delete)
	}

	// 🏷 PRODUCTS
	products := routerv1.Group("/products")
	{
		products.GET("", productHandler.GetAll)
		products.GET("/:id", productHandler.GetByID)
		products.POST("", productHandler.Create)
		products.PUT("/:id", productHandler.Update)
		products.DELETE("/:id", productHandler.Delete)
	}

	// 🧾 ORDERS
	orders := routerv1.Group("/orders")
	{
		orders.GET("", orderHandler.GetAll)
		orders.GET("/:id", orderHandler.GetByID)
		orders.POST("", orderHandler.Create)
		orders.PUT("/:id", orderHandler.Update)
		orders.DELETE("/:id", orderHandler.Delete)
	}

	// 🛒 ORDER ITEMS
	orderItems := routerv1.Group("/order-items")
	{
		orderItems.GET("", orderItemHandler.GetAll)
		orderItems.GET("/:id", orderItemHandler.GetByID)
		orderItems.POST("", orderItemHandler.Create)
		orderItems.PUT("/:id", orderItemHandler.Update)
		orderItems.DELETE("/:id", orderItemHandler.Delete)
	}

	// ❤️ FAVOURITES (пока только продукты)
	favourites := routerv1.Group("/favourites")
	{
		favourites.POST("/product", favouriteHandler.AddProduct)
	}

	// --- GRACEFUL SHUTDOWN ---

	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			utils.Logger.Fatalf("ListenAndServe error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	<-quit
	utils.Logger.Warn("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		utils.Logger.Fatalf("Server forced to shutdown: %v", err)
	}

	utils.Logger.Println("Server exiting")
}
