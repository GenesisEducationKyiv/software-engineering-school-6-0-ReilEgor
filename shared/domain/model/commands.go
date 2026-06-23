package model

type SendNotificationCommand struct {
	Email    string `json:"email"`
	RepoName string `json:"repo_name"`
	Tag      string `json:"tag"`
	Token    string `json:"token"`
}

type SendConfirmationCommand struct {
	Email          string `json:"email"`
	RepoName       string `json:"repo_name"`
	Token          string `json:"token"`
	SagaID         int64  `json:"saga_id"`
	SubscriptionID int64  `json:"subscription_id"`
}

type ConfirmationResultEvent struct {
	SagaID         int64  `json:"saga_id"`
	SubscriptionID int64  `json:"subscription_id"`
	Success        bool   `json:"success"`
	Email          string `json:"email"`
	RepoName       string `json:"repo_name"`
	Token          string `json:"token"`
}

type SubscriptionActivatedEvent struct {
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Token    string `json:"token"`
}

type UnsubscriptionActivatedEvent struct {
	Email    string `json:"email"`
	RepoName string `json:"repo_name"`
}

type TagUpdatedEvent struct {
	FullName string `json:"full_name"`
	Tag      string `json:"tag"`
}
