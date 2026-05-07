# [ADR-0002] Adopting Clean Architecture for Service Design

* **Status:** accepted
* **Deciders:** Yehor Reil (Lead Software Engineer)
* **Date:** 2026-04-07

## Context and Problem Statement

The system includes multiple responsibilities:
- HTTP API
- Business logic
- Database access
- External integrations (GitHub API, Email)

A clear separation of concerns is required to ensure maintainability and testability.

## Decision Drivers

* **Maintainability:** Clear boundaries between layers
* **Testability:** Ability to mock dependencies
* **Scalability:** Easier future refactoring or extension
* **Code Organization:** Structured and predictable project layout

## Considered Options

* **Clean Architecture**
* **Layered Architecture (MVC)**
* **Minimal/Flat Structure**

## Decision Outcome

Chosen option: **Clean Architecture**

### Consequences

* **Good:** Business logic is independent of frameworks and infrastructure
* **Good:** Easy to write unit tests using interfaces and mocks
* **Good:** Flexible to replace DB, HTTP framework, or external APIs
* **Bad:** Increased initial complexity
* **Bad:** More boilerplate code

---

## Pros and Cons of the Options

### Clean Architecture
* **Pros:** Strong separation of concerns, high testability
* **Cons:** More code and abstractions

### Layered Architecture
* **Pros:** Simpler structure, widely understood
* **Cons:** Business logic often tightly coupled with infrastructure

### Minimal Structure
* **Pros:** Fast to start
* **Cons:** Quickly becomes unmaintainable

## More Information

The architecture is divided into:
- **Domain** (entities)
- **UseCase** (business logic)
- **Delivery** (HTTP)
- **Infrastructure** (DB, GitHub, Email)

Dependencies are directed inward to preserve isolation of core logic.