CREATE TABLE IF NOT EXISTS subscriptions (
    id            BIGSERIAL PRIMARY KEY,
    user_id       BIGINT NOT NULL,
    repository_id BIGINT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    email         VARCHAR(255) NOT NULL,
    token         VARCHAR(255) NOT NULL,
    is_confirmed  BOOLEAN NOT NULL DEFAULT FALSE
);
