# [ADR-0001] Choosing Monolithic Architecture for the GitHub Release Notifier

* **Status:** accepted
* **Deciders:** Yehor Reil (Lead Software Engineer)
* **Date:** 2026-04-07

## Context and Problem Statement

The system must provide an API for managing subscriptions to GitHub repository releases, periodically scan for new releases, and send email notifications.

- All functionality must be implemented within a single service
- Simplicity of deployment is required

## Decision Drivers

* **Simplicity:** Fast development and minimal infrastructure complexity
* **Deployment:** Single containerized service
* **Maintainability:** Easier debugging and local development

## Considered Options

* **Monolith**
* **Microservices Architecture**

## Decision Outcome

Chosen option: **Monolithic Architecture**

### Consequences

* **Good:** Faster development and easier onboarding
* **Good:** Simple Docker-based deployment
* **Good:** No network overhead between components
* **Bad:** Reduced scalability compared to microservices
* **Bad:** Harder to split into services in the future

---

## Pros and Cons of the Options

### Monolith
* **Pros:** Simplicity, fewer moving parts, easy to test and deploy
* **Cons:** Tight coupling, limited scalability

### Microservices
* **Pros:** Independent scaling, better separation of concerns
* **Cons:** Overkill for current scope, operational complexity
