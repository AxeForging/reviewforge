package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/AxeForging/reviewforge/helpers"
	"github.com/rs/zerolog/log"
)

// GeminiProvider implements AIProvider using the Google Gemini API
type GeminiProvider struct {
	APIKey  string
	Model   string
	BaseURL string // override for testing
}

func (p *GeminiProvider) Name() string { return "gemini" }

func (p *GeminiProvider) Review(systemPrompt, userPrompt string, temperature float64) (string, error) {
	baseURL := p.BaseURL
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com"
	}

	body := map[string]interface{}{
		"system_instruction": map[string]interface{}{
			"parts": []map[string]string{
				{"text": systemPrompt},
			},
		},
		"contents": []map[string]interface{}{
			{
				"role": "user",
				"parts": []map[string]string{
					{"text": userPrompt},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"responseMimeType": "application/json",
			"temperature":      temperature,
		},
	}

	data, err := json.Marshal(body)
	if err != nil {
		return "", helpers.WrapError(err, "gemini", "failed to marshal request")
	}

	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", baseURL, p.Model, p.APIKey)
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return "", helpers.WrapError(err, "gemini", "failed to create request")
	}
	req.Header.Set("Content-Type", "application/json")

	log.Debug().Str("model", p.Model).Msg("Sending request to Gemini")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", helpers.WrapError(err, "gemini", "request failed")
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", helpers.WrapError(err, "gemini", "failed to read response")
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: Gemini returned %d: %s", helpers.ErrAIRequest, resp.StatusCode, string(respBody))
	}

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", helpers.WrapError(err, "gemini", "failed to parse response")
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("%w: Gemini returned no content", helpers.ErrAIRequest)
	}

	return result.Candidates[0].Content.Parts[0].Text, nil
}
