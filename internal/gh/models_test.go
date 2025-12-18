package gh

import (
	"encoding/json"
	"testing"
	"time"
)

func TestWorkflowRunParsing(t *testing.T) {
	jsonData := `{
		"id": 12345678,
		"name": "CI",
		"run_number": 42,
		"status": "completed",
		"conclusion": "success",
		"created_at": "2024-01-15T10:30:00Z",
		"updated_at": "2024-01-15T10:35:00Z",
		"html_url": "https://github.com/owner/repo/actions/runs/12345678",
		"event": "push",
		"head_branch": "main",
		"actor": {
			"login": "testuser"
		}
	}`

	var run WorkflowRun
	if err := json.Unmarshal([]byte(jsonData), &run); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if run.ID != 12345678 {
		t.Errorf("ID = %d, want 12345678", run.ID)
	}
	if run.Name != "CI" {
		t.Errorf("Name = %q, want %q", run.Name, "CI")
	}
	if run.RunNumber != 42 {
		t.Errorf("RunNumber = %d, want 42", run.RunNumber)
	}
	if run.Status != "completed" {
		t.Errorf("Status = %q, want %q", run.Status, "completed")
	}
	if run.Conclusion == nil || *run.Conclusion != "success" {
		t.Errorf("Conclusion = %v, want %q", run.Conclusion, "success")
	}
	if run.Event != "push" {
		t.Errorf("Event = %q, want %q", run.Event, "push")
	}
	if run.HeadBranch != "main" {
		t.Errorf("HeadBranch = %q, want %q", run.HeadBranch, "main")
	}
	if run.ActorLogin() != "testuser" {
		t.Errorf("ActorLogin() = %q, want %q", run.ActorLogin(), "testuser")
	}
}

func TestJobParsing(t *testing.T) {
	jsonData := `{
		"id": 98765432,
		"name": "build",
		"status": "completed",
		"conclusion": "success",
		"started_at": "2024-01-15T10:30:00Z",
		"completed_at": "2024-01-15T10:32:30Z",
		"html_url": "https://github.com/owner/repo/actions/runs/12345678/job/98765432",
		"runner_name": "ubuntu-latest"
	}`

	var job Job
	if err := json.Unmarshal([]byte(jsonData), &job); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if job.ID != 98765432 {
		t.Errorf("ID = %d, want 98765432", job.ID)
	}
	if job.Name != "build" {
		t.Errorf("Name = %q, want %q", job.Name, "build")
	}
	if job.Status != "completed" {
		t.Errorf("Status = %q, want %q", job.Status, "completed")
	}

	expectedDuration := 2*time.Minute + 30*time.Second
	if job.Duration() != expectedDuration {
		t.Errorf("Duration() = %v, want %v", job.Duration(), expectedDuration)
	}
}

func TestWorkflowRunsResponseParsing(t *testing.T) {
	jsonData := `{
		"total_count": 1,
		"workflow_runs": [
			{
				"id": 12345678,
				"name": "CI",
				"run_number": 42,
				"status": "in_progress",
				"conclusion": null,
				"event": "pull_request"
			}
		]
	}`

	var response WorkflowRunsResponse
	if err := json.Unmarshal([]byte(jsonData), &response); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if response.TotalCount != 1 {
		t.Errorf("TotalCount = %d, want 1", response.TotalCount)
	}
	if len(response.WorkflowRuns) != 1 {
		t.Fatalf("len(WorkflowRuns) = %d, want 1", len(response.WorkflowRuns))
	}

	run := response.WorkflowRuns[0]
	if run.Status != "in_progress" {
		t.Errorf("Status = %q, want %q", run.Status, "in_progress")
	}
	if run.Conclusion != nil {
		t.Errorf("Conclusion should be nil for in_progress run")
	}
}

func TestJobsResponseParsing(t *testing.T) {
	jsonData := `{
		"total_count": 2,
		"jobs": [
			{
				"id": 1,
				"name": "build",
				"status": "completed",
				"conclusion": "success"
			},
			{
				"id": 2,
				"name": "test",
				"status": "completed",
				"conclusion": "failure"
			}
		]
	}`

	var response JobsResponse
	if err := json.Unmarshal([]byte(jsonData), &response); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if response.TotalCount != 2 {
		t.Errorf("TotalCount = %d, want 2", response.TotalCount)
	}
	if len(response.Jobs) != 2 {
		t.Fatalf("len(Jobs) = %d, want 2", len(response.Jobs))
	}
}

func TestWorkflowRunStatus(t *testing.T) {
	tests := []struct {
		name          string
		status        string
		conclusion    *string
		wantCompleted bool
		wantSuccess   bool
		wantFailure   bool
	}{
		{
			name:          "in progress",
			status:        StatusInProgress,
			conclusion:    nil,
			wantCompleted: false,
			wantSuccess:   false,
			wantFailure:   false,
		},
		{
			name:          "success",
			status:        StatusCompleted,
			conclusion:    strPtr(ConclusionSuccess),
			wantCompleted: true,
			wantSuccess:   true,
			wantFailure:   false,
		},
		{
			name:          "failure",
			status:        StatusCompleted,
			conclusion:    strPtr(ConclusionFailure),
			wantCompleted: true,
			wantSuccess:   false,
			wantFailure:   true,
		},
		{
			name:          "cancelled",
			status:        StatusCompleted,
			conclusion:    strPtr(ConclusionCancelled),
			wantCompleted: true,
			wantSuccess:   false,
			wantFailure:   true,
		},
		{
			name:          "skipped",
			status:        StatusCompleted,
			conclusion:    strPtr(ConclusionSkipped),
			wantCompleted: true,
			wantSuccess:   true, // skipped counts as success
			wantFailure:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run := WorkflowRun{Status: tt.status, Conclusion: tt.conclusion}

			if got := run.IsCompleted(); got != tt.wantCompleted {
				t.Errorf("IsCompleted() = %v, want %v", got, tt.wantCompleted)
			}
			if got := run.IsSuccess(); got != tt.wantSuccess {
				t.Errorf("IsSuccess() = %v, want %v", got, tt.wantSuccess)
			}
			if got := run.IsFailure(); got != tt.wantFailure {
				t.Errorf("IsFailure() = %v, want %v", got, tt.wantFailure)
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}
