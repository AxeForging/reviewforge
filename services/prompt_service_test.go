package services

import (
	"os"
	"strings"
	"testing"

	"github.com/AxeForging/reviewforge/domain"
)

func TestPromptService_BuildSystemPrompt(t *testing.T) {
	svc := &PromptService{}

	t.Run("base prompt without update or persona", func(t *testing.T) {
		prompt := svc.BuildSystemPrompt(PromptOptions{})
		if !strings.Contains(prompt, "expert code reviewer") {
			t.Error("should contain base reviewer prompt")
		}
		if strings.Contains(prompt, "updates to a PR") {
			t.Error("should not contain update prompt")
		}
		if strings.Contains(prompt, "Persona") {
			t.Error("should not contain persona section")
		}
	})

	t.Run("with update prompt", func(t *testing.T) {
		prompt := svc.BuildSystemPrompt(PromptOptions{IsUpdate: true})
		if !strings.Contains(prompt, "updates to a PR") {
			t.Error("should contain update prompt")
		}
	})

	t.Run("with persona prompt", func(t *testing.T) {
		prompt := svc.BuildSystemPrompt(PromptOptions{PersonaPrompt: "Be friendly and encouraging."})
		if !strings.Contains(prompt, "Persona Instructions") {
			t.Error("should contain persona section header")
		}
		if !strings.Contains(prompt, "Be friendly and encouraging.") {
			t.Error("should contain persona prompt text")
		}
	})

	t.Run("with both update and persona", func(t *testing.T) {
		prompt := svc.BuildSystemPrompt(PromptOptions{IsUpdate: true, PersonaPrompt: "Be nerdy."})
		if !strings.Contains(prompt, "updates to a PR") {
			t.Error("should contain update prompt")
		}
		if !strings.Contains(prompt, "Be nerdy.") {
			t.Error("should contain persona prompt")
		}
	})

	t.Run("with language", func(t *testing.T) {
		prompt := svc.BuildSystemPrompt(PromptOptions{Language: "Portuguese"})
		if !strings.Contains(prompt, "Language Instructions") {
			t.Error("should contain language section header")
		}
		if !strings.Contains(prompt, "Portuguese") {
			t.Error("should contain language name")
		}
		if !strings.Contains(prompt, "JSON structure keys") {
			t.Error("should instruct to keep JSON keys in English")
		}
	})

	t.Run("with learning report", func(t *testing.T) {
		prompt := svc.BuildSystemPrompt(PromptOptions{IncludeLearning: true})
		if !strings.Contains(prompt, "Learning Report") {
			t.Error("should contain learning section header")
		}
		if !strings.Contains(prompt, "techniques_spotted") {
			t.Error("should instruct to include techniques")
		}
		if !strings.Contains(prompt, "what_went_well") {
			t.Error("should instruct to include what went well")
		}
		if !strings.Contains(prompt, "areas_to_improve") {
			t.Error("should instruct to include areas to improve")
		}
		if !strings.Contains(prompt, "key_takeaways") {
			t.Error("should instruct to include key takeaways")
		}
	})

	t.Run("with review rules", func(t *testing.T) {
		prompt := svc.BuildSystemPrompt(PromptOptions{ReviewRules: "Only comment on bugs."})
		if !strings.Contains(prompt, "Review Rules") {
			t.Error("should contain review rules section header")
		}
		if !strings.Contains(prompt, "Only comment on bugs.") {
			t.Error("should contain the rules text")
		}
	})

	t.Run("with strict changes", func(t *testing.T) {
		prompt := svc.BuildSystemPrompt(PromptOptions{StrictChanges: true})
		if !strings.Contains(prompt, "Strict Changes Mode") {
			t.Error("should contain strict changes section header")
		}
		if !strings.Contains(prompt, "Syntax errors") {
			t.Error("should mention syntax errors")
		}
		if !strings.Contains(prompt, "Degradation") {
			t.Error("should mention degradation")
		}
	})

	t.Run("all options combined", func(t *testing.T) {
		prompt := svc.BuildSystemPrompt(PromptOptions{
			IsUpdate:        true,
			PersonaPrompt:   "Be kind.",
			Language:        "Spanish",
			IncludeLearning: true,
			StrictChanges:   true,
		})
		if !strings.Contains(prompt, "updates to a PR") {
			t.Error("should contain update prompt")
		}
		if !strings.Contains(prompt, "Be kind.") {
			t.Error("should contain persona")
		}
		if !strings.Contains(prompt, "Spanish") {
			t.Error("should contain language")
		}
		if !strings.Contains(prompt, "techniques_spotted") {
			t.Error("should contain learning section")
		}
		if !strings.Contains(prompt, "Strict Changes Mode") {
			t.Error("should contain strict changes section")
		}
	})
}

func TestPromptService_BuildUserPrompt(t *testing.T) {
	svc := &PromptService{}

	req := domain.ReviewRequest{
		Files: []domain.FileDiff{
			{Path: "main.go", Diff: "+import fmt"},
		},
		PullRequest: domain.PRSummary{
			Title: "Add feature",
		},
	}

	prompt := svc.BuildUserPrompt(req)
	if !strings.Contains(prompt, "main.go") {
		t.Error("user prompt should contain file path")
	}
	if !strings.Contains(prompt, "Add feature") {
		t.Error("user prompt should contain PR title")
	}
}

