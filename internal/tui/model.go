package tui

import (
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
	run  *gh.WorkflowRun
	jobs []gh.Job

	// Job details state
	showingJobDetails bool
	selectedJob       *gh.Job
	jobDetailsCursor  int

	// Log viewer state
	showingLogs     bool
	logContent      string
	logScrollOffset int
	logSearchTerm   string
	logSearchIndex  int

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
		config:   cfg,
		client:   client,
		state:    StateLoading,
		styles:   DefaultStyles(colorEnabled),
		keys:     DefaultKeyMap(),
		spinner:  s,
		watching: cfg.Watch,
	}
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.fetchLatestRun(),
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
		return m, nil

	case TickMsg:
		if m.watching {
			return m, m.fetchLatestRun()
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
		m.state = StateLoading
		return m, m.fetchLatestRun()

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
		} else if m.showingJobDetails {
			if m.selectedJob != nil && m.jobDetailsCursor < len(m.selectedJob.Steps)-1 {
				m.jobDetailsCursor++
			}
		} else {
			if m.cursor < len(m.jobs)-1 {
				m.cursor++
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
			return m, m.fetchLogs(job.ID)
		} else if m.state == StateJobDetails && m.selectedJob != nil {
			// View logs for selected job in details view
			m.showingLogs = true
			m.logScrollOffset = 0
			m.logSearchTerm = ""
			m.logSearchIndex = 0
			return m, m.fetchLogs(m.selectedJob.ID)
		} else if m.state == StateLogViewer {
			// Exit log viewer
			m.showingLogs = false
			m.logContent = ""
			m.logScrollOffset = 0
			m.logSearchTerm = ""
			m.logSearchIndex = 0
			if m.selectedJob != nil {
				m.state = StateJobDetails
			} else {
				m.state = StateReady
			}
			return m, nil
		}
		return m, nil
	}

	return m, nil
}

// Commands

func (m Model) fetchLatestRun() tea.Cmd {
	return func() tea.Msg {
		run, err := m.client.FetchLatestRun(m.config.Owner, m.config.Repo, m.config.Branch)
		if err != nil {
			return ErrMsg{Err: err}
		}
		return RunLoadedMsg{Run: run}
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
