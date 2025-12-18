package git

import (
	"errors"
	"regexp"
	"strings"
)

// RepoInfo contains the owner and repo name extracted from a GitHub URL
type RepoInfo struct {
	Owner string
	Repo  string
}

var (
	// ErrInvalidURL is returned when the URL cannot be parsed
	ErrInvalidURL = errors.New("invalid GitHub URL")

	// SSH format: git@github.com:owner/repo.git
	sshPattern = regexp.MustCompile(`^git@github\.com:([^/]+)/([^/]+?)(?:\.git)?$`)

	// HTTPS format: https://github.com/owner/repo.git or https://github.com/owner/repo
	httpsPattern = regexp.MustCompile(`^https?://github\.com/([^/]+)/([^/]+?)(?:\.git)?/?$`)
)

// ParseGitHubURL extracts owner and repo from a GitHub remote URL.
// Supports both SSH (git@github.com:owner/repo.git) and HTTPS
// (https://github.com/owner/repo.git or https://github.com/owner/repo) formats.
func ParseGitHubURL(url string) (RepoInfo, error) {
	url = strings.TrimSpace(url)

	// Try SSH format first
	if matches := sshPattern.FindStringSubmatch(url); matches != nil {
		return RepoInfo{
			Owner: matches[1],
			Repo:  matches[2],
		}, nil
	}

	// Try HTTPS format
	if matches := httpsPattern.FindStringSubmatch(url); matches != nil {
		return RepoInfo{
			Owner: matches[1],
			Repo:  matches[2],
		}, nil
	}

	return RepoInfo{}, ErrInvalidURL
}
