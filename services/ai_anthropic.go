package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/AxeForging/reviewforge/domain"
	"github.com/AxeForging/reviewforge/helpers"
	"github.com/rs/zerolog/log"
)

// AnthropicProvider implements AIProvider using the Anthropic Messages API
type AnthropicProvider struct {
	APIKey  string
	Model   string
	BaseURL string // override for testing
}

func (p *AnthropicProvider) Name() string { return "anthropic" }

func (p *AnthropicProvider) Review(systemPrompt, userPrompt string, temperature float64) (string, *domain.TokenUsage, error) {
	baseURL := p.BaseURL
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}

	body := map[string]interface{}{
		"model":      p.Model,
		"max_tokens": 4096,
		"system":     systemPrompt,
		"messages": []map[string]interface{}{
			{"role": "user", "content": userPrompt},
			{"role": "assistant", "content": "{"},
		},
		"temperature": temperature,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return "", nil, helpers.WrapError(err, "anthropic", "failed to marshal request")
	}

	req, err := http.NewRequest("POST", baseURL+"/v1/messages", bytes.NewReader(data))
	if err != nil {
		return "", nil, helpers.WrapError(err, "anthropic", "failed to create request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	log.Debug().Str("model", p.Model).Msg("Sending request to Anthropic")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", nil, helpers.WrapError(err, "anthropic", "request failed")
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, helpers.WrapError(err, "anthropic", "failed to read response")
	}

	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("%w: Anthropic returned %d: %s", helpers.ErrAIRequest, resp.StatusCode, string(respBody))
	}

	var result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", nil, helpers.WrapError(err, "anthropic", "failed to parse response")
	}

	if len(result.Content) == 0 {
		return "", nil, fmt.Errorf("%w: Anthropic returned no content", helpers.ErrAIRequest)
	}

	usage := &domain.TokenUsage{
		PromptTokens:     result.Usage.InputTokens,
		CompletionTokens: result.Usage.OutputTokens,
		TotalTokens:      result.Usage.InputTokens + result.Usage.OutputTokens,
	}

	// Prepend the "{" we used as assistant prefill
	return "{" + result.Content[0].Text, usage, nil
}
