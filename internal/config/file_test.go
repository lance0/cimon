package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigFile(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantRepos int
		wantErr  bool
	}{
		{
			name: "valid config",
			content: `repositories:
  - owner1/repo1
  - owner2/repo2
`,
			wantRepos: 2,
		},
		{
			name: "empty repos",
			content: `repositories: []
`,
			wantRepos: 0,
		},
		{
			name:     "invalid yaml",
			content:  "invalid: [yaml: content",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			dir := t.TempDir()
			path := filepath.Join(dir, "cimon.yml")
			if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write temp file: %v", err)
			}

			cfg, err := LoadConfigFile(path)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfigFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if len(cfg.Repositories) != tt.wantRepos {
				t.Errorf("LoadConfigFile() got %d repos, want %d", len(cfg.Repositories), tt.wantRepos)
			}
		})
	}
}

func TestLoadConfigFileNotExists(t *testing.T) {
	cfg, err := LoadConfigFile("/nonexistent/cimon.yml")
	if err != nil {
		t.Errorf("LoadConfigFile() error = %v, want nil for missing file", err)
	}
	if cfg != nil {
		t.Error("LoadConfigFile() returned non-nil config for missing file")
	}
}

func TestFileConfigToRepoSpecs(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *FileConfig
		want    []RepoSpec
		wantErr bool
	}{
		{
			name: "valid repos",
			cfg: &FileConfig{
				Repositories: []string{"owner1/repo1", "owner2/repo2"},
			},
			want: []RepoSpec{
				{Owner: "owner1", Repo: "repo1"},
				{Owner: "owner2", Repo: "repo2"},
			},
		},
		{
			name: "nil config",
			cfg:  nil,
			want: nil,
		},
		{
			name: "empty repos",
			cfg:  &FileConfig{Repositories: []string{}},
			want: nil,
		},
		{
			name: "skip empty strings",
			cfg: &FileConfig{
				Repositories: []string{"owner1/repo1", "", "owner2/repo2"},
			},
			want: []RepoSpec{
				{Owner: "owner1", Repo: "repo1"},
				{Owner: "owner2", Repo: "repo2"},
			},
		},
		{
			name: "invalid format",
			cfg: &FileConfig{
				Repositories: []string{"invalid"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.cfg.ToRepoSpecs()
			if (err != nil) != tt.wantErr {
				t.Errorf("ToRepoSpecs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if len(got) != len(tt.want) {
				t.Errorf("ToRepoSpecs() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ToRepoSpecs()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}
