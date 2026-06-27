CREATE TABLE outbox_messages
(
    id         BIGSERIAL    PRIMARY KEY,
    queue      VARCHAR(255) NOT NULL,
    payload    JSONB        NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_outbox_messages_created_at ON outbox_messages (created_at);
