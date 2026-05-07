package dto

// RepositoryResponse represents a tracked GitHub repository.
type RepositoryResponse struct {
	ID          int64  `json:"id"            example:"101"`           // Internal unique identifier of the repository.
	FullName    string `json:"full_name"     example:"gin-gonic/gin"` // Full name of the repository in "owner/repo" format.
	LastSeenTag string `json:"last_seen_tag" example:"v1.9.1"`        // The most recent tag processed by the system.
}
