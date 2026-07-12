# 1. Drop foreign key from notifications to local users table

Date: 2026-07-12

## Status

accepted

## Context

Gossip Monger was built as, and has grown into, a generic notification
proxy: any `io.opencrafts.*` service can already publish to
`gossip.topic.exchange` and reach OneSignal/Resend without prior
registration on the push side. One piece of schema didn't fit that
picture: `notifications.target_user_id` and `notifications.source_user_id`
carried a `FOREIGN KEY ... REFERENCES users(id)`, and `users` is a local
cache that is *only* ever populated by consuming `verisafe.user.*` events
(`internal/broker/consumers/user_consumer.go` hard-rejects any
`source_service_id` other than `io.opencrafts.verisafe`).

In practice this meant: to target a push notification at someone, that
person had to already exist as a Verisafe-issued user in gossip-monger's
own database. Any subsystem whose recipients aren't Verisafe users — or
aren't users at all (devices, org accounts, external contacts) — could not
use `target_user_id`/`source_user_id` at all, even though the rest of the
targeting surface (segments, external ids, email tokens, phone numbers)
was already open to anyone.

We audited what actually depends on the constraint being enforced: there
is no HTTP API beyond `/ping` (`internal/handlers/ping_handler.go`), and
the only place `target_user_id` is read is
`preparePushPayload`/`hasTargeting` in
`internal/service/push_notification_service.go`, which forwards it to
OneSignal as a plain `external_id` alias string — it never dereferences
the `users` row. The FK bought row-existence checking for a read path
that doesn't exist yet, at the cost of blocking every non-Verisafe caller.

## Decision

Drop both foreign keys via
`database/migrations/20260712090000_drop_notifications_user_fk.sql`.
`target_user_id`/`source_user_id` remain nullable UUID columns with no
referential integrity against `users`. Any caller can now supply any
UUID as a target, and gossip-monger forwards it to OneSignal exactly as
it did before for values that happened to be known Verisafe users.

The migration locates the constraint(s) to drop via an
`information_schema` query rather than a hardcoded constraint name, and
the down migration restores them as `NOT VALID` so a rollback can't fail
against rows that by then reference ids no longer present in `users`.

## Consequences

- Any `io.opencrafts.*` service can target push notifications by UUID
  without first getting that UUID synced into gossip-monger via
  Verisafe. This removes the one structural blocker identified in the
  decoupling assessment (`.reports/decoupling-assessment.md`, untracked).
- No behavior change for existing callers: delivery still goes through
  the same `external_id` alias path in `preparePushPayload`.
- Postgres no longer guarantees `target_user_id`/`source_user_id`
  reference a row that exists. If a future feature needs to join
  notifications back to `users` (e.g. a dashboard), it must handle
  missing/foreign ids explicitly rather than relying on the schema.
- This decision deliberately does not change who can populate `users` in
  the first place — see [ADR-0002](0002-keep-local-user-directory-single-sourced-from-verisafe.md).