# ReviewForge

AI-powered code reviewer with personality — GitHub Action & CLI.

ReviewForge reviews GitHub pull requests using AI (OpenAI, Anthropic, or Gemini), posting line-level comments with severity levels and review verdicts. It supports reviewer personas that add personality to reviews without changing the technical analysis.

## Features

- **Multi-provider AI**: OpenAI, Anthropic, and Gemini support
- **Line-level comments**: Precise feedback on specific code lines with severity (critical/warning/suggestion)
- **Review verdicts**: Approve, request changes, or comment with confidence scores
- **Incremental reviews**: Only review new changes since the last bot review (default: on)
- **Reviewer personas**: Built-in personalities (Bob, Robert) or custom personas
- **File filtering**: Exclude files by glob patterns
- **Context files**: Include project files (README, package.json) for better AI context
- **Dry-run mode**: Test locally without posting to GitHub
- **Docker-based GitHub Action**: No Node.js runtime needed, fast cold starts
- **Standalone CLI**: Run reviews from your terminal

## Quick Start (GitHub Action)

```yaml
name: Code Review

on:
  pull_request:
    types: [opened, synchronize]

permissions:
  contents: read
  pull-requests: write

jobs:
  review:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: AxeForging/reviewforge@v1
        with:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          AI_PROVIDER: openai
          AI_MODEL: gpt-4
          AI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
```

## Quick Start (CLI)

```bash
# Install
go install github.com/AxeForging/reviewforge@latest

# Review a PR (dry-run)
reviewforge review \
  --provider openai \
  --model gpt-4 \
  --api-key $OPENAI_API_KEY \
  --github-token $GITHUB_TOKEN \
  --repo owner/repo \
  --pr 42 \
  --dry-run

# With a persona
reviewforge review \
  --provider anthropic \
  --model claude-sonnet-4-20250514 \
  --api-key $ANTHROPIC_API_KEY \
  --github-token $GITHUB_TOKEN \
  --repo owner/repo \
  --pr 42 \
  --persona bob
```

## Personas

ReviewForge includes built-in reviewer personas that modify the review style:

| Persona | Name | Style |
|---------|------|-------|
| `bob` | Bob Lil Swagger | Friendly, encouraging. Celebrates good code, suggests improvements warmly, teaches while reviewing. |
| `robert` | Robert Dover Clow | Nerdy tech expert. Names every pattern spotted, references CS concepts and SOLID principles. |
| _(empty)_ | Default | Standard expert code reviewer. Professional, thorough, no personality overlay. |

### Custom Personas

```bash
# Inline JSON
reviewforge review --custom-persona '{"name":"strict","prompt":"Be extremely strict..."}' ...

# From file
reviewforge review --custom-persona-file ./my-persona.json ...
```

### GitHub Action with Persona

```yaml
- uses: AxeForging/reviewforge@v1
  with:
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    AI_PROVIDER: anthropic
    AI_MODEL: claude-sonnet-4-20250514
    AI_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
    PERSONA: bob
```

## Configuration

### GitHub Action Inputs

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `GITHUB_TOKEN` | Yes | - | GitHub token for API access |
| `AI_PROVIDER` | Yes | `openai` | AI provider: `openai`, `anthropic`, `gemini` |
| `AI_MODEL` | Yes | `gpt-4` | Model name |
| `AI_API_KEY` | Yes | - | API key for the provider |
| `AI_TEMPERATURE` | No | `0` | Temperature (0-1) |
| `APPROVE_REVIEWS` | No | `true` | Allow approve/request changes verdicts |
| `MAX_COMMENTS` | No | `25` | Max line comments (0 = unlimited) |
| `INCREMENTAL` | No | `true` | Only review new changes |
| `EXCLUDE_PATTERNS` | No | `**/*.lock,**/*.json,**/*.md` | Glob patterns to exclude |
| `CONTEXT_FILES` | No | `package.json,README.md` | Files for AI context |
| `PROJECT_CONTEXT` | No | - | Additional project context string |
| `PERSONA` | No | - | Built-in persona name |
| `CUSTOM_PERSONA` | No | - | Custom persona JSON |
| `CUSTOM_PERSONA_FILE` | No | - | Path to persona JSON file |

### CLI Flags

All inputs are available as CLI flags with `--kebab-case` naming. Run `reviewforge review --help` for the full list.

## Commands

```bash
reviewforge review [flags]    # Review a PR
reviewforge personas          # List available personas
reviewforge version           # Show version info
reviewforge --help            # Show help
```

## Development

```bash
# Build
make build-local

# Test
make test

# Cross-platform build
make build

# Install locally
make install
```

## License

MIT
