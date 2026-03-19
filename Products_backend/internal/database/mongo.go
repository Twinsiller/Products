package database

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// DbMongo — глобальный клиент MongoDB.
var DbMongo *mongo.Client

// MongoDatabaseName — имя базы MongoDB (задаётся при подключении).
var MongoDatabaseName string

// MongoConfig описывает настройки подключения к MongoDB.
type MongoConfig struct {
	URI      string
	Database string
}

// LoadMongoConfigFromEnv загружает конфиг MongoDB из переменных окружения.
// По умолчанию использует локальный MongoDB без авторизации.
func LoadMongoConfigFromEnv() MongoConfig {
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		host := getenvOrDefault("MONGO_HOST", "localhost")
		port := getenvOrDefault("MONGO_PORT", "27017")
		uri = fmt.Sprintf("mongodb://%s:%s", host, port)
	}

	dbName := os.Getenv("MONGO_DB")
	if dbName == "" {
		dbName = "products_db"
	}

	return MongoConfig{
		URI:      uri,
		Database: dbName,
	}
}

// ConnectMongoDB устанавливает подключение к MongoDB и сохраняет его в DbMongo.
func ConnectMongoDB(cfg MongoConfig) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.URI))
	if err != nil {
		return err
	}

	// Проверяем подключение.
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return err
	}

	DbMongo = client
	MongoDatabaseName = cfg.Database
	return nil
}

// CloseMongoDB закрывает соединение с MongoDB.
func CloseMongoDB() error {
	if DbMongo == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return DbMongo.Disconnect(ctx)
}

