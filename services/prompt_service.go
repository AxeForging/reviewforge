package services

import (
	"encoding/json"
	"os"
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
### Core Changes
- What is the main purpose/goal of this PR?
- Only highlight the most impactful changes

---
### Concerns (if any)
- Security vulnerabilities
- Performance degradation
- Critical logic flaws
- Breaking API changes without migration path

---
### Verdict
Should be one of the following:
- **Approve**: Changes look good and are safe to merge
- **Comment**: Unsure about the changes, needs more discussion
- **Request Changes**: ONLY for serious issues such as:
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

// Built-in review rules presets
var reviewRulesPresets = map[string]string{
	"concise": `CRITICAL OVERRIDE — These rules REPLACE all previous comment guidelines above.

You MUST ONLY add line comments for:
1. Clear logical errors or bugs that will cause incorrect behavior
2. Security vulnerabilities (exposed secrets, injection, unsafe operations)
3. Breaking changes that degrade existing functionality
4. Crashes or runtime errors

You MUST NOT add line comments for ANY of the following (zero tolerance):
1. Suggestions to use defer, context.Context, or different patterns — these are style preferences
2. Suggesting better approaches, alternative implementations, or "improvements"
3. Variable or function naming
4. Performance optimizations unless they cause measurable degradation
5. Code style or idiom preferences (even if the current approach is less common)
6. Missing tests or test suggestions
7. Linting issues
8. Backoff strategy suggestions (linear vs exponential)
9. Anything that starts with "you could", "consider", "it would be nice", or "for future"
10. Changes to numeric values unless they clearly break functionality

If the code works correctly, compiles, and doesn't break anything: DO NOT comment on it.
If you are unsure whether something is a real bug: DO NOT comment.
An empty comments array is a GOOD outcome when the code is functional.

For the summary, ALWAYS include these sections using Markdown and separate them with ---:
### Core Changes
What the PR does (2-3 bullet points max).
---
### Concerns
Only if there are real bugs/security/breaking issues. Omit this section if there are none.
---
### Verdict
**Approve**, **Comment**, or **Request Changes** — with a short justification.

Do not list suggestions or improvements in the summary.
Prepend ⚠️ for critical issues only.`,

	"thorough": `Comment on ALL of the following:
1. Bugs, logical errors, and edge cases
2. Security vulnerabilities and unsafe patterns
3. Performance issues and inefficiencies
4. Breaking changes or backward compatibility concerns
5. Error handling gaps (missing checks, swallowed errors)
6. Resource leaks (unclosed connections, missing cleanup)
7. Concurrency issues (race conditions, deadlocks)
8. API design concerns (naming, contracts, versioning)

DO NOT comment on:
1. Pure code style or formatting preferences
2. Obvious or trivial changes
3. Compliments or positive-only feedback

Be thorough but actionable — every comment should explain the problem and suggest a fix.`,
}

// PromptOptions configures system prompt generation
type PromptOptions struct {
	IsUpdate        bool
	PersonaPrompt   string
	Language        string
	IncludeLearning bool
	StrictChanges   bool
	ReviewRules     string
}

// BuildSystemPrompt constructs the system prompt for the AI, including persona, language, and learning sections
func (s *PromptService) BuildSystemPrompt(opts PromptOptions) string {
	var b strings.Builder
	b.WriteString(baseCodeReviewPrompt)

	if opts.IsUpdate {
		b.WriteString("\n")
		b.WriteString(updateReviewPrompt)
	}

	if opts.StrictChanges {
		b.WriteString("\n\n------\nStrict Changes Mode:\n")
		b.WriteString(`IMPORTANT: You may ONLY use "request_changes" when the code has:
- Syntax errors that will prevent compilation or execution
- Degradation of existing functionality (breaking what already works)
- Runtime errors that will crash the application

For everything else — style, best practices, performance suggestions, missing tests,
code smells, refactoring opportunities — use "comment" or "approve".
Do NOT request changes for improvements, suggestions, or new patterns to adopt.
Only block the PR if it genuinely breaks something.`)
	}

	if opts.ReviewRules != "" {
		b.WriteString("\n\n------\nReview Rules:\n")
		b.WriteString(opts.ReviewRules)
	}

	if opts.PersonaPrompt != "" {
		b.WriteString("\n\n------\nPersona Instructions:\n")
		b.WriteString(opts.PersonaPrompt)
	}

	if opts.Language != "" {
		lang := resolveLanguage(opts.Language)
		b.WriteString("\n\n------\nLanguage Instructions:\n")
		b.WriteString("Write ALL review comments, summary text, and feedback in " + lang + ".\n")
		b.WriteString("Only keep code references, file paths, and JSON field names in English.\n")
		b.WriteString("The JSON structure keys (summary, comments, suggestedAction, etc.) must remain in English.")
	}

	if opts.IncludeLearning {
		b.WriteString("\n\n------\nLearning Report Instructions:\n")
		b.WriteString(`CRITICAL: The entire "learning" object and all its text content MUST ALWAYS be written in English (en-US), regardless of the language requested for the rest of the review.

Include a "learning" object in your JSON response with these fields:
- "techniques_spotted": array of strings — design patterns, techniques, and best practices you identified in the code (e.g. "Dependency Injection", "Builder Pattern", "Error wrapping")
- "what_went_well": array of strings — things the developer did well that should be reinforced
- "areas_to_improve": array of strings — skills or practices the developer should work on, framed as growth opportunities
- "key_takeaways": array of strings — the most important lessons from this code review that the developer should remember

Be specific and educational. Each item should teach something.`)
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

// ResolveReviewRules resolves review rules from preset name, custom text, or file.
// Priority: custom rules file > custom rules text > preset name > default (concise).
// Use preset "none" to disable default rules entirely.
func ResolveReviewRules(presetName, customRules, customRulesFile string) string {
	// "none" explicitly disables all rules
	if strings.ToLower(presetName) == "none" {
		return ""
	}

	if customRulesFile != "" {
		data, err := os.ReadFile(customRulesFile)
		if err != nil {
			log.Warn().Err(err).Str("file", customRulesFile).Msg("Failed to read custom rules file, falling back")
		} else {
			return strings.TrimSpace(string(data))
		}
	}

	if customRules != "" {
		return customRules
	}

	if presetName != "" {
		if rules, ok := reviewRulesPresets[strings.ToLower(presetName)]; ok {
			return rules
		}
		log.Warn().Str("name", presetName).Msg("Unknown review rules preset, using default")
	}

	// Default to concise rules
	return reviewRulesPresets["concise"]
}

// ListReviewRulesPresets returns the available preset names
func ListReviewRulesPresets() []string {
	return []string{"concise", "thorough"}
}

// resolveLanguage maps locale codes (e.g. "pt-br") to full language names,
// or returns the input as-is if it's already a language name
func resolveLanguage(code string) string {
	locales := map[string]string{
		"en":    "English",
		"en-us": "English (US)",
		"en-gb": "English (UK)",
		"pt":    "Portuguese",
		"pt-br": "Brazilian Portuguese",
		"pt-pt": "European Portuguese",
		"es":    "Spanish",
		"es-mx": "Mexican Spanish",
		"es-es": "European Spanish",
		"fr":    "French",
		"fr-ca": "Canadian French",
		"de":    "German",
		"it":    "Italian",
		"nl":    "Dutch",
		"ja":    "Japanese",
		"ko":    "Korean",
		"zh":    "Chinese",
		"zh-cn": "Simplified Chinese",
		"zh-tw": "Traditional Chinese",
		"ru":    "Russian",
		"ar":    "Arabic",
		"hi":    "Hindi",
		"tr":    "Turkish",
		"pl":    "Polish",
		"sv":    "Swedish",
		"da":    "Danish",
		"no":    "Norwegian",
		"fi":    "Finnish",
	}

	if lang, ok := locales[strings.ToLower(code)]; ok {
		return lang
	}
	return code
}
