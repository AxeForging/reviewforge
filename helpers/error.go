package helpers

import (
	"errors"
	"fmt"
)

// Sentinel errors
var (
	ErrNoAPIKey       = errors.New("AI API key is required")
	ErrNoGitHubToken  = errors.New("GitHub token is required")
	ErrNoPRNumber     = errors.New("PR number is required")
	ErrNoRepo         = errors.New("repository is required (format: owner/repo)")
	ErrInvalidProvider = errors.New("unsupported AI provider (use: openai, anthropic, gemini)")
	ErrAIRequest      = errors.New("AI provider request failed")
	ErrGitHubAPI      = errors.New("GitHub API request failed")
	ErrDiffParse      = errors.New("failed to parse diff")
	ErrJSONParse      = errors.New("failed to parse AI response JSON")
)

// FormatError represents a formatted error with context
type FormatError struct {
	Operation string
	Details   string
	Err       error
}

func (e *FormatError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Operation, e.Details, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Operation, e.Details)
}

func (e *FormatError) Unwrap() error {
	return e.Err
}

// NewFormatError creates a new format error
func NewFormatError(operation, details string, err error) *FormatError {
	return &FormatError{
		Operation: operation,
		Details:   details,
		Err:       err,
	}
}

// WrapError wraps an error with additional context
func WrapError(err error, operation, details string) error {
	if err == nil {
		return nil
	}
	return NewFormatError(operation, details, err)
}
