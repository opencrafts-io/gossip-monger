# 2. Keep local user directory single-sourced from Verisafe

Date: 2026-07-12

## Status

accepted

## Context

While removing the FK from `notifications` to `users`
([ADR-0001](0001-drop-foreign-key-from-notifications-to-local-users-table.md)),
we considered whether to also open the `users` table itself to multiple
identity sources. Today `internal/broker/consumers/user_consumer.go`
binds only to `verisafe.exchange` / `verisafe.user.*` and hard-rejects
any `user.created`/`updated`/`deleted` event whose `source_service_id`
isn't literally `io.opencrafts.verisafe`. That makes gossip-monger's
notion of "a user" entirely Verisafe's.

Making this multi-source (accepting user events from any
`io.opencrafts.*` service, keyed by `(source_service_id, external_id)`
instead of one global UUID space) is a real option, but it's a product
and data-modeling decision, not just an engineering one: it requires
deciding whether "the same user" is meant to mean the same thing across
every OpenCrafts app, how UUID collisions across independently-issued
identity spaces are handled, and whether the `users` table should even
be gossip-monger's responsibility to reconcile. None of that is needed
to unblock other apps from *sending* notifications — that need is
already satisfied by ADR-0001 without touching identity semantics at
all, and nothing in the running service currently reads the `users`
table back out (see ADR-0001's context).

## Decision

Leave `user_consumer.go`'s Verisafe-only source check as-is for now.
`users` continues to exist purely as a Verisafe-sourced directory; it is
not yet a general multi-tenant identity cache. This is a deliberate
deferral, not an oversight: the FK removal in ADR-0001 already achieves
the goal of letting any app target a notification by UUID, without
requiring a stance on cross-app identity.

## Consequences

- Other apps can target notifications (ADR-0001) but cannot yet push
  their *own* user directory into gossip-monger, and any future feature
  that wants to resolve a `target_user_id` back to a name/email/phone
  will only succeed for Verisafe-sourced users.
- Revisit this only when a concrete feature needs the local directory to
  be authoritative for more than one identity source (e.g. a
  cross-service notification dashboard). At that point, the key design
  question is whether identity should be keyed globally or per
  `(source_service_id, external_id)`.
- This ADR should be superseded, not silently contradicted, if that
  decision is made later.