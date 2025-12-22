package tui

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lance0/cimon/internal/config"
	"github.com/lance0/cimon/internal/gh"
	"github.com/lance0/cimon/internal/notify"
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
	StateHelp
	StateWorkflowViewer
	StateArtifactSelection
	StateLogFilter      // v0.6: Log filter selection
	StateMultiJobSelect // v0.6: Multi-job selection for following
	StateCompareSelect  // v0.6: Run selection for comparison
	StateCompareView    // v0.6: Viewing log comparison
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
	showingLogs       bool
	logContent        string
	logScrollOffset   int
	logSearchTerm     string
	logSearchMatches  []int // line numbers with matches
	logSearchIndex    int   // current match index
	logJobID          int64
	logLastFetch      time.Time
	logStreaming      bool
	searchInputMode   bool   // true when typing search term
	searchInputBuffer string // buffer for search input
	logSyntaxEnabled  bool      // v0.6: syntax highlighting on/off
	logExportMessage  string    // v0.6: export success/error message
	logExportTime     time.Time // v0.6: when message was set (for auto-clear)

	// Log filtering state (v0.6)
	parsedLogs           *gh.ParsedLogs // Structured log data with step-level parsing
	logFilterStepNumbers []int          // Currently selected step numbers to display
	logFilterIndex       int            // Current selection in filter menu

	// Multi-job following state (v0.6)
	multiJobMode      bool             // Whether we're in multi-job view mode
	multiJobIDs       []int64          // Selected job IDs for multi-job view
	multiJobContents  map[int64]string // Log contents for each job
	multiJobViewSplit bool             // true=split view, false=combined view
	multiJobSelectIdx int              // Selection cursor for job selection

	// Log comparison state (v0.6)
	compareRunIdx1    int      // First run index for comparison
	compareRunIdx2    int      // Second run index for comparison (-1 = not selected)
	compareSelectStep int      // 0 = selecting first, 1 = selecting second
	compareCursor     int      // Cursor for run selection
	compareLogs1      string   // Logs for first run
	compareLogs2      string   // Logs for second run
	compareDiff       []string // Computed diff lines
	compareDiffColors []int    // 0=normal, 1=added, -1=removed
	compareScrollOff  int      // Scroll offset for diff view

	// Multi-repo state (v0.8)
	multiRepoMode      bool             // True when monitoring multiple repos
	sourcedRuns        []gh.SourcedRun  // Runs from all repos, sorted by time
	selectedSourcedRun int              // Index in sourcedRuns slice

	// Workflow viewer state
	workflowContent      string
	workflowScrollOffset int
	workflowPath         string

	// Artifact selection state
	artifacts             []gh.Artifact
	selectedArtifactIndex int

	// UI state
	cursor           int
	watching         bool
	notificationSent bool // v0.7: Prevent duplicate notifications on completion
	lastFetch        time.Time

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

// WorkflowLoadedMsg is sent when workflow content is loaded
type WorkflowLoadedMsg struct {
	Content string
	Path    string
}

// ArtifactsLoadedMsg is sent when artifacts are loaded
type ArtifactsLoadedMsg struct {
	Artifacts []gh.Artifact
}

// ArtifactDownloadedMsg is sent when an artifact is downloaded
type ArtifactDownloadedMsg struct {
	Filename string
}

// LogExportedMsg is sent when logs are exported to file (v0.6)
type LogExportedMsg struct {
	Filename string
	Error    error
}

// ParsedLogsLoadedMsg is sent when structured logs are loaded (v0.6)
type ParsedLogsLoadedMsg struct {
	Logs *gh.ParsedLogs
}

// MultiJobLogsLoadedMsg is sent when logs for multiple jobs are loaded (v0.6)
type MultiJobLogsLoadedMsg struct {
	Contents map[int64]string
}

// CompareLogsLoadedMsg is sent when logs for comparison are loaded (v0.6)
type CompareLogsLoadedMsg struct {
	Logs1 string
	Logs2 string
}

