package gh

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
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

// Get performs a GET request to the GitHub API with retry logic
func (c *Client) Get(path string, response interface{}) error {
	config := DefaultRetryConfig()
	return RetryWithBackoff(func() error {
		err := c.rest.Get(path, response)
		if err != nil {
			return c.wrapError(err)
		}
		return nil
	}, config)
}

// Post performs a POST request to the GitHub API with retry logic
func (c *Client) Post(path string, payload interface{}) error {
	config := DefaultRetryConfig()
	return RetryWithBackoff(func() error {
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
	}, config)
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

// wrapError converts API errors to our custom error types with retry logic
func (c *Client) wrapError(err error) error {
	if err == nil {
		return nil
	}

	errStr := err.Error()

	// Check for HTTP status codes in error message
	if strings.Contains(errStr, "401") {
		return &AuthError{Err: fmt.Errorf("authentication failed: please check your GitHub token or run 'gh auth login': %w", err)}
	}

	if strings.Contains(errStr, "403") {
		if strings.Contains(errStr, "rate limit") {
			return &RateLimitError{Err: fmt.Errorf("rate limit exceeded: please wait before retrying: %w", err)}
		}
		return &AuthError{Err: fmt.Errorf("access forbidden: please check repository permissions: %w", err)}
	}

	if strings.Contains(errStr, "404") {
		return &NotFoundError{Resource: "resource", Err: fmt.Errorf("resource not found: please check repository and branch names: %w", err)}
	}

	if strings.Contains(errStr, "429") {
		return &RateLimitError{Err: fmt.Errorf("too many requests: GitHub API rate limit exceeded: %w", err)}
	}

	// Network and server errors that should be retried
	if strings.Contains(errStr, "502") || strings.Contains(errStr, "503") || strings.Contains(errStr, "504") {
		return fmt.Errorf("server error (will retry): %w", err)
	}

	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "connection") {
		return fmt.Errorf("network error (will retry): %w", err)
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

// FetchWorkflowContent fetches the content of a workflow file
func (c *Client) FetchWorkflowContent(owner, repo, path string) (string, error) {
	encodedPath := url.PathEscape(path)
	apiPath := fmt.Sprintf("repos/%s/%s/contents/%s",
		url.PathEscape(owner),
		url.PathEscape(repo),
		encodedPath,
	)

	var content Content
	if err := c.Get(apiPath, &content); err != nil {
		return "", err
	}

	// Decode base64 content
	if content.Encoding != "base64" {
		return "", fmt.Errorf("unexpected content encoding: %s", content.Encoding)
	}

	// GitHub API returns base64 encoded content
	decoded, err := base64.StdEncoding.DecodeString(content.Content)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 content: %w", err)
	}

	return string(decoded), nil
}

// FetchWorkflowArtifacts fetches artifacts for a specific workflow run
func (c *Client) FetchWorkflowArtifacts(owner, repo string, runID int64) ([]Artifact, error) {
	path := fmt.Sprintf("repos/%s/%s/actions/runs/%d/artifacts",
		url.PathEscape(owner),
		url.PathEscape(repo),
		runID,
	)

	var response ArtifactsResponse
	if err := c.Get(path, &response); err != nil {
		return nil, err
	}

	return response.Artifacts, nil
}

// DownloadArtifact downloads an artifact to the current directory
func (c *Client) DownloadArtifact(owner, repo string, artifactID int64, filename string) error {
	path := fmt.Sprintf("repos/%s/%s/actions/artifacts/%d/zip",
		url.PathEscape(owner),
		url.PathEscape(repo),
		artifactID,
	)

	// Create a temporary file for download
	tempFile, err := os.CreateTemp("", "cimon-artifact-*.zip")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tempFile.Close()
	defer os.Remove(tempFile.Name()) // Clean up temp file

	// Download the artifact
	resp, err := c.getRawResponse(path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	// Copy the response body to temp file
	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to download artifact: %w", err)
	}

	// Close temp file before renaming
	tempFile.Close()

	// Move temp file to final destination
	if err := os.Rename(tempFile.Name(), filename); err != nil {
		return fmt.Errorf("failed to save artifact: %w", err)
	}

	return nil
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
