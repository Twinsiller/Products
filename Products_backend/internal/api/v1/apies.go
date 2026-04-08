package v1

import (
	"Products_backend/internal/database"
	"Products_backend/internal/handlers"
	"Products_backend/internal/models"
	"Products_backend/internal/repository"
	"Products_backend/internal/services"
	"Products_backend/utils"
	"context"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// enableCORS включает простую CORS‑политику для разработки.
func enableCORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, X-User-Id")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid token"})
			return
		}

		tokenStr := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer"))
		token, err := jwt.ParseWithClaims(tokenStr, &handlers.Claims{}, func(token *jwt.Token) (interface{}, error) {
			return handlers.GetJWTSecret(), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		if claims, ok := token.Claims.(*handlers.Claims); ok {
			c.Set("user_id", claims.UserID)
			c.Set("user_name", claims.Name)
			c.Set("user_role", claims.Role)
		}

		c.Next()
	}
}

// deleteProductAndImage удаляет изображение товара из MongoDB, затем вызывает удаление товара в Postgres.
func deleteProductAndImage(h *handlers.BaseHandler[models.Product]) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		if id, err := strconv.ParseInt(idStr, 10, 64); err == nil {
			_ = repository.DeleteProductImage(c.Request.Context(), id)
		}
		h.Delete(c)
	}
}

// adminOnly — middleware, разрешающий доступ только пользователю с ролью admin.
func adminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get("user_role")
		role, ok := roleVal.(string)
		if !exists || !ok || role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			return
		}
		c.Next()
	}
}

func login(c *gin.Context) {
	h := &handlers.AuthHandler{}
	h.Login(c)
}

func register(c *gin.Context) {
	h := &handlers.AuthHandler{}
	h.Register(c)
}

