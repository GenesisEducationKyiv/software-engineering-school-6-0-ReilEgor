# Testing Guide

This document explains how to run the different types of tests available in the project.

## Requirements

Before running the tests, make sure the following tools are installed on your machine:

- Go
- Docker
- Node.js and npm

---

## Unit Tests

Run unit tests for all services at once:

```bash
go test ./services/subscription/... ./services/tracking/... ./services/notification/... ./shared/...
```

Or per service:

```bash
go test ./services/subscription/internal/...
```
```bash
go test ./services/tracking/internal/...
```
```bash
go test ./services/notification/internal/...
```
```bash
go test ./shared/infrastructure...
```

## Integration Tests

Integration tests verify interactions between application components and external services such as databases or caches.

All required services are automatically started using Docker and Testcontainers.

Run integration tests with:

```bash
go test -v ./services/subscription/tests/integration/...
```

## Architecture Lint

This project uses [go-arch-lint](https://github.com/fe3dback/go-arch-lint) to enforce the
clean-architecture dependency rules described in [ADR-0002](docs/adr/0002-use-clean-architecture.md).

Install (one-time, pinned to the version used in CI):

```bash
go install github.com/fe3dback/go-arch-lint@v1.15.0
```

Run for all modules:

```bash
go-arch-lint check --project-path ./services/subscription
go-arch-lint check --project-path ./services/tracking
go-arch-lint check --project-path ./services/notification
go-arch-lint check --project-path ./shared
```

## End-to-End (E2E) Tests

E2E tests are written using Playwright and validate the application from the user's perspective.

Run E2E tests with:

```bash
npx --prefix tests/e2e playwright test
```
