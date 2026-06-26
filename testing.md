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

## Integration Tests

Integration tests verify interactions between application components and external services such as databases or caches.

All required services are automatically started using Docker and Testcontainers.

Run integration tests with:

```bash
go test -v ./services/subscription/tests/integration/...
```

## End-to-End (E2E) Tests

E2E tests are written using Playwright and validate the application from the user's perspective.

Run E2E tests with:

```bash
npx --prefix tests/e2e playwright test
```
