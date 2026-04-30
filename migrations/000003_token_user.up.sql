ALTER TABLE subscriptions ADD COLUMN token TEXT UNIQUE;
ALTER TABLE subscriptions ADD COLUMN is_confirmed BOOLEAN DEFAULT FALSE;

CREATE INDEX idx_subscriptions_token ON subscriptions(token);