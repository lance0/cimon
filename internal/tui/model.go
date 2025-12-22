package tui

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lance0/cimon/internal/config"
	"github.com/lance0/cimon/internal/gh"
)

// State represents the current state of the TUI
type State int

const (
	StateLoading State = iota
	StateReady
	StateWatching
	StateError
	StateJobDetails
	StateLogViewer
	StateBranchSelection
	StateStatusFilter
)

// Model is the Bubble Tea model for the TUI
type Model struct {
	// Configuration
	config *config.Config

	// GitHub client
	client *gh.Client

	// Current state
	state State

	// Data
	runs     []gh.WorkflowRun // All workflow runs (for history)
	run      *gh.WorkflowRun  // Currently selected run
	jobs     []gh.Job
	branches []gh.Branch // All available branches

	// Navigation state
	selectedRunIndex    int // Index of currently selected run in runs slice
	selectedBranchIndex int // Index of currently selected branch in branch selection

	// Filter state
	currentStatusFilter string   // Current status filter ("", "success", "failure", "in_progress", etc.)
	statusFilterOptions []string // Available filter options
	selectedFilterIndex int      // Index of currently selected filter option

	// Job details state
	showingJobDetails bool
	selectedJob       *gh.Job
	jobDetailsCursor  int

	// Log viewer state
	showingLogs      bool
	logContent       string
	logScrollOffset  int
	logSearchTerm    string
	logSearchMatches []int // line numbers with matches
	logSearchIndex   int   // current match index
	logJobID         int64
	logLastFetch     time.Time
	logStreaming     bool

	// UI state
	cursor    int
	watching  bool
	lastFetch time.Time

	// Error
	err error

	// Styles and keys
	styles *Styles
	keys   KeyMap

	// Spinner for loading state
	spinner spinner.Model

	// Window size
	width  int
	height int

	// Loading state
	loadingMessage string

	// Exit code to return (set when quitting)
	exitCode int
}

// Messages

// RunLoadedMsg is sent when a workflow run is loaded
type RunLoadedMsg struct {
	Run *gh.WorkflowRun
}

// JobsLoadedMsg is sent when jobs are loaded
type JobsLoadedMsg struct {
	Jobs []gh.Job
}

// JobDetailsLoadedMsg is sent when job details are loaded
type JobDetailsLoadedMsg struct {
	Job *gh.Job
}

// LogLoadedMsg is sent when job logs are loaded
type LogLoadedMsg struct {
	Content string
}

// LogUpdatedMsg is sent when logs are updated during streaming
type LogUpdatedMsg struct {
	Content string
}

// RunsLoadedMsg is sent when multiple workflow runs are loaded
type RunsLoadedMsg struct {
	Runs []gh.WorkflowRun
}

// BranchesLoadedMsg is sent when branches are loaded
type BranchesLoadedMsg struct {
	Branches []gh.Branch
}

// ErrMsg is sent when an error occurs
type ErrMsg struct {
	Err error
}

// TickMsg is sent for watch mode polling
type TickMsg struct {
	Time time.Time
}

// NewModel creates a new TUI model
func NewModel(cfg *config.Config, client *gh.Client) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot

	// Colors are enabled unless NO_COLOR is set or --no-color flag is used
	colorEnabled := os.Getenv("NO_COLOR") == "" && !cfg.NoColor

	return Model{
		config:              cfg,
		client:              client,
		state:               StateLoading,
		selectedRunIndex:    0,  // Start with the first (latest) run
		currentStatusFilter: "", // Start with no filter (all runs)
		statusFilterOptions: []string{"", "success", "failure", "in_progress", "completed", "queued"},
		loadingMessage:      "Loading workflow runs...",
		styles:              DefaultStyles(colorEnabled),
		keys:                DefaultKeyMap(),
		spinner:             s,
		watching:            cfg.Watch,
	}
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.fetchWorkflowRuns(),
	)
}

