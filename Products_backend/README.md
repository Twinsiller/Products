# Magazine_backend


## База данных
### Удаление данных из таблицы и обнуление значение serial:
```TRUNCATE TABLE db_name RESTART IDENTITY;```

### Библиотеки:
PostgreSQL:
    ```_ "github.com/lib/pq"```

Hash-Password:
    ```password "github.com/vzglad-smerti/password_hash"```

## Docker:
....
    build: "docker build -t myapp:latest ."
    For launch: "docker run -e POSTGRES_PASSWORD=qwerty -e POSTGRES_DB=ProjectONE -p 8080:8080 -p 5432:5432 myapp:latest"
.......


## Для запуска:
    1. go mod init ProjectONE **Модули**
        1.1 
    2. go mod tidy **Обновление модулей**
    3. go get -u 'Библиотека'
        3.0 go get -u ./... 'Обновить все зависимости до последних совместимых'
        3.1 go get -u github.com/gin-gonic/gin 'Сервер'
        3.2 go get -u github.com/lib/pq 'БД PostgreSQL'
        3.3 go get -u github.com/vzglad-smerti/password_hash 'Хэширование паролей'
        3.4 go get -u github.com/dgrijalva/jwt-go 'Технология JWT'
        3.5 go get -u 
        3.6 go get -u 
        3.7 go get -u github.com/swaggo/files 'Документация Swagger JSON files'
        3.8 go get -u github.com/swaggo/gin-swagger 'Документация Swagger UI'