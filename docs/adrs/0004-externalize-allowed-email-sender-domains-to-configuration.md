# 4. Externalize allowed email sender domains to configuration

Date: 2026-07-12

## Status

accepted

## Context

`internal/service/email_service.go` rejected any `from_address` that
didn't end with a single compiled-in literal,
`const allowedSenderDomain = "@posta.opencrafts.io"`. That's fine as long
as every OpenCrafts product sends under one brand's mail domain forever,
but it means adding a second verified sending domain — a separate
product brand, an acquired app, a white-labeled deployment — requires a
Go code change and a redeploy of gossip-monger itself, for a decision
that is really "which domains has the team verified in Resend," i.e.
operational configuration, not application logic.

## Decision

Move the allowlist into `internal/config/config.go` as
`ResendConfig.AllowedSenderDomains`, sourced from
`RESEND_ALLOWED_SENDER_DOMAINS` (comma-separated) and defaulting to
`@posta.opencrafts.io` so behavior is unchanged for anyone not setting
the new variable. `emailService` now holds `allowedSenderDomains
[]string` and accepts a `from_address` if it has *any* of the configured
domains as a suffix, threaded through `NewEmailService` from
`internal/app/app.go`.

## Consequences

- Adding a new sending domain is now an environment/config change plus
  Resend-side domain verification, not a Go source change.
- The allowlist is still centrally enforced by gossip-monger (not
  per-service-configurable) — any onboarded service can send from any
  domain in the list, there's no `service_id -> domain` mapping. That's
  intentional for now: nothing in the current set of consumers needs
  per-service domain restriction, and adding it would be speculative.
  If that need arises, it's a follow-up decision, not a reversal of this
  one.
- `docs/email_integration_guide.md` was updated to describe the domain
  requirement as configurable/team-communicated rather than a single
  fixed string.