package gh

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
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
	defer func() { _ = resp.Body.Close() }()

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
		defer func() { _ = resp.Body.Close() }()

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
	parsed, err := extractLogsFromZIPStructured(zipData)
	if err != nil {
		return "", err
	}
	return parsed.Combined, nil
}

// extractLogsFromZIPStructured extracts logs with step-level structure preserved (v0.6)
// GitHub Actions log ZIP files have format: "{step_number}_{step_name}.txt"
// e.g., "1_Set up job.txt", "2_Checkout.txt", "3_Build.txt"
func extractLogsFromZIPStructured(zipData []byte) (*ParsedLogs, error) {
	zipReader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("failed to read ZIP: %w", err)
	}

	parsed := &ParsedLogs{
		Steps:      []StepLog{},
		StepsByKey: make(map[string]string),
	}

	// Regex to parse step filename: "number_name.txt" or just "name.txt"
	stepPattern := regexp.MustCompile(`^(\d+)_(.+)\.txt$`)

	// Collect all files first so we can sort them
	type fileEntry struct {
		number  int
		name    string
		key     string
		content string
	}
	var entries []fileEntry

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
		_ = rc.Close()
		if err != nil {
			continue // Skip files we can't read
		}

		// Parse the filename to extract step number and name
		filename := file.Name
		// Handle nested paths (e.g., "job_name/1_step.txt")
		if idx := strings.LastIndex(filename, "/"); idx >= 0 {
			filename = filename[idx+1:]
		}

		var stepNum int
		var stepName string

		if matches := stepPattern.FindStringSubmatch(filename); matches != nil {
			stepNum, _ = strconv.Atoi(matches[1])
			stepName = matches[2]
		} else {
			// Fallback: use filename without extension as name, 0 as number
			stepName = strings.TrimSuffix(filename, ".txt")
			stepNum = 0
		}

		key := fmt.Sprintf("%d_%s", stepNum, stepName)
		entries = append(entries, fileEntry{
			number:  stepNum,
			name:    stepName,
			key:     key,
			content: string(content),
		})
	}

	// Sort by step number
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].number < entries[j].number
	})

	// Build the parsed result
	var combined strings.Builder
	for _, entry := range entries {
		step := StepLog{
			Number:  entry.number,
			Name:    entry.name,
			Content: entry.content,
		}
		parsed.Steps = append(parsed.Steps, step)
		parsed.StepsByKey[entry.key] = entry.content

		// Build combined output
		combined.WriteString(fmt.Sprintf("=== %s ===\n", entry.key))
		combined.WriteString(entry.content)
		combined.WriteString("\n\n")
	}

	parsed.Combined = combined.String()
	return parsed, nil
}

// FetchJobLogsStructured fetches logs with step-level structure (v0.6)
func (c *Client) FetchJobLogsStructured(owner, repo string, jobID int64) (*ParsedLogs, error) {
	path := fmt.Sprintf("repos/%s/%s/actions/jobs/%d/logs",
		url.PathEscape(owner),
		url.PathEscape(repo),
		jobID,
	)

	// Get the redirect URL for the logs ZIP file
	resp, err := c.getRawResponse(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	// Follow the redirect to get the ZIP file
	if resp.StatusCode == http.StatusFound {
		redirectURL := resp.Header.Get("Location")
		if redirectURL == "" {
			return nil, fmt.Errorf("no redirect URL found for logs")
		}

		// Download the ZIP file
		resp, err := http.Get(redirectURL)
		if err != nil {
			return nil, fmt.Errorf("failed to download logs ZIP: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to download logs ZIP: status %d", resp.StatusCode)
		}

		// Read the ZIP content
		zipData, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read ZIP data: %w", err)
		}

		// Extract with structure preserved
		return extractLogsFromZIPStructured(zipData)
	}

	return nil, fmt.Errorf("unexpected response status: %d", resp.StatusCode)
}
