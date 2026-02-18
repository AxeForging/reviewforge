package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/AxeForging/reviewforge/actions"
	"github.com/AxeForging/reviewforge/helpers"
	"github.com/AxeForging/reviewforge/services"

	"github.com/urfave/cli"
)

// Version information - set during build via ldflags
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	personaService := services.NewPersonaService()
	diffService := &services.DiffService{}
	promptService := &services.PromptService{}

	reviewAction := actions.NewReviewAction(personaService, diffService, promptService)

	app := cli.NewApp()
	app.Name = "reviewforge"
	app.Usage = "AI-powered code reviewer with personality — GitHub Action & CLI"
	app.Version = Version

	// Setup logger
	helpers.SetupLogger("info")

	app.Commands = []cli.Command{
		{
			Name:    "review",
			Aliases: []string{"r"},
			Usage:   "Review a GitHub pull request",
			Flags:   reviewFlags,
			Action:  reviewAction.Execute,
		},
		{
			Name:    "personas",
			Aliases: []string{"p"},
			Usage:   "List available reviewer personas",
			Action: func(c *cli.Context) error {
				personas := personaService.ListPersonas()
				fmt.Println("Available reviewer personas:")
				fmt.Println()
				for _, p := range personas {
					fmt.Printf("  %-10s %s\n", p.Name, p.DisplayName)
					fmt.Printf("  %s\n\n", p.Description)
				}
				fmt.Println("Use --persona <name> with the review command.")
				fmt.Println("Use --custom-persona or --custom-persona-file for custom personas.")
				return nil
			},
		},
		{
			Name:  "version",
			Usage: "Show version information",
			Action: func(c *cli.Context) error {
				fmt.Printf("reviewforge version %s\n", Version)
				fmt.Printf("Build time: %s\n", BuildTime)
				fmt.Printf("Git commit: %s\n", GitCommit)
				return nil
			},
		},
	}

	// Auto-route: if GITHUB_EVENT_PATH is set and no subcommand given, run review
	app.Action = func(c *cli.Context) error {
		eventPath := os.Getenv("GITHUB_EVENT_PATH")
		if eventPath == "" {
			return cli.ShowAppHelp(c)
		}

		// Read PR number from event payload
		data, err := os.ReadFile(eventPath)
		if err != nil {
			return helpers.WrapError(err, "auto-route", "failed to read GITHUB_EVENT_PATH")
		}

		var event struct {
			PullRequest struct {
				Number int `json:"number"`
			} `json:"pull_request"`
		}
		if err := json.Unmarshal(data, &event); err != nil {
			return helpers.WrapError(err, "auto-route", "failed to parse event payload")
		}

		if event.PullRequest.Number == 0 {
			return fmt.Errorf("auto-route: no pull_request.number in event payload")
		}

		// Set PR_NUMBER env so flags pick it up, then run review
		os.Setenv("PR_NUMBER", fmt.Sprintf("%d", event.PullRequest.Number))
		return reviewAction.Execute(c)
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
