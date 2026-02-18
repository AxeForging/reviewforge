package services

import (
	"path/filepath"
	"strconv"
	"strings"

	"github.com/AxeForging/reviewforge/domain"
	"github.com/rs/zerolog/log"
)

// DiffService parses unified diffs, filters files by glob patterns, and maps line numbers
type DiffService struct{}

// ParseUnifiedDiff parses a full unified diff string into individual FileDiffs
func (s *DiffService) ParseUnifiedDiff(diffText string) []domain.FileDiff {
	var files []domain.FileDiff
	var current *domain.FileDiff

	lines := strings.Split(diffText, "\n")
	var diffLines []string

	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git") {
			// Save previous file
			if current != nil {
				current.Diff = strings.Join(diffLines, "\n")
				files = append(files, *current)
			}
			current = &domain.FileDiff{}
			diffLines = nil
			continue
		}

		if current == nil {
			continue
		}

		// Parse file path from +++ line
		if strings.HasPrefix(line, "+++ b/") {
			current.Path = strings.TrimPrefix(line, "+++ b/")
			continue
		}

		// Skip --- line
		if strings.HasPrefix(line, "--- ") {
			continue
		}

		// Collect hunk headers and diff content
		if strings.HasPrefix(line, "@@") || strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") || strings.HasPrefix(line, " ") {
			diffLines = append(diffLines, line)
		}
	}

	// Save last file
	if current != nil && current.Path != "" {
		current.Diff = strings.Join(diffLines, "\n")
		files = append(files, *current)
	}

	return files
}

// FilterFiles removes files matching exclude patterns
func (s *DiffService) FilterFiles(files []domain.FileDiff, excludePatterns []string) []domain.FileDiff {
	if len(excludePatterns) == 0 {
		return files
	}

	var result []domain.FileDiff
	for _, f := range files {
		excluded := false
		for _, pattern := range excludePatterns {
			pattern = strings.TrimSpace(pattern)
			if pattern == "" {
				continue
			}
			matched, err := filepath.Match(pattern, f.Path)
			if err != nil {
				// Try matching just the base name for ** patterns
				matched, _ = filepath.Match(pattern, filepath.Base(f.Path))
			}
			// Also try doublestar-style matching: strip leading **/
			if !matched && strings.HasPrefix(pattern, "**/") {
				suffix := strings.TrimPrefix(pattern, "**/")
				matched, _ = filepath.Match(suffix, filepath.Base(f.Path))
			}
			if matched {
				log.Debug().Str("file", f.Path).Str("pattern", pattern).Msg("Excluding file")
				excluded = true
				break
			}
		}
		if !excluded {
			result = append(result, f)
		}
	}
	return result
}

// ParseExcludePatterns splits comma-separated patterns into a slice
func (s *DiffService) ParseExcludePatterns(patterns string) []string {
	if patterns == "" {
		return nil
	}
	parts := strings.Split(patterns, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// ExtractAddedLineNumbers returns the "new file" line numbers from a diff hunk
// These are the lines valid for posting RIGHT-side comments on GitHub
func (s *DiffService) ExtractAddedLineNumbers(diff string) []int {
	var lines []int
	currentLine := 0

	for _, line := range strings.Split(diff, "\n") {
		if strings.HasPrefix(line, "@@") {
			// Parse hunk header: @@ -old,count +new,count @@
			currentLine = parseHunkNewStart(line)
			continue
		}

		if currentLine == 0 {
			continue
		}

		if strings.HasPrefix(line, "+") {
			lines = append(lines, currentLine)
			currentLine++
		} else if strings.HasPrefix(line, "-") {
			// Deleted line doesn't advance the new file line counter
		} else {
			// Context line
			currentLine++
		}
	}
	return lines
}

// ValidateCommentLine checks if a line number is valid for a RIGHT-side comment on the given diff
func (s *DiffService) ValidateCommentLine(diff string, line int) bool {
	for _, l := range s.ExtractAddedLineNumbers(diff) {
		if l == line {
			return true
		}
	}
	// Also allow context lines
	currentLine := 0
	for _, dl := range strings.Split(diff, "\n") {
		if strings.HasPrefix(dl, "@@") {
			currentLine = parseHunkNewStart(dl)
			continue
		}
		if currentLine == 0 {
			continue
		}
		if strings.HasPrefix(dl, "-") {
			continue
		}
		if currentLine == line {
			return true
		}
		currentLine++
	}
	return false
}

// parseHunkNewStart extracts the new file start line from a hunk header like "@@ -1,5 +3,8 @@"
func parseHunkNewStart(hunkHeader string) int {
	// Find the +N part
	idx := strings.Index(hunkHeader, "+")
	if idx < 0 {
		return 0
	}
	rest := hunkHeader[idx+1:]
	// Take until comma or space
	end := strings.IndexAny(rest, ", @")
	if end < 0 {
		end = len(rest)
	}
	n, err := strconv.Atoi(rest[:end])
	if err != nil {
		return 0
	}
	return n
}
