-- name: CreateEmailRequest :one
-- Persists an email request to the database for replayability
INSERT INTO email_requests (
  service_id, 
  queue_message_id,
  exchange,
  routing_key,

  from_address,
  reply_to,
  to_addresses,
  cc_addresses,
  bcc_addresses,
  subject,
  body_html,
  body_text,
  attachments,

  template_id,
  template_vars,

  status,
  processed_at

) VALUES ( $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
RETURNING *;


-- name: GetEmailRequestByService :many
-- Orders the time it was recieved ie the most previous
SELECT * FROM email_requests WHERE service_id = $1 ORDER BY received_at DESC;

-- name: GetEmailRequestByID :one
SELECT * FROM email_requests WHERE id = $1 LIMIT 1;


-- name: CreateEmailDispatch :one
-- Records an email dispatch to the email sending service for compliance
-- purposes
INSERT INTO email_dispatches(
  email_request_id,
  resend_email_id,
  resend_payload,
  status,
  http_status_code,
  resend_error
) VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;
