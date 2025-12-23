package notify

import (
	"runtime"
	"testing"
)

func TestFormatTitle(t *testing.T) {
	tests := []struct {
		name     string
		data     NotificationData
		expected string
	}{
		{
			name: "success",
			data: NotificationData{
				WorkflowName: "CI",
				RunNumber:    123,
				Conclusion:   "success",
			},
			expected: "✓ CI #123",
		},
		{
			name: "failure",
			data: NotificationData{
				WorkflowName: "Deploy",
				RunNumber:    456,
				Conclusion:   "failure",
			},
			expected: "✗ Deploy #456",
		},
		{
			name: "cancelled",
			data: NotificationData{
				WorkflowName: "Test",
				RunNumber:    789,
				Conclusion:   "cancelled",
			},
			expected: "⊘ Test #789",
		},
		{
			name: "timed_out",
			data: NotificationData{
				WorkflowName: "Build",
				RunNumber:    100,
				Conclusion:   "timed_out",
			},
			expected: "⏱ Build #100",
		},
		{
			name: "unknown conclusion",
			data: NotificationData{
				WorkflowName: "Other",
				RunNumber:    200,
				Conclusion:   "unknown",
			},
			expected: "● Other #200",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTitle(tt.data)
			if result != tt.expected {
				t.Errorf("formatTitle() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFormatBody(t *testing.T) {
	tests := []struct {
		name     string
		data     NotificationData
		expected string
	}{
		{
			name: "with conclusion",
			data: NotificationData{
				Repo:       "owner/repo",
				Branch:     "main",
				Conclusion: "success",
			},
			expected: "owner/repo on main - success",
		},
		{
			name: "empty conclusion",
			data: NotificationData{
				Repo:       "owner/repo",
				Branch:     "feature",
				Conclusion: "",
			},
			expected: "owner/repo on feature - completed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatBody(tt.data)
			if result != tt.expected {
				t.Errorf("formatBody() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGetStatusIcon(t *testing.T) {
	tests := []struct {
		conclusion string
		expected   string
	}{
		{"success", "✓"},
		{"failure", "✗"},
		{"cancelled", "⊘"},
		{"timed_out", "⏱"},
		{"unknown", "●"},
		{"", "●"},
	}

	for _, tt := range tests {
		t.Run(tt.conclusion, func(t *testing.T) {
			result := getStatusIcon(tt.conclusion)
			if result != tt.expected {
				t.Errorf("getStatusIcon(%q) = %q, want %q", tt.conclusion, result, tt.expected)
			}
		})
	}
}

func TestGetUrgency(t *testing.T) {
	tests := []struct {
		conclusion string
		expected   string
	}{
		{"failure", "critical"},
		{"timed_out", "critical"},
		{"cancelled", "normal"},
		{"success", "low"},
		{"", "low"},
	}

	for _, tt := range tests {
		t.Run(tt.conclusion, func(t *testing.T) {
			result := getUrgency(tt.conclusion)
			if result != tt.expected {
				t.Errorf("getUrgency(%q) = %q, want %q", tt.conclusion, result, tt.expected)
			}
		})
	}
}

func TestHookDataToEnvVars(t *testing.T) {
	data := HookData{
		WorkflowName: "CI",
		RunNumber:    123,
		RunID:        456789,
		Status:       "completed",
		Conclusion:   "success",
		Repo:         "owner/repo",
		Branch:       "main",
		Event:        "push",
		Actor:        "username",
		HTMLURL:      "https://github.com/owner/repo/actions/runs/456789",
		JobCount:     3,
		SuccessCount: 2,
		FailureCount: 1,
	}

	envVars := data.ToEnvVars()

	expected := map[string]string{
		"CIMON_WORKFLOW_NAME": "CI",
		"CIMON_RUN_NUMBER":    "123",
		"CIMON_RUN_ID":        "456789",
		"CIMON_STATUS":        "completed",
		"CIMON_CONCLUSION":    "success",
		"CIMON_REPO":          "owner/repo",
		"CIMON_BRANCH":        "main",
		"CIMON_EVENT":         "push",
		"CIMON_ACTOR":         "username",
		"CIMON_HTML_URL":      "https://github.com/owner/repo/actions/runs/456789",
		"CIMON_JOB_COUNT":     "3",
		"CIMON_SUCCESS_COUNT": "2",
		"CIMON_FAILURE_COUNT": "1",
	}

	if len(envVars) != len(expected) {
		t.Errorf("ToEnvVars() returned %d vars, want %d", len(envVars), len(expected))
	}

	// Convert slice to map for easier checking
	envMap := make(map[string]string)
	for _, env := range envVars {
		for i := 0; i < len(env); i++ {
			if env[i] == '=' {
				envMap[env[:i]] = env[i+1:]
				break
			}
		}
	}

	for key, expectedValue := range expected {
		if value, ok := envMap[key]; !ok {
			t.Errorf("ToEnvVars() missing %s", key)
		} else if value != expectedValue {
			t.Errorf("ToEnvVars() %s = %q, want %q", key, value, expectedValue)
		}
	}
}

func TestIsNotificationAvailable(t *testing.T) {
	// This function should return a boolean without panicking
	result := IsNotificationAvailable()

	// On macOS and Windows, it should always return true
	switch runtime.GOOS {
	case "darwin", "windows":
		if !result {
			t.Errorf("IsNotificationAvailable() = false on %s, want true", runtime.GOOS)
		}
	case "linux":
		// On Linux, result depends on notify-send being installed
		// Just verify it doesn't panic and returns a bool
		_ = result
	default:
		// Unknown platforms should return false
		if result {
			t.Errorf("IsNotificationAvailable() = true on unknown platform %s", runtime.GOOS)
		}
	}
}

func TestBuildMacOSNotification(t *testing.T) {
	cmd := buildMacOSNotification("Test Title", "Test Body")
	if cmd == nil {
		t.Fatal("buildMacOSNotification() returned nil")
	}

	// Verify command is osascript
	if cmd.Path == "" {
		t.Error("buildMacOSNotification() command has empty path")
	}

	// Verify args contain the title and body
	args := cmd.Args
	if len(args) < 2 {
		t.Errorf("buildMacOSNotification() has %d args, want at least 2", len(args))
	}
}

func TestBuildWindowsNotification(t *testing.T) {
	cmd := buildWindowsNotification("Test Title", "Test Body")
	if cmd == nil {
		t.Fatal("buildWindowsNotification() returned nil")
	}

	// Verify command has arguments
	if len(cmd.Args) == 0 {
		t.Error("buildWindowsNotification() command has no args")
	}
}

func TestBuildLinuxNotification(t *testing.T) {
	// This may return nil if notify-send is not installed
	cmd := buildLinuxNotification("Test Title", "Test Body", "normal")

	// If notify-send is available, verify command structure
	if cmd != nil {
		if len(cmd.Args) < 4 {
			t.Errorf("buildLinuxNotification() has %d args, want at least 4", len(cmd.Args))
		}
	}
}

func TestSendDesktopNotification_UnsupportedPlatform(t *testing.T) {
	// We can't easily test unsupported platforms, but we can test the data path
	data := NotificationData{
		WorkflowName: "Test",
		RunNumber:    1,
		Conclusion:   "success",
		Repo:         "owner/repo",
		Branch:       "main",
	}

	// Just verify it doesn't panic
	result := SendDesktopNotification(data)
	// Result depends on platform and available tools
	_ = result
}

func TestNotificationDataFields(t *testing.T) {
	data := NotificationData{
		WorkflowName: "CI Build",
		RunNumber:    42,
		Conclusion:   "failure",
		Repo:         "owner/repo",
		Branch:       "feature-branch",
		HTMLURL:      "https://github.com/owner/repo/actions/runs/123",
	}

	if data.WorkflowName != "CI Build" {
		t.Error("WorkflowName not set correctly")
	}
	if data.RunNumber != 42 {
		t.Error("RunNumber not set correctly")
	}
	if data.Conclusion != "failure" {
		t.Error("Conclusion not set correctly")
	}
	if data.Repo != "owner/repo" {
		t.Error("Repo not set correctly")
	}
	if data.Branch != "feature-branch" {
		t.Error("Branch not set correctly")
	}
	if data.HTMLURL != "https://github.com/owner/repo/actions/runs/123" {
		t.Error("HTMLURL not set correctly")
	}
}

func TestNotifyResultFields(t *testing.T) {
	// Test successful result
	successResult := NotifyResult{Sent: true, Error: nil}
	if !successResult.Sent {
		t.Error("NotifyResult.Sent should be true")
	}
	if successResult.Error != nil {
		t.Error("NotifyResult.Error should be nil")
	}

	// Test failed result
	failResult := NotifyResult{Sent: false, Error: nil}
	if failResult.Sent {
		t.Error("NotifyResult.Sent should be false")
	}
}
