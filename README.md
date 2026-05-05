## Shortener

HTTP-сервис для создания коротких ссылок.

### Запуск всего стека через Docker Compose

```bash
docker compose up --build
```

Compose поднимает API, PostgreSQL и Redis. API будет доступен на `http://localhost:8080`.
Если локальные порты заняты, их можно переопределить: `API_PORT=18080 REDIS_PORT=6380 docker compose up --build`.

### Локальный запуск API с инфраструктурой из Compose

```bash
docker compose up -d postgres redis
export DATABASE_URL='postgres://shortener:shortener@localhost:5432/shortener?sslmode=disable'
export REDIS_ADDR='localhost:6379'
go run ./cmd/api
```

По умолчанию приложение слушает `:8080`. Адрес можно изменить через `SHORTENER_ADDR`.
Если `REDIS_ADDR` не задан, кэш редиректов отключается.

### Переменные окружения

| Переменная | Назначение | По умолчанию |
| --- | --- | --- |
| `SHORTENER_ADDR` | Адрес HTTP-сервера | `:8080` |
| `DATABASE_URL` | DSN PostgreSQL. Без значения используется memory-хранилище | пусто |
| `REDIS_ADDR` | Адрес Redis для кэша резолва ссылок | пусто |
| `REDIS_USERNAME` | Username для Redis ACL | пусто |
| `REDIS_PASSWORD` | Пароль Redis | пусто |
| `REDIS_DB` | Номер Redis DB | `0` |
| `REDIS_CACHE_TTL` | TTL кэша для ссылок без собственного срока действия | `24h` |
| `REDIS_KEY_PREFIX` | Префикс ключей Redis | `shortener:link:` |

### API

```bash
curl -X POST http://localhost:8080/links \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://example.com"}'
```

Ответ:

```json
{"short_code":"abc123"}
```

Редирект работает по `GET /{short_code}`.

### Тесты

```bash
go test ./...
```

Интеграционные тесты PostgreSQL и Redis запускаются только при наличии DSN/адреса:

```bash
SHORTENER_TEST_DATABASE_URL='postgres://shortener:shortener@localhost:5432/shortener?sslmode=disable' \
SHORTENER_TEST_REDIS_ADDR='localhost:6379' \
go test ./...
```

CI находится в `.github/workflows/ci.yml`: `go vet`, `go test -race ./...` с PostgreSQL/Redis services и Docker build.
CD находится в `.github/workflows/cd.yml`: публикация Docker-образа в GHCR по тегу `v*.*.*` или ручному запуску.

# Осталось сделать
- kafka
- мониторинг
- swagger
- нагрузочное тестирование
