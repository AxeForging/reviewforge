package services

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/AxeForging/reviewforge/domain"
)

func newTestGitHubService(serverURL string) *GitHubService {
	return &GitHubService{
		Token:   "test-token",
		Owner:   "testowner",
		Repo:    "testrepo",
		BaseURL: serverURL,
	}
}

func TestNewGitHubService(t *testing.T) {
	tests := []struct {
		repo    string
		wantErr bool
	}{
		{"owner/repo", false},
		{"owner/", true},
		{"/repo", true},
		{"noslash", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.repo, func(t *testing.T) {
			svc, err := NewGitHubService("token", tt.repo)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if svc.Owner != "owner" || svc.Repo != "repo" {
				t.Errorf("Owner=%q, Repo=%q", svc.Owner, svc.Repo)
			}
		})
	}
}

func TestGitHubService_GetPRDetails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/testowner/testrepo/pulls/42" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			t.Error("missing Authorization header")
		}

		resp := map[string]interface{}{
			"title": "Add feature X",
			"body":  "This PR adds feature X",
			"base":  map[string]string{"sha": "abc123"},
			"head":  map[string]string{"sha": "def456"},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	svc := newTestGitHubService(server.URL)
	pr, err := svc.GetPRDetails(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pr.Title != "Add feature X" {
		t.Errorf("Title = %q", pr.Title)
	}
	if pr.BaseSHA != "abc123" {
		t.Errorf("BaseSHA = %q", pr.BaseSHA)
	}
	if pr.HeadSHA != "def456" {
		t.Errorf("HeadSHA = %q", pr.HeadSHA)
	}
	if pr.Number != 42 {
		t.Errorf("Number = %d", pr.Number)
	}
}

func TestGitHubService_GetPRDiff(t *testing.T) {
	expectedDiff := "diff --git a/main.go b/main.go\n+import fmt"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/vnd.github.v3.diff" {
			t.Errorf("Accept = %q", r.Header.Get("Accept"))
		}
		w.Write([]byte(expectedDiff))
	}))
	defer server.Close()

	svc := newTestGitHubService(server.URL)

	diff, err := svc.GetPRDiff(1, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if diff != expectedDiff {
		t.Errorf("diff = %q", diff)
	}
}

func TestGitHubService_GetFileContent(t *testing.T) {
	fileContent := "package main\n\nfunc main() {}\n"
	encoded := base64.StdEncoding.EncodeToString([]byte(fileContent))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/contents/main.go") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		resp := map[string]string{
			"content":  encoded,
			"encoding": "base64",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	svc := newTestGitHubService(server.URL)

	content, err := svc.GetFileContent("main.go", "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content != fileContent {
		t.Errorf("content = %q", content)
	}
}

func TestGitHubService_GetLastReviewedCommit(t *testing.T) {
	t.Run("found bot review", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reviews := []map[string]interface{}{
				{"user": map[string]string{"login": "human"}, "commit_id": "aaa"},
				{"user": map[string]string{"login": "github-actions[bot]"}, "commit_id": "bbb"},
			}
			json.NewEncoder(w).Encode(reviews)
		}))
		defer server.Close()

		svc := newTestGitHubService(server.URL)
		sha, err := svc.GetLastReviewedCommit(1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sha != "bbb" {
			t.Errorf("sha = %q, want bbb", sha)
		}
	})

	t.Run("no bot review", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode([]map[string]interface{}{})
		}))
		defer server.Close()

		svc := newTestGitHubService(server.URL)
		sha, err := svc.GetLastReviewedCommit(1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sha != "" {
			t.Errorf("sha = %q, want empty", sha)
		}
	})
}

func TestGitHubService_SubmitReview(t *testing.T) {
	t.Run("successful submission", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)

			if body["event"] != "COMMENT" {
				t.Errorf("event = %v", body["event"])
			}

			comments := body["comments"].([]interface{})
			if len(comments) != 1 {
				t.Errorf("expected 1 comment, got %d", len(comments))
			}

			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{}`))
		}))
		defer server.Close()

		svc := newTestGitHubService(server.URL)
		output := &domain.AIReviewOutput{
			Summary: "Looks good",
			Comments: []domain.ReviewComment{
				{Path: "main.go", Line: 5, Comment: "Fix this", Severity: "warning"},
			},
			SuggestedAction: "comment",
		}

		err := svc.SubmitReview(1, output, "comment")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("422 retries without comments", func(t *testing.T) {
		callCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			if callCount == 1 {
				w.WriteHeader(http.StatusUnprocessableEntity)
				w.Write([]byte(`{"message":"invalid"}`))
				return
			}
			// Second call should have empty comments
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			comments := body["comments"].([]interface{})
			if len(comments) != 0 {
				t.Errorf("retry should have 0 comments, got %d", len(comments))
			}
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{}`))
		}))
		defer server.Close()

		svc := newTestGitHubService(server.URL)
		output := &domain.AIReviewOutput{
			Summary: "Review",
			Comments: []domain.ReviewComment{
				{Path: "a.go", Line: 1, Comment: "Issue"},
			},
		}

		err := svc.SubmitReview(1, output, "comment")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if callCount != 2 {
			t.Errorf("expected 2 calls, got %d", callCount)
		}
	})
}
