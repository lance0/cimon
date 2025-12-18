package git

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

var (
	// ErrNotGitRepo is returned when the directory is not a git repository
	ErrNotGitRepo = errors.New("not a git repository")

	// ErrNoRemote is returned when no remote origin is found
	ErrNoRemote = errors.New("no remote origin found")
)

// FindGitRoot walks up from the given directory to find the .git directory.
// Returns the path to the .git directory if found.
func FindGitRoot(startDir string) (string, error) {
	dir := startDir
	for {
		gitPath := filepath.Join(dir, ".git")
		if info, err := os.Stat(gitPath); err == nil {
			if info.IsDir() {
				return gitPath, nil
			}
			// Handle git worktrees where .git is a file
			return gitPath, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			return "", ErrNotGitRepo
		}
		dir = parent
	}
}

// GetRemoteURL reads the git config file and extracts the URL for the
// remote named "origin". If origin doesn't exist, it returns the first
// remote URL found.
func GetRemoteURL(gitDir string) (string, error) {
	configPath := filepath.Join(gitDir, "config")

	file, err := os.Open(configPath)
	if err != nil {
		return "", ErrNotGitRepo
	}
	defer file.Close()

	return parseGitConfig(file)
}

// parseGitConfig parses a git config file and extracts the remote origin URL.
func parseGitConfig(file *os.File) (string, error) {
	scanner := bufio.NewScanner(file)

	var inRemoteOrigin bool
	var inAnyRemote bool
	var firstRemoteURL string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Check for section headers
		if strings.HasPrefix(line, "[") {
			inRemoteOrigin = line == `[remote "origin"]`
			inAnyRemote = strings.HasPrefix(line, `[remote "`)
			continue
		}

		// Look for url = ... lines
		if strings.HasPrefix(line, "url") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				url := strings.TrimSpace(parts[1])
				if inRemoteOrigin {
					return url, nil
				}
				if inAnyRemote && firstRemoteURL == "" {
					firstRemoteURL = url
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	// Fall back to first remote if origin not found
	if firstRemoteURL != "" {
		return firstRemoteURL, nil
	}

	return "", ErrNoRemote
}

// GetRepoInfo finds the git root and parses the remote URL to get owner/repo.
func GetRepoInfo(startDir string) (RepoInfo, error) {
	gitDir, err := FindGitRoot(startDir)
	if err != nil {
		return RepoInfo{}, err
	}

	url, err := GetRemoteURL(gitDir)
	if err != nil {
		return RepoInfo{}, err
	}

	return ParseGitHubURL(url)
}
