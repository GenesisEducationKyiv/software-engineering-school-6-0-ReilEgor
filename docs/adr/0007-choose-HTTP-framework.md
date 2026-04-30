# [ADR-0007] Choosing HTTP framework for RepoNotifier

* **Status:** accepted
* **Deciders:** Yehor Reil (Lead Software Engineer)
* **Date:** 2026-04-09

---

## Context and Problem Statement

The RepoNotifier project requires an HTTP layer to expose REST endpoints for managing subscriptions and interacting with the system.

The framework must be lightweight, performant, easy to integrate with Clean Architecture, and suitable for a Go-based monolith.

---

## Decision Drivers

* **Performance:** Minimal overhead and fast request handling.
* **Simplicity:** Easy to learn and use for rapid development.
* **Ecosystem:** Availability of middleware and community support.
* **Compatibility:** Ability to integrate cleanly with the usecase layer without tight coupling.
* **Maintainability:** Clear structure for handlers and routing.

---

## Considered Options

* **Gin**
* **net/http (standard library)**
* **Echo**
* **Fiber**

---

## Decision Outcome

Chosen option: **Gin** because it provides an optimal balance between performance, simplicity, and developer experience.

---

## Consequences

### Positive

* **High performance:** Gin is built on top of `net/http` and is optimized for speed.
* **Minimal boilerplate:** Simplifies routing, middleware, and request handling.
* **Good ecosystem:** Wide adoption and availability of middleware (logging, recovery, validation).
* **Clean integration:** Handlers can remain thin and delegate logic to usecases.
* **Fast development:** Reduces time to implement REST endpoints.

### Negative

* **Abstraction over net/http:** Slight loss of control compared to raw standard library.
* **Context coupling:** Use of `gin.Context` requires mapping to internal DTOs.
* **Less minimalistic than net/http:** Adds an extra dependency.

---

## Pros and Cons of the Options

### Gin
* **Pros:** Fast, simple API, large community, rich middleware support.
* **Cons:** Additional abstraction layer, dependency on external library.

---

### net/http
* **Pros:** No external dependencies, full control, part of standard library.
* **Cons:** Requires more boilerplate, slower development speed.

---

### Echo
* **Pros:** Similar performance to Gin, slightly more features out-of-the-box.
* **Cons:** Smaller community compared to Gin, less widespread adoption.

---

### Fiber
* **Pros:** Very high performance, Express.js-like API.
* **Cons:** Not fully compatible with net/http, less idiomatic for Go.

---

## More Information

Gin will be used only in the delivery layer (HTTP handlers).  
Business logic will remain in the usecase layer, and domain models will remain independent of the HTTP framework.

This ensures that switching the HTTP framework in the future will require minimal changes.