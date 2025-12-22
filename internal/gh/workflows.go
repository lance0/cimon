package gh

import (
	"fmt"
	"net/url"
)

// RerunWorkflow triggers a rerun of the specified workflow run
func (c *Client) RerunWorkflow(owner, repo string, runID int64) error {
	path := fmt.Sprintf("repos/%s/%s/actions/runs/%d/rerun",
		url.PathEscape(owner),
		url.PathEscape(repo),
		runID,
	)

	// POST request with empty body
	return c.Post(path, nil)
}

// CancelWorkflow cancels the specified workflow run
func (c *Client) CancelWorkflow(owner, repo string, runID int64) error {
	path := fmt.Sprintf("repos/%s/%s/actions/runs/%d/cancel",
		url.PathEscape(owner),
		url.PathEscape(repo),
		runID,
	)

	// POST request with empty body
	return c.Post(path, nil)
}

// DispatchWorkflow triggers a workflow_dispatch event
func (c *Client) DispatchWorkflow(owner, repo, workflowFile, ref string) error {
	path := fmt.Sprintf("repos/%s/%s/actions/workflows/%s/dispatches",
		url.PathEscape(owner),
		url.PathEscape(repo),
		url.PathEscape(workflowFile),
	)

	payload := map[string]interface{}{
		"ref": ref,
	}

	return c.Post(path, payload)
}
