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
	case StateHelp:
		return m.viewHelp()
	case StateWorkflowViewer:
		return m.viewWorkflowViewer()
	case StateArtifactSelection:
		return m.viewArtifactSelection()
	case StateLogFilter:
		return m.viewLogFilter()
	case StateMultiJobSelect:
		return m.viewMultiJobSelect()
	case StateCompareSelect:
		return m.viewCompareSelect()
	case StateCompareView:
		return m.viewCompareView()
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
			bindings = []key.Binding{m.keys.Up, m.keys.Down, m.keys.NextMatch, m.keys.PrevMatch, m.keys.LogFilter, m.keys.Logs, m.keys.Quit}
		} else if m.multiJobMode {
			// Show view toggle in multi-job mode
			bindings = []key.Binding{m.keys.Up, m.keys.Down, m.keys.Search, m.keys.LogViewToggle, m.keys.LogSave, m.keys.Logs, m.keys.Quit}
		} else {
			bindings = []key.Binding{m.keys.Up, m.keys.Down, m.keys.Search, m.keys.LogFilter, m.keys.LogSave, m.keys.LogHighlight, m.keys.Logs, m.keys.Quit}
		}
	} else if len(m.jobs) > 0 && !m.showingJobDetails && len(m.runs) > 1 {
		// Show run navigation, Enter and Logs keys when multiple runs available
		bindings = []key.Binding{m.keys.Refresh, m.keys.Watch, m.keys.Open, m.keys.PrevRun, m.keys.NextRun, m.keys.BranchSelect, m.keys.Filter, m.keys.LogMulti, m.keys.LogCompare, m.keys.Enter, m.keys.Logs, m.keys.Quit}
	} else if len(m.jobs) > 0 && !m.showingJobDetails {
		// Show Enter and Logs keys when jobs are available and not in details mode
		bindings = []key.Binding{m.keys.Refresh, m.keys.Watch, m.keys.Open, m.keys.BranchSelect, m.keys.Filter, m.keys.LogMulti, m.keys.Enter, m.keys.Logs, m.keys.Quit}
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

	// Footer
	b.WriteString("\n")
	b.WriteString(m.viewFooter())

	return b.String()
}

