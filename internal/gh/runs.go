package gh

import (
	"fmt"
	"net/url"
)

// FetchLatestRun fetches the most recent workflow run for a branch.
// Returns ErrNoRuns if no runs are found.
func (c *Client) FetchLatestRun(owner, repo, branch string) (*WorkflowRun, error) {
	path := fmt.Sprintf("repos/%s/%s/actions/runs?branch=%s&per_page=1",
		url.PathEscape(owner),
		url.PathEscape(repo),
		url.QueryEscape(branch),
	)

	var response WorkflowRunsResponse
	if err := c.Get(path, &response); err != nil {
		return nil, err
	}

	if len(response.WorkflowRuns) == 0 {
		return nil, ErrNoRuns
	}

	return &response.WorkflowRuns[0], nil
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
