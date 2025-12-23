package gh

import (
	"errors"
	"strings"
	"testing"
)

func TestAuthError(t *testing.T) {
	innerErr := errors.New("token expired")
	authErr := &AuthError{Err: innerErr}

	// Test Error()
	errStr := authErr.Error()
	if !strings.Contains(errStr, "authentication failed") {
		t.Errorf("AuthError.Error() = %q, want to contain 'authentication failed'", errStr)
	}
	if !strings.Contains(errStr, "token expired") {
		t.Errorf("AuthError.Error() = %q, want to contain inner error", errStr)
	}
	if !strings.Contains(errStr, "gh auth login") {
		t.Errorf("AuthError.Error() = %q, want to contain suggestion", errStr)
	}

	// Test Unwrap()
	if authErr.Unwrap() != innerErr {
		t.Errorf("AuthError.Unwrap() = %v, want %v", authErr.Unwrap(), innerErr)
	}
}

func TestNotFoundError(t *testing.T) {
	innerErr := errors.New("404")
	notFoundErr := &NotFoundError{Resource: "repository", Err: innerErr}

	// Test Error()
	errStr := notFoundErr.Error()
	if !strings.Contains(errStr, "repository") {
		t.Errorf("NotFoundError.Error() = %q, want to contain 'repository'", errStr)
	}
	if !strings.Contains(errStr, "not found") {
		t.Errorf("NotFoundError.Error() = %q, want to contain 'not found'", errStr)
	}
	if !strings.Contains(errStr, "404") {
		t.Errorf("NotFoundError.Error() = %q, want to contain inner error", errStr)
	}

	// Test Unwrap()
	if notFoundErr.Unwrap() != innerErr {
		t.Errorf("NotFoundError.Unwrap() = %v, want %v", notFoundErr.Unwrap(), innerErr)
	}
}

func TestRateLimitError(t *testing.T) {
	innerErr := errors.New("429 Too Many Requests")
	rateLimitErr := &RateLimitError{Err: innerErr}

	// Test Error()
	errStr := rateLimitErr.Error()
	if !strings.Contains(errStr, "rate limited") {
		t.Errorf("RateLimitError.Error() = %q, want to contain 'rate limited'", errStr)
	}
	if !strings.Contains(errStr, "429") {
		t.Errorf("RateLimitError.Error() = %q, want to contain inner error", errStr)
	}

	// Test Unwrap()
	if rateLimitErr.Unwrap() != innerErr {
		t.Errorf("RateLimitError.Unwrap() = %v, want %v", rateLimitErr.Unwrap(), innerErr)
	}
}

func TestErrorsUnwrapChain(t *testing.T) {
	innerErr := errors.New("root cause")

	// Test that errors.Is works with wrapped errors
	authErr := &AuthError{Err: innerErr}
	if !errors.Is(authErr, innerErr) {
		t.Error("errors.Is(AuthError, innerErr) = false, want true")
	}

	notFoundErr := &NotFoundError{Resource: "workflow", Err: innerErr}
	if !errors.Is(notFoundErr, innerErr) {
		t.Error("errors.Is(NotFoundError, innerErr) = false, want true")
	}

	rateLimitErr := &RateLimitError{Err: innerErr}
	if !errors.Is(rateLimitErr, innerErr) {
		t.Error("errors.Is(RateLimitError, innerErr) = false, want true")
	}
}

func TestSentinelErrors(t *testing.T) {
	// Test that sentinel errors are defined and have expected messages
	if ErrNotAuthenticated.Error() == "" {
		t.Error("ErrNotAuthenticated has empty message")
	}
	if ErrRepoNotFound.Error() == "" {
		t.Error("ErrRepoNotFound has empty message")
	}
	if ErrRateLimited.Error() == "" {
		t.Error("ErrRateLimited has empty message")
	}
	if ErrNoRuns.Error() == "" {
		t.Error("ErrNoRuns has empty message")
	}
}
