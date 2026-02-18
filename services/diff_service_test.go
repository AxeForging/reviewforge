package services

import (
	"strings"
	"testing"

	"github.com/AxeForging/reviewforge/domain"
)

func TestDiffService_ParseUnifiedDiff(t *testing.T) {
	svc := &DiffService{}

	diff := `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
 package main

+import "fmt"
 func main() {
diff --git a/helpers/util.go b/helpers/util.go
--- a/helpers/util.go
+++ b/helpers/util.go
@@ -5,3 +5,5 @@
 func helper() {
+	doSomething()
+	doMore()
 }`

	files := svc.ParseUnifiedDiff(diff)
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}

	if files[0].Path != "main.go" {
		t.Errorf("file[0].Path = %q, want %q", files[0].Path, "main.go")
	}
	if files[1].Path != "helpers/util.go" {
		t.Errorf("file[1].Path = %q, want %q", files[1].Path, "helpers/util.go")
	}

	// Verify diff content contains the hunks
	if !strings.Contains(files[0].Diff, `+import "fmt"`) {
		t.Error("file[0].Diff should contain the added import line")
	}
	if !strings.Contains(files[1].Diff, "+\tdoSomething()") {
		t.Error("file[1].Diff should contain the added doSomething line")
	}
}

func TestDiffService_ParseUnifiedDiff_Empty(t *testing.T) {
	svc := &DiffService{}
	files := svc.ParseUnifiedDiff("")
	if len(files) != 0 {
		t.Fatalf("expected 0 files, got %d", len(files))
	}
}

func TestDiffService_FilterFiles(t *testing.T) {
	svc := &DiffService{}

	files := []domain.FileDiff{
		{Path: "src/main.go"},
		{Path: "package-lock.json"},
		{Path: "README.md"},
		{Path: "src/service.go"},
		{Path: "go.sum"},
	}

	tests := []struct {
		name     string
		patterns []string
		expected int
	}{
		{"no patterns", nil, 5},
		{"exclude json", []string{"*.json"}, 4},
		{"exclude md and json", []string{"*.json", "*.md"}, 3},
		{"doublestar json", []string{"**/*.json"}, 4},
		{"exclude sum", []string{"*.sum"}, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.FilterFiles(files, tt.patterns)
			if len(result) != tt.expected {
				names := make([]string, len(result))
				for i, f := range result {
					names[i] = f.Path
				}
				t.Errorf("FilterFiles with %v: got %d files %v, want %d", tt.patterns, len(result), names, tt.expected)
			}
		})
	}
}

func TestDiffService_ParseExcludePatterns(t *testing.T) {
	svc := &DiffService{}

	tests := []struct {
		input    string
		expected int
	}{
		{"", 0},
		{"*.json", 1},
		{"*.json,*.md,*.lock", 3},
		{"*.json, *.md , *.lock", 3},
		{",,,", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := svc.ParseExcludePatterns(tt.input)
			if len(result) != tt.expected {
				t.Errorf("ParseExcludePatterns(%q) = %v (len %d), want len %d", tt.input, result, len(result), tt.expected)
			}
		})
	}
}

func TestDiffService_ExtractAddedLineNumbers(t *testing.T) {
	svc := &DiffService{}

	diff := `@@ -1,3 +1,5 @@
 package main

+import "fmt"
+import "os"
 func main() {`

	lines := svc.ExtractAddedLineNumbers(diff)
	if len(lines) != 2 {
		t.Fatalf("expected 2 added lines, got %d: %v", len(lines), lines)
	}
	if lines[0] != 3 {
		t.Errorf("lines[0] = %d, want 3", lines[0])
	}
	if lines[1] != 4 {
		t.Errorf("lines[1] = %d, want 4", lines[1])
	}
}

func TestDiffService_ValidateCommentLine(t *testing.T) {
	svc := &DiffService{}

	diff := `@@ -1,3 +1,5 @@
 package main

+import "fmt"
+import "os"
 func main() {`

	// Added lines (3,4) should be valid
	if !svc.ValidateCommentLine(diff, 3) {
		t.Error("line 3 (added) should be valid")
	}
	if !svc.ValidateCommentLine(diff, 4) {
		t.Error("line 4 (added) should be valid")
	}
	// Context lines (1,2,5) should also be valid
	if !svc.ValidateCommentLine(diff, 1) {
		t.Error("line 1 (context) should be valid")
	}
	// Line 99 should not be valid
	if svc.ValidateCommentLine(diff, 99) {
		t.Error("line 99 should not be valid")
	}
}

func TestParseHunkNewStart(t *testing.T) {
	tests := []struct {
		header   string
		expected int
	}{
		{"@@ -1,3 +1,5 @@", 1},
		{"@@ -10,8 +15,12 @@ func foo()", 15},
		{"@@ -0,0 +1,30 @@", 1},
		{"invalid", 0},
	}

	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			got := parseHunkNewStart(tt.header)
			if got != tt.expected {
				t.Errorf("parseHunkNewStart(%q) = %d, want %d", tt.header, got, tt.expected)
			}
		})
	}
}
