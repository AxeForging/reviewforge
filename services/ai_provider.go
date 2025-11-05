package services

import (
	"fmt"
	"strings"

	"github.com/AxeForging/reviewforge/helpers"
)

// AIProvider is the interface for AI code review providers
type AIProvider interface {
	// Review sends a code review request and returns the raw JSON response body
	Review(systemPrompt, userPrompt string, temperature float64) (string, error)
	// Name returns the provider name for logging
	Name() string
}

// NewAIProvider creates an AIProvider based on the provider name
func NewAIProvider(provider, apiKey, model string) (AIProvider, error) {
	if apiKey == "" {
		return nil, helpers.ErrNoAPIKey
	}

	switch strings.ToLower(provider) {
	case "openai":
		return &OpenAIProvider{APIKey: apiKey, Model: model}, nil
	case "anthropic":
		return &AnthropicProvider{APIKey: apiKey, Model: model}, nil
	case "gemini", "google":
		return &GeminiProvider{APIKey: apiKey, Model: model}, nil
	default:
		return nil, fmt.Errorf("%w: %s", helpers.ErrInvalidProvider, provider)
	}
}
