package gh

import (
	"fmt"
	"net/url"
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
