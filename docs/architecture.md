# Architecture

Проект реализован как небольшой набор сервисов вокруг shortener-домена. Это не полное production-разделение на все возможные bounded contexts, но здесь уже есть реальные коммуникации через broker-и, gRPC endpoint, cache, transactional storage, analytics storage и worker pools.

## Services

### API Service

Команда: `cmd/api`.

Ответственность:

- Публичный HTTP API.
- Внутренний gRPC Link Service endpoint.
- Создание ссылок.
- Редиректы.
- PostgreSQL writes/reads.
- Redis cache для resolve path.
- Kafka publish `link.clicked`.
- RabbitMQ publish `url.check`.
- Prometheus metrics endpoint `/metrics`.

### Analytics Service

Команда: `cmd/analytics`.

Ответственность:

- Kafka consumer group `shortener-analytics`.
- Чтение `link.clicked`.
- Worker pool для параллельной записи.
- Сохранение raw click events в MongoDB.

### URL Check Service

Команда: `cmd/urlcheck`.

Ответственность:

- RabbitMQ consumer queue `url.check`.
- Worker pool для параллельной проверки URL.
- HTTP `HEAD` проверки с timeout.
- Structured logs по результату проверки.

## Data Flow

### Link Creation

1. Client вызывает `POST /links`.
2. API валидирует URL.
3. Link Service сохраняет ссылку в PostgreSQL.
4. API публикует `URLCheckJob` в RabbitMQ queue `url.check`.
5. URL Check Service забирает job и проверяет URL.

### Redirect And Analytics

1. Client вызывает `GET /{short_code}`.
2. API проверяет Redis cache.
3. При cache miss API читает PostgreSQL и кладет результат в Redis.
4. API возвращает HTTP redirect.
5. API асинхронно отправляет `ClickEvent` в Kafka topic `link.clicked`.
6. Analytics Service читает topic через consumer group.
7. Analytics Service пишет событие в MongoDB.

## Storage

### PostgreSQL

Источник истины для ссылок.

- `short_code`
- `original_url`
- `created_at`
- `expires_at`
- `blocked`

### Redis

Горячий cache для read-heavy redirect path.

- key: `shortener:link:{code}`
- value: original URL
- TTL ограничивается expiration ссылки, если она есть.

### MongoDB

Документное хранилище аналитики.

- database: `shortener`
- collection: `clicks`
- document: code, original_url, clicked_at, remote_addr, user_agent, stored_at

## Messaging

### Kafka

Используется для clickstream.

- topic: `link.clicked`
- producer: API Service
- consumer: Analytics Service

Kafka выбрана для событий, которые полезно replay-ить и читать несколькими consumer groups.

### RabbitMQ

Используется для command/job очереди.

- queue: `url.check`
- producer: API Service
- consumer: URL Check Service

RabbitMQ выбран для фоновых задач, где важна понятная queue-семантика и ack/nack.

## gRPC

API Service поднимает gRPC endpoint на `SHORTENER_GRPC_ADDR`.

Service name:

```text
shortener.link.v1.LinkService
```

Methods:

```text
CreateLink
ResolveLink
```

В этом репозитории gRPC descriptor зарегистрирован вручную, а transport использует JSON codec. Это позволяет держать проект собираемым без `protoc`, но следующий production-шаг: добавить `.proto`, `buf`, generated stubs и grpc-gateway при необходимости.

## Observability

### Prometheus

Prometheus scrape-ит API Service по endpoint:

```text
api:8080/metrics
```

Конфиг находится в `deploy/prometheus/prometheus.yml`.

Собираются:

- стандартные Go/process metrics от Prometheus Go client;
- `shortener_http_requests_total`;
- `shortener_http_request_duration_seconds`;
- `shortener_http_in_flight_requests`.

### Grafana

Grafana получает datasource и dashboard через provisioning.

- datasource: `deploy/grafana/provisioning/datasources/prometheus.yml`
- dashboard provider: `deploy/grafana/provisioning/dashboards/dashboards.yml`
- dashboard JSON: `deploy/grafana/dashboards/shortener-overview.json`

Dashboard `Shortener Overview` показывает request rate, latency, in-flight requests, Go heap и goroutines.

## Concurrency

Многопоточность используется там, где есть реальная польза:

- Kafka publisher в API работает через bounded buffered channel и worker pool.
- Analytics Service читает Kafka и параллельно пишет события в MongoDB.
- URL Check Service параллельно обрабатывает RabbitMQ jobs.
- Все долгоживущие workers останавливаются через `context`.
- HTTP/gRPC server shutdown работает через graceful shutdown.

## Current Tradeoffs

- Link Service пока живёт внутри API binary. gRPC endpoint уже есть, но отдельный `cmd/link` можно вынести следующим шагом.
- Kafka publish и RabbitMQ publish не блокируют основной бизнес-сценарий: создание ссылки и редирект не должны падать только из-за аналитики или фоновой проверки.
- URL Check Service пока только логирует результат проверки. Следующий шаг: писать статус проверки в PostgreSQL или отдельную audit table.