func (m Model) viewArtifactSelection() string {
	var b strings.Builder

	b.WriteString("Select Artifact to Download\n\n")

	if len(m.artifacts) == 0 {
		b.WriteString("  No artifacts available for this workflow run\n")
	} else {
		for i, artifact := range m.artifacts {
			if i == m.selectedArtifactIndex {
				b.WriteString(m.styles.Selected.Render("→ "))
			} else {
				b.WriteString("  ")
			}

			b.WriteString(artifact.Name)
			b.WriteString(" (")
			b.WriteString(fmt.Sprintf("%d bytes", artifact.SizeInBytes))
			b.WriteString(")")

			if artifact.Expired {
				b.WriteString(" EXPIRED")
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

	// Title with mode indicators
	b.WriteString("Job Logs")
	if m.logStreaming {
		b.WriteString(m.styles.Watching.Render(" [LIVE]"))
	}
	if m.logSyntaxEnabled {
		b.WriteString(m.styles.Branch.Render(" [SYNTAX]"))
	}
	if len(m.logFilterStepNumbers) > 0 {
		b.WriteString(m.styles.LogWarning.Render(fmt.Sprintf(" [FILTER: %d steps]", len(m.logFilterStepNumbers))))
	}
	if m.multiJobMode {
		b.WriteString(m.styles.Branch.Render(fmt.Sprintf(" [MULTI: %d jobs]", len(m.multiJobIDs))))
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

			// Truncate long lines to fit width first
			if len(line) > m.width-4 {
				line = line[:m.width-7] + "..."
			}

			// Apply syntax highlighting (v0.6)
			line = m.viewLogLine(line)

			// Highlight search matches (overlay on top of syntax highlighting)
			if m.logSearchTerm != "" {
				lowerLine := strings.ToLower(line)
				lowerTerm := strings.ToLower(m.logSearchTerm)
				if strings.Contains(lowerLine, lowerTerm) {
					// Simple highlighting - could be improved
					line = strings.ReplaceAll(line, m.logSearchTerm, m.styles.StatusFailure.Render(m.logSearchTerm))
				}
			}

			b.WriteString(line)
			b.WriteString("\n")
		}

		// Show search input if in search mode
		if m.searchInputMode {
			b.WriteString(fmt.Sprintf("\nSearch: %s_", m.searchInputBuffer))
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

		if m.logSearchTerm != "" && !m.searchInputMode {
			if len(m.logSearchMatches) > 0 {
				statusParts = append(statusParts, fmt.Sprintf("Search: '%s' (%d/%d)", m.logSearchTerm, m.logSearchIndex+1, len(m.logSearchMatches)))
			} else {
				statusParts = append(statusParts, fmt.Sprintf("Search: '%s' (no matches)", m.logSearchTerm))
			}
		}

		if len(statusParts) > 0 {
			b.WriteString(fmt.Sprintf("\n[%s]", strings.Join(statusParts, " | ")))
		}

		// Show export message (v0.6) - auto-clear after 3 seconds
		if m.logExportMessage != "" && time.Since(m.logExportTime) < 3*time.Second {
			b.WriteString("\n")
			b.WriteString(m.styles.StatusSuccess.Render(m.logExportMessage))
		}
	}

	// Footer
	b.WriteString("\n")
	b.WriteString(m.viewFooter())

	return b.String()
}

func (m Model) viewHelp() string {
	var b strings.Builder

	b.WriteString("Keyboard Shortcuts\n\n")

	// Group shortcuts by category
	sections := []struct {
		title string
		keys  []key.Binding
	}{
		{
			title: "Navigation",
			keys:  []key.Binding{m.keys.Up, m.keys.Down, m.keys.NextRun, m.keys.PrevRun},
		},
		{
			title: "Actions",
			keys:  []key.Binding{m.keys.Refresh, m.keys.Watch, m.keys.Open, m.keys.Enter},
		},
		{
			title: "Filtering & Selection",
			keys:  []key.Binding{m.keys.BranchSelect, m.keys.Filter, m.keys.Logs, m.keys.Search, m.keys.Workflow, m.keys.Artifacts},
		},
		{
			title: "Search Navigation",
			keys:  []key.Binding{m.keys.NextMatch, m.keys.PrevMatch},
		},
		{
			title: "General",
			keys:  []key.Binding{m.keys.Quit, m.keys.Help},
		},
	}

	for _, section := range sections {
		b.WriteString(m.styles.Bold.Render(section.title))
		b.WriteString("\n")

		for _, binding := range section.keys {
			help := binding.Help()
			if help.Key != "" {
				b.WriteString("  ")
				b.WriteString(m.styles.HelpKey.Render(help.Key))
				b.WriteString("  ")
				b.WriteString(help.Desc)
				b.WriteString("\n")
			}
		}

		b.WriteString("\n")
	}

	b.WriteString("Press any key to exit help\n")

	return b.String()
}

func (m Model) viewWorkflowViewer() string {
	var b strings.Builder

	// Header
	b.WriteString(m.viewHeader())
	b.WriteString("\n")

	// Title with file path
	b.WriteString("Workflow Configuration")
	b.WriteString("\n")
	b.WriteString(m.styles.Dim.Render(m.workflowPath))
	b.WriteString("\n\n")

	if m.workflowContent == "" {
		b.WriteString("  ")
		b.WriteString(m.styles.Dim.Render("Loading workflow content"))
		b.WriteString(" ")
		b.WriteString(m.spinner.View())
		b.WriteString("\n")
	} else {
		// Split workflow content into lines
		lines := strings.Split(strings.TrimSuffix(m.workflowContent, "\n"), "\n")

		// Calculate visible area (reserve space for header and footer)
		maxLines := m.height - 10

		// Ensure scroll offset is valid
		if m.workflowScrollOffset < 0 {
			m.workflowScrollOffset = 0
		}
		if m.workflowScrollOffset > len(lines)-maxLines && len(lines) > maxLines {
			m.workflowScrollOffset = len(lines) - maxLines
		}

		// Display visible lines
		start := m.workflowScrollOffset
		end := start + maxLines
		if end > len(lines) {
			end = len(lines)
		}

		for i := start; i < end; i++ {
			line := lines[i]

			// Truncate long lines to fit width
			if len(line) > m.width-4 {
				line = line[:m.width-7] + "..."
			}
			b.WriteString(line)
			b.WriteString("\n")
		}

		// Show scroll status
		if len(lines) > maxLines {
			scrollPercent := float64(m.workflowScrollOffset) / float64(len(lines)-maxLines) * 100
			b.WriteString(fmt.Sprintf("\n[Line %d/%d (%.0f%%)]", m.workflowScrollOffset+1, len(lines), scrollPercent))
		}
	}

	// Footer
	b.WriteString("\n")
	b.WriteString(m.viewFooter())

	return b.String()
}

// viewLogLine applies syntax highlighting to a log line (v0.6)
func (m Model) viewLogLine(line string) string {
	if !m.logSyntaxEnabled {
		return line
	}

	// GitHub Actions error/warning markers
	if strings.Contains(line, "##[error]") {
		return m.styles.LogError.Render(line)
	}
	if strings.Contains(line, "##[warning]") {
		return m.styles.LogWarning.Render(line)
	}

	// Group markers
	if strings.HasPrefix(line, "##[group]") || strings.HasPrefix(line, "##[endgroup]") {
		return m.styles.LogGroup.Render(line)
	}

	// Common error patterns
	lowerLine := strings.ToLower(line)
	if strings.Contains(lowerLine, "error:") ||
		strings.Contains(lowerLine, "fatal:") ||
		strings.Contains(lowerLine, "failed:") ||
		strings.Contains(lowerLine, "exception:") ||
		strings.Contains(lowerLine, "panic:") {
		return m.styles.LogError.Render(line)
	}

	// Common warning patterns
	if strings.Contains(lowerLine, "warning:") ||
		strings.Contains(lowerLine, "warn:") ||
		strings.Contains(lowerLine, "deprecated:") {
		return m.styles.LogWarning.Render(line)
	}

	// Command execution patterns
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "Run ") ||
		strings.HasPrefix(trimmed, "+ ") ||
		strings.HasPrefix(trimmed, "$ ") ||
		strings.HasPrefix(trimmed, "> ") {
		return m.styles.LogCommand.Render(line)
	}

	// Timestamp at start of line (e.g., "2024-01-15T12:34:56.789Z")
	if len(line) >= 24 && line[4] == '-' && line[7] == '-' && line[10] == 'T' {
		return m.styles.LogTimestamp.Render(line[:24]) + line[24:]
	}

	return line
}

// viewLogFilter displays the log filter step selection (v0.6)
func (m Model) viewLogFilter() string {
	var b strings.Builder

	b.WriteString("Filter Log Steps\n\n")

	if m.parsedLogs == nil || len(m.parsedLogs.Steps) == 0 {
		b.WriteString("  No steps available\n")
	} else {
		b.WriteString("  Select steps to display (space to toggle, F/enter to apply):\n\n")

		for i, step := range m.parsedLogs.Steps {
			// Selection cursor
			if i == m.logFilterIndex {
				b.WriteString(m.styles.Selected.Render("→ "))
			} else {
				b.WriteString("  ")
			}

			// Checkbox
			if m.isStepSelected(step.Number) {
				b.WriteString("[✓] ")
			} else {
				b.WriteString("[ ] ")
			}

			// Step number and name
			stepLabel := fmt.Sprintf("%d. %s", step.Number, step.Name)
			if len(stepLabel) > m.width-10 {
				stepLabel = stepLabel[:m.width-13] + "..."
			}
			b.WriteString(stepLabel)
			b.WriteString("\n")
		}

		// Show current selection summary
		b.WriteString("\n")
		if len(m.logFilterStepNumbers) == 0 {
			b.WriteString(m.styles.Dim.Render("  (no filter - showing all steps)"))
		} else {
			b.WriteString(m.styles.Dim.Render(fmt.Sprintf("  (%d step(s) selected)", len(m.logFilterStepNumbers))))
		}
		b.WriteString("\n")
	}

	// Footer with key hints
	b.WriteString("\n")
	b.WriteString("  ")
	b.WriteString(m.styles.HelpKey.Render("space"))
	b.WriteString(" toggle  ")
	b.WriteString(m.styles.HelpKey.Render("F/enter"))
	b.WriteString(" apply  ")
	b.WriteString(m.styles.HelpKey.Render("esc"))
	b.WriteString(" cancel\n")

	return b.String()
}

// viewMultiJobSelect displays the multi-job selection UI (v0.6)
func (m Model) viewMultiJobSelect() string {
	var b strings.Builder

	b.WriteString("Select Jobs to Follow\n\n")

	if len(m.jobs) == 0 {
		b.WriteString("  No jobs available\n")
	} else {
		b.WriteString("  Select up to 4 jobs to view simultaneously (space to toggle):\n\n")

		for i, job := range m.jobs {
			// Selection cursor
			if i == m.multiJobSelectIdx {
				b.WriteString(m.styles.Selected.Render("→ "))
			} else {
				b.WriteString("  ")
			}

			// Checkbox
			if m.isJobSelected(job.ID) {
				b.WriteString("[✓] ")
			} else {
				b.WriteString("[ ] ")
			}

			// Status icon
			b.WriteString(m.styles.StatusIconStyled(job.Status, job.Conclusion))
			b.WriteString(" ")

			// Job name
			jobName := job.Name
			if len(jobName) > m.width-15 {
				jobName = jobName[:m.width-18] + "..."
			}
			b.WriteString(jobName)
			b.WriteString("\n")
		}

		// Show current selection summary
		b.WriteString("\n")
		if len(m.multiJobIDs) == 0 {
			b.WriteString(m.styles.Dim.Render("  (no jobs selected)"))
		} else if len(m.multiJobIDs) >= 4 {
			b.WriteString(m.styles.LogWarning.Render(fmt.Sprintf("  (%d jobs selected - max reached)", len(m.multiJobIDs))))
		} else {
			b.WriteString(m.styles.Dim.Render(fmt.Sprintf("  (%d job(s) selected)", len(m.multiJobIDs))))
		}
		b.WriteString("\n")
	}

	// Footer with key hints
	b.WriteString("\n")
	b.WriteString("  ")
	b.WriteString(m.styles.HelpKey.Render("space"))
	b.WriteString(" toggle  ")
	b.WriteString(m.styles.HelpKey.Render("m/enter"))
	b.WriteString(" apply  ")
	b.WriteString(m.styles.HelpKey.Render("esc"))
	b.WriteString(" cancel\n")

	return b.String()
}

// viewCompareSelect displays the run selection UI for comparison (v0.6)
func (m Model) viewCompareSelect() string {
	var b strings.Builder

	if m.compareSelectStep == 0 {
		b.WriteString("Compare Logs - Select First Run\n\n")
	} else {
		b.WriteString("Compare Logs - Select Second Run\n\n")
		// Show first selection
		if m.compareRunIdx1 >= 0 && m.compareRunIdx1 < len(m.runs) {
			run := m.runs[m.compareRunIdx1]
			b.WriteString(fmt.Sprintf("  First: #%d %s\n\n", run.RunNumber, run.Name))
		}
	}

	if len(m.runs) < 2 {
		b.WriteString("  Need at least 2 runs to compare\n")
	} else {
		for i, run := range m.runs {
			// Selection cursor
			if i == m.compareCursor {
				b.WriteString(m.styles.Selected.Render("→ "))
			} else {
				b.WriteString("  ")
			}

			// Mark already selected run
			if i == m.compareRunIdx1 {
				b.WriteString("[1] ")
			} else {
				b.WriteString("    ")
			}

			// Status icon
			b.WriteString(m.styles.StatusBadge(run.Status, run.Conclusion))
			b.WriteString(" ")

			// Run info
			runLabel := fmt.Sprintf("#%d %s", run.RunNumber, run.Name)
			if len(runLabel) > m.width-20 {
				runLabel = runLabel[:m.width-23] + "..."
			}
			b.WriteString(runLabel)
			b.WriteString(" ")
			b.WriteString(m.styles.Dim.Render(timeAgo(run.UpdatedAt)))
			b.WriteString("\n")
		}
	}

	// Footer with key hints
	b.WriteString("\n")
	b.WriteString("  ")
	b.WriteString(m.styles.HelpKey.Render("c/enter"))
	b.WriteString(" select  ")
	b.WriteString(m.styles.HelpKey.Render("esc"))
	b.WriteString(" cancel\n")

	return b.String()
}

// viewCompareView displays the diff comparison view (v0.6)
func (m Model) viewCompareView() string {
	var b strings.Builder

	// Header
	b.WriteString("Log Comparison\n")

	// Show which runs are being compared
	if m.compareRunIdx1 >= 0 && m.compareRunIdx1 < len(m.runs) &&
		m.compareRunIdx2 >= 0 && m.compareRunIdx2 < len(m.runs) {
		run1 := m.runs[m.compareRunIdx1]
		run2 := m.runs[m.compareRunIdx2]
		b.WriteString(m.styles.Dim.Render(fmt.Sprintf("  Run #%d vs Run #%d\n", run1.RunNumber, run2.RunNumber)))
	}
	b.WriteString("\n")

	// Legend
	b.WriteString("  ")
	b.WriteString(m.styles.DiffRemoved.Render("- removed"))
	b.WriteString("  ")
	b.WriteString(m.styles.DiffAdded.Render("+ added"))
	b.WriteString("\n\n")

	if len(m.compareDiff) == 0 {
		b.WriteString("  No differences found or logs are empty\n")
	} else {
		// Calculate visible area
		maxLines := m.height - 12

		// Display visible diff lines
		start := m.compareScrollOff
		end := start + maxLines
		if end > len(m.compareDiff) {
			end = len(m.compareDiff)
		}

		for i := start; i < end; i++ {
			line := m.compareDiff[i]

			// Truncate long lines
			if len(line) > m.width-4 {
				line = line[:m.width-7] + "..."
			}

			// Apply color based on diff type
			if i < len(m.compareDiffColors) {
				switch m.compareDiffColors[i] {
				case -1:
					line = m.styles.DiffRemoved.Render(line)
				case 1:
					line = m.styles.DiffAdded.Render(line)
				}
			}

			b.WriteString(line)
			b.WriteString("\n")
		}

		// Show scroll status
		if len(m.compareDiff) > maxLines {
			scrollPercent := float64(m.compareScrollOff) / float64(len(m.compareDiff)-maxLines) * 100
			b.WriteString(fmt.Sprintf("\n[Line %d/%d (%.0f%%)]", m.compareScrollOff+1, len(m.compareDiff), scrollPercent))
		}
	}

	// Footer
	b.WriteString("\n")
	b.WriteString("  ")
	b.WriteString(m.styles.HelpKey.Render("↑/↓"))
	b.WriteString(" scroll  ")
	b.WriteString(m.styles.HelpKey.Render("c/esc"))
	b.WriteString(" exit\n")

	return b.String()
}