// Update implements tea.Model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case RunsLoadedMsg:
		m.runs = msg.Runs
		if len(m.runs) > 0 {
			// Ensure selectedRunIndex is valid
			if m.selectedRunIndex >= len(m.runs) {
				m.selectedRunIndex = 0
			}
			m.run = &m.runs[m.selectedRunIndex] // Select the current run
			m.lastFetch = time.Now()
			return m, m.fetchJobs()
		}
		// No runs found - still go to ready state but show message
		m.run = nil
		m.state = StateReady
		return m, nil

	case BranchesLoadedMsg:
		m.branches = msg.Branches
		m.state = StateBranchSelection
		return m, nil

	case RunLoadedMsg:
		m.run = msg.Run
		m.lastFetch = time.Now()
		if m.run != nil {
			return m, m.fetchJobs()
		}
		m.state = StateReady
		return m, nil

	case JobsLoadedMsg:
		m.jobs = msg.Jobs
		// Even if job fetching fails, we can still show the runs
		// Jobs are optional - runs provide the main value
		if m.watching {
			m.state = StateWatching
		} else {
			m.state = StateReady
		}
		// If watching and run is complete, stop watching
		if m.watching && m.run != nil && m.run.IsCompleted() {
			m.watching = false
			m.state = StateReady
		}
		// Set exit code based on run status
		m.updateExitCode()
		return m, m.scheduleNextPoll()

	case JobDetailsLoadedMsg:
		m.selectedJob = msg.Job
		m.state = StateJobDetails
		return m, nil

	case LogLoadedMsg:
		m.logContent = msg.Content
		m.state = StateLogViewer
		// Check if we should enable streaming (job might still be running)
		return m, m.checkStreamingStatus()

	case LogUpdatedMsg:
		// Only update if content has changed
		if msg.Content != m.logContent {
			m.logContent = msg.Content
			// Auto-scroll to bottom for streaming logs
			if m.logStreaming {
				lines := strings.Split(strings.TrimSuffix(m.logContent, "\n"), "\n")
				maxLines := m.height - 8
				if len(lines) > maxLines {
					m.logScrollOffset = len(lines) - maxLines
				}
			}
		}
		// Continue streaming if job is still running
		return m, m.scheduleLogUpdate()

	case TickMsg:
		if m.state == StateLogViewer && m.logStreaming {
			return m, m.updateLogs(m.logJobID)
		} else if m.watching {
			m.loadingMessage = "Watching for updates..."
			m.state = StateLoading
			return m, m.fetchWorkflowRuns()
		}
		return m, nil

	case ErrMsg:
		m.err = msg.Err
		m.state = StateError
		m.exitCode = 2
		return m, nil
	}

	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Refresh):
		if m.err != nil {
			// If we have an error, retry the last operation
			m.err = nil
			m.state = StateLoading
			return m, m.fetchWorkflowRuns()
		} else {
			// Normal refresh
			m.state = StateLoading
			return m, m.fetchWorkflowRuns()
		}

	case key.Matches(msg, m.keys.Watch):
		m.watching = !m.watching
		if m.watching {
			m.state = StateWatching
			return m, m.scheduleNextPoll()
		}
		m.state = StateReady
		return m, nil

	case key.Matches(msg, m.keys.Open):
		return m, m.openInBrowser()

	case key.Matches(msg, m.keys.Up):
		if m.state == StateLogViewer {
			// Scroll up in log viewer
			if m.logScrollOffset > 0 {
				m.logScrollOffset--
			}
		} else if m.state == StateBranchSelection {
			// Navigate branches up
			if m.selectedBranchIndex > 0 {
				m.selectedBranchIndex--
			}
		} else if m.state == StateStatusFilter {
			// Navigate filter options up
			if m.selectedFilterIndex > 0 {
				m.selectedFilterIndex--
			}
		} else {
			if m.cursor > 0 {
				m.cursor--
			}
		}
		return m, nil

	case key.Matches(msg, m.keys.Down):
		if m.state == StateLogViewer {
			// Scroll down in log viewer
			lines := strings.Split(strings.TrimSuffix(m.logContent, "\n"), "\n")
			maxScroll := len(lines) - (m.height - 8) // Approximate visible lines
			if maxScroll > 0 && m.logScrollOffset < maxScroll {
				m.logScrollOffset++
			}
		} else if m.state == StateBranchSelection {
			// Navigate branches down
			if m.selectedBranchIndex < len(m.branches)-1 {
				m.selectedBranchIndex++
			}
		} else if m.state == StateStatusFilter {
			// Navigate filter options down
			if m.selectedFilterIndex < len(m.statusFilterOptions)-1 {
				m.selectedFilterIndex++
			}
		} else if m.showingJobDetails {
			if m.selectedJob != nil && m.jobDetailsCursor < len(m.selectedJob.Steps)-1 {
				m.jobDetailsCursor++
			}
		} else {
			if m.cursor < len(m.jobs)-1 {
				m.cursor--
			}
		}
		return m, nil

	case key.Matches(msg, m.keys.Enter):
		if m.state == StateReady && len(m.jobs) > 0 && m.cursor >= 0 && m.cursor < len(m.jobs) {
			// Enter job details mode
			m.showingJobDetails = true
			m.jobDetailsCursor = 0
			job := m.jobs[m.cursor]
			return m, m.fetchJobDetails(job.ID)
		} else if m.state == StateJobDetails {
			// Exit job details mode
			m.showingJobDetails = false
			m.selectedJob = nil
			m.jobDetailsCursor = 0
			m.state = StateReady
			return m, nil
		} else if m.state == StateBranchSelection {
			// Select the current branch and reload runs
			if len(m.branches) > 0 && m.selectedBranchIndex >= 0 && m.selectedBranchIndex < len(m.branches) {
				selectedBranch := m.branches[m.selectedBranchIndex]
				m.config.Branch = selectedBranch.Name
				m.loadingMessage = fmt.Sprintf("Switching to branch '%s'...", selectedBranch.Name)
				m.state = StateLoading
				m.selectedRunIndex = 0
				return m, m.fetchWorkflowRuns()
			}
		} else if m.state == StateStatusFilter {
			// Apply selected filter and reload runs
			if m.selectedFilterIndex >= 0 && m.selectedFilterIndex < len(m.statusFilterOptions) {
				m.currentStatusFilter = m.statusFilterOptions[m.selectedFilterIndex]
				m.loadingMessage = fmt.Sprintf("Applying '%s' filter...", m.statusFilterOptions[m.selectedFilterIndex])
				m.state = StateLoading
				m.selectedRunIndex = 0
				return m, m.fetchWorkflowRuns()
			}
		}
		return m, nil

	case key.Matches(msg, m.keys.Logs):
		if m.state == StateReady && len(m.jobs) > 0 && m.cursor >= 0 && m.cursor < len(m.jobs) {
			// View logs for selected job
			job := m.jobs[m.cursor]
			m.showingLogs = true
			m.logScrollOffset = 0
			m.logSearchTerm = ""
			m.logSearchIndex = 0
			m.logJobID = job.ID
			m.logLastFetch = time.Now()
			return m, m.fetchLogs(job.ID)
		} else if m.state == StateJobDetails && m.selectedJob != nil {
			// View logs for selected job in details view
			m.showingLogs = true
			m.logScrollOffset = 0
			m.logSearchTerm = ""
			m.logSearchIndex = 0
			m.logJobID = m.selectedJob.ID
			m.logLastFetch = time.Now()
			return m, m.fetchLogs(m.selectedJob.ID)
		} else if m.state == StateLogViewer {
			// Exit log viewer
			m.showingLogs = false
			m.logContent = ""
			m.logScrollOffset = 0
			m.logSearchTerm = ""
			m.logSearchIndex = 0
			m.logJobID = 0
			m.logStreaming = false
			if m.selectedJob != nil {
				m.state = StateJobDetails
			} else {
				m.state = StateReady
			}
			return m, nil
		}
		return m, nil

	case key.Matches(msg, m.keys.Search):
		if m.state == StateLogViewer {
			// For now, just set a demo search term
			// In a real implementation, this would enter input mode
			m.logSearchTerm = "error"
			m.findSearchMatches()
			return m, nil
		}
		return m, nil

	case key.Matches(msg, m.keys.NextMatch):
		if m.state == StateLogViewer && len(m.logSearchMatches) > 0 {
			m.nextSearchMatch()
			return m, nil
		}
		return m, nil

	case key.Matches(msg, m.keys.PrevMatch):
		if m.state == StateLogViewer && len(m.logSearchMatches) > 0 {
			m.prevSearchMatch()
			return m, nil
		}
		return m, nil

	case key.Matches(msg, m.keys.NextRun):
		if !m.showingJobDetails && !m.showingLogs && len(m.runs) > 1 {
			if m.selectedRunIndex < len(m.runs)-1 {
				m.selectedRunIndex++
				m.run = &m.runs[m.selectedRunIndex]
				m.cursor = 0 // Reset job cursor
				return m, m.fetchJobs()
			}
		}
		return m, nil

	case key.Matches(msg, m.keys.PrevRun):
		if !m.showingJobDetails && !m.showingLogs && len(m.runs) > 1 {
			if m.selectedRunIndex > 0 {
				m.selectedRunIndex--
				m.run = &m.runs[m.selectedRunIndex]
				m.cursor = 0 // Reset job cursor
				return m, m.fetchJobs()
			}
		}
		return m, nil

	case key.Matches(msg, m.keys.BranchSelect):
		if m.state == StateReady && !m.showingJobDetails && !m.showingLogs {
			// Enter branch selection mode
			m.selectedBranchIndex = 0 // Start with first branch
			return m, m.fetchBranches()
		} else if m.state == StateBranchSelection {
			// Exit branch selection mode (don't change branch)
			m.state = StateReady
			return m, nil
		}
		return m, nil

	case key.Matches(msg, m.keys.Filter):
		if m.state == StateReady && !m.showingJobDetails && !m.showingLogs {
			// Enter status filter mode
			m.selectedFilterIndex = 0 // Start with first option (All)
			m.state = StateStatusFilter
			return m, nil
		} else if m.state == StateStatusFilter {
			// Apply selected filter and reload runs
			if m.selectedFilterIndex >= 0 && m.selectedFilterIndex < len(m.statusFilterOptions) {
				m.currentStatusFilter = m.statusFilterOptions[m.selectedFilterIndex]
				m.state = StateLoading
				m.selectedRunIndex = 0
				return m, m.fetchWorkflowRuns()
			}
		}
		return m, nil
	}

	return m, nil
}

