package cmd

import (
	v1 "Products_backend/internal/api/v1"
	"Products_backend/internal/database"
	"Products_backend/internal/models"
	"Products_backend/utils"
	"os"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
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
		&models.Country{},
		&models.Dish{},
		&models.DishProduct{},
		&models.FavouriteProduct{},
		&models.FavouriteDish{},
	); err != nil {
		utils.Logger.Fatalf("Failed to migrate models: %v", err)
		return err
	}
	utils.Logger.Info("Успешная миграция моделей")

	// Создание встроенного администратора, если заданы переменные окружения
	adminName := os.Getenv("ADMIN_NAME")
	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminName != "" && adminPassword != "" {
		var count int64
		if err := database.DbPostgres.Model(&models.User{}).Where("name = ? AND role = ?", adminName, "admin").Count(&count).Error; err != nil {
			utils.Logger.Errorf("Failed to check admin user: %v", err)
		} else if count == 0 {
			hash, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
			if err != nil {
				utils.Logger.Errorf("Failed to hash admin password: %v", err)
			} else {
				admin := models.User{
					Name:         adminName,
					Role:         "admin",
					PasswordHash: string(hash),
				}
				if err := database.DbPostgres.Create(&admin).Error; err != nil {
					utils.Logger.Errorf("Failed to create admin user: %v", err)
				} else {
					utils.Logger.Infof("Admin user %q created", adminName)
				}
			}
		}
	}

	// Запуск HTTP API
	v1.Apies()

	return nil
}
