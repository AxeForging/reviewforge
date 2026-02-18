package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AxeForging/reviewforge/domain"
)

func TestPersonaService_ListPersonas(t *testing.T) {
	svc := NewPersonaService()
	personas := svc.ListPersonas()

	if len(personas) != 4 {
		t.Fatalf("expected 4 built-in personas, got %d", len(personas))
	}

	expected := []string{"bob", "robert", "maya", "eli"}
	for i, name := range expected {
		if personas[i].Name != name {
			t.Errorf("persona[%d] = %q, want %q", i, personas[i].Name, name)
		}
	}
}

func TestPersonaService_GetPersona_Builtin(t *testing.T) {
	svc := NewPersonaService()

	t.Run("bob", func(t *testing.T) {
		p := svc.GetPersona(domain.ReviewConfig{PersonaName: "bob"})
		if p == nil {
			t.Fatal("expected non-nil persona for bob")
		}
		if p.DisplayName != "Bob Lil Swagger" {
			t.Errorf("DisplayName = %q", p.DisplayName)
		}
	})

	t.Run("robert", func(t *testing.T) {
		p := svc.GetPersona(domain.ReviewConfig{PersonaName: "robert"})
		if p == nil {
			t.Fatal("expected non-nil persona for robert")
		}
		if p.DisplayName != "Robert Dover Clow" {
			t.Errorf("DisplayName = %q", p.DisplayName)
		}
	})

	t.Run("maya", func(t *testing.T) {
		p := svc.GetPersona(domain.ReviewConfig{PersonaName: "maya"})
		if p == nil {
			t.Fatal("expected non-nil persona for maya")
		}
		if p.DisplayName != "Maya Simplifica" {
			t.Errorf("DisplayName = %q", p.DisplayName)
		}
		if !strings.Contains(p.Prompt, "analogy") {
			t.Error("maya prompt should mention analogies")
		}
	})

	t.Run("eli", func(t *testing.T) {
		p := svc.GetPersona(domain.ReviewConfig{PersonaName: "eli"})
		if p == nil {
			t.Fatal("expected non-nil persona for eli")
		}
		if p.DisplayName != "Eli Passo" {
			t.Errorf("DisplayName = %q", p.DisplayName)
		}
		if !strings.Contains(p.Prompt, "newcomer") {
			t.Error("eli prompt should mention newcomers")
		}
	})

	t.Run("unknown returns nil", func(t *testing.T) {
		p := svc.GetPersona(domain.ReviewConfig{PersonaName: "unknown"})
		if p != nil {
			t.Error("expected nil for unknown persona")
		}
	})

	t.Run("empty returns nil", func(t *testing.T) {
		p := svc.GetPersona(domain.ReviewConfig{})
		if p != nil {
			t.Error("expected nil for empty persona")
		}
	})
}

func TestPersonaService_GetPersona_CustomJSON(t *testing.T) {
	svc := NewPersonaService()

	config := domain.ReviewConfig{
		CustomPersona: `{"name":"custom","display_name":"Custom Bot","description":"A custom bot","prompt":"Be custom."}`,
	}

	p := svc.GetPersona(config)
	if p == nil {
		t.Fatal("expected non-nil persona for custom JSON")
	}
	if p.Name != "custom" {
		t.Errorf("Name = %q, want custom", p.Name)
	}
	if p.Prompt != "Be custom." {
		t.Errorf("Prompt = %q", p.Prompt)
	}
}

func TestPersonaService_GetPersona_CustomJSON_Invalid(t *testing.T) {
	svc := NewPersonaService()

	config := domain.ReviewConfig{
		CustomPersona: "not json",
	}

	p := svc.GetPersona(config)
	if p != nil {
		t.Error("expected nil for invalid custom JSON")
	}
}

func TestPersonaService_GetPersona_CustomFile(t *testing.T) {
	svc := NewPersonaService()

	// Create temp file
	dir := t.TempDir()
	path := filepath.Join(dir, "persona.json")
	err := os.WriteFile(path, []byte(`{"name":"file-persona","prompt":"From file."}`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	config := domain.ReviewConfig{
		CustomPersonaFile: path,
	}

	p := svc.GetPersona(config)
	if p == nil {
		t.Fatal("expected non-nil persona from file")
	}
	if p.Name != "file-persona" {
		t.Errorf("Name = %q", p.Name)
	}
}

func TestPersonaService_GetPersona_Priority(t *testing.T) {
	svc := NewPersonaService()

	// Custom JSON takes priority over built-in name
	config := domain.ReviewConfig{
		PersonaName:   "bob",
		CustomPersona: `{"name":"override","prompt":"Override prompt."}`,
	}

	p := svc.GetPersona(config)
	if p == nil {
		t.Fatal("expected non-nil persona")
	}
	if p.Name != "override" {
		t.Errorf("Custom JSON should take priority, got Name = %q", p.Name)
	}
}

func TestPersonaService_GetPersonaPrompt(t *testing.T) {
	svc := NewPersonaService()

	t.Run("with persona", func(t *testing.T) {
		prompt := svc.GetPersonaPrompt(domain.ReviewConfig{PersonaName: "bob"})
		if prompt == "" {
			t.Error("expected non-empty prompt for bob")
		}
	})

	t.Run("without persona", func(t *testing.T) {
		prompt := svc.GetPersonaPrompt(domain.ReviewConfig{})
		if prompt != "" {
			t.Errorf("expected empty prompt, got %q", prompt)
		}
	})
}

func TestPersonaService_ValidatePersonaName(t *testing.T) {
	svc := NewPersonaService()

	if err := svc.ValidatePersonaName(""); err != nil {
		t.Errorf("empty should be valid: %v", err)
	}
	if err := svc.ValidatePersonaName("bob"); err != nil {
		t.Errorf("bob should be valid: %v", err)
	}
	if err := svc.ValidatePersonaName("robert"); err != nil {
		t.Errorf("robert should be valid: %v", err)
	}
	if err := svc.ValidatePersonaName("maya"); err != nil {
		t.Errorf("maya should be valid: %v", err)
	}
	if err := svc.ValidatePersonaName("eli"); err != nil {
		t.Errorf("eli should be valid: %v", err)
	}
	if err := svc.ValidatePersonaName("unknown"); err == nil {
		t.Error("unknown should be invalid")
	}
}
