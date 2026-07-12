-- name: UpsertService :one
-- Registers a service on first use so email onboarding is self-service;
-- ON CONFLICT DO UPDATE (a no-op) instead of DO NOTHING so RETURNING always
-- yields exactly one row, whether the service already existed or not.
INSERT INTO services (id, name)
VALUES ($1, $2)
ON CONFLICT (id) DO UPDATE SET is_active = services.is_active
RETURNING *;
