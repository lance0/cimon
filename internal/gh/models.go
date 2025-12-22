package gh

import "time"

// WorkflowRun represents a GitHub Actions workflow run
type WorkflowRun struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	RunNumber  int       `json:"run_number"`
	Status     string    `json:"status"`     // queued, in_progress, completed
	Conclusion *string   `json:"conclusion"` // success, failure, cancelled, skipped, timed_out, action_required
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	HTMLURL    string    `json:"html_url"`
	Event      string    `json:"event"` // push, pull_request, workflow_dispatch, etc.
	HeadBranch string    `json:"head_branch"`
	Actor      *User     `json:"actor"`
}

// User represents a GitHub user
type User struct {
	Login string `json:"login"`
}

// Job represents a job within a workflow run
type Job struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Status      string     `json:"status"`     // queued, in_progress, completed
	Conclusion  *string    `json:"conclusion"` // success, failure, cancelled, skipped
	StartedAt   *time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
	HTMLURL     string     `json:"html_url"`
	RunnerName  string     `json:"runner_name"`
}

// WorkflowRunsResponse is the API response for listing workflow runs
type WorkflowRunsResponse struct {
	TotalCount   int           `json:"total_count"`
	WorkflowRuns []WorkflowRun `json:"workflow_runs"`
}

// JobsResponse is the API response for listing jobs
type JobsResponse struct {
	TotalCount int   `json:"total_count"`
	Jobs       []Job `json:"jobs"`
}

// Repository represents a GitHub repository
type Repository struct {
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	DefaultBranch string `json:"default_branch"`
}

// RunStatus constants
const (
	StatusQueued     = "queued"
	StatusInProgress = "in_progress"
	StatusCompleted  = "completed"
)

// Conclusion constants
const (
	ConclusionSuccess        = "success"
	ConclusionFailure        = "failure"
	ConclusionCancelled      = "cancelled"
	ConclusionSkipped        = "skipped"
	ConclusionTimedOut       = "timed_out"
	ConclusionActionRequired = "action_required"
	ConclusionNeutral        = "neutral"
)

// IsCompleted returns true if the run has completed
func (r *WorkflowRun) IsCompleted() bool {
	return r.Status == StatusCompleted
}

// IsSuccess returns true if the run completed successfully
func (r *WorkflowRun) IsSuccess() bool {
	if r.Conclusion == nil {
		return false
	}
	c := *r.Conclusion
	return c == ConclusionSuccess || c == ConclusionNeutral || c == ConclusionSkipped
}

// IsFailure returns true if the run failed
func (r *WorkflowRun) IsFailure() bool {
	if r.Conclusion == nil {
		return false
	}
	c := *r.Conclusion
	return c == ConclusionFailure || c == ConclusionCancelled || c == ConclusionTimedOut || c == ConclusionActionRequired
}

// ActorLogin returns the login of the actor who triggered the run
func (r *WorkflowRun) ActorLogin() string {
	if r.Actor == nil {
		return ""
	}
	return r.Actor.Login
}

// Duration returns the duration of a completed job
func (j *Job) Duration() time.Duration {
	if j.StartedAt == nil || j.CompletedAt == nil {
		return 0
	}
	return j.CompletedAt.Sub(*j.StartedAt)
}

// IsCompleted returns true if the job has completed
func (j *Job) IsCompleted() bool {
	return j.Status == StatusCompleted
}
