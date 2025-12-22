package gh

import (
	"fmt"
	"math"
	"time"
)

// RetryConfig holds retry configuration
type RetryConfig struct {
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
}

// DefaultRetryConfig returns sensible defaults for API retries
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries: 3,
		BaseDelay:  1 * time.Second,
		MaxDelay:   30 * time.Second,
	}
}

// RetryableError represents an error that can be retried
type RetryableError struct {
	Err       error
	Retryable bool
}

func (e RetryableError) Error() string {
	return e.Err.Error()
}

func (e RetryableError) Unwrap() error {
	return e.Err
}

// IsRetryable determines if an error should trigger a retry
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Check for HTTP status codes that indicate temporary issues
	if httpErr, ok := err.(*RetryableError); ok {
		return httpErr.Retryable
	}

	// Check error message for common retryable patterns
	errStr := err.Error()
	retryablePatterns := []string{
		"502", "503", "504", // Server errors
		"429", // Rate limit
		"timeout",
		"connection refused",
		"network is unreachable",
		"temporary failure",
		"service unavailable",
	}

	for _, pattern := range retryablePatterns {
		if containsIgnoreCase(errStr, pattern) {
			return true
		}
	}

	return false
}

// containsIgnoreCase checks if a string contains a substring (case-insensitive)
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsInMiddle(s, substr)))
}

// containsInMiddle checks if substr appears anywhere in s
func containsInMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if equalIgnoreCase(s[i:i+len(substr)], substr) {
			return true
		}
	}
	return false
}

func equalIgnoreCase(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] && a[i] != b[i]-32 && a[i] != b[i]+32 {
			return false
		}
	}
	return true
}

// RetryWithBackoff executes a function with exponential backoff retry logic
func RetryWithBackoff(fn func() error, config RetryConfig) error {
	var lastErr error

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		err := fn()
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Don't retry on the last attempt
		if attempt == config.MaxRetries {
			break
		}

		// Check if error is retryable
		if !IsRetryable(err) {
			break // Don't retry non-retryable errors
		}

		// Calculate delay with exponential backoff
		delay := time.Duration(float64(config.BaseDelay) * math.Pow(2, float64(attempt)))
		if delay > config.MaxDelay {
			delay = config.MaxDelay
		}

		time.Sleep(delay)
	}

	return fmt.Errorf("failed after %d retries: %w", config.MaxRetries, lastErr)
}
