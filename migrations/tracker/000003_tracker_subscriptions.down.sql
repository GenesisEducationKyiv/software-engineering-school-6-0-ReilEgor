ALTER TABLE subscriptions DROP CONSTRAINT IF EXISTS subscriptions_repository_id_email_key;

ALTER TABLE subscriptions ALTER COLUMN user_id SET NOT NULL;
