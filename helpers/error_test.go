package helpers

import (
	"errors"
	"testing"
)

func TestFormatError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *FormatError
		expected string
	}{
		{
			name:     "with underlying error",
			err:      NewFormatError("review", "failed to call AI", errors.New("timeout")),
			expected: "review: failed to call AI (timeout)",
		},
		{
			name:     "without underlying error",
			err:      NewFormatError("validation", "missing API key", nil),
			expected: "validation: missing API key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestFormatError_Unwrap(t *testing.T) {
	inner := errors.New("connection refused")
	err := NewFormatError("github", "API call failed", inner)
	if !errors.Is(err, inner) {
		t.Error("Unwrap should return the inner error")
	}
}

func TestWrapError(t *testing.T) {
	t.Run("nil error returns nil", func(t *testing.T) {
		if got := WrapError(nil, "op", "details"); got != nil {
			t.Errorf("WrapError(nil) = %v, want nil", got)
		}
	})

	t.Run("wraps non-nil error", func(t *testing.T) {
		inner := errors.New("boom")
		got := WrapError(inner, "review", "AI failed")
		if got == nil {
			t.Fatal("WrapError should return non-nil error")
		}
		if got.Error() != "review: AI failed (boom)" {
			t.Errorf("Error() = %q, want %q", got.Error(), "review: AI failed (boom)")
		}
	})
}

func TestSentinelErrors(t *testing.T) {
	// Ensure sentinel errors are distinct
	sentinels := []error{
		ErrNoAPIKey, ErrNoGitHubToken, ErrNoPRNumber, ErrNoRepo,
		ErrInvalidProvider, ErrAIRequest, ErrGitHubAPI, ErrDiffParse, ErrJSONParse,
	}
	for i, a := range sentinels {
		for j, b := range sentinels {
			if i != j && errors.Is(a, b) {
				t.Errorf("sentinel errors %d and %d should be distinct", i, j)
			}
		}
	}
}
