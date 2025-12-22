package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lance0/cimon/internal/config"
	"github.com/lance0/cimon/internal/gh"
	"github.com/lance0/cimon/internal/git"
	"github.com/lance0/cimon/internal/tui"
)

// Build variables (set by goreleaser)
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	os.Exit(run())
}

func run() int {
	// Parse CLI flags
	cfg, err := config.Parse(os.Args[1:])
	if err != nil {
		if err == config.ErrHelp {
			return 0
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}

	// Handle --version
	if cfg.Version {
		fmt.Printf("cimon %s (%s) built %s\n", version, commit, date)
		return 0
	}

	// Create GitHub client (may be needed for detached HEAD resolution)
	var client *gh.Client

	// Resolve repo and branch from git
	if err := cfg.Resolve(); err != nil {
		if err == config.ErrDetachedHead {
			// In detached HEAD state, we need to resolve the default branch
			// First create client to get repository info
			client, clientErr := gh.NewClient()
			if clientErr != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				return 2
			}

			// Get repository info (should be resolved by now)
			cwd, cwdErr := os.Getwd()
			if cwdErr != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				return 2
			}

			repoInfo, repoErr := git.GetRepoInfo(cwd)
			if repoErr != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				return 2
			}

			cfg.Owner = repoInfo.Owner
			cfg.Repo = repoInfo.Repo

			// Get default branch from GitHub
			repo, repoErr := client.GetRepository(cfg.Owner, cfg.Repo)
			if repoErr != nil {
				fmt.Fprintf(os.Stderr, "Error: detached HEAD - could not determine default branch: %v\n", repoErr)
				return 2
			}

			cfg.Branch = repo.DefaultBranch
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 2
		}
	}

	// Create GitHub client if not already created for detached HEAD
	if client == nil {
		var err error
		client, err = gh.NewClient()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 2
		}
	}

	// Handle output modes
	if cfg.Plain && cfg.Json {
		fmt.Fprintf(os.Stderr, "Error: cannot use both --plain and --json flags\n")
		return 2
	}
	if cfg.Plain {
		return runPlain(cfg, client)
	}
	if cfg.Json {
		return runJson(cfg, client)
	}

	// Create and run TUI
	model := tui.NewModel(cfg, client)
	p := tea.NewProgram(model, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		return 2
	}

	// Return exit code based on run status
	if m, ok := finalModel.(tui.Model); ok {
		return m.ExitCode()
	}

	return 0
}

// runPlain runs in plain text mode, fetching and displaying data synchronously
func runPlain(cfg *config.Config, client *gh.Client) int {
	// Fetch latest run
	run, err := client.FetchLatestRun(cfg.Owner, cfg.Repo, cfg.Branch)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching latest run: %v\n", err)
		return 2
	}

	// Fetch jobs if run exists
	var jobs []gh.Job
	if run != nil {
		jobs, err = client.FetchJobs(cfg.Owner, cfg.Repo, run.ID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching jobs: %v\n", err)
			return 2
		}
	}

	// Output plain text
	outputPlain(cfg, run, jobs)

	// Return exit code based on run status
	if run == nil {
		return 2
	}
	if run.IsSuccess() {
		return 0
	} else if run.IsFailure() {
		return 1
	}
	return 0
}

// runJson runs in JSON mode, fetching and displaying data synchronously
func runJson(cfg *config.Config, client *gh.Client) int {
	// Fetch latest run
	run, err := client.FetchLatestRun(cfg.Owner, cfg.Repo, cfg.Branch)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching latest run: %v\n", err)
		return 2
	}

	// Fetch jobs if run exists
	var jobs []gh.Job
	if run != nil {
		jobs, err = client.FetchJobs(cfg.Owner, cfg.Repo, run.ID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching jobs: %v\n", err)
			return 2
		}
	}

	// Output JSON
	outputJson(cfg, run, jobs)

	// Return exit code based on run status
	if run == nil {
		return 2
	}
	if run.IsSuccess() {
		return 0
	} else if run.IsFailure() {
		return 1
	}
	return 0
}

// outputPlain outputs run and job information in plain text format
func outputPlain(cfg *config.Config, run *gh.WorkflowRun, jobs []gh.Job) {
	fmt.Printf("Repository: %s\n", cfg.RepoSlug())
	fmt.Printf("Branch: %s\n", cfg.Branch)
	fmt.Println()

	if run == nil {
		fmt.Println("No workflow runs found")
		return
	}

	// Run information
	fmt.Printf("Run #%d: %s\n", run.RunNumber, run.Name)
	fmt.Printf("Status: %s", run.Status)
	if run.Conclusion != nil {
		fmt.Printf(" (%s)", *run.Conclusion)
	}
	fmt.Println()
	fmt.Printf("Event: %s\n", run.Event)
	if run.Actor != nil {
		fmt.Printf("Triggered by: %s\n", run.Actor.Login)
	}
	fmt.Printf("Created: %s\n", run.CreatedAt.Format("2006-01-02 15:04:05"))
	if run.Status == gh.StatusCompleted {
		fmt.Printf("Updated: %s\n", run.UpdatedAt.Format("2006-01-02 15:04:05"))
	}
	fmt.Printf("URL: %s\n", run.HTMLURL)
	fmt.Println()

	// Jobs
	if len(jobs) == 0 {
		fmt.Println("No jobs found")
		return
	}

	fmt.Printf("Jobs (%d):\n", len(jobs))
	for _, job := range jobs {
		fmt.Printf("  %s: %s", job.Name, job.Status)
		if job.Conclusion != nil {
			fmt.Printf(" (%s)", *job.Conclusion)
		}
		if job.IsCompleted() && job.Duration() > 0 {
			fmt.Printf(" - %s", formatDuration(job.Duration()))
		}
		fmt.Println()
	}
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}

// JsonOutput represents the JSON structure for cimon output
type JsonOutput struct {
	Repository string          `json:"repository"`
	Branch     string          `json:"branch"`
	Run        *gh.WorkflowRun `json:"run,omitempty"`
	Jobs       []gh.Job        `json:"jobs,omitempty"`
	Error      string          `json:"error,omitempty"`
}

// outputJson outputs run and job information in JSON format
func outputJson(cfg *config.Config, run *gh.WorkflowRun, jobs []gh.Job) {
	output := JsonOutput{
		Repository: cfg.RepoSlug(),
		Branch:     cfg.Branch,
		Run:        run,
		Jobs:       jobs,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
	}
}
