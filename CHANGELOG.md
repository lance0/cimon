# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

#### v0.2 Job Details
- Job selection with details pane (`enter` key)
- Show job steps within selected job
- Open individual job URLs in browser
- Interactive job inspection with navigation
- Job metadata display (runner, timing, status)

#### v0.3 Logs
- Log streaming for running jobs with real-time updates
- Log viewer for completed jobs with scrollable interface
- Search within logs (`/` key) with match navigation (`n`/`N`)
- Log export and filtering capabilities
- Syntax highlighting for log content

#### v0.4 Workflow Actions
- Rerun workflow (`cimon retry`) with confirmation prompts
- Cancel running workflow (`cimon cancel`) safely
- Trigger workflow dispatch (`cimon dispatch <workflow>`)
- Interactive CLI commands with safety confirmations
- GitHub API integration for workflow control

#### v0.5 Enhanced UX
- Workflow history with pagination (10+ runs, `h`/`l` navigation)
- Branch filtering and selection (`b` key)
- Status filtering by workflow outcome (`f` key)
 - Keyboard shortcuts help dialog (`?` key)
 - View workflow YAML configuration files (`y` key)
 - Download build artifacts from workflow runs (`a` key)
 - Enhanced loading states with contextual messages
 - Improved error handling with retry logic and suggestions
 - Better visual indicators and status display

### Changed
- Extended TUI with multiple interactive modes
- Enhanced GitHub API integration with additional endpoints
- Improved user experience with better navigation and feedback
- Added comprehensive keyboard shortcuts for power users

### Technical
- Added workflow actions CLI subcommands
- Implemented real-time log streaming
- Added pagination support for workflow runs
- Enhanced error handling and user feedback
- Improved performance with API optimizations

## [0.1.0] - 2024-12-18

### Added

- Initial release of Cimon - terminal-first CI monitor for GitHub Actions
- Interactive TUI built with Bubble Tea
- Auto-detection of repository and branch from git
- Display latest workflow run with job breakdown
- Status icons and color-coded output (success/failure/in-progress)
- Watch mode (`-w`) that polls until run completion
- Open workflow run in browser with `o` key
- Keyboard navigation (`j`/`k` or arrows)
- Manual refresh with `r` key
- CLI flags:
  - `--repo owner/name` to override repository
  - `--branch name` to override branch
  - `--watch` for watch mode
  - `--poll duration` for custom poll interval
  - `--version` to show version info
- Exit codes based on workflow run status:
  - `0` for success/neutral/skipped
  - `1` for failure/cancelled/timed_out
  - `2` for errors (auth, not found, etc.)
- Authentication via `gh` CLI or `GITHUB_TOKEN` environment variable
- Cross-platform support (Linux, macOS, Windows)
