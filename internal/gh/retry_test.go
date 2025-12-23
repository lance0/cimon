package gh

import (
	"errors"
	"testing"
	"time"
)

func TestDefaultRetryConfig(t *testing.T) {
	cfg := DefaultRetryConfig()

	if cfg.MaxRetries != 3 {
		t.Errorf("DefaultRetryConfig().MaxRetries = %d, want 3", cfg.MaxRetries)
	}
	if cfg.BaseDelay != 1*time.Second {
		t.Errorf("DefaultRetryConfig().BaseDelay = %v, want 1s", cfg.BaseDelay)
	}
	if cfg.MaxDelay != 30*time.Second {
		t.Errorf("DefaultRetryConfig().MaxDelay = %v, want 30s", cfg.MaxDelay)
	}
}

func TestRetryableError(t *testing.T) {
	innerErr := errors.New("connection failed")
	retryErr := RetryableError{Err: innerErr, Retryable: true}

	// Test Error()
	if retryErr.Error() != "connection failed" {
		t.Errorf("RetryableError.Error() = %q, want %q", retryErr.Error(), "connection failed")
	}

	// Test Unwrap()
	if retryErr.Unwrap() != innerErr {
		t.Errorf("RetryableError.Unwrap() = %v, want %v", retryErr.Unwrap(), innerErr)
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"502 error", errors.New("HTTP 502 Bad Gateway"), true},
		{"503 error", errors.New("503 Service Unavailable"), true},
		{"504 error", errors.New("504 Gateway Timeout"), true},
		{"429 rate limit", errors.New("429 Too Many Requests"), true},
		{"timeout", errors.New("connection timeout"), true},
		{"connection refused", errors.New("connection refused"), true},
		{"network unreachable", errors.New("network is unreachable"), true},
		{"temporary failure", errors.New("temporary failure in name resolution"), true},
		{"service unavailable", errors.New("service unavailable"), true},
		{"normal error", errors.New("invalid input"), false},
		{"auth error", errors.New("authentication failed"), false},
		{"not found", errors.New("404 Not Found"), false},
		{
			"retryable error type true",
			&RetryableError{Err: errors.New("test"), Retryable: true},
			true,
		},
		{
			"retryable error type false",
			&RetryableError{Err: errors.New("test"), Retryable: false},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRetryable(tt.err); got != tt.want {
				t.Errorf("IsRetryable(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestContainsIgnoreCase(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   bool
	}{
		{"hello world", "world", true},
		{"hello world", "WORLD", true},
		{"HELLO WORLD", "world", true},
		{"hello", "hello", true},
		{"hello", "world", false},
		{"", "test", false},
		{"test", "", true},
		{"abc", "abcd", false},
		{"The quick brown fox", "QUICK", true},
		{"Error 502 occurred", "502", true},
	}

	for _, tt := range tests {
		t.Run(tt.s+"_"+tt.substr, func(t *testing.T) {
			if got := containsIgnoreCase(tt.s, tt.substr); got != tt.want {
				t.Errorf("containsIgnoreCase(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}

func TestRetryWithBackoff_ImmediateSuccess(t *testing.T) {
	callCount := 0
	fn := func() error {
		callCount++
		return nil
	}

	cfg := RetryConfig{
		MaxRetries: 3,
		BaseDelay:  1 * time.Millisecond,
		MaxDelay:   10 * time.Millisecond,
	}

	err := RetryWithBackoff(fn, cfg)
	if err != nil {
		t.Errorf("RetryWithBackoff() error = %v, want nil", err)
	}
	if callCount != 1 {
		t.Errorf("RetryWithBackoff() called fn %d times, want 1", callCount)
	}
}

func TestRetryWithBackoff_NonRetryableError(t *testing.T) {
	callCount := 0
	fn := func() error {
		callCount++
		return errors.New("not retryable")
	}

	cfg := RetryConfig{
		MaxRetries: 3,
		BaseDelay:  1 * time.Millisecond,
		MaxDelay:   10 * time.Millisecond,
	}

	err := RetryWithBackoff(fn, cfg)
	if err == nil {
		t.Error("RetryWithBackoff() error = nil, want error")
	}
	if callCount != 1 {
		t.Errorf("RetryWithBackoff() called fn %d times, want 1 (no retries for non-retryable)", callCount)
	}
}

func TestRetryWithBackoff_SuccessAfterRetry(t *testing.T) {
	callCount := 0
	fn := func() error {
		callCount++
		if callCount < 3 {
			return errors.New("503 Service Unavailable")
		}
		return nil
	}

	cfg := RetryConfig{
		MaxRetries: 5,
		BaseDelay:  1 * time.Millisecond,
		MaxDelay:   10 * time.Millisecond,
	}

	err := RetryWithBackoff(fn, cfg)
	if err != nil {
		t.Errorf("RetryWithBackoff() error = %v, want nil", err)
	}
	if callCount != 3 {
		t.Errorf("RetryWithBackoff() called fn %d times, want 3", callCount)
	}
}

func TestRetryWithBackoff_MaxRetriesExhausted(t *testing.T) {
	callCount := 0
	fn := func() error {
		callCount++
		return errors.New("503 Service Unavailable")
	}

	cfg := RetryConfig{
		MaxRetries: 2,
		BaseDelay:  1 * time.Millisecond,
		MaxDelay:   10 * time.Millisecond,
	}

	err := RetryWithBackoff(fn, cfg)
	if err == nil {
		t.Error("RetryWithBackoff() error = nil, want error")
	}
	// Should be called MaxRetries + 1 times (initial + retries)
	if callCount != 3 {
		t.Errorf("RetryWithBackoff() called fn %d times, want 3", callCount)
	}
}