// Commands

func (m Model) fetchWorkflowRuns() tea.Cmd {
	return func() tea.Msg {
		runs, err := m.client.FetchWorkflowRuns(m.config.Owner, m.config.Repo, m.config.Branch, m.currentStatusFilter, 1, 10) // Fetch 10 most recent runs with current filter
		if err != nil {
			return ErrMsg{Err: err}
		}

		if len(runs) == 0 {
			return ErrMsg{Err: fmt.Errorf("no workflow runs found")}
		}

		return RunsLoadedMsg{Runs: runs}
	}
}

func (m Model) fetchBranches() tea.Cmd {
	return func() tea.Msg {
		branches, err := m.client.FetchBranches(m.config.Owner, m.config.Repo)
		if err != nil {
			return ErrMsg{Err: err}
		}

		return BranchesLoadedMsg{Branches: branches}
	}
}

func (m Model) fetchJobs() tea.Cmd {
	return func() tea.Msg {
		if m.run == nil {
			return JobsLoadedMsg{Jobs: nil}
		}
		jobs, err := m.client.FetchJobs(m.config.Owner, m.config.Repo, m.run.ID)
		if err != nil {
			return ErrMsg{Err: err}
		}
		return JobsLoadedMsg{Jobs: jobs}
	}
}

func (m Model) fetchJobDetails(jobID int64) tea.Cmd {
	return func() tea.Msg {
		job, err := m.client.FetchJobDetails(m.config.Owner, m.config.Repo, jobID)
		if err != nil {
			return ErrMsg{Err: err}
		}
		return JobDetailsLoadedMsg{Job: job}
	}
}

