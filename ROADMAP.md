# Cimon Roadmap

## v0 (Current) - Ship

- [x] TUI default mode with Bubble Tea
- [x] Repo + branch auto-detection from git
- [x] Latest workflow run + jobs view
- [x] Refresh (`r`) and watch toggle (`w`)
- [x] Open run in browser (`o`)
- [x] Proper exit codes (0=success, 1=failure, 2=error)
- [x] Cross-platform builds via goreleaser

## v0.1 - Scripting Support

- [x] `--plain` text output (no TUI)
- [x] `--json` output for scripting
- [x] Better detached HEAD behavior via default branch lookup
- [x] NO_COLOR environment variable support

## v0.2 - Job Details

- [x] Job selection with details pane
- [x] Show job steps within selected job
- [x] Open individual job URL in browser

## v0.3 - Logs

- [x] Log streaming for running jobs
- [x] Log viewer for completed jobs
- [ ] Search within logs

## v0.4 - Workflow Actions

- [ ] Rerun workflow (`cimon retry`) with confirmation
- [ ] Cancel running workflow
- [ ] Trigger workflow dispatch

## Future

- [ ] GitLab CI support behind provider interface
- [ ] CircleCI support
- [ ] Multi-repo dashboard
- [ ] Notifications on completion
- [ ] Custom workflow filtering
