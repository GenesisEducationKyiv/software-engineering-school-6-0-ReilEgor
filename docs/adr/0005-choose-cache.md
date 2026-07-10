# [ADR-0005] Choosing Redis as a Cache for RepoNotifier

* **Status:** accepted
* **Deciders:** Yehor Reil (Lead Software Engineer)
* **Date:** 2026-04-07

## Context and Problem Statement

RepoNotifier frequently checks GitHub repositories for new releases. To reduce load on the GitHub API and speed up repeated access, a fast in-memory cache is required.

We need a caching solution that is fast, reliable, and easy to integrate with Go.

## Decision Drivers

* **Performance:** quick access to frequently read data (`last_seen_tag`)
* **TTL Support:** automatic expiration of stale data
* **Scalability:** must support growing number of repositories and users
* **Go Integration:** mature client libraries
* **Future Extensibility:** possible Pub/Sub usage for notifications

## Considered Options

* Redis
* Memcached
* In-memory application cache

## Decision Outcome

**Chosen option:** Redis

### Reasons

* Extremely fast key-value access
* Supports TTL, preventing stale cache issues
* Pub/Sub support for future notification features
* Mature Go client library (`go-redis`)
* Simple to deploy alongside the monolith

### Consequences

* **Positive:** faster repository checks and reduced GitHub API calls
* **Positive:** easy to expire and update cached `last_seen_tag` values
* **Negative:** requires managing a separate caching service
* **Negative:** must handle cache invalidation and potential consistency issues

---

## More Information

Redis will be used as a **fast in-memory cache** to store temporary or frequently accessed data, such as the last seen release of each repository.