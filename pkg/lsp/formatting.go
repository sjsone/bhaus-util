package lsp

import (
	"regexp"
	"strings"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Top-level keywords that should never be indented.
// Also used by completion.go for context-aware completion filtering.
var topLevelKeywords = []string{
	"CLASS", "STRUCT", "PROTOCOL", "FUNCTION",
	"SYSTEM", "CONTAINER", "COMPONENT", "CONNECTION",
}

// Access modifier keywords that depend on context
var accessModifiers = []string{
	"PUBLIC", "PRIVATE", "PROTECTED", "OVERRIDE",
}

// TextDocumentOnTypeFormatting handles auto-indentation when the user presses Enter
func (h *Handler) TextDocumentOnTypeFormatting(context *glsp.Context, params *protocol.DocumentOnTypeFormattingParams) ([]protocol.TextEdit, error) {
	uri := string(params.TextDocument.URI)
	position := params.Position

	content, ok := h.Documents[uri]
	if !ok {
		return nil, nil
	}

	// Get the text edits for formatting
	edits := getIndentationEdits(content, position)

	logger.Debugf("onTypeFormatting: %s @ line %d -> %d edit(s)", uri, position.Line, len(edits))
	return edits, nil
}

// getIndentationEdits calculates the proper indentation for the current position
func getIndentationEdits(content string, position protocol.Position) []protocol.TextEdit {
	lines := strings.Split(content, "\n")

	if int(position.Line) >= len(lines) {
		return nil
	}

	// Get the previous non-empty line
	prevLineIdx := int(position.Line) - 1
	var prevLine string
	var prevIndent int

	for prevLineIdx >= 0 {
		prevLine = lines[prevLineIdx]
		trimmed := strings.TrimLeft(prevLine, " \t")
		if trimmed != "" {
			// Found a non-empty line
			prevIndent = len(prevLine) - len(trimmed)
			break
		}
		prevLineIdx--
	}

	if prevLineIdx < 0 {
		// No previous content, no indentation
		return nil
	}

	// Get current line content (if any) to check for top-level keywords
	currentLine := ""
	if int(position.Line) < len(lines) {
		currentLine = lines[int(position.Line)]
	}
	currentLineTrimmed := strings.TrimSpace(currentLine)

	// Check if current line starts with a top-level keyword
	isTopLevelKeyword := false
	for _, kw := range topLevelKeywords {
		if strings.HasPrefix(currentLineTrimmed, kw) {
			isTopLevelKeyword = true
			break
		}
	}

	// If typing a top-level keyword, force indent to 0
	if isTopLevelKeyword {
		// Calculate range from start of current line to the first non-whitespace
		currentIndent := len(currentLine) - len(strings.TrimLeft(currentLine, " \t"))
		if currentIndent > 0 {
			return []protocol.TextEdit{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: position.Line, Character: 0},
						End:   protocol.Position{Line: position.Line, Character: uint32(currentIndent)},
					},
					NewText: "",
				},
			}
		}
		return nil
	}

	// Check if the previous line is a container definition (CLASS/STRUCT/PROTOCOL Name:)
	trimmedPrevLine := strings.TrimSpace(prevLine)
	isContainerStart := false

	containerRegex := regexp.MustCompile(`^(CLASS|STRUCT|PROTOCOL)\s+\S+?:`)
	if containerRegex.MatchString(trimmedPrevLine) {
		isContainerStart = true
	}

	// Calculate desired indentation
	desiredIndent := 0

	if isContainerStart {
		// After a container definition, indent by 4 spaces
		desiredIndent = prevIndent + 4
	} else {
		// Check whether the cursor is inside a container, using the
		// previous line's indentation. If the previous line is indented,
		// the cursor is inside a container. Keep that indent.
		if prevIndent >= 4 {
			// The cursor is inside a container. Keep the indentation.
			desiredIndent = prevIndent
		} else {
			// Top-level, no indentation
			desiredIndent = 0
		}
	}

	// Check for access modifiers. They should stay at the same level as the container content.
	if currentLineTrimmed != "" {
		for _, mod := range accessModifiers {
			if strings.HasPrefix(currentLineTrimmed, mod) {
				// Check whether the cursor is at top level. This should not
				// happen for access modifiers. If it is, keep indent at 0.
				// Otherwise, keep the previous indent.
				if prevIndent > 0 {
					desiredIndent = prevIndent
				}
				break
			}
		}
	}

	// Get current indentation
	currentIndent := 0
	if int(position.Line) < len(lines) {
		currentIndent = len(currentLine) - len(strings.TrimLeft(currentLine, " \t"))
	}

	// Only apply edit if indentation differs
	if currentIndent != desiredIndent {
		return []protocol.TextEdit{
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: position.Line, Character: 0},
					End:   protocol.Position{Line: position.Line, Character: uint32(currentIndent)},
				},
				NewText: strings.Repeat(" ", desiredIndent),
			},
		}
	}

	return nil
}
