// Package include resolves INCLUDE directives to concrete file paths.
//
// The language server (incremental, Handler-backed loading) and the CLI linter
// (one-shot loading) share this logic. It extracts INCLUDE patterns from a
// document. It normalizes the patterns. It globs the patterns relative to the
// document's own directory. Callers drive their own recursion and storage.
// Match does not recurse into matched files. Match does not read or parse them.
package include

import (
	"path/filepath"
	"strings"
)

// Match returns the absolute filesystem paths of the files that the document at
// fileURI directly includes. It globs patterns relative to the document's own
// directory. The returned slice has no duplicates and keeps first-seen order.
// Match does not filter out the document's own path; callers already dedupe by URI.
func Match(fileURI, content string) []string {
	patterns := Patterns(content)
	if len(patterns) == 0 {
		return nil
	}

	dir := filepath.Dir(strings.TrimPrefix(fileURI, "file://"))

	var matches []string
	seen := make(map[string]bool)
	for _, pattern := range patterns {
		full := filepath.Join(dir, pattern)
		globbed, err := filepath.Glob(full)
		if err != nil {
			// Only ErrBadPattern can occur here. Skip a malformed pattern
			// instead of failing the whole resolution.
			continue
		}
		for _, m := range globbed {
			if seen[m] {
				continue
			}
			seen[m] = true
			matches = append(matches, m)
		}
	}
	return matches
}

// Patterns extracts the glob patterns from the INCLUDE directives in content.
// If a wildcard pattern lacks a .bhaus extension, Patterns appends one.
// This makes "INCLUDE types/*" resolve the same as "INCLUDE types/*.bhaus".
func Patterns(content string) []string {
	var patterns []string
	for line := range strings.SplitSeq(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "INCLUDE ") {
			continue
		}
		pattern := strings.TrimSpace(strings.TrimPrefix(trimmed, "INCLUDE "))
		if pattern == "" {
			continue
		}
		if !strings.HasSuffix(pattern, ".bhaus") && strings.Contains(pattern, "*") {
			pattern += ".bhaus"
		}
		patterns = append(patterns, pattern)
	}
	return patterns
}
