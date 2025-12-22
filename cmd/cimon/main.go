package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lance0/cimon/internal/config"
	"github.com/lance0/cimon/internal/gh"
	"github.com/lance0/cimon/internal/git"
	"github.com/lance0/cimon/internal/tui"
	"github.com/spf13/pflag"
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
	args := os.Args[1:]

	// Check for subcommands
	if len(args) > 0 {
		switch args[0] {
		case "retry":
			return runRetry(args[1:])
		case "cancel":
			return runCancel(args[1:])
		case "dispatch":
			return runDispatch(args[1:])
		case "help", "-h", "--help":
			printUsage()
			return 0
		}
	}

	// Parse CLI flags for TUI mode
	cfg, err := config.Parse(args)
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

	// Load config file if no --repos flag (v0.8)
	if len(cfg.Repositories) == 0 {
		fileCfg, fileErr := config.LoadConfigFile(config.DefaultConfigPath())
		if fileErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", fileErr)
		} else if fileCfg != nil {
			specs, specErr := fileCfg.ToRepoSpecs()
			if specErr != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", specErr)
				return 2
			}
			cfg.Repositories = specs
		}
	}

	// Create GitHub client (may be needed for detached HEAD resolution)
	var client *gh.Client

	// Multi-repo mode: skip single-repo resolution (v0.8)
	if cfg.IsMultiRepo() {
		var err error
		client, err = gh.NewClient()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 2
		}
	} else if len(cfg.Repositories) == 1 {
		// Single repo from --repos or config file
		cfg.Owner = cfg.Repositories[0].Owner
		cfg.Repo = cfg.Repositories[0].Repo
		cfg.Branch = cfg.Repositories[0].Branch
		cfg.Repositories = nil // Clear to use single-repo mode
	}

	// Resolve repo and branch from git (single-repo mode only)
	if !cfg.IsMultiRepo() && (cfg.Owner == "" || cfg.Repo == "") {
		if err := cfg.Resolve(); err != nil {
			if err == config.ErrDetachedHead {
				// In detached HEAD state, we need to resolve the default branch
				// First create client to get repository info
				client, clientErr := gh.NewClient()
				if clientErr != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", clientErr)
					return 2
				}

				// Get repository info (should be resolved by now)
				cwd, cwdErr := os.Getwd()
				if cwdErr != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", cwdErr)
					return 2
				}

				repoInfo, repoErr := git.GetRepoInfo(cwd)
				if repoErr != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", repoErr)
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

func printUsage() {
	fmt.Printf(`cimon - Terminal-first CI monitor for GitHub Actions

USAGE:
    cimon [flags]                    Monitor CI status (interactive)
    cimon retry [flags]              Rerun the latest workflow
    cimon cancel [flags]             Cancel a running workflow
    cimon dispatch <workflow> [flags] Trigger workflow dispatch

FLAGS:
    -r, --repo string     Repository in owner/name format
        --repos string    Comma-separated repos for multi-repo mode (owner/repo1,owner/repo2)
    -b, --branch string   Branch name
    -w, --watch           Watch mode - poll until completion
    -p, --poll duration   Poll interval for watch mode (default 5s)
        --notify          Desktop notification on completion (watch mode)
        --hook string     Run script on completion with env vars (watch mode)
        --no-color        Disable color output
        --plain           Plain text output (no TUI)
        --json            JSON output for scripting
    -v, --version         Show version

CONFIG FILE (cimon.yml):
    repositories:
      - owner/repo1
      - owner/repo2

EXAMPLES:
    cimon                                   # Monitor current repo
    cimon --repos org/api,org/web           # Monitor multiple repos
    cimon --plain                           # Plain text output
    cimon -w --notify                       # Watch with desktop notification
    cimon -w --hook ./my-script.sh          # Watch with custom hook
    cimon retry                             # Rerun latest workflow
    cimon cancel                            # Cancel running workflow
    cimon dispatch deploy.yml               # Trigger workflow dispatch

HOOK ENVIRONMENT VARIABLES:
    CIMON_WORKFLOW_NAME   Workflow name (e.g., "CI")
    CIMON_RUN_NUMBER      Run number (e.g., "123")
    CIMON_CONCLUSION      Conclusion (success, failure, cancelled)
    CIMON_REPO            Repository (owner/repo)
    CIMON_BRANCH          Branch name
    CIMON_HTML_URL        URL to the run

For more information, see: https://github.com/lance0/cimon
`)
}

func runRetry(args []string) int {
	// Parse flags for retry command
	cfg, err := parseSubcommandFlags(args, "retry")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}

	// Resolve repo and branch
	if err := cfg.Resolve(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}

	// Create client
	client, err := gh.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}

	// Get latest run
	run, err := client.FetchLatestRun(cfg.Owner, cfg.Repo, cfg.Branch)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching latest run: %v\n", err)
		return 2
	}

	if run == nil {
		fmt.Fprintf(os.Stderr, "No workflow runs found for %s/%s on branch %s\n", cfg.Owner, cfg.Repo, cfg.Branch)
		return 2
	}

	// Confirm rerun
	fmt.Printf("Rerun workflow #%d (%s) on %s/%s?\n", run.RunNumber, run.Name, cfg.Owner, cfg.Repo)
	if !getConfirmation() {
		fmt.Println("Cancelled.")
		return 0
	}

	// Rerun the workflow
	err = client.RerunWorkflow(cfg.Owner, cfg.Repo, run.ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error rerunning workflow: %v\n", err)
		return 2
	}

	fmt.Printf("Successfully triggered rerun of workflow #%d\n", run.RunNumber)
	return 0
}

