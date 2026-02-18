package main

import "github.com/urfave/cli"

// AI provider flags
var providerFlag = cli.StringFlag{
	Name:   "provider, p",
	Value:  "openai",
	Usage:  "AI provider: openai, anthropic, gemini",
	EnvVar: "INPUT_AI_PROVIDER,AI_PROVIDER",
}

var modelFlag = cli.StringFlag{
	Name:   "model, m",
	Value:  "gpt-4",
	Usage:  "AI model to use",
	EnvVar: "INPUT_AI_MODEL,AI_MODEL",
}

var apiKeyFlag = cli.StringFlag{
	Name:   "api-key",
	Usage:  "API key for the AI provider",
	EnvVar: "INPUT_AI_API_KEY,AI_API_KEY",
}

var temperatureFlag = cli.Float64Flag{
	Name:   "temperature, t",
	Value:  0,
	Usage:  "Temperature for AI model (0-1)",
	EnvVar: "INPUT_AI_TEMPERATURE,AI_TEMPERATURE",
}

// GitHub flags
var githubTokenFlag = cli.StringFlag{
	Name:   "github-token",
	Usage:  "GitHub token for API access",
	EnvVar: "INPUT_GITHUB_TOKEN,GITHUB_TOKEN",
}

var repoFlag = cli.StringFlag{
	Name:   "repo, r",
	Usage:  "Repository in owner/repo format",
	EnvVar: "GITHUB_REPOSITORY",
}

var prFlag = cli.IntFlag{
	Name:   "pr",
	Usage:  "Pull request number",
	EnvVar: "PR_NUMBER",
}

// Review behavior flags
var approveFlag = cli.BoolTFlag{
	Name:   "approve-reviews",
	Usage:  "Allow approving/requesting changes on PRs (default: true)",
	EnvVar: "INPUT_APPROVE_REVIEWS,APPROVE_REVIEWS",
}

var maxCommentsFlag = cli.IntFlag{
	Name:   "max-comments",
	Value:  25,
	Usage:  "Maximum number of line comments (0 = unlimited)",
	EnvVar: "INPUT_MAX_COMMENTS,MAX_COMMENTS",
}

var incrementalFlag = cli.BoolTFlag{
	Name:   "incremental",
	Usage:  "Only review new changes since last bot review (default: true)",
	EnvVar: "INPUT_INCREMENTAL,INCREMENTAL",
}

var excludePatternsFlag = cli.StringFlag{
	Name:   "exclude-patterns",
	Value:  "**/*.lock,**/*.json,**/*.md",
	Usage:  "Comma-separated glob patterns to exclude",
	EnvVar: "INPUT_EXCLUDE_PATTERNS,EXCLUDE_PATTERNS",
}

var contextFilesFlag = cli.StringFlag{
	Name:   "context-files",
	Value:  "package.json,README.md",
	Usage:  "Comma-separated files to include as AI context",
	EnvVar: "INPUT_CONTEXT_FILES,CONTEXT_FILES",
}

var projectContextFlag = cli.StringFlag{
	Name:   "project-context",
	Usage:  "Additional context about the project",
	EnvVar: "INPUT_PROJECT_CONTEXT,PROJECT_CONTEXT",
}

// Persona flags
var personaFlag = cli.StringFlag{
	Name:   "persona",
	Usage:  "Reviewer persona: bob, robert, maya, eli, or leave empty for default",
	EnvVar: "INPUT_PERSONA,PERSONA",
}

var customPersonaFlag = cli.StringFlag{
	Name:   "custom-persona",
	Usage:  `Custom persona JSON: '{"name":"...","prompt":"..."}'`,
	EnvVar: "INPUT_CUSTOM_PERSONA,CUSTOM_PERSONA",
}

var customPersonaFileFlag = cli.StringFlag{
	Name:   "custom-persona-file",
	Usage:  "Path to custom persona JSON file",
	EnvVar: "INPUT_CUSTOM_PERSONA_FILE,CUSTOM_PERSONA_FILE",
}

// Language flag
var languageFlag = cli.StringFlag{
	Name:   "language, l",
	Usage:  "Language for review comments and summary (e.g. Portuguese, Spanish, French)",
	EnvVar: "INPUT_LANGUAGE,LANGUAGE",
}

// Report flag
var saveReportFlag = cli.StringFlag{
	Name:   "save-report",
	Usage:  "Save review report as JSON to this file path (includes learning insights)",
	EnvVar: "SAVE_REPORT",
}

// Output flags
var dryRunFlag = cli.BoolFlag{
	Name:   "dry-run",
	Usage:  "Print review JSON instead of posting to GitHub",
	EnvVar: "DRY_RUN",
}

var verboseFlag = cli.BoolFlag{
	Name:   "verbose, v",
	Usage:  "Enable verbose logging",
	EnvVar: "VERBOSE",
}

// reviewFlags is the full set of flags for the review command
var reviewFlags = []cli.Flag{
	providerFlag,
	modelFlag,
	apiKeyFlag,
	temperatureFlag,
	githubTokenFlag,
	repoFlag,
	prFlag,
	approveFlag,
	maxCommentsFlag,
	incrementalFlag,
	excludePatternsFlag,
	contextFilesFlag,
	projectContextFlag,
	personaFlag,
	customPersonaFlag,
	customPersonaFileFlag,
	languageFlag,
	saveReportFlag,
	dryRunFlag,
	verboseFlag,
}
