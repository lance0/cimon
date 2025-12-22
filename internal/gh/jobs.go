package gh

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// FetchJobs fetches all jobs for a workflow run.
func (c *Client) FetchJobs(owner, repo string, runID int64) ([]Job, error) {
	path := fmt.Sprintf("repos/%s/%s/actions/runs/%d/jobs?per_page=100",
		url.PathEscape(owner),
		url.PathEscape(repo),
		runID,
	)

	var response JobsResponse
	if err := c.Get(path, &response); err != nil {
		return nil, err
	}

	return response.Jobs, nil
}

// FetchJobDetails fetches detailed information for a specific job including steps.
func (c *Client) FetchJobDetails(owner, repo string, jobID int64) (*Job, error) {
	path := fmt.Sprintf("repos/%s/%s/actions/jobs/%d",
		url.PathEscape(owner),
		url.PathEscape(repo),
		jobID,
	)

	var job Job
	if err := c.Get(path, &job); err != nil {
		return nil, err
	}

	return &job, nil
}

// FetchJobLogs fetches and extracts the logs for a specific job.
// Returns the combined log text from all log files in the ZIP.
func (c *Client) FetchJobLogs(owner, repo string, jobID int64) (string, error) {
	path := fmt.Sprintf("repos/%s/%s/actions/jobs/%d/logs",
		url.PathEscape(owner),
		url.PathEscape(repo),
		jobID,
	)

	// Get the redirect URL for the logs ZIP file
	resp, err := c.getRawResponse(path)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Follow the redirect to get the ZIP file
	if resp.StatusCode == http.StatusFound {
		redirectURL := resp.Header.Get("Location")
		if redirectURL == "" {
			return "", fmt.Errorf("no redirect URL found for logs")
		}

		// Download the ZIP file
		resp, err := http.Get(redirectURL)
		if err != nil {
			return "", fmt.Errorf("failed to download logs ZIP: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("failed to download logs ZIP: status %d", resp.StatusCode)
		}

		// Read the ZIP content
		zipData, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to read ZIP data: %w", err)
		}

		// Extract and combine all text files from the ZIP
		return extractLogsFromZIP(zipData)
	}

	return "", fmt.Errorf("unexpected response status: %d", resp.StatusCode)
}

// getRawResponse performs a GET request and returns the raw HTTP response
func (c *Client) getRawResponse(path string) (*http.Response, error) {
	fullURL := fmt.Sprintf("https://api.github.com/%s", path)

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, err
	}

	// Add authentication header
	if c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.authToken)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	// Use a client with timeout
	client := &http.Client{
		Timeout: 60 * time.Second, // 60 second timeout for large file downloads
	}

	return client.Do(req)
}

// extractLogsFromZIP extracts and combines all text files from a ZIP archive
func extractLogsFromZIP(zipData []byte) (string, error) {
	zipReader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return "", fmt.Errorf("failed to read ZIP: %w", err)
	}

	var logs strings.Builder

	for _, file := range zipReader.File {
		if file.FileInfo().IsDir() {
			continue
		}

		// Open the file in the ZIP
		rc, err := file.Open()
		if err != nil {
			continue // Skip files we can't open
		}

		// Read the file content
		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			continue // Skip files we can't read
		}

		// Add a header for each file
		logs.WriteString(fmt.Sprintf("=== %s ===\n", file.Name))
		logs.Write(content)
		logs.WriteString("\n\n")
	}

	return logs.String(), nil
}
