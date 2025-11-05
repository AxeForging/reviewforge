package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAIProvider_Review(t *testing.T) {
	reviewJSON := `{"summary":"LGTM","comments":[],"suggestedAction":"approve","confidence":90}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			t.Error("missing Authorization header")
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["model"] != "gpt-4" {
			t.Errorf("model = %v", body["model"])
		}

		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": reviewJSON,
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := &OpenAIProvider{
		APIKey:  "test-key",
		Model:   "gpt-4",
		BaseURL: server.URL,
	}

	result, err := provider.Review("system prompt", "user prompt", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != reviewJSON {
		t.Errorf("result = %q", result)
	}
}

func TestOpenAIProvider_Review_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":"rate limited"}`))
	}))
	defer server.Close()

	provider := &OpenAIProvider{
		APIKey:  "test-key",
		Model:   "gpt-4",
		BaseURL: server.URL,
	}

	_, err := provider.Review("system", "user", 0)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "429") {
		t.Errorf("error should mention status code: %v", err)
	}
}

func TestOpenAIProvider_Name(t *testing.T) {
	p := &OpenAIProvider{}
	if p.Name() != "openai" {
		t.Errorf("Name() = %q", p.Name())
	}
}
