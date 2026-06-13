package model

import "errors"

var (
	ErrGitHubUnavailable = errors.New("github service is temporarily unavailable")
	ErrRateLimitExceeded = errors.New("github api rate limit exceeded")
)
