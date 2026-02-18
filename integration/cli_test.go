package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var binaryPath string

func TestMain(m *testing.M) {
	// Build the binary once for all integration tests
	dir, err := os.MkdirTemp("", "reviewforge-test")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	binaryPath = filepath.Join(dir, "reviewforge")
	cmd := exec.Command("go", "build", "-o", binaryPath, "..")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic("failed to build binary: " + err.Error())
	}

	os.Exit(m.Run())
}

func runBinary(args ...string) (string, error) {
	cmd := exec.Command(binaryPath, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestCLI_Help(t *testing.T) {
	out, err := runBinary("--help")
	if err != nil {
		t.Fatalf("help failed: %v\n%s", err, out)
	}

	expected := []string{
		"reviewforge",
		"review",
		"personas",
		"version",
		"AI-powered code reviewer",
	}
	for _, s := range expected {
		if !strings.Contains(out, s) {
			t.Errorf("help output missing %q", s)
		}
	}
}

func TestCLI_Version(t *testing.T) {
	out, err := runBinary("version")
	if err != nil {
		t.Fatalf("version failed: %v\n%s", err, out)
	}

	if !strings.Contains(out, "reviewforge version") {
		t.Errorf("version output missing prefix: %s", out)
	}
	if !strings.Contains(out, "Build time:") {
		t.Errorf("version output missing build time: %s", out)
	}
}

func TestCLI_Personas(t *testing.T) {
	out, err := runBinary("personas")
	if err != nil {
		t.Fatalf("personas failed: %v\n%s", err, out)
	}

	expected := []string{
		"bob", "Bob Lil Swagger",
		"robert", "Robert Dover Clow",
		"--persona",
	}
	for _, s := range expected {
		if !strings.Contains(out, s) {
			t.Errorf("personas output missing %q", s)
		}
	}
}

func TestCLI_ReviewMissingAPIKey(t *testing.T) {
	_, err := runBinary("review", "--repo", "owner/repo", "--pr", "1")
	if err == nil {
		t.Fatal("expected error for missing API key")
	}
}

func TestCLI_ReviewMissingPR(t *testing.T) {
	_, err := runBinary("review", "--api-key", "test", "--github-token", "test", "--repo", "owner/repo")
	if err == nil {
		t.Fatal("expected error for missing PR number")
	}
}

func TestCLI_ReviewHelp(t *testing.T) {
	out, err := runBinary("review", "--help")
	if err != nil {
		t.Fatalf("review help failed: %v\n%s", err, out)
	}

	expected := []string{
		"--provider",
		"--model",
		"--api-key",
		"--github-token",
		"--repo",
		"--pr",
		"--persona",
		"--dry-run",
		"--incremental",
		"--max-comments",
	}
	for _, s := range expected {
		if !strings.Contains(out, s) {
			t.Errorf("review help missing %q", s)
		}
	}
}
