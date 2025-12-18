package git

import (
	"testing"
)

func TestParseGitHubURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		// SSH format
		{
			name:      "SSH with .git suffix",
			url:       "git@github.com:owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "SSH without .git suffix",
			url:       "git@github.com:owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "SSH with org name",
			url:       "git@github.com:my-org/my-repo.git",
			wantOwner: "my-org",
			wantRepo:  "my-repo",
		},

		// HTTPS format
		{
			name:      "HTTPS with .git suffix",
			url:       "https://github.com/owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "HTTPS without .git suffix",
			url:       "https://github.com/owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "HTTPS with trailing slash",
			url:       "https://github.com/owner/repo/",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "HTTP (not HTTPS)",
			url:       "http://github.com/owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
		},

		// Edge cases
		{
			name:      "URL with whitespace",
			url:       "  git@github.com:owner/repo.git  ",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "Repo name with dots",
			url:       "git@github.com:owner/repo.name.git",
			wantOwner: "owner",
			wantRepo:  "repo.name",
		},
		{
			name:      "Owner with numbers",
			url:       "https://github.com/owner123/repo",
			wantOwner: "owner123",
			wantRepo:  "repo",
		},

		// Invalid URLs
		{
			name:    "Empty string",
			url:     "",
			wantErr: true,
		},
		{
			name:    "Not a GitHub URL",
			url:     "https://gitlab.com/owner/repo",
			wantErr: true,
		},
		{
			name:    "Missing repo",
			url:     "git@github.com:owner",
			wantErr: true,
		},
		{
			name:    "Completely invalid",
			url:     "not a url at all",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseGitHubURL(tt.url)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseGitHubURL(%q) expected error, got nil", tt.url)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseGitHubURL(%q) unexpected error: %v", tt.url, err)
				return
			}

			if got.Owner != tt.wantOwner {
				t.Errorf("ParseGitHubURL(%q) Owner = %q, want %q", tt.url, got.Owner, tt.wantOwner)
			}
			if got.Repo != tt.wantRepo {
				t.Errorf("ParseGitHubURL(%q) Repo = %q, want %q", tt.url, got.Repo, tt.wantRepo)
			}
		})
	}
}
