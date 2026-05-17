# Testing Guide

This document explains how to run the different types of tests available in the project.

## Requirements

Before running the tests, make sure the following tools are installed on your machine:

- Go
- Docker
- Node.js and npm

---

## Run All Tests

To run all tests in the project

```bash
go test -v ./...
```

## Unit Tests

Unit tests cover the internal business logic of the application and do not require any external dependencies.

Run all unit tests with:

```bash
go test -v ./internal/...
```

## Integration Tests

Integration tests verify interactions between application components and external services such as databases or caches.

All required services are automatically started using Docker and Testcontainers.

Run integration tests with:

```bash
go test -v ./tests/integration/...

```

## End-to-End (E2E) Tests

E2E tests are written using Playwright and validate the application from the user's perspective.

Run E2E tests with:

```bash
npx --prefix static playwright test
```