package services

import (
	"encoding/json"
	"strings"

	"github.com/AxeForging/reviewforge/domain"
	"github.com/AxeForging/reviewforge/helpers"
	"github.com/rs/zerolog/log"
)

// PromptService builds system and user prompts for AI providers and parses their JSON responses
type PromptService struct{}

const outputFormat = `{
  "summary": "",
  "comments": [{"path": "file_path", "line": 1, "comment": "comment text", "severity": "critical|warning|suggestion"}],
  "suggestedAction": "approve|request_changes|comment",
  "confidence": 85
}`

const baseCodeReviewPrompt = `You are an expert code reviewer. Analyze the provided code changes and provide detailed, actionable feedback.

Follow this JSON format:
` + outputFormat + `

------
Understanding the diff:
- Lines starting with "-" (del) show code that was REMOVED
- Lines starting with "+" (add) show code that was ADDED
- Lines without prefix (normal) show unchanged context

------
For the "summary" field, use Markdown formatting and follow these guidelines:
1. Core Changes
   - What is the main purpose/goal of this PR?
   - Only highlight the most impactful changes

2. Concerns (if any)
   - Security vulnerabilities
   - Performance degradation
   - Critical logic flaws
   - Breaking API changes without migration path

3. Verdict:
   Should be one of the following:
   - Approve: Changes look good and are safe to merge
   - Comment: Unsure about the changes, needs more discussion
   - Request Changes: ONLY for serious issues such as:
     * Security vulnerabilities
     * Critical performance issues
     * Broken core functionality
     * Data integrity risks
     * Production stability threats

   Normal code improvements, refactoring suggestions, or breaking changes
   with clear migration paths should use "Comment" instead.

------
For the "comments" field:

- ONLY add comments for actual issues that need to be addressed
- DO NOT add comments for:
  * Compliments or positive feedback
  * Style preferences
  * Minor suggestions
  * Obvious changes
  * General observations
- Each comment must be:
  * Actionable (something specific that needs to change)
  * Important enough to discuss
  * Related to code quality, performance, or correctness
- Each comment should have the fields: path, line, comment, severity
- severity must be one of: "critical", "warning", "suggestion"
- ONLY use line numbers that appear in the "diff" property of each file
- DO NOT use line number 0 or line numbers not present in the diff
- Focus on new code (lines with "+") and the impact of changes

ABOVE anything else, DO NOT repeat the same comment multiple times.

------
For the "suggestedAction" field, provide one of: "approve", "request_changes", "comment"

------
For the "confidence" field, provide a number between 0 and 100.`

const updateReviewPrompt = `
When reviewing updates to a PR:
1. Focus on the modified sections but consider their context
2. Reference previous comments if they're still relevant
3. Acknowledge fixed issues from previous reviews
4. Only comment on new issues or unresolved previous issues
5. Consider the cumulative impact of changes
6. IMPORTANT: Only use line numbers that appear in the current "diff" field`

// BuildSystemPrompt constructs the system prompt for the AI, including persona if set
func (s *PromptService) BuildSystemPrompt(isUpdate bool, personaPrompt string) string {
	var b strings.Builder
	b.WriteString(baseCodeReviewPrompt)

	if isUpdate {
		b.WriteString("\n")
		b.WriteString(updateReviewPrompt)
	}

	if personaPrompt != "" {
		b.WriteString("\n\n------\nPersona Instructions:\n")
		b.WriteString(personaPrompt)
	}

	return b.String()
}

// BuildUserPrompt constructs the user message containing the PR data for the AI
func (s *PromptService) BuildUserPrompt(req domain.ReviewRequest) string {
	data, err := json.Marshal(req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal review request")
		return "{}"
	}
	return string(data)
}

// ParseAIResponse parses the AI's JSON response into a structured AIReviewOutput
func (s *PromptService) ParseAIResponse(raw string) (*domain.AIReviewOutput, error) {
	// Strip markdown code fences if present
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "```json") {
		raw = strings.TrimPrefix(raw, "```json")
		raw = strings.TrimSuffix(raw, "```")
		raw = strings.TrimSpace(raw)
	} else if strings.HasPrefix(raw, "```") {
		raw = strings.TrimPrefix(raw, "```")
		raw = strings.TrimSuffix(raw, "```")
		raw = strings.TrimSpace(raw)
	}

	var output domain.AIReviewOutput
	if err := json.Unmarshal([]byte(raw), &output); err != nil {
		return nil, helpers.WrapError(err, "parse", "failed to parse AI response JSON")
	}

	// Normalize suggested action
	output.SuggestedAction = strings.ToLower(output.SuggestedAction)
	switch output.SuggestedAction {
	case "approve", "request_changes", "comment":
		// valid
	default:
		output.SuggestedAction = "comment"
	}

	// Default severity for comments missing it
	for i := range output.Comments {
		if output.Comments[i].Severity == "" {
			output.Comments[i].Severity = "suggestion"
		}
	}

	return &output, nil
}

// FormatModelFooter returns the model info line to append to the review summary
func (s *PromptService) FormatModelFooter(provider, model string) string {
	return "_Code review performed by `" + strings.ToUpper(provider) + " - " + model + "`._"
}
