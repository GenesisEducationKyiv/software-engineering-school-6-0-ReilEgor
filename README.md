# RepoNotifier
 
> Go system that tracks GitHub repository releases and sends real-time email notifications to subscribers. Built as three independently deployable services communicating over gRPC and RabbitMQ.
 
[![codecov](https://codecov.io/gh/ReilEgor/NotifierTest/graph/badge.svg?token=S8KWDBMUQ7)](https://codecov.io/gh/ReilEgor/NotifierTest)
![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)
 
---

## Table of Contents
 
- [Overview](#overview)
- [How It Works](#how-it-works)
- [Features](#features)
- [Architecture](#architecture)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [API Reference](#api-reference)
- [Testing](#testing)
- [Observability](#observability)
- [Tech Stack](#tech-stack)
- [Contributing](#contributing)
 
---

## Overview
 
RepoNotifier continuously monitors GitHub repositories and notifies users when new releases are published. It is split into three services - **subscription**, **tracking**, and **notification** - each with Clean Architecture internals, resilience patterns, and observability built in. See [ARCHITECTURE.md](ARCHITECTURE.md) for the full breakdown and [docs/adr](docs/adr/) for the reasoning behind each major decision.
 
---

## How It Works

1. **Subscribe** - a user registers their email and a target GitHub repository via the subscription service's REST or gRPC API; a confirmation email is sent (saga-coordinated).
2. **Scan** - the tracking service periodically queries the GitHub API for each tracked repository.
3. **Detect** - new releases are identified by comparing the current tag against the stored `last_seen_tag`.
4. **Publish** - release/notification events are written transactionally via the outbox pattern and relayed to RabbitMQ.
5. **Notify** - the notification service consumes the queue and emails matching subscribers via SMTP.
 
---

## Features
 
| Feature | Description |
|---|---|
| 🔔 Automated tracking | Background scanner detects new releases using `last_seen_tag` strategy |
| 📫 Email notifications | Instant alerts via SMTP - compatible with Mailtrap, SendGrid, Gmail |
| 🛡 Rate limit handling | Graceful handling of GitHub API `429 Too Many Requests` |
| ⚡ Caching layer | Redis caching reduces redundant API calls and prevents duplicate emails |
| 🌐 Dual interface | REST API (Gin) + gRPC support |
| 📨 Async messaging | RabbitMQ + outbox pattern for reliable event delivery between services |
| 🔥 Resilience patterns | Circuit Breaker (gobreaker), retry strategy, graceful shutdown |
| 📊 Observability | Prometheus metrics, Grafana dashboards, ELK log aggregation |
| 🔐 API key auth | All sensitive endpoints require `X-API-Key` header |
| 🧱 Clean Architecture | Decoupled layers per service, enforced in CI via go-arch-lint |
 
---

## Architecture

RepoNotifier is split into three independently deployable Go services, each with its own database, plus a shared `shared/` module for common infrastructure code. Full details, diagrams, and the "why" behind each decision live in [ARCHITECTURE.md](ARCHITECTURE.md).

| Service | Responsibility | Storage | Ports |
|---|---|---|---|
| **subscription** | Manages users/subscriptions, confirmation saga; REST (Gin) + gRPC API | PostgreSQL (`subscription_db`) | HTTP `8080`, gRPC `9091` |
| **tracking** | Polls GitHub API, detects new releases via `last_seen_tag`, gRPC API | PostgreSQL (`tracker_db`) | health/metrics `8081`, gRPC `50051` |
| **notification** | Consumes RabbitMQ commands and sends email via SMTP | stateless | health/metrics `8082` |

### C4 Model

![component_worker.png](docs/%D1%814/component_worker.png)
![container.png](docs/%D1%814/container.png)
![component_api.png](docs/%D1%814/component_api.png)
![component_sender.png](docs/%D1%814/component_sender.png)
<img width="4524" height="1768" src="https://github.com/user-attachments/assets/15231bf2-ac06-43d8-b861-b3b8e1e63163" />
<img width="1837" height="849" alt="image" src="https://github.com/user-attachments/assets/a45bff06-2bcd-4f16-9b7a-f9ba8a153202" />

---
 
## Quick Start
 
> [!IMPORTANT]
> Complete the env configuration before starting the app. Without valid credentials, the email and GitHub API integrations will fail.
 
```bash
# Clone the repository
git clone https://github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor.git
cd software-engineering-school-6-0-ReilEgor
 
# Copy and fill in environment variables:
# - deployments/.env holds shared/infra values (DB, Redis, RabbitMQ, ports)
# - deployments/env/*.env holds per-service config (subscription, tracking, notification)
cp deployments/.env.example deployments/.env
cp deployments/env/subscription.env.example deployments/env/subscription.env
cp deployments/env/tracking.env.example deployments/env/tracking.env
cp deployments/env/notification.env.example deployments/env/notification.env
# Edit the copied files - see Configuration section below
 
# Build and start all services
docker compose --profile observability --profile docs -f deployments/docker-compose.yml up --build
```
 
Once running, verify the services are healthy:
 
| Service | URL |
|---|---|
| Subscription REST API | http://localhost:8080 |
| Tracking health/metrics | http://localhost:8081/health |
| Notification health/metrics | http://localhost:8082/health |
| Swagger UI | http://localhost:9080 |
| RabbitMQ management | http://localhost:15672 |
| Prometheus | http://localhost:9090 |
| Grafana | http://localhost:3030 |
| Kibana | http://localhost:5601 |
 
---
 
## Configuration
 
Env files live under `deployments/`: `.env` for shared/infra values, `env/subscription.env`, `env/tracking.env` and `env/notification.env` for each service's own config. Edit them with your credentials before starting:
 
| Variable | File | Required | Description |
|---|---|---|---|
| `APP_API_KEY` | `env/subscription.env`, `env/tracking.env` | **Required** | Secret key for `X-API-Key` authentication. All protected endpoints reject requests without this. |
| `GITHUB_TOKEN` | `env/tracking.env` | **Required** | GitHub personal access token used to poll the releases API. |
| `EMAIL_USER` | `env/notification.env` | **Required** | SMTP sender address (e.g. `you@gmail.com`). |
| `EMAIL_PASSWORD` | `env/notification.env` | **Required** | SMTP app password - not your account login password. |
| `RABBITMQ_USER` / `RABBITMQ_PASSWORD` | `.env` | **Required** | Credentials for the shared RabbitMQ broker. |
| `APP_HTTP_PORT` | `.env` | Optional | Port for the subscription REST API. Default: `8080`. |
| `APP_GRPC_PORT` | `.env` | Optional | Port for the subscription gRPC server. Default: `9091`. |
| `TRACKING_GRPC_PORT` | `.env` | Optional | Port for the tracking gRPC server. Default: `50051`. |
| `WORKER_HEALTH_PORT` / `SENDER_HEALTH_PORT` | `.env` | Optional | Health/metrics ports for tracking and notification. Defaults: `8081` / `8082`. |
 
> **Gmail users**: generate an [App Password](https://myaccount.google.com/apppasswords) - standard account passwords are rejected by Gmail SMTP.
 
---
 
## API Reference
 
All endpoints below belong to the **subscription** service (`http://localhost:8080/api/v1`). Protected endpoints require the `X-API-Key` header. Public endpoints (confirm, unsubscribe, Swagger, healthcheck) do not.
 
### Subscribe to a repository
 
```bash
curl -X 'POST' \
  'http://localhost:8080/api/v1/subscribe' \
  -H 'accept: application/json' \
  -H 'X-API-Key: my-super-secret-token-123' \
  -H 'Content-Type: application/json' \
  -d '{
  "email": "test@gmail.com",
  "repository": "ReilEgor/NotifierTest"
}'
```
 
**Success response** `202 Accepted`:
```json
{
  "message": "Subscription initiated. Please check your email to confirm."
}
```
 
---

### Confirm a subscription

```bash
curl -X 'GET' \
  'http://localhost:8080/api/v1/confirm/{token}' \
  -H 'accept: application/json'
```

The `{token}` comes from the confirmation link emailed to the subscriber.

---
 
### Unsubscribe from a repository
 
```bash
curl -X 'GET' \
  'http://localhost:8080/api/v1/unsubscribe/{token}' \
  -H 'accept: application/json'
```
 
**Success response** `200 OK`:
```json
{
  "message": "You have been successfully unsubscribed"
}
```

---

### List subscriptions

```bash
curl -X 'GET' \
  'http://localhost:8080/api/v1/subscriptions?email=test@gmail.com' \
  -H 'accept: application/json' \
  -H 'X-API-Key: my-super-secret-token-123'
```

---
 
### Error responses
 
| Status | Meaning |
|---|---|
| `400 Bad Request` | Missing or malformed request body |
| `401 Unauthorized` | Missing or invalid `X-API-Key` |
| `404 Not Found` | Subscription or token not found/expired |
| `429 Too Many Requests` | GitHub API rate limit reached |
| `500 Internal Server Error` | Unexpected server error |
 
Full interactive documentation is available at **http://localhost:9080** (Swagger UI).

---

## Testing

Unit tests, integration tests (Testcontainers), architecture-boundary checks (go-arch-lint), and E2E tests (Playwright) are all documented in [testing.md](testing.md).
 
---

## Observability
 
Each service exposes Prometheus metrics at `/metrics` (subscription on `8080`, tracking on `8081`, notification on `8082`) and structured logs shipped to Elasticsearch via Fluent Bit. Grafana and Kibana dashboards cover:
 
- GitHub API request rate and error rate
- Email delivery success/failure
- Background scanner cycle duration
- Circuit breaker state transitions
- Redis cache hit/miss ratio
- Centralized service logs (Kibana)

```bash
# Core only
docker compose -f deployments/docker-compose.yml up -d

# With monitoring (Prometheus, Grafana, ELK)
docker compose -f deployments/docker-compose.yml --profile observability up -d

# With API documentation (Swagger UI)
docker compose -f deployments/docker-compose.yml --profile docs up -d

# All at once
docker compose -f deployments/docker-compose.yml --profile observability --profile docs up -d
```

---
 
## Tech Stack
 
| Layer | Technology |
|---|---|
| Language | Go 1.25+ |
| HTTP framework | Gin |
| RPC | gRPC (google.golang.org/grpc) |
| Message broker | RabbitMQ |
| Database | PostgreSQL |
| Cache | Redis |
| Resilience | gobreaker (Circuit Breaker) |
| Metrics | Prometheus |
| Dashboards | Grafana |
| Logs | Fluent Bit + Elasticsearch + Kibana |
| API docs | Swagger / OpenAPI |
| Architecture linting | go-arch-lint |
| Infrastructure | Docker, Docker Compose |
| External API | GitHub REST API |
 
---
 
## Contributing
 
Contributions, bug reports, and feature requests are welcome.
 
1. Fork the repository
2. Create a feature branch: `git checkout -b feat/your-feature`
3. Commit your changes: `git commit -m 'feat: add your feature'`
4. Push and open a pull request
 
Please follow the existing code style, keep changes within their service's architecture boundaries (`go-arch-lint check`), and add tests for any new functionality.
