package config

import (
	"testing"
)

func TestRepoSpecSlug(t *testing.T) {
	spec := RepoSpec{Owner: "owner", Repo: "repo"}
	if got := spec.Slug(); got != "owner/repo" {
		t.Errorf("Slug() = %q, want %q", got, "owner/repo")
	}
}

func TestParseReposFlag(t *testing.T) {
	tests := []struct {
		name    string
		flag    string
		want    []RepoSpec
		wantErr bool
	}{
		{
			name: "single repo",
			flag: "owner/repo",
			want: []RepoSpec{{Owner: "owner", Repo: "repo"}},
		},
		{
			name: "multiple repos",
			flag: "owner1/repo1,owner2/repo2",
			want: []RepoSpec{
				{Owner: "owner1", Repo: "repo1"},
				{Owner: "owner2", Repo: "repo2"},
			},
		},
		{
			name: "with spaces",
			flag: " owner1/repo1 , owner2/repo2 ",
			want: []RepoSpec{
				{Owner: "owner1", Repo: "repo1"},
				{Owner: "owner2", Repo: "repo2"},
			},
		},
		{
			name: "empty string",
			flag: "",
			want: nil,
		},
		{
			name:    "invalid format - no slash",
			flag:    "invalid",
			wantErr: true,
		},
		{
			name:    "invalid format - empty owner",
			flag:    "/repo",
			wantErr: true,
		},
		{
			name:    "invalid format - empty repo",
			flag:    "owner/",
			wantErr: true,
		},
		{
			name: "skip empty entries",
			flag: "owner1/repo1,,owner2/repo2",
			want: []RepoSpec{
				{Owner: "owner1", Repo: "repo1"},
				{Owner: "owner2", Repo: "repo2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseReposFlag(tt.flag)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseReposFlag() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if len(got) != len(tt.want) {
				t.Errorf("ParseReposFlag() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ParseReposFlag()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestConfigIsMultiRepo(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want bool
	}{
		{
			name: "empty repos",
			cfg:  Config{},
			want: false,
		},
		{
			name: "single repo",
			cfg:  Config{Repositories: []RepoSpec{{Owner: "o", Repo: "r"}}},
			want: false,
		},
		{
			name: "multiple repos",
			cfg: Config{Repositories: []RepoSpec{
				{Owner: "o1", Repo: "r1"},
				{Owner: "o2", Repo: "r2"},
			}},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.IsMultiRepo(); got != tt.want {
				t.Errorf("IsMultiRepo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseWithReposFlag(t *testing.T) {
	args := []string{"--repos", "owner1/repo1,owner2/repo2"}
	cfg, err := Parse(args)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(cfg.Repositories) != 2 {
		t.Errorf("Parse() got %d repos, want 2", len(cfg.Repositories))
	}

	if !cfg.IsMultiRepo() {
		t.Error("Parse() IsMultiRepo() = false, want true")
	}
}
