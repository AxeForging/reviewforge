package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/AxeForging/reviewforge/helpers"
	"github.com/rs/zerolog/log"
)

// OpenAIProvider implements AIProvider using the OpenAI Chat Completions API
type OpenAIProvider struct {
	APIKey  string
	Model   string
	BaseURL string // override for testing
}

func (p *OpenAIProvider) Name() string { return "openai" }

func (p *OpenAIProvider) Review(systemPrompt, userPrompt string, temperature float64) (string, error) {
	baseURL := p.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}

	isO1 := strings.Contains(p.Model, "o1-mini") || strings.Contains(p.Model, "o1-preview")

	systemRole := "system"
	if isO1 {
		systemRole = "user"
	}

	temp := temperature
	if isO1 {
		temp = 1
	}

	respFormat := map[string]string{"type": "json_object"}
	if isO1 {
		respFormat = map[string]string{"type": "text"}
	}

	body := map[string]interface{}{
		"model": p.Model,
		"messages": []map[string]string{
			{"role": systemRole, "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"temperature":     temp,
		"response_format": respFormat,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return "", helpers.WrapError(err, "openai", "failed to marshal request")
	}

	req, err := http.NewRequest("POST", baseURL+"/v1/chat/completions", bytes.NewReader(data))
	if err != nil {
		return "", helpers.WrapError(err, "openai", "failed to create request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIKey)

	log.Debug().Str("model", p.Model).Msg("Sending request to OpenAI")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", helpers.WrapError(err, "openai", "request failed")
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", helpers.WrapError(err, "openai", "failed to read response")
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: OpenAI returned %d: %s", helpers.ErrAIRequest, resp.StatusCode, string(respBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", helpers.WrapError(err, "openai", "failed to parse response")
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("%w: OpenAI returned no choices", helpers.ErrAIRequest)
	}

	return result.Choices[0].Message.Content, nil
}
