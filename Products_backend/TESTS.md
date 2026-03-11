# Тесты проекта Products_backend

## Как запустить

```bash
# Все тесты
go test ./...

# С выводом логов
go test -v ./...

# Только один пакет
go test ./internal/handlers/...
go test ./internal/database/...
go test ./internal/services/...
go test ./internal/api/v1/...
```

На Windows с локальным Go:

```powershell
$env:GOTOOLCHAIN="local"
go test ./...
```

---

## Что именно тестируется

### 1. Конфигурация БД (`internal/database/`)

| Файл | Что проверяем |
|------|----------------|
| `postgres_test.go` | **LoadConfigFromEnv**: если задан `POSTGRES_DSN` — он используется как есть; иначе DSN собирается из `POSTGRES_HOST`, `POSTGRES_PORT`, `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`. При пустых переменных подставляются значения по умолчанию (localhost, 5432, postgres, …). |
| `mongo_test.go` | **LoadMongoConfigFromEnv**: при заданном `MONGO_URI` он используется; иначе URI собирается из `MONGO_HOST` и `MONGO_PORT`. `MONGO_DB` задаёт имя БД, по умолчанию `products_mongo`. |

Подключение к реальному PostgreSQL/Mongo в тестах не используется — проверяется только формирование конфига из окружения.

---

### 2. Обработчики HTTP (`internal/handlers/`)

Для тестов handlers используется **in-memory SQLite** (драйвер `github.com/glebarez/sqlite`), чтобы не зависеть от запущенного PostgreSQL.

| Файл | Что проверяем |
|------|----------------|
| `base_test.go` | **BaseHandler[Category]**: создание категории (POST), список (GET), обновление (PUT), удаление (DELETE); ответ 404 для несуществующего id; ответ 400 при невалидном JSON. |
| `auth_test.go` | **AuthHandler.Login**: POST /login возвращает 200 и тело с JSON (в реализации — `{"token":"dummy"}`). |
| `favourite_test.go` | **FavouriteHandler.AddProduct**: добавление избранного товара (user_id, product_id), ответ 201 и сохранение в БД; при невалидном JSON — 400. |

Каждый тест поднимает свой экземпляр БД в памяти и при необходимости делает миграции (AutoMigrate) только для нужных моделей.

---

### 3. Сервис заказов (`internal/services/`)

| Файл | Что проверяем |
|------|----------------|
| `order_service_test.go` | **OrderService.CreateOrder**: создание заказа и нескольких позиций (OrderItem) в одной транзакции; проставление `OrderID` у позиций; расчёт `TotalAmount` (сумма quantity × price_per_unit). Отдельный кейс с пустым списком позиций (TotalAmount = 0). |

БД снова in-memory SQLite, мигрируются только `Order` и `OrderItem`.

---

### 4. API v1 (`internal/api/v1/`)

| Файл | Что проверяем |
|------|----------------|
| `apies_test.go` | Публичный маршрут **POST /login** возвращает 200 и JSON. Middleware **enableCORS** выставляет заголовок `Access-Control-Allow-Origin: *`; на запрос **OPTIONS** возвращается 204. |

Эндпоинты под префиксом `/v1/*` (users, products, categories и т.д.) в этом пакете не вызываются: они завязаны на глобальное подключение `database.DbPostgres`. Их поведение покрыто тестами в `internal/handlers/*_test.go`, где handler вызывается с подставной in-memory БД.

---

## Где что лежит и как устроено

- **Конфиг** — только логика чтения из `os.Getenv`, без реальных подключений.
- **Handlers и Service** — тесты подменяют БД на SQLite в памяти, поэтому не требуют Docker/Postgres.
- **API** — проверяются только публичные маршруты и CORS без поднятия полного сервера и БД.

Чтобы в будущем тестировать полный роутинг `/v1/*` с одной тестовой БД, можно вынести построение роутера в отдельную функцию, принимающую `*gorm.DB`, и в тестах передавать туда in-memory SQLite.
