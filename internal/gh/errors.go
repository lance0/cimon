package gh

import (
	"errors"
	"fmt"
)

var (
	// ErrNotAuthenticated is returned when GitHub authentication fails
	ErrNotAuthenticated = errors.New("not authenticated to GitHub")

	// ErrRepoNotFound is returned when the repository is not found
	ErrRepoNotFound = errors.New("repository not found")

	// ErrRateLimited is returned when GitHub API rate limit is exceeded
	ErrRateLimited = errors.New("GitHub API rate limit exceeded")

	// ErrNoRuns is returned when no workflow runs are found
	ErrNoRuns = errors.New("no workflow runs found for this branch")
)

// AuthError wraps authentication-related errors with helpful messages
type AuthError struct {
	Err error
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("GitHub authentication failed: %v\nInstall gh and run 'gh auth login' or set GITHUB_TOKEN", e.Err)
}

func (e *AuthError) Unwrap() error {
	return e.Err
}

// NotFoundError wraps 404 errors
type NotFoundError struct {
	Resource string
	Err      error
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s not found: %v\nCheck that the repository exists and you have access", e.Resource, e.Err)
}

func (e *NotFoundError) Unwrap() error {
	return e.Err
}

// RateLimitError wraps rate limit errors
type RateLimitError struct {
	Err error
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("GitHub API rate limited: %v\nWait a moment and try again, or authenticate with gh auth login", e.Err)
}

func (e *RateLimitError) Unwrap() error {
	return e.Err
}
