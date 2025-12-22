# cimon

Terminal-first CI monitor for GitHub Actions. Check your workflow status without leaving the terminal.

![cimon demo](https://github.com/lance0/cimon/assets/demo.gif)

## Features

### Core Monitoring
- **Zero friction** - Run inside any git repo and auto-detect repository/branch
- **Fast feedback** - See workflow runs, jobs, and status instantly
- **Watch mode** - Poll until completion with real-time updates (`-w`)
- **Multi-run history** - Browse 10+ recent workflow runs with pagination
- **Branch switching** - Monitor CI across different branches (`b` key)
- **Status filtering** - Filter by success, failure, running, queued (`f` key)

### Deep Inspection
- **Job details** - Drill into individual jobs with step-by-step breakdown
- **Live logs** - Stream logs from running jobs with automatic refresh
- **Log search** - Find specific errors or messages within logs (`/` key)
- **Interactive navigation** - Full keyboard-driven interface

### Workflow Control
- **Rerun workflows** - Restart failed builds with confirmation (`cimon retry`)
- **Cancel runs** - Stop running workflows safely (`cimon cancel`)
- **Trigger dispatches** - Start manual workflows (`cimon dispatch <workflow>`)

### Developer Experience
- **Great terminal UX** - Clean TUI with comprehensive keyboard shortcuts
- **Scriptable** - JSON/plain output modes for automation
- **Multiple output formats** - Human-readable, JSON, and plain text
- **Cross-platform** - Works on Linux, macOS, and Windows
- **Accessibility** - NO_COLOR support and clear visual feedback

## Installation

### From source

```bash
go install github.com/lance0/cimon/cmd/cimon@latest
```

### From releases

Download the latest binary from [Releases](https://github.com/lance0/cimon/releases).

## Usage

```bash
# Monitor current repo and branch
cimon

# Monitor a specific branch
cimon --branch main

# Watch until completion
cimon --watch

# Override repo detection
cimon --repo owner/name --branch main
```

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `r` | Refresh |
| `w` | Toggle watch mode |
| `o` | Open run/job in browser |
| `b` | Select branch |
| `f` | Filter by status |
| `h/l` or `←/→` | Navigate between runs |
| `j/k` or `↑/↓` | Navigate jobs/steps/logs/branches/filters |
| `enter` | Show job details / select branch/filter |
| `l` | View/exit job logs |
| `/` | Search in logs |
| `n` | Next search match |
| `N` | Previous search match |
| `s` | Save logs to file |
| `H` | Toggle syntax highlighting |
| `y` | View workflow YAML |
| `a` | Download artifacts |
| `?` | Show help |
| `q` | Quit |

### Flags

```
-b, --branch string   Branch name
-r, --repo string     Repository in owner/name format
-w, --watch           Watch mode - poll until completion
-p, --poll duration   Poll interval for watch mode (default 5s)
    --json            JSON output for scripting
    --no-color        Disable color output
    --plain           Plain text output (no TUI)
-v, --version         Show version
```

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success (or neutral/skipped) |
| 1 | Failure (or cancelled/timed out) |
| 2 | Error (auth, not found, etc.) |

## Authentication

cimon uses GitHub authentication in this order:

1. **gh CLI** (recommended) - If you have [gh](https://cli.github.com/) installed and authenticated
2. **GITHUB_TOKEN** - Environment variable

```bash
# Option 1: Use gh CLI
gh auth login

# Option 2: Set token directly
export GITHUB_TOKEN=ghp_xxxxxxxxxxxx
```

## Environment Variables

- **NO_COLOR** - Disable colored output in TUI mode (equivalent to `--no-color` flag)

## Examples

```bash
# Quick CI check before pushing
cimon && git push

# Wait for CI to finish
cimon -w

# Check CI on main branch
cimon -b main

# Get plain text output for scripting
cimon --plain

# Get JSON output for automation/scripting
cimon --json

# Monitor a different repo
cimon -r octocat/hello-world -b main
```

## License

MIT
