package services

import (
	"encoding/json"
	"os"

	"github.com/AxeForging/reviewforge/domain"
	"github.com/AxeForging/reviewforge/helpers"
	"github.com/rs/zerolog/log"
)

// PersonaService manages built-in and custom reviewer personas
type PersonaService struct {
	builtins map[string]domain.Persona
}

// NewPersonaService creates a PersonaService with built-in personas
func NewPersonaService() *PersonaService {
	return &PersonaService{
		builtins: map[string]domain.Persona{
			"bob": {
				Name:        "bob",
				DisplayName: "Bob Lil Swagger",
				Description: "Friendly, encouraging reviewer. Celebrates good code, suggests improvements warmly, teaches while reviewing, uses humor.",
				Prompt: `You are "Bob Lil Swagger", a friendly and encouraging code reviewer.
Your reviewing style:
- Celebrate good code patterns you spot — point out what's done well before diving into issues
- When suggesting improvements, frame them warmly: "Nice approach! One thing that could make it even better..."
- Teach while reviewing: briefly explain WHY something matters, not just what to change
- Use light humor to keep the review approachable (but never at the author's expense)
- Encourage the developer — reviewing code is mentoring
- Still be thorough and honest about real problems, but deliver feedback constructively
- Sign off your summary with a short motivational note`,
			},
			"robert": {
				Name:        "robert",
				DisplayName: "Robert Dover Clow",
				Description: "Nerdy tech expert. Names every pattern/technique spotted, references CS concepts and SOLID principles, suggests where patterns could be applied.",
				Prompt: `You are "Robert Dover Clow", a deeply knowledgeable and nerdy tech expert code reviewer.
Your reviewing style:
- Name every design pattern, technique, and principle you spot (e.g. "This is a classic Strategy pattern", "Good use of the Open/Closed Principle here")
- Explain WHY patterns are appropriate for the context, not just that they exist
- Reference CS concepts freely: SOLID, DRY, YAGNI, Law of Demeter, separation of concerns, etc.
- When you find issues, suggest which pattern or principle could improve the code
- Point out where patterns could be applied elsewhere in the codebase
- Compare approaches: "This uses X pattern, but Y pattern might give you better extensibility here because..."
- Be precise with terminology — use correct names for things
- Get excited about elegant solutions and clever implementations
- Still be practical: don't over-engineer, recommend patterns only when they genuinely help`,
			},
		},
	}
}

// ListPersonas returns all built-in personas
func (s *PersonaService) ListPersonas() []domain.Persona {
	result := make([]domain.Persona, 0, len(s.builtins))
	// Return in deterministic order
	for _, name := range []string{"bob", "robert"} {
		if p, ok := s.builtins[name]; ok {
			result = append(result, p)
		}
	}
	return result
}

// GetPersona resolves a persona from config. Priority: custom JSON > custom file > built-in name > nil
func (s *PersonaService) GetPersona(config domain.ReviewConfig) *domain.Persona {
	// Custom persona JSON takes highest priority
	if config.CustomPersona != "" {
		var p domain.Persona
		if err := json.Unmarshal([]byte(config.CustomPersona), &p); err != nil {
			log.Warn().Err(err).Msg("Failed to parse custom persona JSON, using default reviewer")
			return nil
		}
		log.Info().Str("persona", p.Name).Msg("Using custom persona")
		return &p
	}

	// Custom persona file
	if config.CustomPersonaFile != "" {
		data, err := os.ReadFile(config.CustomPersonaFile)
		if err != nil {
			log.Warn().Err(err).Str("file", config.CustomPersonaFile).Msg("Failed to read custom persona file, using default reviewer")
			return nil
		}
		var p domain.Persona
		if err := json.Unmarshal(data, &p); err != nil {
			log.Warn().Err(err).Str("file", config.CustomPersonaFile).Msg("Failed to parse custom persona file, using default reviewer")
			return nil
		}
		log.Info().Str("persona", p.Name).Msg("Using custom persona from file")
		return &p
	}

	// Built-in persona by name
	if config.PersonaName != "" {
		if p, ok := s.builtins[config.PersonaName]; ok {
			log.Info().Str("persona", p.Name).Msg("Using built-in persona")
			return &p
		}
		log.Warn().Str("name", config.PersonaName).Msg("Unknown persona name, using default reviewer")
		return nil
	}

	return nil
}

// GetPersonaPrompt returns the persona prompt string, or empty if no persona
func (s *PersonaService) GetPersonaPrompt(config domain.ReviewConfig) string {
	p := s.GetPersona(config)
	if p == nil {
		return ""
	}
	return p.Prompt
}

// ValidatePersonaName checks if a persona name is valid (built-in or empty)
func (s *PersonaService) ValidatePersonaName(name string) error {
	if name == "" {
		return nil
	}
	if _, ok := s.builtins[name]; ok {
		return nil
	}
	return helpers.NewFormatError("persona", "unknown persona: "+name, nil)
}
