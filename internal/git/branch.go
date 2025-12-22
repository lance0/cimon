package git

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

var (
	// ErrDetachedHead is returned when HEAD is not on a branch
	ErrDetachedHead = errors.New("detached HEAD state - please checkout a branch or use --branch")
)

// GetCurrentBranch reads the current branch name from .git/HEAD.
// Returns ErrDetachedHead if in detached HEAD state.
func GetCurrentBranch(gitDir string) (string, error) {
	headPath := filepath.Join(gitDir, "HEAD")

	data, err := os.ReadFile(headPath)
	if err != nil {
		return "", ErrNotGitRepo
	}

	content := strings.TrimSpace(string(data))

	// Normal branch reference format: ref: refs/heads/branch-name
	const refPrefix = "ref: refs/heads/"
	if strings.HasPrefix(content, refPrefix) {
		return strings.TrimPrefix(content, refPrefix), nil
	}

	// If it's a commit SHA, we're in detached HEAD state
	// SHA-1 hashes are 40 hex characters
	if len(content) == 40 && isHexString(content) {
		return "", ErrDetachedHead
	}

	// Unknown format
	return "", ErrDetachedHead
}

// isHexString checks if a string contains only hexadecimal characters
func isHexString(s string) bool {
	for _, c := range s {
		isDigit := c >= '0' && c <= '9'
		isLowerHex := c >= 'a' && c <= 'f'
		isUpperHex := c >= 'A' && c <= 'F'
		if !isDigit && !isLowerHex && !isUpperHex {
			return false
		}
	}
	return true
}

// GetBranch is a convenience function that finds the git root and gets the current branch.
func GetBranch(startDir string) (string, error) {
	gitDir, err := FindGitRoot(startDir)
	if err != nil {
		return "", err
	}

	return GetCurrentBranch(gitDir)
}
