ALTER TABLE outbox_messages
    ADD COLUMN status     VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    ADD COLUMN attempts   INT         NOT NULL DEFAULT 0,
    ADD COLUMN last_error TEXT;

CREATE INDEX idx_outbox_messages_status ON outbox_messages (status);
