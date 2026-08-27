// Package util provides zero-dependency shared utilities for text manipulation
// and URI handling. The LSP server and the CLI both use this package.
package util

import (
	"path/filepath"
	"strings"
)

// GetWordAtPosition extracts the word at a given line/column position.
// Includes '/' in word characters for contextual paths like Domain/Entity/User.
// Returns empty string if the position is out of bounds.
func GetWordAtPosition(content string, line, col int) string {
	lines := strings.Split(content, "\n")
	if line < 0 || line >= len(lines) {
		return ""
	}

	currentLine := lines[line]
	if col < 0 || col > len(currentLine) {
		return ""
	}

	// At end of line, back up to the last character so cursor-after-last-char works
	if col == len(currentLine) {
		if col == 0 {
			return ""
		}
		col = col - 1
	}

	start := col
	end := col

	for start > 0 {
		c := currentLine[start-1]
		if !IsWordChar(c) && c != '/' {
			break
		}
		start--
	}

	for end < len(currentLine) {
		c := currentLine[end]
		if !IsWordChar(c) && c != '/' {
			break
		}
		end++
	}

	return currentLine[start:end]
}

// GetWordPrefixAtPosition extracts the word prefix up to (but not including) col.
// Completion uses this to get the text the user has typed so far.
// It does not include '/'. Use it only for completion prefixes.
func GetWordPrefixAtPosition(content string, line, col int) string {
	lines := strings.Split(content, "\n")
	if line < 0 || line >= len(lines) {
		return ""
	}

	currentLine := lines[line]
	if col < 0 || col > len(currentLine) {
		return ""
	}

	start := col
	for start > 0 {
		c := currentLine[start-1]
		if !IsWordChar(c) {
			break
		}
		start--
	}

	return currentLine[start:col]
}

// IsWordChar returns true if the byte is valid in an identifier.
func IsWordChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}

// URIToPath converts a file:// URI to a filesystem path.
func URIToPath(uri string) string {
	if after, ok := strings.CutPrefix(uri, "file://"); ok {
		path := after
		if len(path) > 1 && path[1] == '/' {
			path = path[1:]
		}
		return path
	}
	return uri
}

// PathToURI converts a filesystem path to a file:// URI.
func PathToURI(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "file://" + path
	}
	return "file://" + absPath
}
