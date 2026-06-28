# Social Media API

Бэкенд социальной сети на Go для курса Highload Architect в OTUS. REST API на базе OpenAPI-спецификации с JWT-аутентификацией и PostgreSQL.

## Технологии

- **Go 1.26**
- **PostgreSQL 18**
- **chi** — HTTP-роутер
- **oapi-codegen** — кодогенерация из OpenAPI-спецификации
- **goose** — миграции БД
- **zerolog** — структурированные логи
- **golang-jwt** — JWT-токены

## Переменные окружения

Конфиденциальные значения передаются только через переменные окружения. Скопируйте `.env.example` в `.env` и заполните:

```env
DB_USER=postgres
DB_PASSWORD=secret
DB_NAME=social_media_db
JWT_SECRET=your-secret-key-min-32-chars
```

Остальные параметры настраиваются в `app-config.yaml`:

| Параметр             | YAML-ключ                  | Переменная окружения | Значение по умолчанию |
|----------------------|----------------------------|----------------------|-----------------------|
| Окружение            | `env`                      | `ENV`                | —                     |
| Уровень логов        | `log_level`                | `LOG_LEVEL`          | `debug`               |
| Хост сервера         | `app_server.host`          | `APP_HOST`           | `0.0.0.0`             |
| Порт сервера         | `app_server.port`          | `APP_PORT`           | `8080`                |
| Хост БД              | `app_db.host`              | `DB_HOST`            | —                     |
| Порт БД              | `app_db.port`              | `DB_PORT`            | —                     |
| SSL-режим БД         | `app_db.ssl_mode`          | `DB_SSL_MODE`        | —                     |
| Макс открытых        |                            |                      |                       |
| соединений           | `app_db.max_open_conns`    | -                    | `25`                  |
| Макс idle            |                            |                      |                       |
| соединений           | `app_db.max_idle_conns`    | -                    | `25`                  |
| Время жизни соедин-я | `app_db.max_conn_lifetime` | -                    | `5m`                  |

Значение `ENV=local` включает цветной вывод логов в консоль. Любое другое значение — JSON-формат.

## Запуск

### Локально

Требуется запущенный PostgreSQL и заполненный `.env`.

```bash
cp .env.example .env
# заполните .env

CONFIG_PATH=configs/app-config.yaml go run ./cmd/server
```

### Docker Compose

Поднимает приложение вместе с PostgreSQL одной командой:

```bash
cp .env.example .env
# заполните .env

docker compose --project-directory . -f ./deployments/docker-compose.yaml up --build
```

Сервер будет доступен на `http://localhost:8282`.

## Миграции

Миграции применяются автоматически при старте приложения. Файлы находятся в `internal/db/migrations/`.

## API

Спецификация OpenAPI — `docs/openapi.json`.

Основные маршруты:

| Метод | Путь                           | Описание                 | Доступ    |
|-------|--------------------------------|--------------------------|-----------|
| POST  | `/api/v1/user/register`        | Регистрация пользователя | Публичный |
| POST  | `/api/v1/login`                | Вход, получение токена   | Публичный |
| GET   | `/api/v1/user/get/{id}`        | Получение профиля        | Публичный |
| GET   | `/api/v1/user/search`          | Поиск пользователей      | Публичный |
| PUT   | `/api/v1/friend/set/{user_id}` | Добавить друга           | Токен     |
| POST  | `/api/v1/post/create`          | Создать пост             | Токен     |
| GET   | `/api/v1/post/feed`            | Лента постов             | Токен     |

Защищённые маршруты требуют заголовок:

```http
Authorization: Bearer <token>
```

## Разработка

```bash
# запустить тесты
go test ./...

# линтер
golangci-lint run

# сборка бинаря
go build -o server ./cmd/server
```

### Сгенерировать миграцию

```bash
# замени <NEW_MIGRATION_NAME> на название новой миграции
go tool goose -dir internal/db/migrations create <NEW_MIGRATION_NAME> sql
```

## Тестирование

### Нагрузочное тестирование

```bash
# скрипт для тестирования ручки поиска (требует установленного cli инструмента hey)
hey -c 10 -z 20s http://localhost:8282/api/v1/user/search?first_name=Александр&last_name=Абрамов
```