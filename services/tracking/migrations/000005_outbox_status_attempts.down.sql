DROP INDEX IF EXISTS idx_outbox_messages_status;

ALTER TABLE outbox_messages
    DROP COLUMN status,
    DROP COLUMN attempts,
    DROP COLUMN last_error;
