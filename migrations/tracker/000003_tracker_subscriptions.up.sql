ALTER TABLE subscriptions ALTER COLUMN user_id DROP NOT NULL;

ALTER TABLE subscriptions ADD CONSTRAINT subscriptions_repository_id_email_key UNIQUE (repository_id, email);
