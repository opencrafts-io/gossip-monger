-- name: CreateEmailRecipient :one
INSERT INTO email_recipients (
  email_id, recipient_id, recieve_type
) VALUES ( $1, $2, $3)
RETURNING *;


-- name: GetEmailRecipients :many
SELECT id, email_id, recipient_id, recieve_type, created_at, updated_at
FROM email_recipients WHERE email_id = $1
ORDER BY created_at ASC
LIMIT $2
OFFSET $3;