func Apies() {
	router := gin.Default()

	router.Use(enableCORS())

	// 🔓 Публичные маршруты
	router.POST("/login", login)
	router.POST("/register", register)

	routerv1Public := router.Group("/v1")
	routerv1Protected := router.Group("/v1")
	routerv1Protected.Use(authMiddleware())

	userHandler := handlers.BaseHandler[models.User]{DB: database.DbPostgres}
	productHandler := handlers.BaseHandler[models.Product]{DB: database.DbPostgres}
	categoryHandler := handlers.BaseHandler[models.Category]{DB: database.DbPostgres}
	manufacturerHandler := handlers.BaseHandler[models.Manufacturer]{DB: database.DbPostgres}
	orderItemHandler := handlers.BaseHandler[models.OrderItem]{DB: database.DbPostgres}
	favouriteHandler := handlers.NewFavouriteHandler(database.DbPostgres)
	recommendationHandler := &handlers.RecommendationHandler{
		Service: &services.RecommendationService{DB: database.DbPostgres},
	}

	// 👤 USERS
	// Просмотр и рекомендации — для любых авторизованных.
	users := routerv1Protected.Group("/users")
	{
		users.GET("", userHandler.GetAll)
		// Блюда по заказу (КБЖУ) — до /:id, чтобы не пересекаться с более короткими шаблонами при необходимости
		users.GET("/:id/meals/from-order/:orderId", recommendationHandler.GetMealsFromOrder)
		users.POST("/:id/meals/from-cart", recommendationHandler.GetMealsFromCart)
		users.GET("/:id", userHandler.GetByID)

		// Рекомендации для пользователя
		users.GET("/:id/recommendations/products", recommendationHandler.GetProductRecommendations)
		users.GET("/:id/recommendations/dishes", recommendationHandler.GetDishRecommendations)
		users.GET("/:id/recommendations/final", recommendationHandler.GetFinalRecommendations)
	}
	// Изменение и удаление пользователей — только админ.
	usersAdmin := routerv1Protected.Group("/users")
	usersAdmin.Use(adminOnly())
	{
		usersAdmin.GET("/full", handlers.ListUsersDetailed)
		usersAdmin.GET("/:id/full", handlers.GetUserDetailed)
		usersAdmin.POST("", userHandler.Create)
		usersAdmin.PUT("/:id", userHandler.Update)
		usersAdmin.DELETE("/:id", userHandler.Delete)
	}

	// 📂 CATEGORIES (чтение — гостям, изменения — только админ)
	categories := routerv1Public.Group("/categories")
	{
		categories.GET("", categoryHandler.GetAll)
		categories.GET("/:id", categoryHandler.GetByID)
	}
	categoriesAdmin := routerv1Protected.Group("/categories")
	categoriesAdmin.Use(adminOnly())
	{
		categoriesAdmin.POST("", handlers.CreateCategory)
		categoriesAdmin.PUT("/:id", categoryHandler.Update)
		categoriesAdmin.DELETE("/:id", categoryHandler.Delete)
	}

	// 🏭 MANUFACTURERS (чтение — гостям, изменения — только админ)
	manufacturers := routerv1Public.Group("/manufacturers")
	{
		manufacturers.GET("", manufacturerHandler.GetAll)
		manufacturers.GET("/:id", manufacturerHandler.GetByID)
	}
	manufacturersAdmin := routerv1Protected.Group("/manufacturers")
	manufacturersAdmin.Use(adminOnly())
	{
		manufacturersAdmin.POST("", handlers.CreateManufacturer)
		manufacturersAdmin.PUT("/:id", manufacturerHandler.Update)
		manufacturersAdmin.DELETE("/:id", manufacturerHandler.Delete)
	}

	// 🍽 БЛЮДА (справочник с составом — для подбора по заказу)
	dishes := routerv1Public.Group("/dishes")
	{
		dishes.GET("", handlers.ListDishes)
		dishes.GET("/:id", handlers.GetDishByID)
	}
	dishesAdmin := routerv1Protected.Group("/dishes")
	dishesAdmin.Use(adminOnly())
	{
		dishesAdmin.POST("", handlers.CreateDish)
		dishesAdmin.PUT("/:id", handlers.UpdateDish)
		dishesAdmin.DELETE("/:id", handlers.DeleteDish)
		dishesAdmin.POST("/:id/products", handlers.AddDishProduct)
		dishesAdmin.DELETE("/:id/products/:dishProductId", handlers.DeleteDishProduct)
		dishesAdmin.POST("/:id/category-requirements", handlers.AddDishCategoryRequirement)
		dishesAdmin.PUT("/:id/category-requirements/:reqId", handlers.UpdateDishCategoryRequirement)
		dishesAdmin.DELETE("/:id/category-requirements/:reqId", handlers.DeleteDishCategoryRequirement)
	}

	// 🏷 PRODUCTS (чтение — гостям, изменения — только админ)
	products := routerv1Public.Group("/products")
	{
		products.GET("", productHandler.GetAll)
		products.GET("/:id/image", handlers.GetProductImage)
		products.GET("/:id", productHandler.GetByID)
	}
	productsAdmin := routerv1Protected.Group("/products")
	productsAdmin.Use(adminOnly())
	{
		productsAdmin.POST("", handlers.CreateProduct)
		productsAdmin.POST("/:id/image", handlers.UploadProductImage)
		productsAdmin.PUT("/:id", handlers.UpdateProduct)
		productsAdmin.DELETE("/:id", deleteProductAndImage(&productHandler))
	}

	// 🧾 ORDERS (требует авторизации)
	orders := routerv1Protected.Group("/orders")
	{
		orders.GET("", handlers.ListOrders)
		orders.GET("/:id/items", handlers.ListOrderItemsByOrder)
		orders.GET("/:id", handlers.GetOrderByID)
		orders.POST("", handlers.CreateOrder)
		orders.PUT("/:id", handlers.UpdateOrder)
		orders.DELETE("/:id", handlers.DeleteOrder)
	}

	// 🛒 ORDER ITEMS (требует авторизации)
	orderItems := routerv1Protected.Group("/order-items")
	{
		orderItems.GET("", orderItemHandler.GetAll)
		orderItems.GET("/:id", orderItemHandler.GetByID)
		orderItems.POST("", orderItemHandler.Create)
		orderItems.PUT("/:id", orderItemHandler.Update)
		orderItems.DELETE("/:id", orderItemHandler.Delete)
	}

	// ❤️ FAVOURITES (только авторизованные пользователи)
	favourites := routerv1Protected.Group("/favourites")
	{
		favourites.GET("/products", favouriteHandler.ListMyProducts)
		favourites.POST("/product", favouriteHandler.AddProduct)
		favourites.DELETE("/product/:productId", favouriteHandler.RemoveProduct)
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
