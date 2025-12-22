package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
)

// View implements tea.Model
func (m Model) View() string {
	switch m.state {
	case StateLoading:
		return m.viewLoading()
	case StateError:
		return m.viewError()
	case StateJobDetails:
		return m.viewJobDetails()
	case StateLogViewer:
		return m.viewLogViewer()
	case StateBranchSelection:
		return m.viewBranchSelection()
	case StateStatusFilter:
		return m.viewStatusFilter()
	default:
		return m.viewReady()
	}
}

func (m Model) viewLoading() string {
	message := m.loadingMessage
	if message == "" {
		message = "Fetching latest run..."
	}
	return fmt.Sprintf("\n  %s %s\n", m.spinner.View(), message)
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
		b.WriteString(m.styles.ErrorHint.Render("  Suggestion: "))
		b.WriteString(hint)
		b.WriteString("\n\n")
	}

	// Add recovery options
	b.WriteString("  Press 'r' to retry or 'q' to quit\n")

	return b.String()
}

func (m Model) getErrorHint() string {
	if m.err == nil {
		return ""
	}

	errStr := strings.ToLower(m.err.Error())

	if strings.Contains(errStr, "authentication") || strings.Contains(errStr, "401") {
		return "Run 'gh auth login' to authenticate with GitHub, or set GITHUB_TOKEN environment variable"
	}
	if strings.Contains(errStr, "403") || strings.Contains(errStr, "forbidden") {
		return "Check that you have access to this repository and the correct permissions"
	}
	if strings.Contains(errStr, "not found") || strings.Contains(errStr, "404") {
		return "Verify the repository exists and the branch name is correct"
	}
	if strings.Contains(errStr, "rate limit") || strings.Contains(errStr, "429") || strings.Contains(errStr, "too many requests") {
		return "GitHub API rate limit exceeded - wait a few minutes before retrying"
	}
	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "connection") {
		return "Network connectivity issue - check your internet connection and try again"
	}
	if strings.Contains(errStr, "502") || strings.Contains(errStr, "503") || strings.Contains(errStr, "504") {
		return "GitHub servers are temporarily unavailable - try again in a moment"
	}
	if strings.Contains(errStr, "no workflow runs") {
		return "No CI runs found - push a commit or check that workflows are configured for this branch"
	}
	if strings.Contains(errStr, "detached head") {
		return "Currently in detached HEAD state - checkout a branch or use --branch flag"
	}

	return "Press 'r' to retry the operation or check your configuration"
}

