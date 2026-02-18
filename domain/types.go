package domain

// ReviewConfig holds all configuration for a review run
type ReviewConfig struct {
	// AI settings
	AIProvider  string  `json:"ai_provider"`
	AIModel     string  `json:"ai_model"`
	AIAPIKey    string  `json:"-"`
	Temperature float64 `json:"temperature"`

	// GitHub settings
	GitHubToken string `json:"-"`
	Repo        string `json:"repo"`
	PRNumber    int    `json:"pr_number"`

	// Review settings
	ApproveReviews  bool     `json:"approve_reviews"`
	MaxComments     int      `json:"max_comments"`
	Incremental     bool     `json:"incremental"`
	ExcludePatterns []string `json:"exclude_patterns"`
	ContextFiles    []string `json:"context_files"`
	ProjectContext  string   `json:"project_context"`

	// Persona
	PersonaName       string `json:"persona_name"`
	CustomPersona     string `json:"custom_persona"`
	CustomPersonaFile string `json:"custom_persona_file"`

	// Language
	Language string `json:"language"`

	// Verdict behavior
	StrictChanges bool `json:"strict_changes"`

	// Report
	SaveReport string `json:"save_report"`

	// Output
	DryRun  bool `json:"dry_run"`
	Verbose bool `json:"verbose"`
}

// Persona represents a reviewer personality
type Persona struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	Prompt      string `json:"prompt"`
}

// PRDetails holds pull request metadata
type PRDetails struct {
	Owner       string `json:"owner"`
	Repo        string `json:"repo"`
	Number      int    `json:"number"`
	Title       string `json:"title"`
	Description string `json:"description"`
	BaseSHA     string `json:"base_sha"`
	HeadSHA     string `json:"head_sha"`
}

// FileDiff represents a single file's diff and content
type FileDiff struct {
	Path            string `json:"path"`
	Diff            string `json:"diff"`
	Content         string `json:"content,omitempty"`
	OriginalContent string `json:"original_content,omitempty"`
}

// LineMapping maps diff line numbers to file line numbers
type LineMapping struct {
	DiffLine int `json:"diff_line"`
	FileLine int `json:"file_line"`
	Type     string `json:"type"` // "add", "del", "normal"
}

// ReviewComment represents a single line-level comment from the AI
type ReviewComment struct {
	Path     string `json:"path"`
	Line     int    `json:"line"`
	Comment  string `json:"comment"`
	Severity string `json:"severity,omitempty"` // "critical", "warning", "suggestion"
}

// AIReviewOutput is the parsed JSON response from the AI provider
type AIReviewOutput struct {
	Summary         string          `json:"summary"`
	Comments        []ReviewComment `json:"comments"`
	SuggestedAction string          `json:"suggestedAction"`
	Confidence      int             `json:"confidence"`
	Learning        *LearningReport `json:"learning,omitempty"`
}

// LearningReport contains insights for developer growth, included when --save-report is used
type LearningReport struct {
	TechniquesSpotted []string `json:"techniques_spotted"`
	WhatWentWell      []string `json:"what_went_well"`
	AreasToImprove    []string `json:"areas_to_improve"`
	KeyTakeaways      []string `json:"key_takeaways"`
}

// ReviewReport is the full saved report including metadata
type ReviewReport struct {
	Repo       string          `json:"repo"`
	PRNumber   int             `json:"pr_number"`
	PRTitle    string          `json:"pr_title"`
	Provider   string          `json:"provider"`
	Model      string          `json:"model"`
	Persona    string          `json:"persona,omitempty"`
	Language   string          `json:"language,omitempty"`
	Review     AIReviewOutput  `json:"review"`
	FilesReviewed []string     `json:"files_reviewed"`
}

// ReviewRequest is what gets sent to the AI provider
type ReviewRequest struct {
	Files           []FileDiff      `json:"files"`
	ContextFiles    []ContextFile   `json:"context_files,omitempty"`
	PreviousReviews []PreviousReview `json:"previous_reviews,omitempty"`
	PullRequest     PRSummary       `json:"pr"`
	Context         ReviewContext   `json:"context"`
}

// ContextFile is a file included for AI context (e.g. README, package.json)
type ContextFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// PreviousReview holds data from a prior bot review for incremental reviews
type PreviousReview struct {
	CommitSHA    string          `json:"commit"`
	Summary      string          `json:"summary"`
	LineComments []ReviewComment `json:"line_comments"`
}

// PRSummary is the PR info included in AI requests
type PRSummary struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Base        string `json:"base"`
	Head        string `json:"head"`
}

// ReviewContext provides repository context for the AI
type ReviewContext struct {
	Repository     string `json:"repository"`
	Owner          string `json:"owner"`
	ProjectContext string `json:"project_context,omitempty"`
	IsUpdate       bool   `json:"is_update"`
}
