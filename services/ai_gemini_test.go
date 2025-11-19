package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGeminiProvider_Review(t *testing.T) {
	reviewJSON := `{"summary":"Solid PR","comments":[],"suggestedAction":"approve","confidence":88}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "gemini-pro") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("key") != "test-key" {
			t.Error("missing API key in query")
		}

		resp := map[string]interface{}{
			"candidates": []map[string]interface{}{
				{
					"content": map[string]interface{}{
						"parts": []map[string]string{
							{"text": reviewJSON},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := &GeminiProvider{
		APIKey:  "test-key",
		Model:   "gemini-pro",
		BaseURL: server.URL,
	}

	result, _, err := provider.Review("system prompt", "user prompt", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != reviewJSON {
		t.Errorf("result = %q", result)
	}
}

func TestGeminiProvider_Review_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":"forbidden"}`))
	}))
	defer server.Close()

	provider := &GeminiProvider{
		APIKey:  "test-key",
		Model:   "gemini-pro",
		BaseURL: server.URL,
	}

	_, _, err := provider.Review("system", "user", 0)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("error should mention status code: %v", err)
	}
}

func TestGeminiProvider_Name(t *testing.T) {
	p := &GeminiProvider{}
	if p.Name() != "gemini" {
		t.Errorf("Name() = %q", p.Name())
	}
}
