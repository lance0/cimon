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
- [x] Search within logs

## v0.4 - Workflow Actions

- [x] Rerun workflow (`cimon retry`) with confirmation
- [x] Cancel running workflow
- [x] Trigger workflow dispatch

## v0.5 - Enhanced UX

- [ ] Workflow history with pagination (beyond just latest run)
- [ ] Branch filtering and selection
- [ ] Better error handling with retry logic
- [ ] Loading states and progress indicators
- [ ] Keyboard shortcuts help dialog (?)
- [ ] Filter workflows by status (success/failure/running)
- [ ] View workflow YAML/files
- [ ] Download job artifacts

## v0.6 - Advanced Logs

- [ ] Log filtering by job/step
- [ ] Log export to file
- [ ] Syntax highlighting for different log types
- [ ] Log comparison between runs
- [ ] Follow specific job logs in multi-job workflows

## v0.7 - Notifications & Automation

- [ ] Desktop notifications on completion
- [ ] Webhook support for external integrations
- [ ] Slack/Discord integration
- [ ] Custom completion hooks/scripts
- [ ] Email notifications

## v0.8 - Multi-Repo & Enterprise

- [ ] Multi-repo dashboard
- [ ] Organization-wide CI monitoring
- [ ] Team-based access controls
- [ ] Custom CI server support (Jenkins, etc.)
- [ ] Enterprise GitHub support

## Future

- [ ] GitLab CI support behind provider interface
- [ ] CircleCI support
- [ ] Jenkins support
- [ ] Custom themes and color schemes
- [ ] Plugin system for extensibility
- [ ] Configuration file support (.cimonrc)
- [ ] Performance profiling and optimization
