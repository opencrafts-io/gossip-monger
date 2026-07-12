-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';
-- +goose StatementEnd

ALTER TABLE notifications ADD COLUMN queue_message_id TEXT;
ALTER TABLE notifications ADD CONSTRAINT notifications_queue_message_id_key UNIQUE (queue_message_id);

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd

ALTER TABLE notifications DROP CONSTRAINT IF EXISTS notifications_queue_message_id_key;
ALTER TABLE notifications DROP COLUMN IF EXISTS queue_message_id;
