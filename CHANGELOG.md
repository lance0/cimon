# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

#### v0.7 Notifications & Hooks
- **Desktop Notifications**: OS-native notifications when workflow completes in watch mode (`--notify` flag)
- **Custom Hooks**: Execute user scripts on completion with environment variables (`--hook` flag)
- **Cross-Platform Support**: Linux (notify-send), macOS (osascript), Windows (PowerShell)
- **Hook Environment Variables**: Pass workflow data to scripts via CIMON_* environment variables

#### v0.6 Advanced Logs
- **Syntax Highlighting**: Color-coded log lines for errors (red), warnings (yellow), commands (cyan), and groups (bold) (`H` to toggle)
- **Log Export**: Save current log view to timestamped file with metadata header (`s` key)
- **Log Filtering**: Filter logs by step with multi-select checkbox UI (`F` key in log viewer)
- **Multi-Job Following**: View logs from multiple jobs simultaneously (`m` key, select up to 4 jobs)
- **Log Comparison**: Compare logs between two different workflow runs with diff view (`c` key)

#### v0.5 Enhanced UX
- **Workflow History**: Browse 10+ recent runs with pagination (`h`/`l` keys)
- **Branch Filtering**: Select and switch between branches (`b` key)
- **Status Filtering**: Filter runs by success/failure/in-progress status (`f` key)
- **Help Dialog**: Comprehensive keyboard shortcuts reference (`?` key)
- **Workflow YAML Viewer**: View CI configuration files (`y` key)
- **Artifact Downloads**: Download build artifacts from workflow runs (`a` key)
- **Enhanced Loading States**: Contextual progress messages throughout UI
- **Improved Error Handling**: Automatic retry logic with actionable suggestions
- **Better Visual Indicators**: Enhanced status display and navigation feedback

#### v0.4 Workflow Actions
- **Workflow Rerun**: Restart failed workflows with confirmation (`cimon retry`)
- **Workflow Cancellation**: Safely cancel running workflows (`cimon cancel`)
- **Workflow Dispatch**: Trigger manual workflows (`cimon dispatch <workflow>`)
- **Safety Confirmations**: Interactive prompts for destructive operations

#### v0.3 Logs
- **Live Log Streaming**: Real-time log updates for running jobs
- **Log Viewer**: Scrollable interface for completed job logs
- **Log Search**: Find specific content with match navigation (`/` `n` `N` keys)
- **Log Export**: Save logs for external analysis

#### v0.2 Job Details
- **Job Selection**: Interactive job inspection with details pane (`enter` key)
- **Step Breakdown**: View individual job steps and their status
- **Job Metadata**: Display runner information, timing, and execution details
- **Browser Integration**: Open jobs in GitHub with `o` key

### Changed
- **TUI Enhancements**: Multiple interactive modes with consistent navigation
- **API Integration**: Extended GitHub API support with comprehensive error handling
- **User Experience**: Streamlined workflows with keyboard shortcuts and visual feedback
- **Performance**: Optimized API calls and improved response times

### Technical
- **CLI Subcommands**: Added workflow control commands (`retry`, `cancel`, `dispatch`)
- **Real-time Updates**: Implemented streaming for live job monitoring
- **Pagination Support**: Efficient handling of large workflow histories
- **Retry Logic**: Automatic recovery from transient API failures

### Fixed
- **Navigation Bug**: Fixed down key moving cursor in wrong direction in job list
- **Log Search**: Implemented missing search input handler (`/` key now works)
- **Error Messages**: Fixed incorrect error variable being displayed in detached HEAD handling
- **Help Dialog**: Any key now exits help (not just `?`)
- **Authentication**: Fixed raw HTTP requests missing authentication headers
- **HTTP Timeout**: Added 60-second timeout to prevent hanging requests
- **Resource Leak**: Fixed double-close issue in artifact download
- **Nil Check**: Added defensive nil check in artifact fetching

## [0.1.0] - 2025-12-18

### Added
- **Initial Release**: Terminal-first CI monitor for GitHub Actions
- **Interactive TUI**: Built with Bubble Tea for smooth terminal experience
- **Repository Detection**: Auto-detect repository and branch from git context
- **Workflow Monitoring**: Display latest workflow run with job breakdown
- **Visual Status**: Color-coded icons for success/failure/in-progress states
- **Watch Mode**: Poll for updates until workflow completion (`-w` flag)
- **Browser Integration**: Open workflow runs in GitHub (`o` key)
- **Navigation**: Keyboard controls (`j`/`k` or arrow keys) for job selection
- **Manual Refresh**: Force update workflow data (`r` key)
- **Configuration Flags**:
  - `--repo owner/name`: Override repository detection
  - `--branch name`: Override branch detection
  - `--poll duration`: Customize watch mode polling interval
- **Exit Codes**: Status-based exit codes (0=success, 1=failure, 2=error)
- **Authentication**: Support for `gh` CLI and `GITHUB_TOKEN` environment variable
- **Cross-Platform**: Works on Linux, macOS, and Windows
