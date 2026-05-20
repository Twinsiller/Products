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
		&models.Dish{},
		&models.DishProduct{},
		&models.DishCategoryRequirement{},
		&models.FavouriteProduct{},
		&models.FavouriteDish{},
	); err != nil {
		utils.Logger.Fatalf("Failed to migrate models: %v", err)
		return err
	}
	utils.Logger.Info("Успешная миграция моделей")

	// Чистим устаревшие столбцы, оставшиеся от прежних версий моделей.
	// GORM AutoMigrate никогда не удаляет колонки сам — приходится вручную.
	dropLegacyColumns()

	// Если каталог пуст — заполняем демо-данными (продукты, блюда, пользователи).
	// При повторных запусках seed автоматически не сработает: проверка по products.
	if err := database.SeedIfEmpty(database.DbPostgres); err != nil {
		utils.Logger.Errorf("Ошибка заполнения демо-данными: %v", err)
	} else {
		var pcount int64
		_ = database.DbPostgres.Model(&models.Product{}).Count(&pcount).Error
		utils.Logger.Infof("Демо-данные готовы: в каталоге %d продуктов", pcount)
	}

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

// dropLegacyColumns удаляет устаревшие столбцы, оставшиеся в таблицах после
// переименования полей в моделях (GORM AutoMigrate не удаляет колонки сам).
func dropLegacyColumns() {
	type legacyCol struct {
		Table  string
		Column string
	}
	legacy := []legacyCol{
		{"orders", "cashier_id"}, // переименовано в user_id
	}
	for _, c := range legacy {
		if database.DbPostgres.Migrator().HasColumn(&models.Order{}, c.Column) ||
			hasRawColumn(c.Table, c.Column) {
			if err := database.DbPostgres.Exec(
				"ALTER TABLE " + c.Table + " DROP COLUMN IF EXISTS " + c.Column,
			).Error; err != nil {
				utils.Logger.Warnf("Не удалось удалить устаревший столбец %s.%s: %v", c.Table, c.Column, err)
			} else {
				utils.Logger.Infof("Удалён устаревший столбец %s.%s", c.Table, c.Column)
			}
		}
	}
}

// hasRawColumn — низкоуровневая проверка наличия колонки в Postgres,
// на случай если Migrator() не видит колонку, отсутствующую в модели.
func hasRawColumn(table, column string) bool {
	var count int64
	err := database.DbPostgres.Raw(
		"SELECT COUNT(*) FROM information_schema.columns WHERE table_name = ? AND column_name = ?",
		table, column,
	).Scan(&count).Error
	if err != nil {
		return false
	}
	return count > 0
}
