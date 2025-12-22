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