// MultiRepoRunsLoadedMsg is sent when runs from multiple repos are loaded (v0.8)
type MultiRepoRunsLoadedMsg struct {
	SourcedRuns []gh.SourcedRun
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

	// v0.8: Determine loading message based on mode
	loadingMsg := "Loading workflow runs..."
	if cfg.IsMultiRepo() {
		loadingMsg = "Loading runs from multiple repositories..."
	}

	return Model{
		config:              cfg,
		client:              client,
		state:               StateLoading,
		multiRepoMode:       cfg.IsMultiRepo(), // v0.8
		selectedRunIndex:    0,                 // Start with the first (latest) run
		currentStatusFilter: "",                // Start with no filter (all runs)
		statusFilterOptions: []string{"", "success", "failure", "in_progress", "completed", "queued"},
		loadingMessage:      loadingMsg,
		styles:              DefaultStyles(colorEnabled),
		keys:                DefaultKeyMap(),
		spinner:             s,
		watching:            cfg.Watch,
		logSyntaxEnabled:    true, // v0.6: syntax highlighting on by default
	}
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	// v0.8: Branch based on multi-repo mode
	if m.multiRepoMode {
		return tea.Batch(
			m.spinner.Tick,
			m.fetchMultiRepoRuns(),
		)
	}
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

	case MultiRepoRunsLoadedMsg:
		// v0.8: Handle multi-repo runs loading
		m.sourcedRuns = msg.SourcedRuns
		m.lastFetch = time.Now()
		if len(m.sourcedRuns) > 0 {
			// Ensure selectedSourcedRun is valid
			if m.selectedSourcedRun >= len(m.sourcedRuns) {
				m.selectedSourcedRun = 0
			}
			// Set current run and context from selected sourced run
			sr := m.sourcedRuns[m.selectedSourcedRun]
			m.run = sr.Run
			m.config.Owner = sr.Owner
			m.config.Repo = sr.Repo
			return m, m.fetchJobs()
		}
		// No runs found
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
		// If watching and run is complete, stop watching and trigger notifications
		if m.watching && m.run != nil && m.run.IsCompleted() {
			m.watching = false
			m.state = StateReady
			// v0.7: Send notification and execute hook (only once per completion)
			if !m.notificationSent {
				m.notificationSent = true
				m.triggerNotifications()
			}
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

	case WorkflowLoadedMsg:
		m.workflowContent = msg.Content
		m.workflowPath = msg.Path
		m.state = StateWorkflowViewer
		return m, nil

	case ArtifactsLoadedMsg:
		m.artifacts = msg.Artifacts
		m.selectedArtifactIndex = 0
		m.state = StateArtifactSelection
		return m, nil

	case ArtifactDownloadedMsg:
		// Show success message and return to previous state
		m.state = StateReady
		return m, nil

	case LogExportedMsg:
		// v0.6: Handle log export result
		if msg.Error != nil {
			m.logExportMessage = fmt.Sprintf("Export failed: %v", msg.Error)
		} else {
			m.logExportMessage = fmt.Sprintf("Saved to %s", msg.Filename)
		}
		m.logExportTime = time.Now()
		return m, nil

	case ParsedLogsLoadedMsg:
		// v0.6: Handle structured log loading for filtering
		m.parsedLogs = msg.Logs
		if m.parsedLogs != nil {
			m.logContent = m.parsedLogs.Combined
		}
		m.state = StateLogFilter
		return m, nil

	case MultiJobLogsLoadedMsg:
		// v0.6: Handle multi-job log loading
		m.multiJobContents = msg.Contents
		m.multiJobMode = true
		m.state = StateLogViewer
		// Build combined content from all selected jobs
		m.logContent = m.buildMultiJobContent()
		return m, nil

	case CompareLogsLoadedMsg:
		// v0.6: Handle comparison log loading
		m.compareLogs1 = msg.Logs1
		m.compareLogs2 = msg.Logs2
		m.compareDiff, m.compareDiffColors = m.computeDiff(msg.Logs1, msg.Logs2)
		m.compareScrollOff = 0
		m.state = StateCompareView
		return m, nil

	case TickMsg:
		{
			if m.state == StateLogViewer && m.logStreaming {
				return m, m.updateLogs(m.logJobID)
			} else if m.watching {
				m.loadingMessage = "Watching for updates..."
				m.state = StateLoading
				return m, m.fetchWorkflowRuns()
			}
		}
		return m, nil

	case ErrMsg:
		{
			m.err = msg.Err
			m.state = StateError
			m.exitCode = 2
			return m, nil
		}

	default:
		{
			return m, nil
		}
	}
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle search input mode first
	if m.searchInputMode {
		switch msg.Type {
		case tea.KeyEnter:
			// Confirm search
			m.logSearchTerm = m.searchInputBuffer
			m.searchInputMode = false
			m.findSearchMatches()
			if len(m.logSearchMatches) > 0 {
				m.scrollToLine(m.logSearchMatches[0])
			}
			return m, nil
		case tea.KeyEsc:
			// Cancel search
			m.searchInputMode = false
			m.searchInputBuffer = ""
			return m, nil
		case tea.KeyBackspace:
			// Remove last character
			if len(m.searchInputBuffer) > 0 {
				m.searchInputBuffer = m.searchInputBuffer[:len(m.searchInputBuffer)-1]
			}
			return m, nil
		default:
			// Add character to search buffer
			if msg.Type == tea.KeyRunes {
				m.searchInputBuffer += string(msg.Runes)
			}
			return m, nil
		}
	}

	// Handle help state - any key exits (except q which quits)
	if m.state == StateHelp && !key.Matches(msg, m.keys.Quit) {
		m.state = StateReady
		return m, nil
	}

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
			m.notificationSent = false // v0.7: Reset for new watch session
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
		} else if m.state == StateArtifactSelection {
			// Navigate artifacts up
			if m.selectedArtifactIndex > 0 {
				m.selectedArtifactIndex--
			}
		} else if m.state == StateLogFilter {
			// v0.6: Navigate log filter steps up
			if m.logFilterIndex > 0 {
				m.logFilterIndex--
			}
		} else if m.state == StateMultiJobSelect {
			// v0.6: Navigate multi-job selection up
			if m.multiJobSelectIdx > 0 {
				m.multiJobSelectIdx--
			}
		} else if m.state == StateCompareSelect {
			// v0.6: Navigate compare selection up
			if m.compareCursor > 0 {
				m.compareCursor--
			}
		} else if m.state == StateCompareView {
			// v0.6: Scroll up in compare view
			if m.compareScrollOff > 0 {
				m.compareScrollOff--
			}
		} else if m.multiRepoMode && m.state == StateReady {
			// v0.8: Navigate multi-repo runs up
			if m.selectedSourcedRun > 0 {
				m.selectedSourcedRun--
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
		} else if m.state == StateArtifactSelection {
			// Navigate artifacts down
			if m.selectedArtifactIndex < len(m.artifacts)-1 {
				m.selectedArtifactIndex++
			}
		} else if m.state == StateLogFilter {
			// v0.6: Navigate log filter steps down
			if m.parsedLogs != nil && m.logFilterIndex < len(m.parsedLogs.Steps)-1 {
				m.logFilterIndex++
			}
		} else if m.state == StateMultiJobSelect {
			// v0.6: Navigate multi-job selection down
			if m.multiJobSelectIdx < len(m.jobs)-1 {
				m.multiJobSelectIdx++
			}
		} else if m.state == StateCompareSelect {
			// v0.6: Navigate compare selection down
			if m.compareCursor < len(m.runs)-1 {
				m.compareCursor++
			}
		} else if m.state == StateCompareView {
			// v0.6: Scroll down in compare view
			maxScroll := len(m.compareDiff) - (m.height - 10)
			if maxScroll > 0 && m.compareScrollOff < maxScroll {
				m.compareScrollOff++
			}
		} else if m.multiRepoMode && m.state == StateReady {
			// v0.8: Navigate multi-repo runs down
			if m.selectedSourcedRun < len(m.sourcedRuns)-1 {
				m.selectedSourcedRun++
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
		if m.state == StateLogFilter {
			// v0.6: Apply filter and return to log viewer
			m.applyLogFilter()
			m.state = StateLogViewer
			return m, nil
		} else if m.state == StateMultiJobSelect {
			// v0.6: Apply multi-job selection and load logs
			if len(m.multiJobIDs) > 0 {
				m.loadingMessage = fmt.Sprintf("Loading logs for %d jobs...", len(m.multiJobIDs))
				m.state = StateLoading
				return m, m.fetchMultiJobLogs()
			}
			// No jobs selected, go back
			m.state = StateReady
			return m, nil
		} else if m.state == StateCompareSelect {
			// v0.6: Select run for comparison
			if m.compareSelectStep == 0 {
				m.compareRunIdx1 = m.compareCursor
				m.compareSelectStep = 1
				// Move cursor to a different run
				if m.compareCursor == 0 && len(m.runs) > 1 {
					m.compareCursor = 1
				}
			} else {
				if m.compareCursor != m.compareRunIdx1 {
					m.compareRunIdx2 = m.compareCursor
					m.loadingMessage = "Loading logs for comparison..."
					m.state = StateLoading
					return m, m.fetchComparisonLogs()
				}
			}
			return m, nil
		} else if m.multiRepoMode && m.state == StateReady && len(m.sourcedRuns) > 0 {
			// v0.8: Select multi-repo run and load its jobs
			sr := m.sourcedRuns[m.selectedSourcedRun]
			m.run = sr.Run
			m.config.Owner = sr.Owner
			m.config.Repo = sr.Repo
			m.cursor = 0 // Reset job cursor
			m.loadingMessage = fmt.Sprintf("Loading jobs for %s...", sr.RepoSlug())
			m.state = StateLoading
			return m, m.fetchJobs()
		} else if m.state == StateReady && len(m.jobs) > 0 && m.cursor >= 0 && m.cursor < len(m.jobs) {
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
		} else if m.state == StateArtifactSelection {
			// Download selected artifact
			if len(m.artifacts) > 0 && m.selectedArtifactIndex >= 0 && m.selectedArtifactIndex < len(m.artifacts) {
				selectedArtifact := m.artifacts[m.selectedArtifactIndex]
				if !selectedArtifact.Expired {
					m.loadingMessage = fmt.Sprintf("Downloading %s...", selectedArtifact.Name)
					m.state = StateLoading
					return m, m.downloadArtifact(selectedArtifact)
				}
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

	case key.Matches(msg, m.keys.Search):
		if m.state == StateLogViewer && !m.searchInputMode {
			// Enter search input mode
			m.searchInputMode = true
			m.searchInputBuffer = ""
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
				m.loadingMessage = fmt.Sprintf("Applying '%s' filter...", m.statusFilterOptions[m.selectedFilterIndex])
				m.state = StateLoading
				m.selectedRunIndex = 0
				return m, m.fetchWorkflowRuns()
			}
		}
		return m, nil

	case key.Matches(msg, m.keys.Help):
		if m.state != StateHelp {
			// Enter help mode
			m.state = StateHelp
		}
		// Note: exiting help with any key is handled at the top of handleKey
		return m, nil

	case key.Matches(msg, m.keys.Workflow):
		if m.run != nil && m.run.Path != "" {
			// Enter workflow viewer mode
			m.workflowScrollOffset = 0
			m.workflowPath = m.run.Path
			m.loadingMessage = fmt.Sprintf("Loading workflow file %s...", m.run.Path)
			m.state = StateLoading
			return m, m.fetchWorkflowContent()
		}
		return m, nil

	case key.Matches(msg, m.keys.Artifacts):
		if m.run != nil {
			// Enter artifact selection mode
			m.loadingMessage = "Loading artifacts..."
			m.state = StateLoading
			return m, m.fetchArtifacts()
		}
		return m, nil

	case key.Matches(msg, m.keys.LogHighlight):
		// v0.6: Toggle syntax highlighting in log viewer
		if m.state == StateLogViewer {
			m.logSyntaxEnabled = !m.logSyntaxEnabled
		}
		return m, nil

	case key.Matches(msg, m.keys.LogSave):
		// v0.6: Export logs to file
		if m.state == StateLogViewer && m.logContent != "" {
			return m, m.exportCurrentLogs()
		}
		return m, nil

	case key.Matches(msg, m.keys.LogFilter):
		// v0.6: Enter log filter selection mode
		if m.state == StateLogViewer && m.logJobID != 0 {
			m.logFilterIndex = 0
			m.loadingMessage = "Loading step structure..."
			m.state = StateLoading
			return m, m.fetchLogsStructured(m.logJobID)
		} else if m.state == StateLogFilter {
			// Apply filter and return to log viewer
			m.applyLogFilter()
			m.state = StateLogViewer
			return m, nil
		}
		return m, nil

	case key.Matches(msg, m.keys.Escape):
		// Exit from filter mode without applying
		if m.state == StateLogFilter {
			m.state = StateLogViewer
			return m, nil
		}
		// v0.6: Exit from multi-job selection without applying
		if m.state == StateMultiJobSelect {
			m.state = StateReady
			return m, nil
		}
		// v0.6: Exit from compare selection or view
		if m.state == StateCompareSelect || m.state == StateCompareView {
			m.state = StateReady
			return m, nil
		}
		return m, nil

	case key.Matches(msg, m.keys.Space):
		// v0.6: Toggle step selection in log filter mode
		if m.state == StateLogFilter && m.parsedLogs != nil && len(m.parsedLogs.Steps) > 0 {
			stepNum := m.parsedLogs.Steps[m.logFilterIndex].Number
			m.toggleStepFilter(stepNum)
			return m, nil
		}
		// v0.6: Toggle job selection in multi-job select mode
		if m.state == StateMultiJobSelect && len(m.jobs) > 0 {
			jobID := m.jobs[m.multiJobSelectIdx].ID
			m.toggleMultiJobSelection(jobID)
			return m, nil
		}
		return m, nil

	case key.Matches(msg, m.keys.LogMulti):
		// v0.6: Enter multi-job selection mode
		if (m.state == StateReady || m.state == StateLogViewer) && len(m.jobs) > 1 {
			m.multiJobSelectIdx = 0
			m.state = StateMultiJobSelect
			return m, nil
		} else if m.state == StateMultiJobSelect {
			// Apply selection and load logs
			if len(m.multiJobIDs) > 0 {
				m.loadingMessage = fmt.Sprintf("Loading logs for %d jobs...", len(m.multiJobIDs))
				m.state = StateLoading
				return m, m.fetchMultiJobLogs()
			}
			// No jobs selected, go back
			m.state = StateReady
			return m, nil
		}
		return m, nil

	case key.Matches(msg, m.keys.LogViewToggle):
		// v0.6: Toggle between split and combined view in multi-job mode
		if m.state == StateLogViewer && m.multiJobMode {
			m.multiJobViewSplit = !m.multiJobViewSplit
			m.logContent = m.buildMultiJobContent()
			return m, nil
		}
		return m, nil

	case key.Matches(msg, m.keys.LogCompare):
		// v0.6: Enter comparison mode
		if m.state == StateReady && len(m.runs) >= 2 {
			m.compareCursor = 0
			m.compareSelectStep = 0
			m.compareRunIdx1 = -1
			m.compareRunIdx2 = -1
			m.state = StateCompareSelect
			return m, nil
		} else if m.state == StateCompareSelect {
			// Select current run
			if m.compareSelectStep == 0 {
				m.compareRunIdx1 = m.compareCursor
				m.compareSelectStep = 1
				// Move cursor to a different run
				if m.compareCursor == 0 && len(m.runs) > 1 {
					m.compareCursor = 1
				}
			} else {
				if m.compareCursor != m.compareRunIdx1 {
					m.compareRunIdx2 = m.compareCursor
					// Load logs for both runs
					m.loadingMessage = "Loading logs for comparison..."
					m.state = StateLoading
					return m, m.fetchComparisonLogs()
				}
			}
			return m, nil
		} else if m.state == StateCompareView {
			// Exit comparison view
			m.state = StateReady
			return m, nil
		}
		return m, nil

	default:
		return m, nil
	}
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

// fetchMultiRepoRuns fetches runs from all configured repositories (v0.8)
func (m Model) fetchMultiRepoRuns() tea.Cmd {
	return func() tea.Msg {
		var allRuns []gh.SourcedRun

		for _, repo := range m.config.Repositories {
			runs, err := m.client.FetchWorkflowRuns(
				repo.Owner, repo.Repo, repo.Branch,
				m.currentStatusFilter, 1, 5, // Fetch 5 recent runs per repo
			)
			if err != nil {
				// Log error but continue with other repos
				continue
			}

			for i := range runs {
				allRuns = append(allRuns, gh.SourcedRun{
					Owner: repo.Owner,
					Repo:  repo.Repo,
					Run:   &runs[i],
				})
			}
		}

		// Sort by UpdatedAt descending (most recent first)
		sort.Slice(allRuns, func(i, j int) bool {
			return allRuns[i].Run.UpdatedAt.After(allRuns[j].Run.UpdatedAt)
		})

		if len(allRuns) == 0 {
			return ErrMsg{Err: fmt.Errorf("no workflow runs found across repositories")}
		}

		return MultiRepoRunsLoadedMsg{SourcedRuns: allRuns}
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

func (m Model) fetchWorkflowContent() tea.Cmd {
	return func() tea.Msg {
		content, err := m.client.FetchWorkflowContent(m.config.Owner, m.config.Repo, m.workflowPath)
		if err != nil {
			return ErrMsg{Err: err}
		}
		return WorkflowLoadedMsg{Content: content, Path: m.workflowPath}
	}
}

func (m Model) fetchArtifacts() tea.Cmd {
	return func() tea.Msg {
		if m.run == nil {
			return ArtifactsLoadedMsg{Artifacts: nil}
		}
		artifacts, err := m.client.FetchWorkflowArtifacts(m.config.Owner, m.config.Repo, m.run.ID)
		if err != nil {
			return ErrMsg{Err: err}
		}
		return ArtifactsLoadedMsg{Artifacts: artifacts}
	}
}

func (m Model) downloadArtifact(artifact gh.Artifact) tea.Cmd {
	return func() tea.Msg {
		filename := fmt.Sprintf("%s.zip", artifact.Name)
		err := m.client.DownloadArtifact(m.config.Owner, m.config.Repo, artifact.ID, filename)
		if err != nil {
			return ErrMsg{Err: err}
		}
		return ArtifactDownloadedMsg{Filename: filename}
	}
}

// exportCurrentLogs exports the current log content to a file (v0.6)
func (m Model) exportCurrentLogs() tea.Cmd {
	return func() tea.Msg {
		// Generate filename: cimon-logs-REPO-RUNID-TIMESTAMP.txt
		timestamp := time.Now().Format("20060102-150405")
		filename := fmt.Sprintf("cimon-logs-%s-%d-%s.txt",
			m.config.Repo, m.run.ID, timestamp)

		// Build content with metadata header
		var content strings.Builder
		content.WriteString("# Cimon Log Export\n")
		content.WriteString(fmt.Sprintf("# Repository: %s/%s\n", m.config.Owner, m.config.Repo))
		content.WriteString(fmt.Sprintf("# Branch: %s\n", m.config.Branch))
		if m.run != nil {
			content.WriteString(fmt.Sprintf("# Run: #%d (ID: %d)\n", m.run.RunNumber, m.run.ID))
		}
		content.WriteString(fmt.Sprintf("# Job ID: %d\n", m.logJobID))
		content.WriteString(fmt.Sprintf("# Exported: %s\n", time.Now().Format(time.RFC3339)))
		content.WriteString("#\n\n")
		content.WriteString(m.logContent)

		err := os.WriteFile(filename, []byte(content.String()), 0644)
		return LogExportedMsg{Filename: filename, Error: err}
	}
}

// fetchLogsStructured fetches logs with step-level structure for filtering (v0.6)
func (m Model) fetchLogsStructured(jobID int64) tea.Cmd {
	return func() tea.Msg {
		logs, err := m.client.FetchJobLogsStructured(m.config.Owner, m.config.Repo, jobID)
		if err != nil {
			return ErrMsg{Err: err}
		}
		return ParsedLogsLoadedMsg{Logs: logs}
	}
}

// toggleStepFilter toggles a step number in the filter selection (v0.6)
func (m *Model) toggleStepFilter(stepNum int) {
	// Check if step is already selected
	for i, n := range m.logFilterStepNumbers {
		if n == stepNum {
			// Remove it
			m.logFilterStepNumbers = append(m.logFilterStepNumbers[:i], m.logFilterStepNumbers[i+1:]...)
			return
		}
	}
	// Add it
	m.logFilterStepNumbers = append(m.logFilterStepNumbers, stepNum)
}

// applyLogFilter applies the current filter selection to log content (v0.6)
func (m *Model) applyLogFilter() {
	if m.parsedLogs == nil {
		return
	}

	if len(m.logFilterStepNumbers) == 0 {
		// No filter - show all
		m.logContent = m.parsedLogs.Combined
	} else {
		// Apply filter
		m.logContent = m.parsedLogs.FilteredContent(m.logFilterStepNumbers)
	}
	m.logScrollOffset = 0 // Reset scroll position
}

// isStepSelected returns true if a step number is in the filter selection (v0.6)
func (m Model) isStepSelected(stepNum int) bool {
	for _, n := range m.logFilterStepNumbers {
		if n == stepNum {
			return true
		}
	}
	return false
}

// toggleMultiJobSelection toggles a job ID in the multi-job selection (v0.6)
func (m *Model) toggleMultiJobSelection(jobID int64) {
	// Check if job is already selected
	for i, id := range m.multiJobIDs {
		if id == jobID {
			// Remove it
			m.multiJobIDs = append(m.multiJobIDs[:i], m.multiJobIDs[i+1:]...)
			return
		}
	}
	// Add it (max 4 jobs for reasonable display)
	if len(m.multiJobIDs) < 4 {
		m.multiJobIDs = append(m.multiJobIDs, jobID)
	}
}

// isJobSelected returns true if a job ID is in the multi-job selection (v0.6)
func (m Model) isJobSelected(jobID int64) bool {
	for _, id := range m.multiJobIDs {
		if id == jobID {
			return true
		}
	}
	return false
}

// fetchMultiJobLogs fetches logs for all selected jobs (v0.6)
func (m Model) fetchMultiJobLogs() tea.Cmd {
	return func() tea.Msg {
		contents := make(map[int64]string)
		for _, jobID := range m.multiJobIDs {
			logs, err := m.client.FetchJobLogs(m.config.Owner, m.config.Repo, jobID)
			if err != nil {
				contents[jobID] = fmt.Sprintf("Error loading logs: %v", err)
			} else {
				contents[jobID] = logs
			}
		}
		return MultiJobLogsLoadedMsg{Contents: contents}
	}
}

// buildMultiJobContent builds the combined log content from multiple jobs (v0.6)
func (m *Model) buildMultiJobContent() string {
	if len(m.multiJobIDs) == 0 || m.multiJobContents == nil {
		return ""
	}

	var b strings.Builder

	// Find job names by ID
	jobNames := make(map[int64]string)
	for _, job := range m.jobs {
		jobNames[job.ID] = job.Name
	}

	for _, jobID := range m.multiJobIDs {
		content, ok := m.multiJobContents[jobID]
		if !ok {
			continue
		}

		jobName := jobNames[jobID]
		if jobName == "" {
			jobName = fmt.Sprintf("Job %d", jobID)
		}

		b.WriteString("\n══════════════════════════════════════════════════════════════════════════════\n")
		b.WriteString(fmt.Sprintf("  JOB: %s\n", jobName))
		b.WriteString("══════════════════════════════════════════════════════════════════════════════\n\n")
		b.WriteString(content)
		b.WriteString("\n")
	}

	return b.String()
}

// fetchComparisonLogs fetches logs for both runs to compare (v0.6)
func (m Model) fetchComparisonLogs() tea.Cmd {
	return func() tea.Msg {
		if m.compareRunIdx1 < 0 || m.compareRunIdx2 < 0 ||
			m.compareRunIdx1 >= len(m.runs) || m.compareRunIdx2 >= len(m.runs) {
			return ErrMsg{Err: fmt.Errorf("invalid run selection for comparison")}
		}

		run1 := m.runs[m.compareRunIdx1]
		run2 := m.runs[m.compareRunIdx2]

		// Get jobs for both runs and fetch logs for the first job of each
		jobs1, err := m.client.FetchJobs(m.config.Owner, m.config.Repo, run1.ID)
		if err != nil || len(jobs1) == 0 {
			return ErrMsg{Err: fmt.Errorf("failed to fetch jobs for run #%d", run1.RunNumber)}
		}

		jobs2, err := m.client.FetchJobs(m.config.Owner, m.config.Repo, run2.ID)
		if err != nil || len(jobs2) == 0 {
			return ErrMsg{Err: fmt.Errorf("failed to fetch jobs for run #%d", run2.RunNumber)}
		}

		// Fetch logs for the first job of each run
		logs1, err := m.client.FetchJobLogs(m.config.Owner, m.config.Repo, jobs1[0].ID)
		if err != nil {
			logs1 = fmt.Sprintf("Error loading logs: %v", err)
		}

		logs2, err := m.client.FetchJobLogs(m.config.Owner, m.config.Repo, jobs2[0].ID)
		if err != nil {
			logs2 = fmt.Sprintf("Error loading logs: %v", err)
		}

		return CompareLogsLoadedMsg{Logs1: logs1, Logs2: logs2}
	}
}

// computeDiff computes a simple line-by-line diff between two log contents (v0.6)
func (m *Model) computeDiff(logs1, logs2 string) ([]string, []int) {
	lines1 := strings.Split(logs1, "\n")
	lines2 := strings.Split(logs2, "\n")

	var result []string
	var colors []int

	// Simple diff: show lines that differ
	// This is a basic implementation; a full diff algorithm would be more complex
	maxLen := len(lines1)
	if len(lines2) > maxLen {
		maxLen = len(lines2)
	}

	// Limit to 10000 lines for performance
	if maxLen > 10000 {
		maxLen = 10000
	}

	for i := 0; i < maxLen; i++ {
		var line1, line2 string
		if i < len(lines1) {
			line1 = lines1[i]
		}
		if i < len(lines2) {
			line2 = lines2[i]
		}

		if line1 == line2 {
			// Same line
			result = append(result, "  "+line1)
			colors = append(colors, 0)
		} else {
			// Different - show both with markers
			if line1 != "" {
				result = append(result, "- "+line1)
				colors = append(colors, -1) // removed
			}
			if line2 != "" {
				result = append(result, "+ "+line2)
				colors = append(colors, 1) // added
			}
		}
	}

	return result, colors
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

// triggerNotifications sends desktop notifications and executes hooks (v0.7)
func (m *Model) triggerNotifications() {
	if m.run == nil {
		return
	}

	conclusion := ""
	if m.run.Conclusion != nil {
		conclusion = *m.run.Conclusion
	}

	// Count job successes and failures
	successCount := 0
	failureCount := 0
	for _, job := range m.jobs {
		if job.Conclusion != nil {
			switch *job.Conclusion {
			case gh.ConclusionSuccess:
				successCount++
			case gh.ConclusionFailure:
				failureCount++
			}
		}
	}

	// Build notification data
	notifyData := notify.NotificationData{
		WorkflowName: m.run.Name,
		RunNumber:    m.run.RunNumber,
		Conclusion:   conclusion,
		Repo:         m.config.RepoSlug(),
		Branch:       m.config.Branch,
		HTMLURL:      m.run.HTMLURL,
	}

	// Build hook data
	hookData := notify.HookData{
		WorkflowName: m.run.Name,
		RunNumber:    m.run.RunNumber,
		RunID:        m.run.ID,
		Status:       m.run.Status,
		Conclusion:   conclusion,
		Repo:         m.config.RepoSlug(),
		Branch:       m.config.Branch,
		Event:        m.run.Event,
		Actor:        m.run.ActorLogin(),
		HTMLURL:      m.run.HTMLURL,
		JobCount:     len(m.jobs),
		SuccessCount: successCount,
		FailureCount: failureCount,
	}

	// Send desktop notification if enabled
	if m.config.Notify {
		notify.SendDesktopNotification(notifyData)
	}

	// Execute hook if configured
	if m.config.Hook != "" {
		notify.ExecuteHook(m.config.Hook, hookData)
	}
}
