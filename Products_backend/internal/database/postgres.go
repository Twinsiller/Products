package database

import (
	"fmt"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// DbPostgres — глобальное подключение к PostgreSQL
var DbPostgres *gorm.DB

// Config описывает настройки подключения к базе данных.
type Config struct {
	DSN string
}

// LoadConfigFromEnv формирует DSN из переменных окружения.
// Минимально достаточно задать POSTGRES_DSN, либо по-отдельности:
// POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB.
func LoadConfigFromEnv() Config {
	// Если задана готовая строка подключения — используем её.
	if dsn := os.Getenv("POSTGRES_DSN"); dsn != "" {
		return Config{DSN: dsn}
	}

	host := getenvOrDefault("POSTGRES_HOST", "localhost")
	port := getenvOrDefault("POSTGRES_PORT", "5432")
	user := getenvOrDefault("POSTGRES_USER", "postgres")
	password := getenvOrDefault("POSTGRES_PASSWORD", "postgres")
	dbname := getenvOrDefault("POSTGRES_DB", "postgres")

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname,
	)

	return Config{DSN: dsn}
}

func getenvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// Connect инициализирует подключение к PostgreSQL и сохраняет его в DbPostgres.
func Connect(cfg Config) error {
	db, err := gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{})
	if err != nil {
		return err
	}

	DbPostgres = db
	return nil
}

// Close корректно закрывает соединение с базой данных.
func Close() error {
	if DbPostgres == nil {
		return nil
	}

	sqlDB, err := DbPostgres.DB()
	if err != nil {
		return err
	}

	return sqlDB.Close()
}

// CreateObjDB выполняет авто-миграцию переданных моделей.
func CreateObjDB(models ...interface{}) error {
	if DbPostgres == nil {
		return fmt.Errorf("database is not connected")
	}

	return DbPostgres.AutoMigrate(models...)
}

