package tui

import (
	"fmt"
	"strings"
	"time"
)

// View implements tea.Model
func (m Model) View() string {
	switch m.state {
	case StateLoading:
		return m.viewLoading()
	case StateError:
		return m.viewError()
	default:
		return m.viewReady()
	}
}

func (m Model) viewLoading() string {
	return fmt.Sprintf("\n  %s Fetching latest run...\n", m.spinner.View())
}

func (m Model) viewError() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(m.styles.Error.Render("  Error: "))
	b.WriteString(m.err.Error())
	b.WriteString("\n\n")

	// Add hints based on error type
	hint := m.getErrorHint()
	if hint != "" {
		b.WriteString(m.styles.ErrorHint.Render("  " + hint))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(m.viewFooter())

	return b.String()
}

func (m Model) getErrorHint() string {
	if m.err == nil {
		return ""
	}

	errStr := m.err.Error()

	if strings.Contains(errStr, "authentication") || strings.Contains(errStr, "401") || strings.Contains(errStr, "403") {
		return "Install gh and run 'gh auth login' or set GITHUB_TOKEN"
	}
	if strings.Contains(errStr, "not found") || strings.Contains(errStr, "404") {
		return "Check that the repository exists and you have access"
	}
	if strings.Contains(errStr, "rate limit") || strings.Contains(errStr, "429") {
		return "Rate limited - wait a moment and try again"
	}
	if strings.Contains(errStr, "no workflow runs") {
		return "No CI runs found for this branch - push a commit to trigger a workflow"
	}

	return ""
}

func (m Model) viewReady() string {
	var b strings.Builder

	// Header
	b.WriteString(m.viewHeader())
	b.WriteString("\n")

	// Run summary
	if m.run != nil {
		b.WriteString(m.viewRunSummary())
		b.WriteString("\n")
	}

	// Jobs table
	if len(m.jobs) > 0 {
		b.WriteString(m.viewJobs())
	} else if m.run != nil {
		b.WriteString("\n  No jobs found\n")
	}

	// Footer
	b.WriteString("\n")
	b.WriteString(m.viewFooter())

	return b.String()
}

func (m Model) viewHeader() string {
	var b strings.Builder

	b.WriteString("\n  ")
	b.WriteString(m.styles.RepoName.Render(m.config.RepoSlug()))
	b.WriteString(m.styles.Separator.Render(" • "))
	b.WriteString(m.styles.Branch.Render(m.config.Branch))

	if m.watching {
		b.WriteString("  ")
		b.WriteString(m.styles.Watching.Render("◉ Watching"))
	}

	b.WriteString("\n")

	return b.String()
}

func (m Model) viewRunSummary() string {
	var b strings.Builder

	run := m.run

	b.WriteString("  ")

	// Workflow name and run number
	if run.Name != "" {
		b.WriteString(m.styles.Dim.Render(run.Name))
		b.WriteString(m.styles.Separator.Render(" #"))
		b.WriteString(m.styles.Dim.Render(fmt.Sprintf("%d", run.RunNumber)))
		b.WriteString("  ")
	}

	// Status badge
	b.WriteString(m.styles.StatusBadge(run.Status, run.Conclusion))

	// Event and actor
	b.WriteString("\n  ")
	b.WriteString(m.styles.Dim.Render(run.Event))
	if actor := run.ActorLogin(); actor != "" {
		b.WriteString(m.styles.Dim.Render(" by "))
		b.WriteString(m.styles.Dim.Render(actor))
	}

	// Time ago
	b.WriteString(m.styles.Separator.Render(" • "))
	b.WriteString(m.styles.Dim.Render(timeAgo(run.UpdatedAt)))

	b.WriteString("\n")

	return b.String()
}

func (m Model) viewJobs() string {
	var b strings.Builder

	b.WriteString("\n")

	for i, job := range m.jobs {
		// Icon
		b.WriteString("  ")
		b.WriteString(m.styles.StatusIconStyled(job.Status, job.Conclusion))
		b.WriteString(" ")

		// Job name (highlight if selected)
		name := job.Name
		if i == m.cursor {
			b.WriteString(m.styles.Selected.Render(name))
		} else {
			b.WriteString(m.styles.JobName.Render(name))
		}

		// Duration (if completed)
		if job.IsCompleted() && job.Duration() > 0 {
			b.WriteString("  ")
			b.WriteString(m.styles.JobDuration.Render(formatDuration(job.Duration())))
		}

		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) viewFooter() string {
	var b strings.Builder

	b.WriteString("  ")

	bindings := m.keys.ShortHelp()
	for i, binding := range bindings {
		if i > 0 {
			b.WriteString("  ")
		}
		b.WriteString(m.styles.HelpKey.Render(binding.Help().Key))
		b.WriteString(" ")
		b.WriteString(m.styles.HelpDesc.Render(binding.Help().Desc))
	}

	b.WriteString("\n")

	return b.String()
}

// timeAgo returns a human-readable relative time string
func timeAgo(t time.Time) string {
	d := time.Since(t)

	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		mins := int(d.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case d < 24*time.Hour:
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	default:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}

// formatDuration formats a duration as a human-readable string
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		secs := int(d.Seconds()) % 60
		if secs == 0 {
			return fmt.Sprintf("%dm", mins)
		}
		return fmt.Sprintf("%dm %ds", mins, secs)
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh %dm", hours, mins)
}
