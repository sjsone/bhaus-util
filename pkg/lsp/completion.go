package lsp

import (
	"slices"
	"strings"

	"github.com/sjsone/bhaus-util/pkg/analysis"
	"github.com/sjsone/bhaus-util/pkg/ast"
	"github.com/sjsone/bhaus-util/pkg/util"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// visibilityKeywords are the visibility modifiers valid in member declarations.
// topLevelKeywords (declared in formatting.go) are used for top-level completion.
var visibilityKeywords = []string{
	"PUBLIC", "PRIVATE", "PROTECTED",
}

// TextDocumentCompletion handles the textDocument/completion request.
func (h *Handler) TextDocumentCompletion(context *glsp.Context, params *protocol.CompletionParams) (any, error) {
	uri := string(params.TextDocument.URI)
	position := params.Position
	logger.Debugf("completion: %s @ %d:%d", uri, position.Line, position.Character)

	content, ok := h.Documents[uri]
	if !ok {
		return []protocol.CompletionItem{}, nil
	}

	lines := strings.Split(content, "\n")
	if int(position.Line) >= len(lines) {
		return []protocol.CompletionItem{}, nil
	}

	currentLine := lines[position.Line]

	// Inside a functional intent ("> ..." line), the body is free prose. No
	// keyword, type or symbol completion applies here. The function returns
	// nothing in this case. A leading ">" (after optional indentation) is
	// unambiguous: no other construct starts a line with it. Connection
	// arrows live on "CONNECTION ..." lines, not here. This check tests the
	// whole line, so it stays robust when the cursor sits at the end of the
	// line while typing. An AST span test (half-open range) would miss that
	// case.
	if strings.HasPrefix(strings.TrimSpace(currentLine), ">") {
		logger.Debugf("completion: cursor in functional intent, suppressing")
		return []protocol.CompletionItem{}, nil
	}

	// Get leading whitespace (indentation) of current line
	var leadingWS strings.Builder
	for _, c := range currentLine {
		if c == ' ' || c == '\t' {
			leadingWS.WriteString(string(c))
		} else {
			break
		}
	}

	// Get the word prefix being typed (up to cursor position)
	word := util.GetWordPrefixAtPosition(content, int(position.Line), int(position.Character))
	wordLower := strings.ToLower(word)

	// Determine what token comes before the current word prefix on this line.
	// This gives us context: are we after OVERRIDE? After PUBLIC? Or at the start?
	wordStartCol := int(position.Character) - len(word)
	prevWord := lastWordBefore(currentLine, wordStartCol)

	// Detect whether we are inside a container body (CLASS/STRUCT/PROTOCOL)
	// using the AST ancestor chain from PathTo.
	insideContainer, containerType := false, ""
	if file, ok := h.Files[uri]; ok && file != nil {
		cursorLine := uint32(position.Line)
		path := ast.PathTo(file, cursorLine, uint32(position.Character))
		var found bool
		insideContainer, containerType, found = detectContainerFromPath(path, cursorLine)
		if !found {
			// Blank lines between members fall outside every declaration's
			// span, so PathTo won't see the enclosing container. Fall back to
			// the nearest container declaration ending above the cursor.
			insideContainer, containerType = detectContainerInFile(file, cursorLine)
		}
	}

	// Detect class-header contexts (IMPLEMENTS/EXTENDS) for symbol filtering.
	upperLine := strings.ToUpper(strings.TrimSpace(currentLine))
	implementsContext := strings.Contains(upperLine, "IMPLEMENTS")
	extendsContext := strings.Contains(upperLine, "EXTENDS")

	var items []protocol.CompletionItem

	if insideContainer {
		// ── Inside a container body (member level) ──
		switch prevWord {
		case "OVERRIDE":
			// After OVERRIDE: only visibility keywords are legal.
			addKeywordCompletions(&items, word, wordLower, leadingWS.String(), visibilityKeywords)
		case "PUBLIC", "PRIVATE", "PROTECTED":
			// After a completed visibility keyword, no more keywords apply.
			// The user is typing a type or symbol name (added below).
		default:
			// Start of a member line (or typing the first token).
			addKeywordCompletions(&items, word, wordLower, leadingWS.String(), visibilityKeywords)
			// OVERRIDE is only valid in CLASS members, not STRUCT/PROTOCOL.
			if containerType == "class" {
				addKeywordCompletions(&items, word, wordLower, leadingWS.String(), []string{"OVERRIDE"})
			}
		}
	} else {
		// ── Top level ──
		// IMPLEMENTS/EXTENDS on a class header line: suppress keywords.
		// The user is typing a type name (protocol or class).
		if !implementsContext && !extendsContext {
			addKeywordCompletions(&items, word, wordLower, leadingWS.String(), topLevelKeywords)
		}
	}

	// Add built-in type completions. These are inline, so they need no indentation.
	for t := range BuiltInTypes {
		if word == "" || strings.HasPrefix(strings.ToLower(t), wordLower) {
			items = append(items, protocol.CompletionItem{
				Label:      t,
				Kind:       func() *protocol.CompletionItemKind { k := protocol.CompletionItemKindTypeParameter; return &k }(),
				InsertText: &t,
			})
		}
	}

	// Add symbol completions from the symbol table.
	var symbols []analysis.Symbol
	if word == "" {
		// For an empty prefix, offer every declared symbol.
		symbols = h.Table.All()
	} else {
		symbols = h.Table.Search(word)
	}
	for _, sym := range symbols {
		// Context-aware filtering: only show relevant symbol types.
		if implementsContext && sym.Kind != analysis.KindProtocol {
			continue
		}
		if extendsContext && sym.Kind != analysis.KindClass {
			continue
		}

		candidates := []string{sym.Name, sym.FullName}
		for _, candidate := range candidates {
			if candidate == "" {
				continue
			}
			if word == "" || strings.HasPrefix(strings.ToLower(candidate), wordLower) {
				// Deduplicate.
				exists := false
				for _, item := range items {
					if item.InsertText != nil && *item.InsertText == candidate {
						exists = true
						break
					}
				}
				if !exists {
					var kind protocol.CompletionItemKind
					switch sym.Kind {
					case analysis.KindClass:
						kind = protocol.CompletionItemKindClass
					case analysis.KindStruct:
						kind = protocol.CompletionItemKindStruct
					case analysis.KindProtocol:
						kind = protocol.CompletionItemKindInterface
					default:
						kind = protocol.CompletionItemKindVariable
					}
					insertText := candidate
					items = append(items, protocol.CompletionItem{
						Label:      candidate,
						Kind:       &kind,
						InsertText: &insertText,
					})
				}
			}
		}
	}

	logger.Debugf("completion: word=%q -> %d item(s)", word, len(items))
	return items, nil
}

// detectContainerFromPath checks whether the cursor is inside a
// CLASS/STRUCT/PROTOCOL body using the AST ancestor chain from PathTo.
// A container's own header line counts as top-level. This matches the old
// line-scanning behavior. The function compares the cursor line against each
// container's start line. The third return value reports whether the path
// contains any container at all.
func detectContainerFromPath(path []ast.Node, line uint32) (inside bool, containerType string, found bool) {
	for _, p := range slices.Backward(path) {
		switch d := p.(type) {
		case *ast.ClassDecl:
			return line > d.Pos().Line, "class", true
		case *ast.StructDecl:
			return line > d.Pos().Line, "struct", true
		case *ast.ProtocolDecl:
			return line > d.Pos().Line, "protocol", true
		}
	}
	return false, "", false
}

// detectContainerInFile finds the nearest container declaration whose span
// ends strictly above the cursor line. It is a fallback for positions that
// fall outside every declaration's span. This typically happens on blank
// lines between members. Those blank lines still belong to the enclosing
// container.
func detectContainerInFile(file *ast.File, line uint32) (inside bool, containerType string) {
	var last ast.Decl
	for _, d := range file.Decls {
		if d.End().Line < line {
			last = d
		} else {
			break // declarations are in source order
		}
	}
	switch last.(type) {
	case *ast.ClassDecl:
		return true, "class"
	case *ast.StructDecl:
		return true, "struct"
	case *ast.ProtocolDecl:
		return true, "protocol"
	}
	return false, ""
}

// lastWordBefore returns the last complete word on the current line
// before the given column, uppercased for easy comparison.
// Returns "" if there is no word before the cursor prefix.
func lastWordBefore(line string, col int) string {
	if col <= 0 || col > len(line) {
		return ""
	}
	before := strings.TrimSpace(line[:col])
	words := strings.Fields(before)
	if len(words) == 0 {
		return ""
	}
	return strings.ToUpper(words[len(words)-1])
}

// addKeywordCompletions appends matching keyword completions to items.
// Each keyword is wrapped with leadingWS so indentation is preserved.
func addKeywordCompletions(items *[]protocol.CompletionItem, word, wordLower, leadingWS string, keywords []string) {
	for _, kw := range keywords {
		if word == "" || strings.HasPrefix(strings.ToLower(kw), wordLower) {
			insertText := leadingWS + kw
			*items = append(*items, protocol.CompletionItem{
				Label:      kw,
				Kind:       func() *protocol.CompletionItemKind { k := protocol.CompletionItemKindKeyword; return &k }(),
				InsertText: &insertText,
			})
		}
	}
}
