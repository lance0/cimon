package gh

import (
	"fmt"
	"net/url"
)

// FetchLatestRun fetches the most recent workflow run for a branch.
// Returns ErrNoRuns if no runs are found.
func (c *Client) FetchLatestRun(owner, repo, branch string) (*WorkflowRun, error) {
	runs, err := c.FetchWorkflowRuns(owner, repo, branch, "", 1, 1)
	if err != nil {
		return nil, err
	}

	if len(runs) == 0 {
		return nil, ErrNoRuns
	}

	return &runs[0], nil
}

// FetchWorkflowRuns fetches workflow runs with pagination and optional filtering.
func (c *Client) FetchWorkflowRuns(owner, repo, branch, status string, page, perPage int) ([]WorkflowRun, error) {
	path := fmt.Sprintf("repos/%s/%s/actions/runs?page=%d&per_page=%d",
		url.PathEscape(owner),
		url.PathEscape(repo),
		page,
		perPage,
	)

	// Add branch filter if specified
	if branch != "" {
		path += "&branch=" + url.QueryEscape(branch)
	}

	// Add status filter if specified
	if status != "" {
		path += "&status=" + url.QueryEscape(status)
	}

	var response WorkflowRunsResponse
	if err := c.Get(path, &response); err != nil {
		return nil, err
	}

	return response.WorkflowRuns, nil
}

// FetchRun fetches a specific workflow run by ID.
func (c *Client) FetchRun(owner, repo string, runID int64) (*WorkflowRun, error) {
	path := fmt.Sprintf("repos/%s/%s/actions/runs/%d",
		url.PathEscape(owner),
		url.PathEscape(repo),
		runID,
	)

	var run WorkflowRun
	if err := c.Get(path, &run); err != nil {
		return nil, err
	}

	return &run, nil
}
