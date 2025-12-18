# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
