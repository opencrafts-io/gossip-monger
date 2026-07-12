-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN
        SELECT tc.constraint_name
        FROM information_schema.table_constraints tc
        JOIN information_schema.key_column_usage kcu
          ON tc.constraint_name = kcu.constraint_name
         AND tc.table_schema = kcu.table_schema
        WHERE tc.table_name = 'notifications'
          AND tc.constraint_type = 'FOREIGN KEY'
          AND kcu.column_name IN ('target_user_id', 'source_user_id')
    LOOP
        EXECUTE format('ALTER TABLE notifications DROP CONSTRAINT %I', r.constraint_name);
    END LOOP;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd

ALTER TABLE notifications
  ADD CONSTRAINT notifications_target_user_id_fkey
  FOREIGN KEY (target_user_id) REFERENCES users(id) ON DELETE SET NULL NOT VALID;

ALTER TABLE notifications
  ADD CONSTRAINT notifications_source_user_id_fkey
  FOREIGN KEY (source_user_id) REFERENCES users(id) ON DELETE SET NULL NOT VALID;
