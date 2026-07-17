# 6. Add circuit breaker and dead-letter retry for third-party notification providers

Date: 2026-07-12

## Status

accepted

## Context

Verified directly in code that a transient Resend/OneSignal outage previously resulted in a message being silently, permanently lost:

- `internal/broker/consumer.go`'s consume loop called `msg.Nack(false, false)` on any handler error, and the queues had no dead-letter exchange configured — RabbitMQ simply discarded the message. No DLQ, no backoff, no second attempt.
- `internal/service/email_service.go`'s `Send` swallowed the Resend error — it recorded `status = "failed"` in the database, then **returned `nil`**, so the consumer `Ack`'d the message and it was gone. `docs/email_integration_guide.md` claimed automatic retry; that was aspirational and never actually implemented.
- `internal/service/push_notification_service.go`'s `Send` returned early on a OneSignal error **before** any database row was ever written — worse than email's case, with zero audit trail for a failed push.
- No circuit breaker existed anywhere, so a sustained outage meant every queued message still triggered a live, blocking HTTP call certain to fail, burning through consumer prefetch slots.

## Decision

Add a per-provider circuit breaker (`internal/resilience`, wrapping `sony/gobreaker/v2`) around the OneSignal and Resend calls, and pair it with RabbitMQ-native retry: the two provider queues now dead-letter into a shared `gossip.retry.exchange` on any handler error, land in `gossip.retry.queue` for a configurable delay (`RETRY_DELAY_SECONDS`), then RabbitMQ automatically redelivers them to their original queue. After `MAX_RETRY_ATTEMPTS` (tracked via RabbitMQ's own `x-death` header — no schema needed for that), a message is routed to `gossip.parked.queue` for manual triage instead of retried forever.

Both services now persist every outcome — including `circuit_open` (the breaker rejected the call outright) and `failed` (the provider itself errored) — as a distinct status, and both `email_requests` and `notifications` are now upserted by `queue_message_id` rather than always inserted, so a dead-lettered redelivery updates the same row instead of hitting a unique-constraint violation on the second attempt (which would have silently defeated the whole retry mechanism).

`internal/broker/publisher.go`'s previously-unused `Publisher`/`MessagePublisher` now has its first real caller: routing exhausted messages to the parked queue.

Deliberately deferred, not forgotten:
- Metrics/alerting on breaker state — no metrics stack exists in this service yet; breaker transitions are logged via `slog` only.
- Multi-tier exponential backoff — one configurable fixed retry delay for v1.
- Extending this DLX/retry pattern to `user_consumer.go` (Verisafe sync) — it doesn't call a third-party API, so it's a different failure mode and out of scope here.

## Consequences

- A provider outage now fails fast (breaker open) instead of piling up blocking HTTP calls, and a failed send gets several automatic retries with a delay before landing somewhere a human can see it — the two problems ("silently fails", "no retry policy") this ADR was written to fix.
- Operators can now distinguish, per attempt: `circuit_open` (we didn't even try, provider already known-down), `failed` (we tried and the provider rejected it or errored), and `sent`/`dispatched` (success) — previously push had no record of failures at all, and email conflated "failed" with what is now split out as `circuit_open`.
- Retrying via requeue means idempotency now matters where it didn't structurally before: `CreateEmailRequest`/`CreateNotification` becoming `UpsertEmailRequest`/`UpsertNotification` is a load-bearing part of this change, not incidental — removing the upsert behavior while keeping DLX retry would reintroduce silent failure via unique-constraint errors on every redelivery.
- New operational surface: three new RabbitMQ queues/exchanges to monitor (`gossip.retry.queue` depth indicates ongoing trouble; `gossip.parked.queue` depth indicates messages needing manual attention), and five new environment variables (`RETRY_DELAY_SECONDS`, `MAX_RETRY_ATTEMPTS`, `BREAKER_CONSECUTIVE_FAILURES`, `BREAKER_OPEN_TIMEOUT_SECONDS`, `BREAKER_HALF_OPEN_MAX_REQUESTS`), all with defaults so no operator action was required to adopt this.