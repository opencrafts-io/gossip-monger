-- +goose Up
-- +goose StatementBegin
select 'up SQL query'
;
DROP INDEX IF EXISTS email_recipients_email_id_idx;
DROP INDEX IF EXISTS email_recipients_recipient_id_idx;
DROP TABLE IF EXISTS email_recipients;
DROP TYPE IF EXISTS recipient_type;

DROP INDEX IF EXISTS emails_external_id_idx;
DROP TABLE IF EXISTS emails;
DROP TYPE IF EXISTS sent_status;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
select 'down SQL query'
;
CREATE TYPE sent_status AS ENUM (
  'pending',
  'scheduled',
  'delivered',
  'failed'
);


CREATE TABLE emails (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  external_id UUID, -- External ID from the email service,

-- Email information
  subject VARCHAR(255),
  body TEXT, 

-- Reply information
  reply_to VARCHAR(255), -- The email to reply to

-- Source information
  source_service_id VARCHAR(100), -- The service that triggered the email
  source_user_id UUID, -- The user who sent the email
  source_user_email VARCHAR(255),

-- Schedule
  schedule_for TIMESTAMP, -- Schedule the email to be sent at a specific time

-- Delivery
  delivered_at TIMESTAMP,
  delivery_status sent_status NOT NULL DEFAULT 'pending',

-- Creation update delete
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS emails_external_id_idx ON emails(external_id);


CREATE TYPE recipient_type AS ENUM (
  'cc',
  'bcc',
  'primary'
);


CREATE TABLE IF NOT EXISTS email_recipients (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email_id UUID NOT NULL,
  recipient_id UUID NOT NULL,
  recieve_type recipient_type NOT NULL DEFAULT 'primary',
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

  FOREIGN KEY(email_id) REFERENCES emails(id)
);


CREATE INDEX IF NOT EXISTS email_recipients_email_id_idx ON email_recipients(email_id);
CREATE INDEX IF NOT EXISTS email_recipients_recipient_id_idx ON email_recipients(recipient_id);

-- +goose StatementEnd
