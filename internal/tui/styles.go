package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/lance0/cimon/internal/gh"
)

// Status icons
const (
	IconSuccess    = "✓"
	IconFailure    = "✗"
	IconWarning    = "!"
	IconInProgress = "●"
	IconQueued     = "…"
	IconSkipped    = "-"
)

// Colors
var (
	ColorGreen  = lipgloss.Color("2")  // Green
	ColorRed    = lipgloss.Color("1")  // Red
	ColorYellow = lipgloss.Color("3")  // Yellow
	ColorDim    = lipgloss.Color("8")  // Dim gray
	ColorWhite  = lipgloss.Color("15") // White
	ColorCyan   = lipgloss.Color("6")  // Cyan
)

// Styles holds all the lipgloss styles used in the TUI
type Styles struct {
	// Header styles
	RepoName  lipgloss.Style
	Branch    lipgloss.Style
	Separator lipgloss.Style

	// Status badge styles
	StatusSuccess    lipgloss.Style
	StatusFailure    lipgloss.Style
	StatusInProgress lipgloss.Style
	StatusQueued     lipgloss.Style

	// Job table styles
	JobName     lipgloss.Style
	JobDuration lipgloss.Style
	JobTimeAgo  lipgloss.Style

	// Icon styles
	IconSuccess    lipgloss.Style
	IconFailure    lipgloss.Style
	IconInProgress lipgloss.Style
	IconQueued     lipgloss.Style
	IconSkipped    lipgloss.Style

	// Footer styles
	HelpKey  lipgloss.Style
	HelpDesc lipgloss.Style

	// General
	Dim      lipgloss.Style
	Bold     lipgloss.Style
	Selected lipgloss.Style

	// Error
	Error     lipgloss.Style
	ErrorHint lipgloss.Style

	// Watch indicator
	Watching lipgloss.Style

	// Log syntax highlighting (v0.6)
	LogError     lipgloss.Style
	LogWarning   lipgloss.Style
	LogCommand   lipgloss.Style
	LogGroup     lipgloss.Style
	LogTimestamp lipgloss.Style

	// Diff styles (v0.6)
	DiffAdded   lipgloss.Style
	DiffRemoved lipgloss.Style
}

// DefaultStyles returns the default style set
func DefaultStyles(colorEnabled bool) *Styles {
	if !colorEnabled {
		return &Styles{
			// Header
			RepoName:  lipgloss.NewStyle().Bold(true),
			Branch:    lipgloss.NewStyle(),
			Separator: lipgloss.NewStyle(),

			// Status badges
			StatusSuccess:    lipgloss.NewStyle().Bold(true),
			StatusFailure:    lipgloss.NewStyle().Bold(true),
			StatusInProgress: lipgloss.NewStyle().Bold(true),
			StatusQueued:     lipgloss.NewStyle(),

			// Job table
			JobName:     lipgloss.NewStyle(),
			JobDuration: lipgloss.NewStyle(),
			JobTimeAgo:  lipgloss.NewStyle(),

			// Icons
			IconSuccess:    lipgloss.NewStyle(),
			IconFailure:    lipgloss.NewStyle(),
			IconInProgress: lipgloss.NewStyle(),
			IconQueued:     lipgloss.NewStyle(),
			IconSkipped:    lipgloss.NewStyle(),

			// Footer
			HelpKey:  lipgloss.NewStyle(),
			HelpDesc: lipgloss.NewStyle(),

			// General
			Dim:      lipgloss.NewStyle(),
			Bold:     lipgloss.NewStyle().Bold(true),
			Selected: lipgloss.NewStyle(),

			// Error
			Error:     lipgloss.NewStyle(),
			ErrorHint: lipgloss.NewStyle(),

			// Watch
			Watching: lipgloss.NewStyle(),

			// Log syntax (no color)
			LogError:     lipgloss.NewStyle(),
			LogWarning:   lipgloss.NewStyle(),
			LogCommand:   lipgloss.NewStyle(),
			LogGroup:     lipgloss.NewStyle().Bold(true),
			LogTimestamp: lipgloss.NewStyle(),

			// Diff (no color)
			DiffAdded:   lipgloss.NewStyle(),
			DiffRemoved: lipgloss.NewStyle(),
		}
	}

	return &Styles{
		// Header
		RepoName:  lipgloss.NewStyle().Bold(true).Foreground(ColorWhite),
		Branch:    lipgloss.NewStyle().Foreground(ColorCyan),
		Separator: lipgloss.NewStyle().Foreground(ColorDim),

		// Status badges
		StatusSuccess:    lipgloss.NewStyle().Bold(true).Foreground(ColorGreen),
		StatusFailure:    lipgloss.NewStyle().Bold(true).Foreground(ColorRed),
		StatusInProgress: lipgloss.NewStyle().Bold(true).Foreground(ColorYellow),
		StatusQueued:     lipgloss.NewStyle().Foreground(ColorDim),

		// Job table
		JobName:     lipgloss.NewStyle().Foreground(ColorWhite),
		JobDuration: lipgloss.NewStyle().Foreground(ColorDim),
		JobTimeAgo:  lipgloss.NewStyle().Foreground(ColorDim),

		// Icons
		IconSuccess:    lipgloss.NewStyle().Foreground(ColorGreen),
		IconFailure:    lipgloss.NewStyle().Foreground(ColorRed),
		IconInProgress: lipgloss.NewStyle().Foreground(ColorYellow),
		IconQueued:     lipgloss.NewStyle().Foreground(ColorDim),
		IconSkipped:    lipgloss.NewStyle().Foreground(ColorDim),

		// Footer
		HelpKey:  lipgloss.NewStyle().Foreground(ColorCyan),
		HelpDesc: lipgloss.NewStyle().Foreground(ColorDim),

		// General
		Dim:      lipgloss.NewStyle().Foreground(ColorDim),
		Bold:     lipgloss.NewStyle().Bold(true),
		Selected: lipgloss.NewStyle().Background(lipgloss.Color("8")),

		// Error
		Error:     lipgloss.NewStyle().Foreground(ColorRed),
		ErrorHint: lipgloss.NewStyle().Foreground(ColorDim),

		// Watch
		Watching: lipgloss.NewStyle().Foreground(ColorYellow),

		// Log syntax highlighting
		LogError:     lipgloss.NewStyle().Foreground(ColorRed),
		LogWarning:   lipgloss.NewStyle().Foreground(ColorYellow),
		LogCommand:   lipgloss.NewStyle().Foreground(ColorCyan),
		LogGroup:     lipgloss.NewStyle().Bold(true).Foreground(ColorWhite),
		LogTimestamp: lipgloss.NewStyle().Foreground(ColorDim),

		// Diff styles
		DiffAdded:   lipgloss.NewStyle().Foreground(ColorGreen),
		DiffRemoved: lipgloss.NewStyle().Foreground(ColorRed),
	}
}