func (m Model) viewReady() string {
	if m.showingJobDetails {
		return m.viewSplit()
	}

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
		b.WriteString("\n  No jobs available\n")
	} else if len(m.runs) > 0 {
		b.WriteString("\n  Run history available - use h/l to navigate\n")
	} else {
		b.WriteString("\n  No workflow data available\n")
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

	// Show current filter if active
	if m.currentStatusFilter != "" {
		filterLabels := map[string]string{
			"success":     "✓",
			"failure":     "✗",
			"in_progress": "●",
			"completed":   "○",
			"queued":      "…",
		}
		if icon, ok := filterLabels[m.currentStatusFilter]; ok {
			filterInfo := fmt.Sprintf(" [%s]", icon)
			b.WriteString(m.styles.Separator.Render(filterInfo))
		}
	}

	// Show run navigation info if we have multiple runs
	if len(m.runs) > 1 {
		runInfo := fmt.Sprintf(" [%d/%d]", m.selectedRunIndex+1, len(m.runs))
		b.WriteString(m.styles.Separator.Render(runInfo))
	}

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

	var bindings []key.Binding
	if m.state == StateStatusFilter {
		// In status filter, show navigation and selection options
		bindings = []key.Binding{m.keys.Up, m.keys.Down, m.keys.Enter, m.keys.Filter, m.keys.Quit}
	} else if m.state == StateBranchSelection {
		// In branch selection, show navigation and selection options
		bindings = []key.Binding{m.keys.Up, m.keys.Down, m.keys.Enter, m.keys.BranchSelect, m.keys.Quit}
	} else if m.state == StateLogViewer {
		// In log viewer, show navigation and exit options
		if m.logSearchTerm != "" && len(m.logSearchMatches) > 0 {
			bindings = []key.Binding{m.keys.Up, m.keys.Down, m.keys.NextMatch, m.keys.PrevMatch, m.keys.Logs, m.keys.Quit}
		} else {
			bindings = []key.Binding{m.keys.Up, m.keys.Down, m.keys.Search, m.keys.Logs, m.keys.Quit}
		}
	} else if len(m.jobs) > 0 && !m.showingJobDetails && len(m.runs) > 1 {
		// Show run navigation, Enter and Logs keys when multiple runs available
		bindings = []key.Binding{m.keys.Refresh, m.keys.Watch, m.keys.Open, m.keys.PrevRun, m.keys.NextRun, m.keys.BranchSelect, m.keys.Filter, m.keys.Enter, m.keys.Logs, m.keys.Quit}
	} else if len(m.jobs) > 0 && !m.showingJobDetails {
		// Show Enter and Logs keys when jobs are available and not in details mode
		bindings = []key.Binding{m.keys.Refresh, m.keys.Watch, m.keys.Open, m.keys.BranchSelect, m.keys.Filter, m.keys.Enter, m.keys.Logs, m.keys.Quit}
	} else if m.showingJobDetails {
		// Show Enter and Logs keys in job details mode
		bindings = []key.Binding{m.keys.Refresh, m.keys.Open, m.keys.Logs, m.keys.Enter, m.keys.Quit}
	} else {
		bindings = []key.Binding{m.keys.Refresh, m.keys.Watch, m.keys.BranchSelect, m.keys.Filter, m.keys.Quit}
	}

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

func (m Model) viewSplit() string {
	var b strings.Builder

	// Header
	b.WriteString(m.viewHeader())
	b.WriteString("\n")

	// Run summary
	if m.run != nil {
		b.WriteString(m.viewRunSummary())
		b.WriteString("\n")
	}

	// Split view: jobs on left, details on right
	leftWidth := m.width / 2
	if m.width > 80 {
		leftWidth = m.width * 3 / 5 // 60% for jobs, 40% for details
	}

	jobsView := m.viewJobsList(leftWidth)
	detailsView := m.viewJobDetailsPanel(m.width - leftWidth - 3) // -3 for separator

	// Combine with separator
	linesJobs := strings.Split(strings.TrimSuffix(jobsView, "\n"), "\n")
	linesDetails := strings.Split(strings.TrimSuffix(detailsView, "\n"), "\n")

	maxLines := len(linesJobs)
	if len(linesDetails) > maxLines {
		maxLines = len(linesDetails)
	}

	for i := 0; i < maxLines; i++ {
		if i < len(linesJobs) {
			b.WriteString(linesJobs[i])
		} else {
			b.WriteString(strings.Repeat(" ", leftWidth))
		}

		b.WriteString(" │ ")

		if i < len(linesDetails) {
			b.WriteString(linesDetails[i])
		}

		b.WriteString("\n")
	}

	// Footer
	b.WriteString(m.viewFooter())

	return b.String()
}

func (m Model) viewJobsList(width int) string {
	var b strings.Builder

	b.WriteString("Jobs:\n")

	for i, job := range m.jobs {
		// Icon
		b.WriteString("  ")
		b.WriteString(m.styles.StatusIconStyled(job.Status, job.Conclusion))
		b.WriteString(" ")

		// Job name (highlight if selected)
		name := job.Name
		if len(name) > width-8 { // Truncate if too long
			name = name[:width-11] + "..."
		}
		if i == m.cursor {
			b.WriteString(m.styles.Selected.Render(name))
		} else {
			b.WriteString(m.styles.JobName.Render(name))
		}

		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) viewJobDetailsPanel(width int) string {
	if m.selectedJob == nil {
		return "Job Details:\n  Loading..."
	}

	var b strings.Builder

	job := m.selectedJob

	b.WriteString("Job Details:\n")

	// Job name and status
	b.WriteString("  ")
	b.WriteString(m.styles.StatusIconStyled(job.Status, job.Conclusion))
	b.WriteString(" ")
	b.WriteString(m.styles.JobName.Render(job.Name))
	b.WriteString("\n")

	// Job metadata
	if job.RunnerName != "" {
		b.WriteString("  Runner: ")
		b.WriteString(m.styles.Dim.Render(job.RunnerName))
		b.WriteString("\n")
	}

	if job.StartedAt != nil {
		b.WriteString("  Started: ")
		b.WriteString(m.styles.Dim.Render(job.StartedAt.Format("15:04:05")))
		b.WriteString("\n")
	}

	if job.CompletedAt != nil {
		b.WriteString("  Completed: ")
		b.WriteString(m.styles.Dim.Render(job.CompletedAt.Format("15:04:05")))
		b.WriteString("\n")
	}

	// Steps
	if len(job.Steps) > 0 {
		b.WriteString("  Steps:\n")

		for i, step := range job.Steps {
			b.WriteString("    ")
			b.WriteString(m.styles.StatusIconStyled(step.Status, step.Conclusion))
			b.WriteString(" ")

			stepName := step.Name
			if len(stepName) > width-12 { // Truncate if too long
				stepName = stepName[:width-15] + "..."
			}

			if i == m.jobDetailsCursor {
				b.WriteString(m.styles.Selected.Render(stepName))
			} else {
				b.WriteString(m.styles.JobName.Render(stepName))
			}

			b.WriteString("\n")
		}
	} else {
		b.WriteString("  No steps available\n")
	}

	return b.String()
}

func (m Model) viewBranchSelection() string {
	var b strings.Builder

	b.WriteString("Select Branch\n\n")

	if len(m.branches) == 0 {
		b.WriteString("  ")
		b.WriteString(m.styles.Dim.Render("Loading branches"))
		b.WriteString(" ")
		b.WriteString(m.spinner.View())
		b.WriteString("\n")
	} else {
		for i, branch := range m.branches {
			if i == m.selectedBranchIndex {
				b.WriteString(m.styles.Selected.Render("→ "))
			} else {
				b.WriteString("  ")
			}

			// Show branch name
			if branch.Name == m.config.Branch {
				b.WriteString(m.styles.StatusSuccess.Render(branch.Name))
				b.WriteString(" (current)")
			} else {
				b.WriteString(branch.Name)
			}

			// Show protection status
			if branch.Protected {
				b.WriteString(" 🔒")
			}

			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(m.viewFooter())

	return b.String()
}

func (m Model) viewStatusFilter() string {
	var b strings.Builder

	b.WriteString("Filter by Status\n\n")

	filterLabels := map[string]string{
		"":            "All",
		"success":     "Success",
		"failure":     "Failure",
		"in_progress": "In Progress",
		"completed":   "Completed",
		"queued":      "Queued",
	}

	for i, filterValue := range m.statusFilterOptions {
		if i == m.selectedFilterIndex {
			b.WriteString(m.styles.Selected.Render("→ "))
		} else {
			b.WriteString("  ")
		}

		label := filterLabels[filterValue]
		if filterValue == m.currentStatusFilter {
			b.WriteString(m.styles.StatusSuccess.Render(label))
			b.WriteString(" (current)")
		} else {
			b.WriteString(label)
		}

		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(m.viewFooter())

	return b.String()
}

func (m Model) viewJobDetails() string {
	var b strings.Builder

	// Header
	b.WriteString(m.viewHeader())
	b.WriteString("\n")

	b.WriteString("Job Details\n\n")

	if m.selectedJob == nil {
		b.WriteString("Loading job details...")
	} else {
		job := m.selectedJob

		// Job info
		b.WriteString("Job: ")
		b.WriteString(m.styles.JobName.Render(job.Name))
		b.WriteString("\n")

		b.WriteString("Status: ")
		b.WriteString(m.styles.StatusBadge(job.Status, job.Conclusion))
		b.WriteString("\n")

		if job.RunnerName != "" {
			b.WriteString("Runner: ")
			b.WriteString(m.styles.Dim.Render(job.RunnerName))
			b.WriteString("\n")
		}

		if job.StartedAt != nil {
			b.WriteString("Started: ")
			b.WriteString(m.styles.Dim.Render(job.StartedAt.Format("2006-01-02 15:04:05")))
			b.WriteString("\n")
		}

		if job.CompletedAt != nil {
			b.WriteString("Completed: ")
			b.WriteString(m.styles.Dim.Render(job.CompletedAt.Format("2006-01-02 15:04:05")))
			b.WriteString("\n")
		}

		// Steps
		if len(job.Steps) > 0 {
			b.WriteString("\nSteps:\n")

			for i, step := range job.Steps {
				b.WriteString("  ")
				b.WriteString(m.styles.StatusIconStyled(step.Status, step.Conclusion))
				b.WriteString(" ")

				if i == m.jobDetailsCursor {
					b.WriteString(m.styles.Selected.Render(step.Name))
				} else {
					b.WriteString(m.styles.JobName.Render(step.Name))
				}

				b.WriteString("\n")
			}
		}
	}

	// Footer
	b.WriteString("\n")
	b.WriteString(m.viewFooter())

	return b.String()
}

func (m Model) viewLogViewer() string {
	var b strings.Builder

	// Header
	b.WriteString(m.viewHeader())
	b.WriteString("\n")

	// Title with streaming indicator
	b.WriteString("Job Logs")
	if m.logStreaming {
		b.WriteString(m.styles.Watching.Render(" [LIVE]"))
	}
	b.WriteString("\n\n")

	if m.logContent == "" {
		b.WriteString("  ")
		b.WriteString(m.styles.Dim.Render("Loading logs"))
		b.WriteString(" ")
		b.WriteString(m.spinner.View())
		b.WriteString("\n")
	} else {
		// Split log content into lines
		lines := strings.Split(strings.TrimSuffix(m.logContent, "\n"), "\n")

		// Calculate visible area (reserve space for header and footer)
		maxLines := m.height - 10 // Reserve more space for streaming indicator

		// Ensure scroll offset is valid
		if m.logScrollOffset < 0 {
			m.logScrollOffset = 0
		}
		if m.logScrollOffset > len(lines)-maxLines && len(lines) > maxLines {
			m.logScrollOffset = len(lines) - maxLines
		}

		// Display visible lines
		start := m.logScrollOffset
		end := start + maxLines
		if end > len(lines) {
			end = len(lines)
		}

		for i := start; i < end; i++ {
			line := lines[i]

			// Highlight search matches
			if m.logSearchTerm != "" {
				lowerLine := strings.ToLower(line)
				lowerTerm := strings.ToLower(m.logSearchTerm)
				if strings.Contains(lowerLine, lowerTerm) {
					// Simple highlighting - could be improved
					line = strings.ReplaceAll(line, m.logSearchTerm, m.styles.StatusFailure.Render(m.logSearchTerm))
				}
			}

			// Truncate long lines to fit width
			if len(line) > m.width-4 {
				line = line[:m.width-7] + "..."
			}
			b.WriteString(line)
			b.WriteString("\n")
		}

		// Show status information
		var statusParts []string

		if len(lines) > maxLines {
			scrollPercent := float64(m.logScrollOffset) / float64(len(lines)-maxLines) * 100
			statusParts = append(statusParts, fmt.Sprintf("Line %d/%d (%.0f%%)", m.logScrollOffset+1, len(lines), scrollPercent))
		}

		if m.logStreaming {
			statusParts = append(statusParts, "STREAMING")
		}

		if m.logSearchTerm != "" {
			if len(m.logSearchMatches) > 0 {
				statusParts = append(statusParts, fmt.Sprintf("Search: '%s' (%d/%d)", m.logSearchTerm, m.logSearchIndex+1, len(m.logSearchMatches)))
			} else {
				statusParts = append(statusParts, fmt.Sprintf("Search: '%s' (no matches)", m.logSearchTerm))
			}
		}

		if len(statusParts) > 0 {
			b.WriteString(fmt.Sprintf("\n[%s]", strings.Join(statusParts, " | ")))
		}
	}

	// Footer
	b.WriteString("\n")
	b.WriteString(m.viewFooter())

	return b.String()
}
