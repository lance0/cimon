package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/lance0/cimon/internal/git"
	"github.com/spf13/pflag"
)

// ErrHelp is returned when --help is requested
var ErrHelp = pflag.ErrHelp

// RepoSpec represents a single repository specification (v0.8)
type RepoSpec struct {
	Owner  string
	Repo   string
	Branch string // Optional: if empty, fetch all branches
}

// Slug returns "owner/repo" format
func (r *RepoSpec) Slug() string {
	return r.Owner + "/" + r.Repo
}

// Config holds all runtime configuration for cimon
type Config struct {
	Owner        string
	Repo         string
	Branch       string
	Watch        bool
	Poll         time.Duration
	NoColor      bool
	Plain        bool
	Json         bool
	Version      bool
	Notify       bool       // v0.7 - Enable desktop notifications on completion
	Hook         string     // v0.7 - Path to hook script to execute on completion
	Repositories []RepoSpec // v0.8 - Multiple repos for multi-repo mode
}

// IsMultiRepo returns true if multiple repos are configured (v0.8)
func (c *Config) IsMultiRepo() bool {
	return len(c.Repositories) > 1
}

// Default values
const (
	DefaultPollInterval = 5 * time.Second
)

var (
	// ErrNoRepo is returned when repo cannot be determined
	ErrNoRepo = errors.New("could not determine repository")

	// ErrNoBranch is returned when branch cannot be determined
	ErrNoBranch = errors.New("could not determine branch")

	// ErrDetachedHead is returned when in detached HEAD state
	ErrDetachedHead = errors.New("detached HEAD - will use default branch")
)

// Parse parses command-line flags and resolves configuration.
// It auto-detects repo and branch from git if not specified.
func Parse(args []string) (*Config, error) {
	cfg := &Config{}

	fs := pflag.NewFlagSet("cimon", pflag.ContinueOnError)

	var repoFlag string
	var reposFlag string
	fs.StringVarP(&repoFlag, "repo", "r", "", "Repository in owner/name format")
	fs.StringVar(&reposFlag, "repos", "", "Comma-separated repos for multi-repo mode (owner/repo1,owner/repo2)")
	fs.StringVarP(&cfg.Branch, "branch", "b", "", "Branch name")
	fs.BoolVarP(&cfg.Watch, "watch", "w", false, "Watch mode - poll until completion")
	fs.DurationVarP(&cfg.Poll, "poll", "p", DefaultPollInterval, "Poll interval for watch mode")
	fs.BoolVar(&cfg.NoColor, "no-color", false, "Disable color output")
	fs.BoolVar(&cfg.Plain, "plain", false, "Plain text output (no TUI)")
	fs.BoolVar(&cfg.Json, "json", false, "JSON output for scripting")
	fs.BoolVarP(&cfg.Version, "version", "v", false, "Show version")
	fs.BoolVar(&cfg.Notify, "notify", false, "Show desktop notification on completion (watch mode)")
	fs.StringVar(&cfg.Hook, "hook", "", "Run script on completion with env vars (watch mode)")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	// Handle --repos flag (v0.8 multi-repo mode)
	if reposFlag != "" {
		specs, err := ParseReposFlag(reposFlag)
		if err != nil {
			return nil, err
		}
		cfg.Repositories = specs
	}

	// Handle --repo flag (single repo mode)
	if repoFlag != "" {
		parts := strings.SplitN(repoFlag, "/", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("invalid repo format %q: expected owner/name", repoFlag)
		}
		cfg.Owner = parts[0]
		cfg.Repo = parts[1]
	}

	return cfg, nil
}

// ParseReposFlag parses the --repos flag into RepoSpec slice (v0.8)
func ParseReposFlag(flag string) ([]RepoSpec, error) {
	if flag == "" {
		return nil, nil
	}

	repos := strings.Split(flag, ",")
	var specs []RepoSpec

	for _, r := range repos {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		parts := strings.SplitN(r, "/", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("invalid repo format %q: expected owner/repo", r)
		}
		specs = append(specs, RepoSpec{Owner: parts[0], Repo: parts[1]})
	}

	return specs, nil
}

// Resolve fills in missing Owner, Repo, and Branch from git.
// Should be called after Parse.
func (c *Config) Resolve() error {
	// Get current directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get working directory: %w", err)
	}

	// Resolve repo if not specified
	if c.Owner == "" || c.Repo == "" {
		info, err := git.GetRepoInfo(cwd)
		if err != nil {
			return fmt.Errorf("%w: %v\nRun inside a git repo or pass --repo owner/name", ErrNoRepo, err)
		}
		c.Owner = info.Owner
		c.Repo = info.Repo
	}

	// Resolve branch if not specified
	if c.Branch == "" {
		branch, err := git.GetBranch(cwd)
		if err != nil {
			// If in detached HEAD state, we'll handle it after client creation
			if err == git.ErrDetachedHead {
				return ErrDetachedHead
			}
			return fmt.Errorf("%w: %v", ErrNoBranch, err)
		}
		c.Branch = branch
	}

	return nil
}

// RepoSlug returns the owner/repo format
func (c *Config) RepoSlug() string {
	return c.Owner + "/" + c.Repo
}