// StatusIcon returns the appropriate icon for a status/conclusion combination
func StatusIcon(status string, conclusion *string) string {
	switch status {
	case gh.StatusQueued:
		return IconQueued
	case gh.StatusInProgress:
		return IconInProgress
	case gh.StatusCompleted:
		if conclusion == nil {
			return IconSkipped
		}
		switch *conclusion {
		case gh.ConclusionSuccess:
			return IconSuccess
		case gh.ConclusionFailure:
			return IconFailure
		case gh.ConclusionCancelled, gh.ConclusionTimedOut, gh.ConclusionActionRequired:
			return IconWarning
		case gh.ConclusionSkipped, gh.ConclusionNeutral:
			return IconSkipped
		default:
			return IconSkipped
		}
	default:
		return IconQueued
	}
}

// StatusIconStyled returns a styled icon for a status/conclusion
func (s *Styles) StatusIconStyled(status string, conclusion *string) string {
	icon := StatusIcon(status, conclusion)

	switch status {
	case gh.StatusQueued:
		return s.IconQueued.Render(icon)
	case gh.StatusInProgress:
		return s.IconInProgress.Render(icon)
	case gh.StatusCompleted:
		if conclusion == nil {
			return s.IconSkipped.Render(icon)
		}
		switch *conclusion {
		case gh.ConclusionSuccess:
			return s.IconSuccess.Render(icon)
		case gh.ConclusionFailure:
			return s.IconFailure.Render(icon)
		case gh.ConclusionCancelled, gh.ConclusionTimedOut, gh.ConclusionActionRequired:
			return s.IconFailure.Render(icon)
		default:
			return s.IconSkipped.Render(icon)
		}
	default:
		return s.IconQueued.Render(icon)
	}
}

// StatusBadge returns a styled status badge text
func (s *Styles) StatusBadge(status string, conclusion *string) string {
	switch status {
	case gh.StatusQueued:
		return s.StatusQueued.Render("QUEUED")
	case gh.StatusInProgress:
		return s.StatusInProgress.Render("IN PROGRESS")
	case gh.StatusCompleted:
		if conclusion == nil {
			return s.Dim.Render("UNKNOWN")
		}
		switch *conclusion {
		case gh.ConclusionSuccess:
			return s.StatusSuccess.Render("PASSED")
		case gh.ConclusionFailure:
			return s.StatusFailure.Render("FAILED")
		case gh.ConclusionCancelled:
			return s.StatusFailure.Render("CANCELLED")
		case gh.ConclusionTimedOut:
			return s.StatusFailure.Render("TIMED OUT")
		case gh.ConclusionActionRequired:
			return s.StatusFailure.Render("ACTION REQUIRED")
		case gh.ConclusionSkipped:
			return s.Dim.Render("SKIPPED")
		case gh.ConclusionNeutral:
			return s.Dim.Render("NEUTRAL")
		default:
			return s.Dim.Render(*conclusion)
		}
	default:
		return s.Dim.Render(status)
	}
}
