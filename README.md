# RepoNotifier
 
> Production-ready Go service that tracks GitHub repository releases and sends real-time email notifications to subscribers.
 
[![codecov](https://codecov.io/gh/ReilEgor/NotifierTest/graph/badge.svg?token=S8KWDBMUQ7)](https://codecov.io/gh/ReilEgor/NotifierTest)
![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)
![License](https://img.shields.io/badge/license-MIT-green)
 
---

## Table of Contents
 
- [Overview](#overview)
- [How It Works](#how-it-works)
- [Features](#features)
- [Architecture](#architecture)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [API Reference](#api-reference)
- [Observability](#observability)
- [Tech Stack](#tech-stack)
- [Contributing](#contributing)
 
---

## Overview
 
RepoNotifier continuously monitors GitHub repositories and notifies users when new releases are published. It is built with **Clean Architecture**, resilience patterns, and observability in mind - ready to run in production from day one.
 
---

## How It Works
 
```
User subscribes to a repo
        |
        ▼
Background worker polls GitHub API on schedule
        |
        ▼
New release detected via last_seen_tag comparison
        |
        ▼
Release event cached in Redis
        |
        ▼
Email notification dispatched to all subscribers
```

1. **Subscribe** - a user registers their email and a target GitHub repository via REST or gRPC.
2. **Scan** - a background worker periodically queries the GitHub API for each tracked repository.
3. **Detect** - new releases are identified by comparing the current tag against the stored `last_seen_tag`.
4. **Cache** - release events are cached in Redis to prevent duplicate notifications and reduce API pressure.
5. **Notify** - matching subscribers receive an email with release details.
 
---

## Features
 
| Feature | Description |
|---|---|
| 🔔 Automated tracking | Background scanner detects new releases using `last_seen_tag` strategy |
| 📫 Email notifications | Instant alerts via SMTP - compatible with Mailtrap, SendGrid, Gmail |
| 🛡 Rate limit handling | Graceful handling of GitHub API `429 Too Many Requests` |
| ⚡ Caching layer | Redis caching reduces redundant API calls and prevents duplicate emails |
| 🌐 Dual interface | REST API (Gin) + gRPC support |
| 🔥 Resilience patterns | Circuit Breaker (gobreaker), retry strategy, graceful shutdown |
| 📊 Observability | Prometheus metrics + Grafana dashboards |
| 🔐 API key auth | All sensitive endpoints require `X-API-Key` header |
| 🧱 Clean Architecture | Decoupled layers with dependency injection |
 
---

## Architecture

### C4 Model

<img width="4524" height="1768" src="https://github.com/user-attachments/assets/15231bf2-ac06-43d8-b861-b3b8e1e63163" />
<img width="1837" height="849" alt="image" src="https://github.com/user-attachments/assets/a45bff06-2bcd-4f16-9b7a-f9ba8a153202" />

### Database Schema

<img width="617" height="671" alt="image" src="https://github.com/user-attachments/assets/f7fd9d6b-6119-4cc7-82bf-0edf38f16ba9" />

---
 
## Quick Start
 
> [!IMPORTANT]
> Complete the `.env` configuration before starting the app. Without valid credentials, the email and GitHub API integrations will fail.
 
```bash
# Clone the repository
git clone https://github.com/ReilEgor/RepoNotifier.git
cd RepoNotifier
 
# Copy and fill in environment variables
cp deployments/.env.example deployments/.env
# Edit deployments/.env - see Configuration section below
 
# Build and start all services
docker-compose -f deployments/docker-compose.yml up --build
```
 
Once running, verify the services are healthy:
 
| Service | URL |
|---|---|
| REST API | http://localhost:8080 |
| Swagger UI | http://localhost:9080 |
| Prometheus metrics | http://localhost:8080/metrics |
 
---
 
## Configuration
 
Edit `deployments/.env` with your credentials before starting:
 
| Variable | Required | Description |
|---|---|---|
| `APP_API_KEY` | **Required** | Secret key for `X-API-Key` authentication. All protected endpoints reject requests without this. |
| `EMAIL_USER` | **Required** | SMTP sender address (e.g. `you@gmail.com`). |
| `EMAIL_PASSWORD` | **Required** | SMTP app password - not your account login password. |
| `HTTP_PORT` | Optional | Port for the REST API. Default: `8080`. |
| `GRPC_PORT` | Optional | Port for the gRPC server. Default: `50051`. |
 
> **Gmail users**: generate an [App Password](https://myaccount.google.com/apppasswords) - standard account passwords are rejected by Gmail SMTP.
 
---
 
## API Reference
 
All protected endpoints require the `X-API-Key` header. Public endpoints (Swagger, healthcheck) do not.
 
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
 
**Success response** `201 Created`:
```json
{
  "message": "Subscription initiated. Please check your email to confirm."
}
```
 
---
 
### Unsubscribe from a repository
 
```bash
curl -X 'GET' \
  'http://localhost:8080/api/v1/unsubscribe/123' \
  -H 'accept: application/json'
```
 
**Success response** `200 OK`:
```json
{
  "error": "invalid or expired unsubscribe link"
}
```

---
 
### Error responses
 
| Status | Meaning |
|---|---|
| `400 Bad Request` | Missing or malformed request body |
| `401 Unauthorized` | Missing or invalid `X-API-Key` |
| `404 Not Found` | Subscription not found |
| `429 Too Many Requests` | GitHub API rate limit reached |
| `500 Internal Server Error` | Unexpected server error |
 
Full interactive documentation is available at **http://localhost:9080** (Swagger UI).
 
---

## Observability
 
RepoNotifier exposes Prometheus metrics at `/metrics`. Recommended Grafana dashboards cover:
 
- GitHub API request rate and error rate
- Email delivery success/failure
- Background scanner cycle duration
- Circuit breaker state transitions
- Redis cache hit/miss ratio
 
---
 
## Tech Stack
 
| Layer | Technology |
|---|---|
| Language | Go 1.25+ |
| HTTP framework | Gin |
| RPC | gRPC |
| Database | PostgreSQL |
| Cache | Redis |
| Resilience | gobreaker (Circuit Breaker) |
| Metrics | Prometheus |
| Dashboards | Grafana |
| API docs | Swagger / OpenAPI |
| Infrastructure | Docker, Docker Compose |
| External API | GitHub REST API |
 
---
 
## Contributing
 
Contributions, bug reports, and feature requests are welcome.
 
1. Fork the repository
2. Create a feature branch: `git checkout -b feat/your-feature`
3. Commit your changes: `git commit -m 'feat: add your feature'`
4. Push and open a pull request
 
Please follow the existing code style and add tests for any new functionality.
 
---
 
## License
 
This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.
