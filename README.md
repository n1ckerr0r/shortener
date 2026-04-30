## Shortener

HTTP-сервис для создания коротких ссылок.

### Запуск с PostgreSQL

```bash
docker compose up -d postgres
export DATABASE_URL='postgres://shortener:shortener@localhost:5432/shortener?sslmode=disable'
go run ./cmd/api
```

По умолчанию приложение слушает `:8080`. Адрес можно изменить через `SHORTENER_ADDR`.

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

Интеграционный тест PostgreSQL запускается только при наличии DSN:

```bash
SHORTENER_TEST_DATABASE_URL='postgres://shortener:shortener@localhost:5432/shortener?sslmode=disable' go test ./internal/link/postgres
```
