# [ADR-0004] Using a Separate Swagger UI Service for API Documentation

* **Status:** accepted
* **Deciders:** Yehor Reil (Lead Software Engineer)
* **Date:** 2026-04-07

## Context and Problem Statement

The RepoNotifier monolith exposes a REST API for managing GitHub repository subscriptions. To provide developers with a **friendly interface for exploring and testing the API**, we need a Swagger UI.

Two main approaches were considered:

1. Serve Swagger UI directly from the RepoNotifier monolith
2. Create a dedicated, separate Swagger UI service that hosts the OpenAPI specification

Although there is currently only one service, we want the architecture to be **flexible for potential future expansion**, such as splitting functionality into multiple services or adding more APIs.

## Decision Drivers

* **Developer Experience:** Easy access to API docs
* **Maintainability:** Avoid duplicating Swagger setup inside the monolith
* **Future Scalability:** Ability to aggregate multiple APIs if the project grows
* **Decoupling:** Keep the monolith focused solely on business logic

## Considered Options

### 1. Embedded Swagger UI inside the monolith

**Pros:**

- Simple to implement for a single service
- No additional container required

**Cons:**

- Tightly couples documentation with business logic
- Harder to expand if more APIs/services are added in the future
- Updating Swagger UI or docs might require redeploying the monolith

### 2. Dedicated Swagger UI Service

**Pros:**

- Central location for all API documentation
- Monolith remains focused on core functionality
- Easy to add future API specs without changing the monolith
- Can mount or proxy multiple Swagger/OpenAPI files if more services appear

**Cons:**

- Requires an additional container
- Slightly more complex initial setup

## Decision Outcome

Chosen option: **Dedicated Swagger UI Service**

This approach provides a **central, future-proof way to expose API documentation**. Even with only one service now, the system is prepared to handle multiple APIs in the future without modifying the RepoNotifier monolith.

### Consequences

* **Good:** Monolith stays lightweight and focused
* **Good:** Unified, extendable interface for API documentation
* **Good:** Easy to add future services’ APIs
* **Bad:** Another container to manage
* **Bad:** Slightly more complex deployment compared to embedding Swagger UI