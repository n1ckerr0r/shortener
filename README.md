## Shortener

Shortener-платформа для создания коротких ссылок, быстрых редиректов и асинхронной аналитики.

### Что используется

- HTTP API для публичного интерфейса.
- gRPC для внутреннего Link Service API.
- PostgreSQL как transactional source of truth для ссылок.
- Redis как cache для редиректов.
- Kafka для событий кликов `link.clicked`.
- MongoDB для хранения click analytics.
- RabbitMQ для фоновых URL-check jobs.
- Prometheus для сбора метрик.
- Grafana для dashboards.
- Worker pools и graceful shutdown через `context`.

### Запуск всего стека

```bash
docker compose up --build
```

Compose поднимает:

- `api` на `http://localhost:8080`, gRPC на `localhost:9090`
- `analytics` Kafka consumer service
- `urlcheck` RabbitMQ worker service
- PostgreSQL `localhost:5432`
- Redis `localhost:6379`
- Kafka `localhost:9092`
- RabbitMQ `localhost:5672`, UI `http://localhost:15672`
- MongoDB `localhost:27017`
- Prometheus `http://localhost:9091`
- Grafana `http://localhost:3000`

RabbitMQ UI:

```text
login: shortener
password: shortener
```

Grafana:

```text
login: admin
password: admin
```

### Проверка сценария

Создать ссылку:

```bash
curl -X POST http://localhost:8080/links \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://example.com"}'
```

Ответ:

```json
{"short_code":"abc123"}
```

Перейти по ссылке:

```bash
curl -i http://localhost:8080/abc123
```

Что произойдёт внутри:

- `api` сохранит ссылку в PostgreSQL.
- `api` отправит `URLCheckJob` в RabbitMQ.
- `urlcheck` заберёт job из RabbitMQ и проверит URL через HTTP `HEAD`.
- При редиректе `api` прочитает Redis cache или PostgreSQL.
- `api` отправит `ClickEvent` в Kafka topic `link.clicked`.
- `analytics` прочитает событие из Kafka и запишет его в MongoDB collection `shortener.clicks`.

Посмотреть события в MongoDB:

```bash
docker compose exec mongo mongosh --quiet --eval 'db.getSiblingDB("shortener").clicks.find().pretty()'
```

### Health endpoints

```bash
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
```

`/healthz` проверяет, что процесс жив. `/readyz` проверяет PostgreSQL и Redis, если они включены.

### Metrics and Monitoring

API service отдаёт Prometheus metrics:

```bash
curl http://localhost:8080/metrics
```

Prometheus UI:

```text
http://localhost:9091
```

Grafana UI:

```text
http://localhost:3000
```

В Grafana datasource `Prometheus` и dashboard `Shortener Overview` создаются автоматически через provisioning. Dashboard находится в папке `Shortener`.

Полезные PromQL-запросы:

```promql
sum by (path, method, status) (rate(shortener_http_requests_total[1m]))
histogram_quantile(0.95, sum by (le, path) (rate(shortener_http_request_duration_seconds_bucket[5m])))
shortener_http_in_flight_requests
go_goroutines
go_memstats_alloc_bytes
```

Чтобы графики начали двигаться, создай несколько ссылок и сделай несколько редиректов:

```bash
for i in $(seq 1 20); do
  curl -s -X POST http://localhost:8080/links \
    -H 'Content-Type: application/json' \
    -d "{\"url\":\"https://example.com/?n=$i\"}" >/dev/null
done
```

### Локальный запуск без Docker для app-кода

Поднять инфраструктуру:

```bash
docker compose up -d postgres redis kafka rabbitmq mongo
```

Запустить API:

```bash
export DATABASE_URL='postgres://shortener:shortener@localhost:5432/shortener?sslmode=disable'
export REDIS_ADDR='localhost:6379'
export KAFKA_BROKERS='localhost:9092'
export RABBITMQ_URL='amqp://shortener:shortener@localhost:5672/'
go run ./cmd/api
```

Запустить analytics:

```bash
export KAFKA_BROKERS='localhost:9092'
export MONGO_URI='mongodb://localhost:27017'
go run ./cmd/analytics
```

Запустить urlcheck:

```bash
export RABBITMQ_URL='amqp://shortener:shortener@localhost:5672/'
go run ./cmd/urlcheck
```

### Переменные окружения

| Переменная | Назначение | По умолчанию |
| --- | --- | --- |
| `SHORTENER_ADDR` | HTTP address API service | `:8080` |
| `SHORTENER_GRPC_ADDR` | gRPC address Link Service | `:9090` |
| `SHORTENER_STARTUP_TIMEOUT` | Таймаут подключения к инфраструктуре при старте | `5s` |
| `SHORTENER_SHUTDOWN_TIMEOUT` | Таймаут graceful shutdown | `10s` |
| `SHORTENER_READ_TIMEOUT` | HTTP read timeout | `5s` |
| `SHORTENER_WRITE_TIMEOUT` | HTTP write timeout | `10s` |
| `SHORTENER_IDLE_TIMEOUT` | HTTP idle timeout | `60s` |
| `DATABASE_URL` | DSN PostgreSQL. Без значения используется memory storage | пусто |
| `REDIS_ADDR` | Redis cache address | пусто |
| `REDIS_USERNAME` | Redis ACL username | пусто |
| `REDIS_PASSWORD` | Redis password | пусто |
| `REDIS_DB` | Redis DB | `0` |
| `REDIS_CACHE_TTL` | Cache TTL для ссылок без expiration | `24h` |
| `REDIS_KEY_PREFIX` | Redis key prefix | `shortener:link:` |
| `KAFKA_BROKERS` | Kafka brokers через запятую | пусто для API, `localhost:9092` для analytics |
| `KAFKA_CLICK_TOPIC` | Topic для click events | `link.clicked` |
| `KAFKA_GROUP_ID` | Analytics consumer group | `shortener-analytics` |
| `KAFKA_PUBLISH_WORKERS` | Количество async Kafka producer workers | `4` |
| `KAFKA_PUBLISH_BUFFER` | Размер producer buffer | `1024` |
| `MONGO_URI` | MongoDB URI для analytics | `mongodb://localhost:27017` |
| `MONGO_DATABASE` | MongoDB database | `shortener` |
| `MONGO_CLICK_COLLECTION` | MongoDB collection для кликов | `clicks` |
| `ANALYTICS_WORKERS` | Количество MongoDB writer workers | `4` |
| `RABBITMQ_URL` | RabbitMQ URL | пусто для API, `amqp://guest:guest@localhost:5672/` для urlcheck |
| `RABBITMQ_URL_CHECK_QUEUE` | RabbitMQ queue для URL checks | `url.check` |
| `URLCHECK_WORKERS` | Количество URL-check workers | `4` |
| `URLCHECK_TIMEOUT` | Timeout HTTP проверки URL | `5s` |

### Команды разработки

```bash
make check
make build
make run
make test
make test-race
make lint
make docker-build
make compose-up
make compose-down
```

Интеграционные тесты PostgreSQL и Redis запускаются только при наличии DSN/адреса:

```bash
SHORTENER_TEST_DATABASE_URL='postgres://shortener:shortener@localhost:5432/shortener?sslmode=disable' \
SHORTENER_TEST_REDIS_ADDR='localhost:6379' \
go test ./...
```

Архитектура описана в `docs/architecture.md`.
