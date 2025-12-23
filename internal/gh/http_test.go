package gh

import (
	"errors"
	"net/http"
	"testing"
)

func TestWrapError(t *testing.T) {
	c := &Client{}

	tests := []struct {
		name       string
		err        error
		wantType   string // "auth", "notfound", "ratelimit", "retry", "other"
		wantNil    bool
	}{
		{
			name:    "nil error",
			err:     nil,
			wantNil: true,
		},
		{
			name:     "401 error",
			err:      errors.New("HTTP 401 Unauthorized"),
			wantType: "auth",
		},
		{
			name:     "403 error",
			err:      errors.New("HTTP 403 Forbidden"),
			wantType: "auth",
		},
		{
			name:     "403 rate limit error",
			err:      errors.New("HTTP 403 rate limit exceeded"),
			wantType: "ratelimit",
		},
		{
			name:     "404 error",
			err:      errors.New("HTTP 404 Not Found"),
			wantType: "notfound",
		},
		{
			name:     "429 error",
			err:      errors.New("HTTP 429 Too Many Requests"),
			wantType: "ratelimit",
		},
		{
			name:     "502 error",
			err:      errors.New("HTTP 502 Bad Gateway"),
			wantType: "retry",
		},
		{
			name:     "503 error",
			err:      errors.New("HTTP 503 Service Unavailable"),
			wantType: "retry",
		},
		{
			name:     "504 error",
			err:      errors.New("HTTP 504 Gateway Timeout"),
			wantType: "retry",
		},
		{
			name:     "timeout error",
			err:      errors.New("connection timeout"),
			wantType: "retry",
		},
		{
			name:     "connection error",
			err:      errors.New("connection refused"),
			wantType: "retry",
		},
		{
			name:     "other error",
			err:      errors.New("some other error"),
			wantType: "other",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.wrapError(tt.err)

			if tt.wantNil {
				if result != nil {
					t.Errorf("wrapError() = %v, want nil", result)
				}
				return
			}

			if result == nil {
				t.Fatal("wrapError() = nil, want error")
			}

			switch tt.wantType {
			case "auth":
				if _, ok := result.(*AuthError); !ok {
					t.Errorf("wrapError() type = %T, want *AuthError", result)
				}
			case "notfound":
				if _, ok := result.(*NotFoundError); !ok {
					t.Errorf("wrapError() type = %T, want *NotFoundError", result)
				}
			case "ratelimit":
				if _, ok := result.(*RateLimitError); !ok {
					t.Errorf("wrapError() type = %T, want *RateLimitError", result)
				}
			case "retry":
				// Server errors are wrapped but not in a specific type
				errStr := result.Error()
				if !contains(errStr, "will retry") {
					t.Errorf("wrapError() = %q, want error containing 'will retry'", errStr)
				}
			case "other":
				// Should return the original error
				if result != tt.err {
					t.Errorf("wrapError() = %v, want original error %v", result, tt.err)
				}
			}
		})
	}
}

func TestCheckHTTPError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		statusCode int
		want       bool
	}{
		{"nil error", nil, 404, false},
		{"404 matches 404", errors.New("HTTP 404 Not Found"), 404, true},
		{"404 not matches 500", errors.New("HTTP 404 Not Found"), 500, false},
		{"401 matches 401", errors.New("401 Unauthorized"), 401, true},
		{"no status code", errors.New("network error"), 500, false},
		{"500 matches 500", errors.New("error 500 server"), 500, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CheckHTTPError(tt.err, tt.statusCode); got != tt.want {
				t.Errorf("CheckHTTPError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsHTTPError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"400 Bad Request", errors.New("HTTP 400 Bad Request"), true},
		{"401 Unauthorized", errors.New("401 Unauthorized"), true},
		{"403 Forbidden", errors.New("403 Forbidden"), true},
		{"404 Not Found", errors.New("404 Not Found"), true},
		{"429 Too Many Requests", errors.New("429 rate limit"), true},
		{"500 Internal Server Error", errors.New("500 error"), true},
		{"502 Bad Gateway", errors.New("502 Bad Gateway"), true},
		{"503 Service Unavailable", errors.New("503"), true},
		{"network error", errors.New("network timeout"), false},
		{"generic error", errors.New("something failed"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsHTTPError(tt.err); got != tt.want {
				t.Errorf("IsHTTPError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHTTPStatusCodes(t *testing.T) {
	// Verify common HTTP status codes are handled
	codes := []struct {
		code int
		name string
	}{
		{http.StatusBadRequest, "Bad Request"},
		{http.StatusUnauthorized, "Unauthorized"},
		{http.StatusForbidden, "Forbidden"},
		{http.StatusNotFound, "Not Found"},
		{http.StatusTooManyRequests, "Too Many Requests"},
		{http.StatusInternalServerError, "Internal Server Error"},
		{http.StatusBadGateway, "Bad Gateway"},
		{http.StatusServiceUnavailable, "Service Unavailable"},
	}

	for _, tc := range codes {
		t.Run(tc.name, func(t *testing.T) {
			// Test that CheckHTTPError can detect the code
			errMsg := errors.New("error: " + http.StatusText(tc.code) + " (" + itoa(tc.code) + ")")
			if !CheckHTTPError(errMsg, tc.code) {
				t.Errorf("CheckHTTPError should detect %d in error message", tc.code)
			}
		})
	}
}

// contains checks if s contains substr
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// itoa converts int to string (simple version)
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
