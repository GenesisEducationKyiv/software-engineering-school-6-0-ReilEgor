# [ADR-0006] Choosing Monitoring Stack for RepoNotifier

* **Status:** accepted
* **Deciders:** Yehor Reil (Lead Software Engineer)
* **Date:** 2026-04-07

## Context and Problem Statement

RepoNotifier requires observability to monitor its health, performance, and errors. We need a monitoring solution that allows:

* Metrics collection (HTTP requests, background tasks, DB stats)
* Log aggregation and search
* Easy visualization for developers and operators
* Future scalability for multiple services if the system grows

## Decision Drivers

* **Metrics monitoring:** track uptime, latency, errors
* **Logging:** centralized collection of logs from all services
* **Visualization:** dashboards for metrics and logs
* **Extensibility:** support adding new services in the future
* **Open-source and community support:** stable and well-supported tools
* **Integration with Docker:** easy to deploy in containers

## Considered Options

1. **Prometheus + Grafana + Loki + Promtail**
2. **ELK Stack (Elasticsearch, Logstash, Kibana)**
3. **Cloud solutions (Datadog, New Relic, Grafana Cloud)**

---

## Decision Outcome

**Chosen option:** Prometheus + Grafana + Loki + Promtail

### Reasons

* **Prometheus:** industry-standard monitoring for metrics, works well with Go, supports PromQL for flexible queries, integrates easily with Docker.
* **Grafana:** powerful dashboards and visualization for metrics and logs, supports multiple data sources.
* **Loki:** log aggregation optimized for Grafana; simpler and lighter than Elasticsearch, stores logs efficiently.
* **Promtail:** lightweight agent to collect container logs and send them to Loki.
* **Extensibility:** this stack allows adding new microservices or metrics without major refactoring.

### Comparison with alternatives

* **ELK Stack:** heavy, higher resource consumption, more complex to manage; overkill for single monolith initially.
* **Cloud solutions:** introduces vendor lock-in, may increase costs; for early development and self-hosting, open-source stack is preferable.

---

## Consequences

* **Positive:** unified monitoring stack for metrics and logs; easy to add alerts in the future
* **Positive:** open-source, stable, widely adopted stack
* **Positive:** lightweight enough to run alongside the monolith in Docker
* **Negative:** requires maintenance of four separate components (Prometheus, Grafana, Loki, Promtail)
* **Negative:** need to configure retention policies, backups, and resource limits  