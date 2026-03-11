// Тесты конфигурации PostgreSQL: проверяем, что DSN собирается
// из переменных окружения (POSTGRES_DSN или POSTGRES_HOST/PORT/USER/PASSWORD/DB).
package database

import (
	"os"
	"strings"
	"testing"
)

func TestLoadConfigFromEnv_usesPOSTGRES_DSN_when_set(t *testing.T) {
	// Если задана готовая строка POSTGRES_DSN — она должна использоваться целиком.
	const want = "host=db.example.com port=5433 user=u password=p dbname=mydb sslmode=require"
	os.Setenv("POSTGRES_DSN", want)
	defer func() {
		os.Unsetenv("POSTGRES_DSN")
	}()

	cfg := LoadConfigFromEnv()
	if cfg.DSN != want {
		t.Errorf("LoadConfigFromEnv() DSN = %q, want %q", cfg.DSN, want)
	}
}

func TestLoadConfigFromEnv_builds_DSN_from_components_when_DSN_empty(t *testing.T) {
	// Если POSTGRES_DSN не задан, DSN собирается из отдельных переменных.
	os.Unsetenv("POSTGRES_DSN")
	os.Setenv("POSTGRES_HOST", "myhost")
	os.Setenv("POSTGRES_PORT", "5433")
	os.Setenv("POSTGRES_USER", "myuser")
	os.Setenv("POSTGRES_PASSWORD", "mypass")
	os.Setenv("POSTGRES_DB", "mydb")
	defer func() {
		os.Unsetenv("POSTGRES_HOST")
		os.Unsetenv("POSTGRES_PORT")
		os.Unsetenv("POSTGRES_USER")
		os.Unsetenv("POSTGRES_PASSWORD")
		os.Unsetenv("POSTGRES_DB")
	}()

	cfg := LoadConfigFromEnv()
	if !strings.Contains(cfg.DSN, "host=myhost") || !strings.Contains(cfg.DSN, "port=5433") ||
		!strings.Contains(cfg.DSN, "user=myuser") || !strings.Contains(cfg.DSN, "password=mypass") ||
		!strings.Contains(cfg.DSN, "dbname=mydb") {
		t.Errorf("LoadConfigFromEnv() DSN = %q, expected host=myhost, port=5433, user=myuser, password=mypass, dbname=mydb", cfg.DSN)
	}
}

func TestLoadConfigFromEnv_uses_defaults_when_env_empty(t *testing.T) {
	// При пустых переменных должны подставляться значения по умолчанию.
	os.Unsetenv("POSTGRES_DSN")
	os.Unsetenv("POSTGRES_HOST")
	os.Unsetenv("POSTGRES_PORT")
	os.Unsetenv("POSTGRES_USER")
	os.Unsetenv("POSTGRES_PASSWORD")
	os.Unsetenv("POSTGRES_DB")

	cfg := LoadConfigFromEnv()
	// Должны быть дефолты: localhost, 5432, postgres, postgres, postgres
	if !strings.Contains(cfg.DSN, "host=localhost") || !strings.Contains(cfg.DSN, "port=5432") ||
		!strings.Contains(cfg.DSN, "user=postgres") || !strings.Contains(cfg.DSN, "dbname=postgres") {
		t.Errorf("LoadConfigFromEnv() with empty env DSN = %q, expected defaults (localhost, 5432, postgres)", cfg.DSN)
	}
}
