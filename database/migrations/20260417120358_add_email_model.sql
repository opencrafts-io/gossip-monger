-- +goose Up
-- +goose StatementBegin
select 'up SQL query'
;
-- +goose StatementEnd
CREATE TABLE services (
    id          VARCHAR(255) PRIMARY KEY,
    name        VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO services (
  id, name, description, is_active
) VALUES
('io.opencrafts.verisafe', 'Verisafe', 'The authentication platform', TRUE),
('io.opencrafts.sherehe', 'Sherehe', 'Events, events events!!!', TRUE),
('io.opencrafts.veribroke', 'Veribroke', 'Everything money', TRUE),
('io.opencrafts.keepup', 'Keep Up', 'Everything money', TRUE),
('io.opencrafts.professor', 'Professor', 'Managing your institutions profile', TRUE);

CREATE TABLE email_requests (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service_id      VARCHAR(255) NOT NULL REFERENCES services(id),
    
    -- Routing
    queue_message_id TEXT NOT NULL UNIQUE, -- RabbitMQ message ID (from delivery tag or header)
    exchange        VARCHAR(100) NOT NULL, -- The exchange it was published on
    routing_key     VARCHAR(100) NOT NULL, -- The routing key that was specified

-- Email content (full compliance record)
    from_address    VARCHAR(255) NOT NULL,
    reply_to        VARCHAR(255),
    to_addresses    TEXT[] NOT NULL,
    cc_addresses    TEXT[],
    bcc_addresses   TEXT[],
    subject         TEXT NOT NULL,
    body_html       TEXT,
    body_text       TEXT,
    attachments     JSONB,         -- [{ filename, content_type, size_bytes }] — metadata only, not blobs

    template_id     VARCHAR(100),
    template_vars   JSONB,

-- Processing state
    status          VARCHAR(50) NOT NULL DEFAULT 'received',
    -- received | processing | dispatched | failed
    received_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at    TIMESTAMPTZ
);


CREATE TABLE email_dispatches (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email_request_id    UUID NOT NULL REFERENCES email_requests(id),

-- Resend reference
    resend_email_id     VARCHAR(255) UNIQUE,  -- ID returned by Resend on success
    
    resend_payload      JSONB NOT NULL,

-- Outcome
    status              VARCHAR(50) NOT NULL,
    -- sent | failed
    http_status_code    INT,
    resend_error        TEXT,       -- raw error message from Resend if failed

    dispatched_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE email_delivery_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dispatch_id UUID NOT NULL REFERENCES email_dispatches(id),
    resend_email_id VARCHAR(255) NOT NULL,

    event_type      VARCHAR(100) NOT NULL,
    -- email.sent | email.delivered | email.bounced | email.complained | email.opened
    -- | email.clicked
    recipient       VARCHAR(255),   -- which recipient this event is for
    raw_payload     JSONB NOT NULL, -- full Resend webhook body for compliance
    occurred_at     TIMESTAMPTZ NOT NULL,
    recorded_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Dashboard: all emails for a service
CREATE INDEX idx_email_requests_service_id ON email_requests(service_id);

-- Dashboard: filter by status and time
CREATE INDEX idx_email_requests_status_received ON email_requests(status, received_at DESC);

-- Join dispatches to requests
CREATE INDEX idx_dispatches_email_request_id ON email_dispatches(email_request_id);

-- Webhook lookup by resend ID
CREATE INDEX idx_delivery_events_resend_email_id ON email_delivery_events(resend_email_id);
CREATE INDEX idx_delivery_events_dispatch_id ON email_delivery_events(dispatch_id);

-- +goose Down
-- +goose StatementBegin
select 'down SQL query'
;
-- +goose StatementEnd
-- Drop indexes first
DROP INDEX IF EXISTS idx_delivery_events_dispatch_id;
DROP INDEX IF EXISTS idx_delivery_events_resend_email_id;
DROP INDEX IF EXISTS idx_dispatches_email_request_id;
DROP INDEX IF EXISTS idx_email_requests_status_received;
DROP INDEX IF EXISTS idx_email_requests_service_id;

DROP TABLE IF EXISTS email_delivery_events;
DROP TABLE IF EXISTS email_dispatches;
DROP TABLE IF EXISTS email_requests;
DROP TABLE IF EXISTS services;
