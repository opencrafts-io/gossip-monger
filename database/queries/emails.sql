-- name: UpsertEmailRequest :one
-- Persists an email request to the database for replayability, or updates
-- it in place if this is a retry of the same queue_message_id (dead-lettered
-- redelivery) — retrying must not hit the queue_message_id UNIQUE
-- constraint as a fresh insert, or the retry mechanism would just fail
-- forever on the second attempt without ever reaching Resend again.
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
ON CONFLICT (queue_message_id) DO UPDATE SET
  status = EXCLUDED.status,
  processed_at = EXCLUDED.processed_at
RETURNING *;


-- name: GetEmailRequestByQueueMessageID :one
-- Used to detect a duplicate send before calling Resend: if a request with
-- this queue_message_id was already dispatched, the caller must skip
-- resending rather than upsert-and-retry, or a legitimate DLX redelivery
-- and an external duplicate republish become indistinguishable and both
-- would trigger a second real send.
select *
from email_requests
where queue_message_id = $1
limit 1
;

-- name: GetEmailRequestByService :many
-- Orders the time it was recieved ie the most previous
select *
from email_requests
where service_id = $1
order by received_at desc
limit $2 offset $3
;

-- name: GetEmailRequestByID :one
select *
from email_requests
where id = $1
limit 1
;

-- name: UpdateEmailRequestStatusByID :one
-- Updates an email_request record effectively setting its status to one of
-- the predefined statuses
UPDATE email_requests
  SET status = $2
  WHERE id = $1
RETURNING *;


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
