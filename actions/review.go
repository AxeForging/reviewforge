package actions

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/AxeForging/reviewforge/domain"
	"github.com/AxeForging/reviewforge/helpers"
	"github.com/AxeForging/reviewforge/services"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli"
)

// ReviewAction orchestrates the full code review pipeline
type ReviewAction struct {
	PersonaService *services.PersonaService
	DiffService    *services.DiffService
	PromptService  *services.PromptService
}

// NewReviewAction creates a ReviewAction with the required services
func NewReviewAction(ps *services.PersonaService, ds *services.DiffService, promptSvc *services.PromptService) *ReviewAction {
	return &ReviewAction{
		PersonaService: ps,
		DiffService:    ds,
		PromptService:  promptSvc,
	}
}

// Execute runs the full review pipeline
func (a *ReviewAction) Execute(c *cli.Context) error {
	config := a.resolveConfig(c)

	if config.Verbose {
		helpers.SetupLogger("debug")
	}

	// Validate required fields
	if err := a.validate(config); err != nil {
		return err
	}

	log.Info().
		Str("provider", config.AIProvider).
		Str("model", config.AIModel).
		Str("repo", config.Repo).
		Int("pr", config.PRNumber).
		Bool("incremental", config.Incremental).
		Bool("dry_run", config.DryRun).
		Msg("Starting review")

	// Create GitHub service (needs runtime token)
	ghService, err := services.NewGitHubService(config.GitHubToken, config.Repo)
	if err != nil {
		return err
	}

	// Get PR details
	pr, err := ghService.GetPRDetails(config.PRNumber)
	if err != nil {
		return helpers.WrapError(err, "review", "failed to get PR details")
	}
	log.Info().Str("title", pr.Title).Msg("PR details fetched")

	// Detect incremental review
	var lastReviewedCommit string
	isUpdate := false
	if config.Incremental {
		lastReviewedCommit, err = ghService.GetLastReviewedCommit(config.PRNumber)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to check for previous reviews, doing full review")
		} else if lastReviewedCommit != "" {
			isUpdate = true
			log.Info().Str("last_commit", lastReviewedCommit).Msg("Incremental review: only reviewing new changes")
		}
	}

	// Fetch diff
	diffText, err := ghService.GetPRDiff(config.PRNumber, lastReviewedCommit)
	if err != nil {
		return helpers.WrapError(err, "review", "failed to fetch PR diff")
	}

	// Parse and filter diff
	files := a.DiffService.ParseUnifiedDiff(diffText)
	excludePatterns := a.DiffService.ParseExcludePatterns(strings.Join(config.ExcludePatterns, ","))
	files = a.DiffService.FilterFiles(files, excludePatterns)
	log.Info().Int("files", len(files)).Msg("Files to review after filtering")

	if len(files) == 0 {
		log.Info().Msg("No files to review after filtering")
		return nil
	}

	// Fetch full content for each file
	for i := range files {
		content, _ := ghService.GetFileContent(files[i].Path, pr.HeadSHA)
		files[i].Content = content
		original, _ := ghService.GetFileContent(files[i].Path, pr.BaseSHA)
		files[i].OriginalContent = original
	}

	// Fetch context files
	var contextFiles []domain.ContextFile
	for _, cf := range config.ContextFiles {
		cf = strings.TrimSpace(cf)
		if cf == "" {
			continue
		}
		content, _ := ghService.GetFileContent(cf, pr.HeadSHA)
		if content != "" {
			contextFiles = append(contextFiles, domain.ContextFile{Path: cf, Content: content})
		}
	}

	// Build AI request
	reviewReq := domain.ReviewRequest{
		Files:        files,
		ContextFiles: contextFiles,
		PullRequest: domain.PRSummary{
			Title:       pr.Title,
			Description: pr.Description,
			Base:        pr.BaseSHA,
			Head:        pr.HeadSHA,
		},
		Context: domain.ReviewContext{
			Repository:     config.Repo,
			Owner:          strings.Split(config.Repo, "/")[0],
			ProjectContext: config.ProjectContext,
			IsUpdate:       isUpdate,
		},
	}

	// Build prompts
	personaPrompt := a.PersonaService.GetPersonaPrompt(config)
	reviewRules := services.ResolveReviewRules(config.ReviewRules, config.CustomRules, config.CustomRulesFile)
	systemPrompt := a.PromptService.BuildSystemPrompt(services.PromptOptions{
		IsUpdate:        isUpdate,
		PersonaPrompt:   personaPrompt,
		Language:        config.Language,
		IncludeLearning: config.SaveReport != "",
		StrictChanges:   config.StrictChanges,
		ReviewRules:     reviewRules,
	})
	userPrompt := a.PromptService.BuildUserPrompt(reviewReq)

	// Create AI provider (needs runtime API key)
	aiProvider, err := services.NewAIProvider(config.AIProvider, config.AIAPIKey, config.AIModel)
	if err != nil {
		return err
	}

	// Call AI
	log.Info().Str("provider", aiProvider.Name()).Msg("Sending review request to AI")
	rawResponse, err := aiProvider.Review(systemPrompt, userPrompt, config.Temperature)
	if err != nil {
		return helpers.WrapError(err, "review", "AI review failed")
	}

	// Parse AI response
	output, err := a.PromptService.ParseAIResponse(rawResponse)
	if err != nil {
		return helpers.WrapError(err, "review", "failed to parse AI response")
	}

	// Apply max comments limit
	if config.MaxComments > 0 && len(output.Comments) > config.MaxComments {
		output.Comments = output.Comments[:config.MaxComments]
	}

	// Add model footer
	footer := a.PromptService.FormatModelFooter(config.AIProvider, config.AIModel)
	output.Summary = output.Summary + "\n\n------\n\n" + footer

	log.Info().
		Int("comments", len(output.Comments)).
		Str("action", output.SuggestedAction).
		Int("confidence", output.Confidence).
		Msg("Review complete")

	// Save report if requested
	if config.SaveReport != "" {
		fileNames := make([]string, len(files))
		for i, f := range files {
			fileNames[i] = f.Path
		}

		report := domain.ReviewReport{
			Repo:          config.Repo,
			PRNumber:      config.PRNumber,
			PRTitle:       pr.Title,
			Provider:      config.AIProvider,
			Model:         config.AIModel,
			Persona:       config.PersonaName,
			Language:      config.Language,
			Review:        *output,
			FilesReviewed: fileNames,
		}

		reportData, _ := json.MarshalIndent(report, "", "  ")
		if err := os.WriteFile(config.SaveReport, reportData, 0644); err != nil {
			log.Warn().Err(err).Str("path", config.SaveReport).Msg("Failed to save report")
		} else {
			log.Info().Str("path", config.SaveReport).Msg("Report saved")
		}
	}

	// Dry run: print JSON and exit
	if config.DryRun {
		data, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	// Determine review event
	event := a.resolveEvent(output.SuggestedAction, config.ApproveReviews)

	// Submit review to GitHub
	if err := ghService.SubmitReview(config.PRNumber, output, event); err != nil {
		return helpers.WrapError(err, "review", "failed to submit review")
	}

	return nil
}

