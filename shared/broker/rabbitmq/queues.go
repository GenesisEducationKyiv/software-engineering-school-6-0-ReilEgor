package rabbitmq

const (
	QueueNotifications           = "notifications"
	QueueConfirmations           = "confirmations"
	QueueSagaConfirmationResults = "saga.confirmation.results"
	QueueSubscriptionActivated   = "subscription.activated"
	QueueTagUpdated              = "repository.tag_updated"
	QueueUnsubscriptionActivated = "unsubscription.activated"
)
