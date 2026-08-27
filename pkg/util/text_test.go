package util

import (
	"testing"
)

func TestGetWordAtPosition(t *testing.T) {
	content := "CLASS TestClass:\n    PUBLIC testMethod()\n    name: String\n"

	tests := []struct {
		name     string
		line     int
		col      int
		expected string
	}{
		{"keyword at start of line", 0, 2, "CLASS"},
		{"type name mid-line", 0, 9, "TestClass"},
		{"indented keyword", 1, 5, "PUBLIC"},
		{"method name", 1, 17, "testMethod"},
		{"field name", 2, 4, "name"},
		{"type annotation value", 2, 12, "String"},
		{"cursor at end backs up", 0, 16, "TestClass"}, // col=16 is len("CLASS TestClass:") → backs up to "TestClass"
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetWordAtPosition(content, tt.line, tt.col)
			if got != tt.expected {
				t.Errorf("GetWordAtPosition(line=%d, col=%d) = %q, want %q",
					tt.line, tt.col, got, tt.expected)
			}
		})
	}
}

func TestGetWordAtPosition_ContextualPath(t *testing.T) {
	content := "CLASS Domain/Entity/User:\n"

	got := GetWordAtPosition(content, 0, 10)
	if got != "Domain/Entity/User" {
		t.Errorf("GetWordAtPosition for contextual path = %q, want %q", got, "Domain/Entity/User")
	}
}

func TestGetWordAtPosition_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		line     int
		col      int
		expected string
	}{
		{"negative line", "CLASS A:", -1, 0, ""},
		{"line beyond content", "CLASS A:", 10, 0, ""},
		{"negative column", "CLASS A:", 0, -1, ""},
		{"col beyond line", "CLASS A:", 0, 50, ""},
		{"empty content", "", 0, 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetWordAtPosition(tt.content, tt.line, tt.col)
			if got != tt.expected {
				t.Errorf("GetWordAtPosition() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestIsWordChar(t *testing.T) {
	tests := []struct {
		c        byte
		expected bool
	}{
		{'a', true}, {'z', true},
		{'A', true}, {'Z', true},
		{'0', true}, {'9', true},
		{'_', true},
		{' ', false}, {':', false}, {'/', false},
		{'(', false}, {')', false}, {',', false},
	}

	for _, tt := range tests {
		got := IsWordChar(tt.c)
		if got != tt.expected {
			t.Errorf("IsWordChar(%q) = %v, want %v", tt.c, got, tt.expected)
		}
	}
}

func TestURIToPath(t *testing.T) {
	tests := []struct {
		uri      string
		expected string
	}{
		{"file:///home/user/file.bhaus", "/home/user/file.bhaus"},
		{"file://relative/path.bhaus", "relative/path.bhaus"},
		{"/already/a/path.bhaus", "/already/a/path.bhaus"},
	}

	for _, tt := range tests {
		got := URIToPath(tt.uri)
		if got != tt.expected {
			t.Errorf("URIToPath(%q) = %q, want %q", tt.uri, got, tt.expected)
		}
	}
}
