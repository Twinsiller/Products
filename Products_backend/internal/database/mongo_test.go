// Тесты конфигурации MongoDB: проверяем формирование URI и имени БД
// из переменных окружения (MONGO_URI или MONGO_HOST/PORT, MONGO_DB).
package database

import (
	"os"
	"strings"
	"testing"
)

func TestLoadMongoConfigFromEnv_usesMONGO_URI_when_set(t *testing.T) {
	wantURI := "mongodb://user:pass@host:27017/?authSource=admin"
	os.Setenv("MONGO_URI", wantURI)
	os.Setenv("MONGO_DB", "custom_db")
	defer func() {
		os.Unsetenv("MONGO_URI")
		os.Unsetenv("MONGO_DB")
	}()

	cfg := LoadMongoConfigFromEnv()
	if cfg.URI != wantURI {
		t.Errorf("LoadMongoConfigFromEnv() URI = %q, want %q", cfg.URI, wantURI)
	}
	if cfg.Database != "custom_db" {
		t.Errorf("LoadMongoConfigFromEnv() Database = %q, want custom_db", cfg.Database)
	}
}

func TestLoadMongoConfigFromEnv_builds_URI_from_host_port_when_URI_empty(t *testing.T) {
	os.Unsetenv("MONGO_URI")
	os.Setenv("MONGO_HOST", "mongo.example.com")
	os.Setenv("MONGO_PORT", "27018")
	os.Unsetenv("MONGO_DB")
	defer func() {
		os.Unsetenv("MONGO_HOST")
		os.Unsetenv("MONGO_PORT")
	}()

	cfg := LoadMongoConfigFromEnv()
	wantURI := "mongodb://mongo.example.com:27018"
	if cfg.URI != wantURI {
		t.Errorf("LoadMongoConfigFromEnv() URI = %q, want %q", cfg.URI, wantURI)
	}
	// Имя БД по умолчанию
	if cfg.Database != "products_mongo" {
		t.Errorf("LoadMongoConfigFromEnv() Database = %q, want products_mongo", cfg.Database)
	}
}

func TestLoadMongoConfigFromEnv_defaults(t *testing.T) {
	os.Unsetenv("MONGO_URI")
	os.Unsetenv("MONGO_HOST")
	os.Unsetenv("MONGO_PORT")
	os.Unsetenv("MONGO_DB")

	cfg := LoadMongoConfigFromEnv()
	if !strings.HasPrefix(cfg.URI, "mongodb://") {
		t.Errorf("LoadMongoConfigFromEnv() URI = %q, expected mongodb://...", cfg.URI)
	}
	if !strings.Contains(cfg.URI, "localhost") || !strings.Contains(cfg.URI, "27017") {
		t.Errorf("LoadMongoConfigFromEnv() URI = %q, expected localhost:27017", cfg.URI)
	}
	if cfg.Database != "products_mongo" {
		t.Errorf("LoadMongoConfigFromEnv() Database = %q, want products_mongo", cfg.Database)
	}
}