func runCancel(args []string) int {
	// Parse flags for cancel command
	cfg, err := parseSubcommandFlags(args, "cancel")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}

	// Resolve repo and branch
	if err := cfg.Resolve(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}

	// Create client
	client, err := gh.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}

	// Get latest run
	run, err := client.FetchLatestRun(cfg.Owner, cfg.Repo, cfg.Branch)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching latest run: %v\n", err)
		return 2
	}

	if run == nil {
		fmt.Fprintf(os.Stderr, "No workflow runs found for %s/%s on branch %s\n", cfg.Owner, cfg.Repo, cfg.Branch)
		return 2
	}

	if run.Status != gh.StatusInProgress && run.Status != gh.StatusQueued {
		fmt.Fprintf(os.Stderr, "Workflow #%d is not running (status: %s)\n", run.RunNumber, run.Status)
		return 2
	}

	// Confirm cancellation
	fmt.Printf("Cancel workflow #%d (%s) on %s/%s?\n", run.RunNumber, run.Name, cfg.Owner, cfg.Repo)
	if !getConfirmation() {
		fmt.Println("Cancelled.")
		return 0
	}

	// Cancel the workflow
	err = client.CancelWorkflow(cfg.Owner, cfg.Repo, run.ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error cancelling workflow: %v\n", err)
		return 2
	}

	fmt.Printf("Successfully cancelled workflow #%d\n", run.RunNumber)
	return 0
}

func runDispatch(args []string) int {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: workflow file required\nUsage: cimon dispatch <workflow-file> [flags]\n")
		return 2
	}

	workflowFile := args[0]
	flags := args[1:]

	// Parse flags for dispatch command
	cfg, err := parseSubcommandFlags(flags, "dispatch")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}

	// Resolve repo and branch
	if err := cfg.Resolve(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}

	// Create client
	client, err := gh.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}

	// Confirm dispatch
	fmt.Printf("Trigger workflow dispatch for %s on %s/%s (branch: %s)?\n", workflowFile, cfg.Owner, cfg.Repo, cfg.Branch)
	if !getConfirmation() {
		fmt.Println("Cancelled.")
		return 0
	}

	// Dispatch the workflow
	err = client.DispatchWorkflow(cfg.Owner, cfg.Repo, workflowFile, cfg.Branch)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error dispatching workflow: %v\n", err)
		return 2
	}

	fmt.Printf("Successfully triggered workflow dispatch for %s\n", workflowFile)
	return 0
}

func parseSubcommandFlags(args []string, command string) (*config.Config, error) {
	cfg := &config.Config{}

	fs := pflag.NewFlagSet(command, pflag.ContinueOnError)

	var repoFlag string
	fs.StringVarP(&repoFlag, "repo", "r", "", "Repository in owner/name format")
	fs.StringVarP(&cfg.Branch, "branch", "b", "", "Branch name")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	// Handle --repo flag
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

func getConfirmation() bool {
	fmt.Print("Confirm? (y/N): ")
	var response string
	_, _ = fmt.Scanln(&response) // Ignore error - empty input is valid (defaults to "N")
	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
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
