# [ADR-0008] Choosing RabbitMQ as Message Broker for RepoNotifier

* **Status:** accepted
* **Deciders:** Yehor Reil (Lead Software Engineer)
* **Date:** 2026-06-16

## Context and Problem Statement

RepoNotifier needs to decouple release detection from notification delivery. When a new GitHub release is detected, the event must be forwarded to the notification sender asynchronously, without tight coupling between components.

We need a message broker that is reliable, supports delivery guarantees, and is easy to integrate with Go.

## Decision Drivers

* **Decoupling:** tracking and notification services must be independent
* **Reliability:** messages must not be lost if the consumer is temporarily unavailable
* **Delivery Guarantees:** at-least-once delivery with manual acknowledgements
* **Go Integration:** mature client libraries
* **Operational Simplicity:** easy to run locally and in Docker

## Considered Options

* RabbitMQ
* Apache Kafka
* Redis Pub/Sub

## Decision Outcome

**Chosen option:** RabbitMQ

### Reasons

* Supports durable queues and persistent messages — no event loss on restart
* Manual acknowledgement model fits the "process once, ack on success" pattern
* Lightweight and easy to set up with the official Docker image
* Mature Go client library (`amqp091-go`)
* Management UI included for visibility during development
* Sufficient for current throughput — no need for Kafka's complexity at this scale

### Consequences

* **Positive:** tracking and notification services are fully decoupled
* **Positive:** failed notification attempts do not block release detection
* **Positive:** easy to add more consumers without changing the producer
* **Negative:** requires managing a separate broker service
* **Negative:** RabbitMQ clustering is more complex if horizontal scaling is needed later

---

## More Information

RabbitMQ serves as the **event bus** between the `notification-worker` (producer) and `notification-sender` (consumer). The worker publishes a release event; the sender consumes it and delivers the email notification.
