package cmd

import (
	v1 "Products_backend/internal/api/v1"
	"Products_backend/internal/database"
	"Products_backend/internal/models"
	"Products_backend/utils"

	"github.com/joho/godotenv"
)

func Run() error {
	// Запуск логгера
	utils.InitLogger("utils/app.log")

	// Загружаем переменные окружения из файла .env (необязательно, но удобно в dev)
	if err := godotenv.Load(); err != nil {
		utils.Logger.Warnf("Error loading .env file: %v", err)
	}

	// Подключение к PostgreSQL
	if err := database.Connect(database.LoadConfigFromEnv()); err != nil {
		utils.Logger.Fatalf("Failed to connect to Postgres: %v", err)
		return err
	}
	defer func() {
		if err := database.Close(); err != nil {
			utils.Logger.Errorf("Error closing database: %v", err)
		}
	}()

	// Подключение к MongoDB (опционально, если нужно)
	mongoCfg := database.LoadMongoConfigFromEnv()
	if err := database.ConnectMongoDB(mongoCfg); err != nil {
		utils.Logger.Fatalf("Failed to connect to MongoDB: %v", err)
		return err
	}
	defer func() {
		if err := database.CloseMongoDB(); err != nil {
			utils.Logger.Errorf("Error closing MongoDB: %v", err)
		}
	}()

	utils.Logger.Info("Миграция моделей в базу данных")
	if err := database.CreateObjDB(
		&models.User{},
		&models.Product{},
		&models.Order{},
		&models.OrderItem{},
		&models.Manufacturer{},
		&models.Category{},
	); err != nil {
		utils.Logger.Fatalf("Failed to migrate models: %v", err)
		return err
	}
	utils.Logger.Info("Успешная миграция моделей")

	// Запуск HTTP API
	v1.Apies()

	return nil
}