func (a *ReviewAction) resolveConfig(c *cli.Context) domain.ReviewConfig {
	contextFiles := strings.Split(c.String("context-files"), ",")
	excludePatterns := strings.Split(c.String("exclude-patterns"), ",")

	return domain.ReviewConfig{
		AIProvider:        c.String("provider"),
		AIModel:           c.String("model"),
		AIAPIKey:          c.String("api-key"),
		Temperature:       c.Float64("temperature"),
		GitHubToken:       c.String("github-token"),
		Repo:              c.String("repo"),
		PRNumber:          c.Int("pr"),
		ApproveReviews:    c.BoolT("approve-reviews"),
		MaxComments:       c.Int("max-comments"),
		Incremental:       c.BoolT("incremental"),
		ExcludePatterns:   excludePatterns,
		ContextFiles:      contextFiles,
		ProjectContext:    c.String("project-context"),
		PersonaName:       c.String("persona"),
		CustomPersona:     c.String("custom-persona"),
		CustomPersonaFile: c.String("custom-persona-file"),
		Language:          c.String("language"),
		StrictChanges:     c.Bool("strict-changes"),
		ReviewRules:       c.String("review-rules"),
		CustomRules:       c.String("custom-rules"),
		CustomRulesFile:   c.String("custom-rules-file"),
		SaveReport:        c.String("save-report"),
		DryRun:            c.Bool("dry-run"),
		Verbose:           c.Bool("verbose"),
	}
}

func (a *ReviewAction) validate(config domain.ReviewConfig) error {
	if config.AIAPIKey == "" {
		return helpers.ErrNoAPIKey
	}
	if config.GitHubToken == "" && !config.DryRun {
		return helpers.ErrNoGitHubToken
	}
	if config.PRNumber == 0 {
		return helpers.ErrNoPRNumber
	}
	if config.Repo == "" {
		return helpers.ErrNoRepo
	}
	return nil
}

func (a *ReviewAction) resolveEvent(suggestedAction string, approveReviews bool) string {
	if !approveReviews {
		return "COMMENT"
	}

	switch strings.ToLower(suggestedAction) {
	case "approve":
		return "APPROVE"
	case "request_changes":
		return "REQUEST_CHANGES"
	default:
		return "COMMENT"
	}
}
