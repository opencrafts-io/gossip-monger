-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';
-- +goose StatementEnd
DROP INDEX IF EXISTS email_recipients_email_id_idx;
DROP INDEX IF EXISTS email_recipients_recipient_id_idx;
DROP TABLE IF EXISTS email_recipients;
DROP TYPE IF EXISTS recipient_type;

DROP INDEX IF EXISTS emails_external_id_idx;
DROP TABLE IF EXISTS emails;
DROP TYPE IF EXISTS sent_status;

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd
