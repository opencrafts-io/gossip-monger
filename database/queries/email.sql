-- name: CreateEmail :one
INSERT INTO emails (
  external_id,
  subject,
  body,
  reply_to,
  source_service_id,
  source_user_id,
  source_user_email,
  schedule_for,
  delivery_status
)
VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING *;


-- name: GetEmailByExternalID :one
SELECT id, external_id, subject, body, reply_to, source_service_id,
source_user_id, source_user_email, schedule_for, delivered_at, delivery_status,
created_at, updated_at
FROM emails WHERE external_id=$1 LIMIT 1;

-- name: GetEmailByID :one
SELECT id, external_id, subject, body, reply_to, source_service_id,
source_user_id, source_user_email, schedule_for, delivered_at, delivery_status,
created_at, updated_at
FROM emails WHERE id=$1 LIMIT 1;

