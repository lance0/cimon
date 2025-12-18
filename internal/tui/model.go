package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lance0/cimon/internal/config"
	"github.com/lance0/cimon/internal/gh"
	"github.com/pkg/browser"
)

// State represents the current state of the TUI
type State int

const (
	StateLoading State = iota
	StateReady
	StateWatching
	StateError
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

	return Model{
		config:  cfg,
		client:  client,
		state:   StateLoading,
		styles:  DefaultStyles(),
		keys:    DefaultKeyMap(),
		spinner: s,
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
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil

	case key.Matches(msg, m.keys.Down):
		if m.cursor < len(m.jobs)-1 {
			m.cursor++
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
		if m.run == nil {
			return nil
		}
		// Will be implemented with browser package
		openURL(m.run.HTMLURL)
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

// openURL opens a URL in the default browser
var openURL = func(url string) {
	_ = browser.OpenURL(url)
}