func (m Model) fetchLogs(jobID int64) tea.Cmd {
	return func() tea.Msg {
		logs, err := m.client.FetchJobLogs(m.config.Owner, m.config.Repo, jobID)
		if err != nil {
			return ErrMsg{Err: err}
		}
		return LogLoadedMsg{Content: logs}
	}
}

func (m Model) updateLogs(jobID int64) tea.Cmd {
	return func() tea.Msg {
		logs, err := m.client.FetchJobLogs(m.config.Owner, m.config.Repo, jobID)
		if err != nil {
			// Don't return error for streaming updates, just ignore
			return LogUpdatedMsg{Content: m.logContent}
		}
		return LogUpdatedMsg{Content: logs}
	}
}

func (m Model) checkStreamingStatus() tea.Cmd {
	// Check if the current job is still running
	for _, job := range m.jobs {
		if job.ID == m.logJobID {
			m.logStreaming = job.Status == gh.StatusInProgress || job.Status == gh.StatusQueued
			if m.logStreaming {
				return m.scheduleLogUpdate()
			}
			break
		}
	}
	return nil
}

func (m Model) scheduleLogUpdate() tea.Cmd {
	if !m.logStreaming {
		return nil
	}
	// Update logs every 3 seconds for running jobs
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
		return TickMsg{Time: t}
	})
}

