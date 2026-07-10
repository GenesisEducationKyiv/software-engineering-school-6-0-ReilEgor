package model

type SendNotificationCommand struct {
	Email    string `json:"email"`
	RepoName string `json:"repo_name"`
	Tag      string `json:"tag"`
	Token    string `json:"token"`
}

type SendConfirmationCommand struct {
	Email    string `json:"email"`
	RepoName string `json:"repo_name"`
	Token    string `json:"token"`
}
