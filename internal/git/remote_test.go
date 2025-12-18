package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindGitRoot(t *testing.T) {
	// Create a temp directory structure
	tmpDir := t.TempDir()

	// Create a nested directory structure
	nestedDir := filepath.Join(tmpDir, "a", "b", "c")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}

	// Create .git directory at root
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}

	tests := []struct {
		name     string
		startDir string
		wantGit  string
		wantErr  bool
	}{
		{
			name:     "from git root",
			startDir: tmpDir,
			wantGit:  gitDir,
		},
		{
			name:     "from nested directory",
			startDir: nestedDir,
			wantGit:  gitDir,
		},
		{
			name:     "from non-git directory",
			startDir: os.TempDir(),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FindGitRoot(tt.startDir)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if got != tt.wantGit {
				t.Errorf("FindGitRoot() = %q, want %q", got, tt.wantGit)
			}
		})
	}
}

func TestGetRemoteURL(t *testing.T) {
	tests := []struct {
		name       string
		configData string
		wantURL    string
		wantErr    bool
	}{
		{
			name: "origin remote",
			configData: `[core]
	repositoryformatversion = 0
[remote "origin"]
	url = git@github.com:owner/repo.git
	fetch = +refs/heads/*:refs/remotes/origin/*
`,
			wantURL: "git@github.com:owner/repo.git",
		},
		{
			name: "origin with https",
			configData: `[remote "origin"]
	url = https://github.com/owner/repo.git
`,
			wantURL: "https://github.com/owner/repo.git",
		},
		{
			name: "multiple remotes with origin",
			configData: `[remote "upstream"]
	url = git@github.com:upstream/repo.git
[remote "origin"]
	url = git@github.com:owner/repo.git
`,
			wantURL: "git@github.com:owner/repo.git",
		},
		{
			name: "fallback to first remote when no origin",
			configData: `[remote "upstream"]
	url = git@github.com:upstream/repo.git
[remote "fork"]
	url = git@github.com:fork/repo.git
`,
			wantURL: "git@github.com:upstream/repo.git",
		},
		{
			name: "no remotes",
			configData: `[core]
	repositoryformatversion = 0
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp git directory with config
			tmpDir := t.TempDir()
			gitDir := filepath.Join(tmpDir, ".git")
			if err := os.Mkdir(gitDir, 0755); err != nil {
				t.Fatalf("failed to create .git dir: %v", err)
			}

			configPath := filepath.Join(gitDir, "config")
			if err := os.WriteFile(configPath, []byte(tt.configData), 0644); err != nil {
				t.Fatalf("failed to write config: %v", err)
			}

			got, err := GetRemoteURL(gitDir)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if got != tt.wantURL {
				t.Errorf("GetRemoteURL() = %q, want %q", got, tt.wantURL)
			}
		})
	}
}

func TestGetRepoInfo(t *testing.T) {
	// Create a temp git repo
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}

	configData := `[remote "origin"]
	url = git@github.com:testowner/testrepo.git
`
	if err := os.WriteFile(filepath.Join(gitDir, "config"), []byte(configData), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	info, err := GetRepoInfo(tmpDir)
	if err != nil {
		t.Fatalf("GetRepoInfo() error: %v", err)
	}

	if info.Owner != "testowner" {
		t.Errorf("Owner = %q, want %q", info.Owner, "testowner")
	}
	if info.Repo != "testrepo" {
		t.Errorf("Repo = %q, want %q", info.Repo, "testrepo")
	}
}
