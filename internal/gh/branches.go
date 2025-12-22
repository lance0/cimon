package gh

import (
	"fmt"
	"net/url"
)

// Branch represents a git branch
type Branch struct {
	Name      string `json:"name"`
	Commit    Commit `json:"commit"`
	Protected bool   `json:"protected"`
}

// Commit represents a commit (simplified for branch listing)
type Commit struct {
	SHA string `json:"sha"`
	URL string `json:"url"`
}

// BranchesResponse is the API response for listing branches
type BranchesResponse []Branch

// FetchBranches fetches all branches for a repository.
func (c *Client) FetchBranches(owner, repo string) ([]Branch, error) {
	path := fmt.Sprintf("repos/%s/%s/branches?per_page=100",
		url.PathEscape(owner),
		url.PathEscape(repo),
	)

	var branches BranchesResponse
	if err := c.Get(path, &branches); err != nil {
		return nil, err
	}

	return branches, nil
}

// FetchBranch fetches information about a specific branch.
func (c *Client) FetchBranch(owner, repo, branch string) (*Branch, error) {
	path := fmt.Sprintf("repos/%s/%s/branches/%s",
		url.PathEscape(owner),
		url.PathEscape(repo),
		url.PathEscape(branch),
	)

	var branchInfo Branch
	if err := c.Get(path, &branchInfo); err != nil {
		return nil, err
	}

	return &branchInfo, nil
}
