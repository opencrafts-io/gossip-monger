# 3. Auto-register services on first email send

Date: 2026-07-12

## Status

accepted

## Context

`email_requests.service_id` has a `NOT NULL REFERENCES services(id)`
(`database/migrations/20260417120358_add_email_model.sql`), and `services`
was created with exactly five hardcoded, seeded rows (`verisafe`,
`sherehe`, `veribroke`, `keepup`, `professor`). There is no code path that
inserts into `services` at runtime. Consequently, onboarding a new email
sender today means a human has to write and ship a migration inserting a
row into `services` *before* that service's first `email.send` message
can be processed — confirmed in `docs/email_integration_guide.md`, which
tells integrators to "confirm the following with the Gossip team"
including service registration.

This is inconsistent with push, which has no equivalent gate: anything
whose `source_service_id` passes the `io.opencrafts.*` namespace check in
`internal/broker/consumers/push_notification_consumer.go` can already send
push notifications with zero pre-registration. Email's registry
requirement is the odd one out, and it's a pure process bottleneck, not a
safety mechanism — the namespace check already does the gatekeeping that
matters (only requests over gossip-monger's own RabbitMQ credentials,
already namespaced `io.opencrafts.*`, get this far).

We still want to keep a `services` table as an audit/registry record of
who has sent email through the system, rather than dropping the
foreign key and validating by namespace prefix alone (which was the
alternative considered).

## Decision

Add a `UpsertService` query (`database/queries/services.sql`) and call it
from `internal/service/email_service.go`'s `Send`, inside the same
transaction as `CreateEmailRequest`, using the `source_service_id` as both
the row's `id` and `name` (ids are already unique, so this satisfies
`services.name`'s `NOT NULL UNIQUE` constraint without inventing a
separate human-assigned display name). The insert uses
`ON CONFLICT (id) DO UPDATE` (a no-op update) rather than `DO NOTHING`, so
`RETURNING *` always yields exactly one row whether the service already
existed or not.

The five pre-seeded rows and the FK itself are untouched — this is
additive. `email_consumer.go`'s existing `strings.HasPrefix(id,
"io.opencrafts.")` check still runs before this, so auto-registration
can't be used to register a non-namespaced or spoofed id.

## Consequences

- A new `io.opencrafts.*` service can send its first email without a
  migration or a support ticket; `services` now reflects "every service
  that has ever sent mail" rather than "every service someone remembered
  to add."
- `services.is_active` can still be flipped to `false` to hard-block a
  service after the fact (the FK plus this flag remain useful even
  though registration itself is automatic).
- The registry loses its previous meaning of "explicitly approved
  senders" — it now means "senders seen so far." If a future requirement
  needs pre-approval gating again, that's a new decision, not a
  reversion of this one.