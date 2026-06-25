package dto

import "time"

// ReleaseInfoResponse represents the information about a GitHub release.
type ReleaseInfoResponse struct {
	TagName     string    `json:"tag_name"     example:"v1.2.3"`                                        // Latest release version tag.
	PublishedAt time.Time `json:"published_at" example:"2026-04-08T13:21:24Z"`                          // Date and time when the release was published.
	URL         string    `json:"url"          example:"https://github.com/owner/repo/releases/v1.2.3"` // Direct link to the release on GitHub.
}
