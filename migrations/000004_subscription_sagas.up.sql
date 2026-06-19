CREATE TABLE subscription_sagas
(
    id              BIGSERIAL PRIMARY KEY,
    subscription_id BIGINT      NOT NULL,
    status          VARCHAR(50) NOT NULL DEFAULT 'STARTED',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);