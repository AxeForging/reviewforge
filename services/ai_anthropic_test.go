package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAnthropicProvider_Review(t *testing.T) {
	// Anthropic response text (without the leading "{" since assistant prefill adds it)
	responseText := `"summary":"Nice work","comments":[],"suggestedAction":"approve","confidence":95}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/messages" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("x-api-key") != "test-key" {
			t.Error("missing x-api-key header")
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("anthropic-version = %q", r.Header.Get("anthropic-version"))
		}

		resp := map[string]interface{}{
			"content": []map[string]string{
				{"type": "text", "text": responseText},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := &AnthropicProvider{
		APIKey:  "test-key",
		Model:   "claude-sonnet-4-20250514",
		BaseURL: server.URL,
	}

	result, err := provider.Review("system prompt", "user prompt", 0.3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be valid JSON with the prepended "{"
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("result should be valid JSON: %v\nresult: %s", err, result)
	}
	if parsed["summary"] != "Nice work" {
		t.Errorf("summary = %v", parsed["summary"])
	}
}

func TestAnthropicProvider_Review_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid key"}`))
	}))
	defer server.Close()

	provider := &AnthropicProvider{
		APIKey:  "bad-key",
		Model:   "claude-sonnet-4-20250514",
		BaseURL: server.URL,
	}

	_, err := provider.Review("system", "user", 0)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error should mention status code: %v", err)
	}
}

func TestAnthropicProvider_Name(t *testing.T) {
	p := &AnthropicProvider{}
	if p.Name() != "anthropic" {
		t.Errorf("Name() = %q", p.Name())
	}
}
