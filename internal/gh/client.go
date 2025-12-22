package gh

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
)

// Client wraps the GitHub REST API client
type Client struct {
	rest *api.RESTClient
}

// NewClient creates a new GitHub API client.
// It tries to use gh CLI authentication first, then falls back to GITHUB_TOKEN.
func NewClient() (*Client, error) {
	// Try go-gh which uses gh CLI auth
	opts := api.ClientOptions{
		EnableCache: false,
	}

	// Check if GITHUB_TOKEN is set as override
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		opts.AuthToken = token
	}

	rest, err := api.NewRESTClient(opts)
	if err != nil {
		return nil, &AuthError{Err: err}
	}

	return &Client{rest: rest}, nil
}

// Get performs a GET request to the GitHub API
func (c *Client) Get(path string, response interface{}) error {
	err := c.rest.Get(path, response)
	if err != nil {
		return c.wrapError(err)
	}
	return nil
}

// Post performs a POST request to the GitHub API
func (c *Client) Post(path string, payload interface{}) error {
	var body bytes.Buffer
	if payload != nil {
		if err := json.NewEncoder(&body).Encode(payload); err != nil {
			return fmt.Errorf("failed to encode payload: %w", err)
		}
	}

	err := c.rest.Post(path, &body, nil)
	if err != nil {
		return c.wrapError(err)
	}
	return nil
}

// GetRepository fetches repository information from GitHub API
func (c *Client) GetRepository(owner, repo string) (*Repository, error) {
	path := fmt.Sprintf("repos/%s/%s",
		url.PathEscape(owner),
		url.PathEscape(repo),
	)

	var repository Repository
	if err := c.Get(path, &repository); err != nil {
		return nil, err
	}

	return &repository, nil
}

// wrapError converts API errors to our custom error types
func (c *Client) wrapError(err error) error {
	if err == nil {
		return nil
	}

	errStr := err.Error()

	// Check for HTTP status codes in error message
	if strings.Contains(errStr, "401") || strings.Contains(errStr, "403") {
		return &AuthError{Err: err}
	}

	if strings.Contains(errStr, "404") {
		return &NotFoundError{Resource: "resource", Err: err}
	}

	if strings.Contains(errStr, "429") || strings.Contains(errStr, "rate limit") {
		return &RateLimitError{Err: err}
	}

	return err
}

// CheckHTTPError checks if an error is an HTTP error with the given status code
func CheckHTTPError(err error, statusCode int) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), fmt.Sprintf("%d", statusCode))
}

// IsHTTPError checks if the error is an HTTP error
func IsHTTPError(err error) bool {
	if err == nil {
		return false
	}

	// Check for common HTTP status codes
	httpCodes := []int{
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
	}

	errStr := err.Error()
	for _, code := range httpCodes {
		if strings.Contains(errStr, fmt.Sprintf("%d", code)) {
			return true
		}
	}

	return false
}
