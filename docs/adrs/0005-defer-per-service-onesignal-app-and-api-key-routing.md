# 5. Defer per-service OneSignal app and API key routing

Date: 2026-07-12

## Status

accepted

## Context

`internal/service/push_notification_service.go` always constructs the
OneSignal notification against a single, globally configured app:
`onesignal.NewNotification(os.Getenv("ONESIGNAL_APP_ID"))`. The
`Notification.AppID` column (`app_id` in
`database/migrations/20250828171345_add_notification_model.sql`) is
faithfully persisted but never read back to select a different OneSignal
app/API key — `docs/push_notification_integration.md` already documents
this explicitly: "The `app_id` field on the notification object is
ignored. The service uses its own configured OneSignal app ID."

This is fine while every current and near-term OpenCrafts subsystem
shares one mobile app / one OneSignal project. It becomes a real
blocker only once a subsystem ships its own mobile app with its own
OneSignal project, at which point gossip-monger would need a
`source_service_id -> {app_id, rest_api_key}` lookup (env-driven map or a
small table, mirroring how `services` works for email) with a fallback to
today's single global env vars for anything not in the map.

Building that now, with no second OneSignal project in existence, would
be speculative: we'd be guessing at the shape of a requirement (per-id
config? per-id table? multiple OneSignal clients cached in memory?)
without a real case to validate it against.

## Decision

Do not build multi-app OneSignal routing yet. Leave
`ONESIGNAL_APP_ID`/`ONESIGNAL_REST_API_KEY` as single global values and
`Notification.AppID` as persisted-but-unused, exactly as documented
today. Revisit this ADR (supersede it) when a second OneSignal
project/app is actually being onboarded.

## Consequences

- Push notifications remain effectively single-tenant on OneSignal:
  every `io.opencrafts.*` service can already send push (no
  registration gate, unlike the pre-ADR-0003 email path), but all of
  them deliver through the one configured OneSignal app.
  `notification.app_id` in a published message continues to be silently
  ignored, matching current documented behavior — no consumer-facing
  change results from this ADR.
  - Note: this is orthogonal to the FK removal in ADR-0001 — a
    `target_user_id`/`source_user_id` no longer needing to be a known
    Verisafe user does not change which OneSignal app receives the
    request.
- When a second OneSignal project does appear, the migration path is a
  small config/lookup addition, not a schema change (`app_id` is already
  there) — the cost of deferring this is low.