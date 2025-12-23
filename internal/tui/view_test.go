package tui

import (
	"errors"
	"testing"
	"time"
)

func TestTimeAgo(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{"just now", 30 * time.Second, "just now"},
		{"1 minute", 1 * time.Minute, "1 minute ago"},
		{"5 minutes", 5 * time.Minute, "5 minutes ago"},
		{"1 hour", 1 * time.Hour, "1 hour ago"},
		{"3 hours", 3 * time.Hour, "3 hours ago"},
		{"1 day", 24 * time.Hour, "1 day ago"},
		{"5 days", 5 * 24 * time.Hour, "5 days ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a time that is tt.duration ago
			testTime := time.Now().Add(-tt.duration)
			got := timeAgo(testTime)
			if got != tt.want {
				t.Errorf("timeAgo() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		want     string
	}{
		{30 * time.Second, "30s"},
		{60 * time.Second, "1m"},
		{90 * time.Second, "1m 30s"},
		{5 * time.Minute, "5m"},
		{5*time.Minute + 30*time.Second, "5m 30s"},
		{1 * time.Hour, "1h 0m"},
		{1*time.Hour + 30*time.Minute, "1h 30m"},
	}

	for _, tt := range tests {
		t.Run(tt.duration.String(), func(t *testing.T) {
			got := formatDuration(tt.duration)
			if got != tt.want {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.duration, got, tt.want)
			}
		})
	}
}

func TestStatusIcon(t *testing.T) {
	success := "success"
	failure := "failure"
	cancelled := "cancelled"
	skipped := "skipped"

	tests := []struct {
		name       string
		status     string
		conclusion *string
		want       string
	}{
		{"queued", "queued", nil, IconQueued},
		{"in progress", "in_progress", nil, IconInProgress},
		{"success", "completed", &success, IconSuccess},
		{"failure", "completed", &failure, IconFailure},
		{"cancelled", "completed", &cancelled, IconWarning},
		{"skipped", "completed", &skipped, IconSkipped},
		{"completed no conclusion", "completed", nil, IconSkipped},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StatusIcon(tt.status, tt.conclusion)
			if got != tt.want {
				t.Errorf("StatusIcon(%q, %v) = %q, want %q", tt.status, tt.conclusion, got, tt.want)
			}
		})
	}
}

func TestGetErrorHint(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantIn  string // substring that should be in the result
	}{
		{"nil error", nil, ""},
		{"authentication error", errors.New("authentication failed"), "gh auth login"},
		{"401 error", errors.New("API returned 401"), "gh auth login"},
		{"403 forbidden", errors.New("403 Forbidden access"), "permissions"},
		{"not found", errors.New("repository not found"), "Verify the repository"},
		{"404 error", errors.New("404 Not Found"), "Verify the repository"},
		{"rate limit", errors.New("rate limit exceeded"), "rate limit"},
		{"429 error", errors.New("429 Too Many Requests"), "rate limit"},
		{"timeout error", errors.New("connection timeout"), "internet connection"},
		{"502 error", errors.New("502 Bad Gateway"), "temporarily unavailable"},
		{"503 error", errors.New("503 Service Unavailable"), "temporarily unavailable"},
		{"unknown error", errors.New("something weird happened"), "retry"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{err: tt.err}
			got := m.getErrorHint()
			if tt.wantIn == "" {
				if got != "" {
					t.Errorf("getErrorHint() = %q, want empty", got)
				}
			} else {
				if got == "" || !containsIgnoreCase(got, tt.wantIn) {
					t.Errorf("getErrorHint() = %q, want to contain %q", got, tt.wantIn)
				}
			}
		})
	}
}

func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findIgnoreCase(s, substr)))
}

func findIgnoreCase(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if eqIgnoreCase(s[i:i+len(substr)], substr) {
			return true
		}
	}
	return false
}

func eqIgnoreCase(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}

func TestDefaultKeyMap(t *testing.T) {
	km := DefaultKeyMap()

	// Test that essential keys are defined
	if len(km.Quit.Keys()) == 0 {
		t.Error("Quit key not defined")
	}
	if len(km.Up.Keys()) == 0 {
		t.Error("Up key not defined")
	}
	if len(km.Down.Keys()) == 0 {
		t.Error("Down key not defined")
	}
	if len(km.Enter.Keys()) == 0 {
		t.Error("Enter key not defined")
	}
	if len(km.Refresh.Keys()) == 0 {
		t.Error("Refresh key not defined")
	}
}

func TestDefaultStyles(t *testing.T) {
	// Test with color enabled
	styles := DefaultStyles(true)
	if styles == nil {
		t.Fatal("DefaultStyles(true) returned nil")
	}

	// Test with color disabled
	stylesNoColor := DefaultStyles(false)
	if stylesNoColor == nil {
		t.Fatal("DefaultStyles(false) returned nil")
	}
}