func TestPromptService_ParseAIResponse(t *testing.T) {
	svc := &PromptService{}

	t.Run("valid JSON", func(t *testing.T) {
		raw := `{
			"summary": "Looks good overall.",
			"comments": [
				{"path": "main.go", "line": 5, "comment": "Missing error check", "severity": "critical"}
			],
			"suggestedAction": "comment",
			"confidence": 85
		}`

		out, err := svc.ParseAIResponse(raw)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out.Summary != "Looks good overall." {
			t.Errorf("Summary = %q", out.Summary)
		}
		if len(out.Comments) != 1 {
			t.Fatalf("expected 1 comment, got %d", len(out.Comments))
		}
		if out.Comments[0].Severity != "critical" {
			t.Errorf("Comment severity = %q, want critical", out.Comments[0].Severity)
		}
		if out.SuggestedAction != "comment" {
			t.Errorf("SuggestedAction = %q", out.SuggestedAction)
		}
		if out.Confidence != 85 {
			t.Errorf("Confidence = %d", out.Confidence)
		}
	})

	t.Run("JSON wrapped in code fence", func(t *testing.T) {
		raw := "```json\n{\"summary\":\"OK\",\"comments\":[],\"suggestedAction\":\"approve\",\"confidence\":90}\n```"
		out, err := svc.ParseAIResponse(raw)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out.Summary != "OK" {
			t.Errorf("Summary = %q", out.Summary)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		_, err := svc.ParseAIResponse("not json")
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("unknown suggestedAction defaults to comment", func(t *testing.T) {
		raw := `{"summary":"x","comments":[],"suggestedAction":"unknown","confidence":50}`
		out, err := svc.ParseAIResponse(raw)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out.SuggestedAction != "comment" {
			t.Errorf("SuggestedAction = %q, want comment", out.SuggestedAction)
		}
	})

	t.Run("missing severity defaults to suggestion", func(t *testing.T) {
		raw := `{"summary":"x","comments":[{"path":"a.go","line":1,"comment":"fix"}],"suggestedAction":"comment","confidence":50}`
		out, err := svc.ParseAIResponse(raw)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out.Comments[0].Severity != "suggestion" {
			t.Errorf("Severity = %q, want suggestion", out.Comments[0].Severity)
		}
	})

	t.Run("with learning section", func(t *testing.T) {
		raw := `{
			"summary": "Good PR",
			"comments": [],
			"suggestedAction": "approve",
			"confidence": 90,
			"learning": {
				"techniques_spotted": ["Error wrapping", "Dependency injection"],
				"what_went_well": ["Clean separation of concerns"],
				"areas_to_improve": ["Add context.Context support"],
				"key_takeaways": ["Always close response bodies"]
			}
		}`
		out, err := svc.ParseAIResponse(raw)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out.Learning == nil {
			t.Fatal("Learning should not be nil")
		}
		if len(out.Learning.TechniquesSpotted) != 2 {
			t.Errorf("TechniquesSpotted = %v", out.Learning.TechniquesSpotted)
		}
		if len(out.Learning.WhatWentWell) != 1 {
			t.Errorf("WhatWentWell = %v", out.Learning.WhatWentWell)
		}
	})
}

func TestResolveReviewRules(t *testing.T) {
	t.Run("preset concise", func(t *testing.T) {
		rules := ResolveReviewRules("concise", "", "")
		if !strings.Contains(rules, "ONLY comment on") {
			t.Error("concise preset should contain comment rules")
		}
	})

	t.Run("preset thorough", func(t *testing.T) {
		rules := ResolveReviewRules("thorough", "", "")
		if !strings.Contains(rules, "Comment on ALL") {
			t.Error("thorough preset should contain thorough rules")
		}
	})

	t.Run("unknown preset returns empty", func(t *testing.T) {
		rules := ResolveReviewRules("unknown", "", "")
		if rules != "" {
			t.Errorf("unknown preset should return empty, got %q", rules)
		}
	})

	t.Run("custom rules override preset", func(t *testing.T) {
		rules := ResolveReviewRules("concise", "My custom rules", "")
		if rules != "My custom rules" {
			t.Errorf("custom rules should override preset, got %q", rules)
		}
	})

	t.Run("custom rules file", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := tmpDir + "/rules.txt"
		os.WriteFile(path, []byte("Rules from file"), 0o644)

		rules := ResolveReviewRules("concise", "inline", path)
		if rules != "Rules from file" {
			t.Errorf("file should override all, got %q", rules)
		}
	})

	t.Run("missing file falls back to custom rules", func(t *testing.T) {
		rules := ResolveReviewRules("", "fallback", "/nonexistent/rules.txt")
		if rules != "fallback" {
			t.Errorf("should fall back to custom rules, got %q", rules)
		}
	})

	t.Run("empty everything returns empty", func(t *testing.T) {
		rules := ResolveReviewRules("", "", "")
		if rules != "" {
			t.Errorf("should return empty, got %q", rules)
		}
	})
}

func TestPromptService_FormatModelFooter(t *testing.T) {
	svc := &PromptService{}
	footer := svc.FormatModelFooter("openai", "gpt-4")
	expected := "_Code review performed by `OPENAI - gpt-4`._"
	if footer != expected {
		t.Errorf("FormatModelFooter = %q, want %q", footer, expected)
	}
}
