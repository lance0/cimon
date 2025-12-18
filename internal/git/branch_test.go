package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetCurrentBranch(t *testing.T) {
	tests := []struct {
		name       string
		headData   string
		wantBranch string
		wantErr    error
	}{
		{
			name:       "main branch",
			headData:   "ref: refs/heads/main",
			wantBranch: "main",
		},
		{
			name:       "master branch",
			headData:   "ref: refs/heads/master",
			wantBranch: "master",
		},
		{
			name:       "feature branch with slashes",
			headData:   "ref: refs/heads/feature/my-feature",
			wantBranch: "feature/my-feature",
		},
		{
			name:       "branch with newline",
			headData:   "ref: refs/heads/develop\n",
			wantBranch: "develop",
		},
		{
			name:     "detached HEAD (SHA)",
			headData: "abc123def456789012345678901234567890abcd",
			wantErr:  ErrDetachedHead,
		},
		{
			name:     "detached HEAD (uppercase SHA)",
			headData: "ABC123DEF456789012345678901234567890ABCD",
			wantErr:  ErrDetachedHead,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp git directory
			tmpDir := t.TempDir()
			gitDir := filepath.Join(tmpDir, ".git")
			if err := os.Mkdir(gitDir, 0755); err != nil {
				t.Fatalf("failed to create .git dir: %v", err)
			}

			headPath := filepath.Join(gitDir, "HEAD")
			if err := os.WriteFile(headPath, []byte(tt.headData), 0644); err != nil {
				t.Fatalf("failed to write HEAD: %v", err)
			}

			got, err := GetCurrentBranch(gitDir)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("GetCurrentBranch() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if got != tt.wantBranch {
				t.Errorf("GetCurrentBranch() = %q, want %q", got, tt.wantBranch)
			}
		})
	}
}

func TestGetCurrentBranch_MissingHEAD(t *testing.T) {
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}

	// Don't create HEAD file
	_, err := GetCurrentBranch(gitDir)
	if err == nil {
		t.Error("expected error for missing HEAD file")
	}
}

func TestGetBranch(t *testing.T) {
	// Create a temp git repo
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}

	headData := "ref: refs/heads/test-branch"
	if err := os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte(headData), 0644); err != nil {
		t.Fatalf("failed to write HEAD: %v", err)
	}

	branch, err := GetBranch(tmpDir)
	if err != nil {
		t.Fatalf("GetBranch() error: %v", err)
	}

	if branch != "test-branch" {
		t.Errorf("GetBranch() = %q, want %q", branch, "test-branch")
	}
}

func TestIsHexString(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"abc123", true},
		{"ABC123", true},
		{"0123456789abcdef", true},
		{"ghijkl", false},
		{"abc 123", false},
		{"", true}, // empty string is technically valid
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := isHexString(tt.input); got != tt.want {
				t.Errorf("isHexString(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
