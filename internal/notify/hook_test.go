package notify

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestResolveHookPath(t *testing.T) {
	cwd, _ := os.Getwd()

	tests := []struct {
		name        string
		hookPath    string
		wantAbspath bool
	}{
		{
			name:        "relative path",
			hookPath:    "test-hook.sh",
			wantAbspath: true,
		},
		{
			name:        "absolute path",
			hookPath:    "/usr/bin/test",
			wantAbspath: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolveHookPath(tt.hookPath)
			if err != nil {
				t.Errorf("resolveHookPath() error = %v", err)
				return
			}

			if tt.wantAbspath && !filepath.IsAbs(result) {
				t.Errorf("resolveHookPath() = %q, want absolute path", result)
			}

			// For relative paths, verify it's resolved to cwd
			if !filepath.IsAbs(tt.hookPath) {
				expected := filepath.Join(cwd, tt.hookPath)
				if result != expected {
					t.Errorf("resolveHookPath() = %q, want %q", result, expected)
				}
			}
		})
	}
}

func TestValidateHookFile(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create a regular file (non-executable)
	nonExecFile := filepath.Join(tmpDir, "non-exec.sh")
	if err := os.WriteFile(nonExecFile, []byte("#!/bin/bash\necho hello"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create an executable file
	execFile := filepath.Join(tmpDir, "exec.sh")
	if err := os.WriteFile(execFile, []byte("#!/bin/bash\necho hello"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create a directory
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "non-existent file",
			path:    filepath.Join(tmpDir, "nonexistent.sh"),
			wantErr: true,
			errMsg:  "not found",
		},
		{
			name:    "directory instead of file",
			path:    subDir,
			wantErr: true,
			errMsg:  "directory",
		},
		{
			name:    "executable file",
			path:    execFile,
			wantErr: false,
		},
	}

	// On Unix, non-executable files should fail
	if runtime.GOOS != "windows" {
		tests = append(tests, struct {
			name    string
			path    string
			wantErr bool
			errMsg  string
		}{
			name:    "non-executable file on Unix",
			path:    nonExecFile,
			wantErr: true,
			errMsg:  "not executable",
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHookFile(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateHookFile() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !containsStr(err.Error(), tt.errMsg) {
					t.Errorf("validateHookFile() error = %v, want error containing %q", err, tt.errMsg)
				}
			}
		})
	}
}

func TestValidateHookPath(t *testing.T) {
	// Empty path should be valid (no hook configured)
	err := ValidateHookPath("")
	if err != nil {
		t.Errorf("ValidateHookPath(\"\") = %v, want nil", err)
	}

	// Non-existent path should fail
	err = ValidateHookPath("/nonexistent/path/to/hook.sh")
	if err == nil {
		t.Error("ValidateHookPath() for non-existent file should return error")
	}
}

func TestExecuteHook_EmptyPath(t *testing.T) {
	result := ExecuteHook("", HookData{})
	if result.Executed {
		t.Error("ExecuteHook() with empty path should not execute")
	}
	if result.Error == nil {
		t.Error("ExecuteHook() with empty path should return error")
	}
}

func TestExecuteHook_NonExistentFile(t *testing.T) {
	result := ExecuteHook("/nonexistent/hook.sh", HookData{})
	if result.Executed {
		t.Error("ExecuteHook() with non-existent file should not execute")
	}
	if result.Error == nil {
		t.Error("ExecuteHook() with non-existent file should return error")
	}
}

// containsStr checks if s contains substr
func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestBuildHookCommand(t *testing.T) {
	data := HookData{
		WorkflowName: "CI",
		RunNumber:    123,
		RunID:        456789,
		Status:       "completed",
		Conclusion:   "success",
		Repo:         "owner/repo",
		Branch:       "main",
	}

	// Test with a simple path
	cmd := buildHookCommand("/path/to/hook.sh", data)
	if cmd == nil {
		t.Fatal("buildHookCommand() returned nil")
	}

	// Verify environment variables are set
	envVars := cmd.Env
	foundWorkflow := false
	for _, env := range envVars {
		if containsSubstring(env, "CIMON_WORKFLOW_NAME=CI") {
			foundWorkflow = true
			break
		}
	}
	if !foundWorkflow {
		t.Error("buildHookCommand() should set CIMON_WORKFLOW_NAME in env")
	}
}

func TestBuildHookCommand_WindowsExtensions(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific test")
	}

	data := HookData{}

	// Test .ps1 extension
	ps1Cmd := buildHookCommand("script.ps1", data)
	if ps1Cmd == nil {
		t.Fatal("buildHookCommand() for .ps1 returned nil")
	}

	// Test .bat extension
	batCmd := buildHookCommand("script.bat", data)
	if batCmd == nil {
		t.Fatal("buildHookCommand() for .bat returned nil")
	}

	// Test .cmd extension
	cmdCmd := buildHookCommand("script.cmd", data)
	if cmdCmd == nil {
		t.Fatal("buildHookCommand() for .cmd returned nil")
	}
}

func TestHookResultFields(t *testing.T) {
	// Test successful result
	successResult := HookResult{Executed: true, Error: nil}
	if !successResult.Executed {
		t.Error("HookResult.Executed should be true")
	}
	if successResult.Error != nil {
		t.Error("HookResult.Error should be nil")
	}

	// Test failed result
	failResult := HookResult{Executed: false, Error: nil}
	if failResult.Executed {
		t.Error("HookResult.Executed should be false")
	}
}

func TestExecuteHook_ValidScript(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}

	// Create a temporary executable script
	tmpDir := t.TempDir()
	hookPath := filepath.Join(tmpDir, "test-hook.sh")
	script := "#!/bin/bash\nexit 0\n"
	if err := os.WriteFile(hookPath, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	data := HookData{
		WorkflowName: "Test",
		RunNumber:    1,
		Conclusion:   "success",
	}

	result := ExecuteHook(hookPath, data)
	if !result.Executed {
		t.Errorf("ExecuteHook() Executed = false, want true, error: %v", result.Error)
	}
	if result.Error != nil {
		t.Errorf("ExecuteHook() Error = %v, want nil", result.Error)
	}
}

func TestHookDataFields(t *testing.T) {
	data := HookData{
		WorkflowName: "Build",
		RunNumber:    100,
		RunID:        999,
		Status:       "completed",
		Conclusion:   "failure",
		Repo:         "org/project",
		Branch:       "develop",
		Event:        "pull_request",
		Actor:        "developer",
		HTMLURL:      "https://github.com/org/project/actions/runs/999",
		JobCount:     5,
		SuccessCount: 3,
		FailureCount: 2,
	}

	if data.WorkflowName != "Build" {
		t.Error("WorkflowName not set correctly")
	}
	if data.RunNumber != 100 {
		t.Error("RunNumber not set correctly")
	}
	if data.RunID != 999 {
		t.Error("RunID not set correctly")
	}
	if data.Status != "completed" {
		t.Error("Status not set correctly")
	}
	if data.Conclusion != "failure" {
		t.Error("Conclusion not set correctly")
	}
	if data.JobCount != 5 {
		t.Error("JobCount not set correctly")
	}
	if data.SuccessCount != 3 {
		t.Error("SuccessCount not set correctly")
	}
	if data.FailureCount != 2 {
		t.Error("FailureCount not set correctly")
	}
}
