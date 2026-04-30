# [ADR-0004] Choosing Primary Database for RepoNotifier

* **Status:** accepted
* **Deciders:** Yehor Reil (Lead Software Engineer)
* **Date:** 2026-04-07

## Context and Problem Statement

RepoNotifier needs to store user subscriptions to GitHub repositories and track the last seen release for each repository. The service requires a reliable, consistent database with support for complex queries and migrations.

We need to select a **primary database** that is reliable, scalable, and well-supported in Go.

## Decision Drivers

* **ACID Compliance:** subscription and release data must be stored reliably
* **Maintainability:** easy schema migrations and data management
* **Scalability:** able to handle increasing number of users and repositories
* **Go Integration:** mature libraries for database interaction
* **Community Support:** large ecosystem and documentation

## Considered Options

* PostgreSQL
* MySQL
* SQLite

## Decision Outcome

**Chosen option:** PostgreSQL

### Reasons

* Full ACID compliance ensures data correctness
* Mature support in Go (`pgx`, `gorm`, `sqlx`)
* Flexible schema management with migrations
* Reliable for production and widely used in the industry

### Consequences

* **Positive:** reliable storage of subscriptions and repository release history
* **Positive:** easy to extend schema as the system grows
* **Negative:** requires running a separate database service

---

## More Information

PostgreSQL will serve as the **primary persistent store** for all critical RepoNotifier data.