func (m *Model) findSearchMatches() {
	m.logSearchMatches = []int{}
	if m.logSearchTerm == "" || m.logContent == "" {
		return
	}

	lines := strings.Split(strings.TrimSuffix(m.logContent, "\n"), "\n")
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), strings.ToLower(m.logSearchTerm)) {
			m.logSearchMatches = append(m.logSearchMatches, i)
		}
	}
	m.logSearchIndex = 0
}

func (m *Model) nextSearchMatch() {
	if len(m.logSearchMatches) == 0 {
		return
	}
	m.logSearchIndex = (m.logSearchIndex + 1) % len(m.logSearchMatches)
	lineNum := m.logSearchMatches[m.logSearchIndex]
	m.scrollToLine(lineNum)
}

func (m *Model) prevSearchMatch() {
	if len(m.logSearchMatches) == 0 {
		return
	}
	m.logSearchIndex--
	if m.logSearchIndex < 0 {
		m.logSearchIndex = len(m.logSearchMatches) - 1
	}
	lineNum := m.logSearchMatches[m.logSearchIndex]
	m.scrollToLine(lineNum)
}

func (m *Model) scrollToLine(lineNum int) {
	maxLines := m.height - 10
	if lineNum < m.logScrollOffset {
		m.logScrollOffset = lineNum
	} else if lineNum >= m.logScrollOffset+maxLines {
		m.logScrollOffset = lineNum - maxLines + 1
	}
}

func (m Model) scheduleNextPoll() tea.Cmd {
	if !m.watching {
		return nil
	}
	return tea.Tick(m.config.Poll, func(t time.Time) tea.Msg {
		return TickMsg{Time: t}
	})
}

func (m Model) openInBrowser() tea.Cmd {
	return func() tea.Msg {
		if m.showingJobDetails && m.selectedJob != nil {
			openURL(m.selectedJob.HTMLURL)
		} else if m.run != nil {
			openURL(m.run.HTMLURL)
		}
		return nil
	}
}

func (m *Model) updateExitCode() {
	if m.run == nil {
		m.exitCode = 2
		return
	}
	if m.run.IsSuccess() {
		m.exitCode = 0
	} else if m.run.IsFailure() {
		m.exitCode = 1
	} else {
		// Still running or queued
		m.exitCode = 0
	}
}

// ExitCode returns the exit code to use when quitting
func (m Model) ExitCode() int {
	return m.exitCode
}

// openURL opens a URL in the default browser silently (no stderr output)
var openURL = func(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	// Suppress all output - we don't want to pollute the TUI
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	// Detach from terminal
	cmd.Env = os.Environ()
	_ = cmd.Start()
}
