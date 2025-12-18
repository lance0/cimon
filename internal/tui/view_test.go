package tui

import (
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
