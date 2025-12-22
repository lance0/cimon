# cimon

Terminal-first CI monitor for GitHub Actions. Check your workflow status without leaving the terminal.

![cimon demo](https://github.com/lance0/cimon/assets/demo.gif)

## Features

- **Zero friction** - Run inside any git repo and it just works
- **Fast feedback** - See latest run status and job breakdown instantly
- **Watch mode** - Poll until completion with `-w`
- **Great terminal UX** - Clean TUI with keyboard navigation
- **Scriptable** - Exit codes reflect CI status (0=pass, 1=fail)

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
| `h/l` or `←/→` | Navigate between runs |
| `j/k` or `↑/↓` | Navigate jobs/steps/logs/branches |
| `enter` | Show job details / select branch |
| `l` | View/exit job logs |
| `/` | Search in logs |
| `n` | Next search match |
| `N` | Previous search match |
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
