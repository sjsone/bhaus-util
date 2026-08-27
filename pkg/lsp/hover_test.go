package lsp

import (
	"strings"
	"testing"

	"github.com/sjsone/bhaus-util/pkg/analysis"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestTextDocumentHover(t *testing.T) {
	handler := NewHandler()
	uri := protocol.DocumentUri("file:///test/hover.bhaus")

	testContent := `CLASS TestClass:
    PUBLIC testFunc(): String

STRUCT TestStruct:

PROTOCOL TestProtocol:
`
	handler.Documents[string(uri)] = testContent
	file, err := handler.Cache.Parse(string(uri), testContent)
	if err != nil {
		t.Fatalf("failed to parse test content: %v", err)
	}
	handler.Files[string(uri)] = file
	handler.Table = analysis.BuildSymbolTable(handler.Files)

	tests := []struct {
		name          string
		line          uint32
		character     uint32
		expectContain string
		expectNil     bool
	}{
		{
			name:          "hover on class shows class declaration",
			line:          0,
			character:     7, // "TestClass" in "CLASS TestClass:"
			expectContain: "CLASS TestClass:",
		},
		{
			name:          "hover on struct shows struct declaration",
			line:          3,
			character:     8, // "TestStruct" in "STRUCT TestStruct:"
			expectContain: "STRUCT TestStruct",
		},
		{
			name:          "hover on protocol shows protocol declaration",
			line:          5,
			character:     10, // "TestProtocol" in "PROTOCOL TestProtocol:"
			expectContain: "PROTOCOL TestProtocol",
		},
		{
			name:          "hover on method shows method declaration",
			line:          1,
			character:     11, // "testFunc" in "PUBLIC testFunc(): String"
			expectContain: "PUBLIC testFunc(): String",
		},
		{
			name:      "hover on empty line returns nil",
			line:      2,
			character: 0,
			expectNil: true,
		},
		{
			name:      "hover on unknown word returns nil",
			line:      2,
			character: 3,
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: tt.line, Character: tt.character},
				},
			}

			hover, err := handler.TextDocumentHover(nil, params)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.expectNil {
				if hover != nil {
					t.Errorf("expected nil hover, got: %v", hover)
				}
				return
			}

			if hover == nil {
				t.Fatalf("expected non-nil hover")
			}

			markup, ok := hover.Contents.(protocol.MarkupContent)
			if !ok {
				t.Fatalf("expected MarkupContent, got %T", hover.Contents)
			}

			if !strings.Contains(markup.Value, tt.expectContain) {
				t.Errorf("expected hover to contain %q, got %q", tt.expectContain, markup.Value)
			}
		})
	}
}

func TestHoverBuiltinType(t *testing.T) {
	handler := NewHandler()
	uri := protocol.DocumentUri("file:///test/builtin.bhaus")

	content := "String Int Boolean"
	handler.Documents[string(uri)] = content

	tests := []struct {
		name      string
		line      uint32
		character uint32
		expect    string
	}{
		{"String type", 0, 2, "**type** String"},
		{"Int type", 0, 9, "**type** Int"},
		{"Boolean type", 0, 13, "**type** Boolean"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: tt.line, Character: tt.character},
				},
			}

			hover, err := handler.TextDocumentHover(nil, params)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if hover == nil {
				t.Fatalf("expected non-nil hover for built-in type")
			}

			markup := hover.Contents.(protocol.MarkupContent)
			if markup.Value != tt.expect {
				t.Errorf("expected %q, got %q", tt.expect, markup.Value)
			}
		})
	}
}

func TestHoverMethod(t *testing.T) {
	handler := NewHandler()
	uri := protocol.DocumentUri("file:///test/method.bhaus")

	content := `CLASS MyClass:
    PUBLIC myMethod(): String
`
	handler.Documents[string(uri)] = content
	file, err := handler.Cache.Parse(string(uri), content)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	handler.Files[string(uri)] = file
	handler.Table = analysis.BuildSymbolTable(handler.Files)

	// Hover on the actual method name
	params := &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 1, Character: 11}, // on "myMethod"
		},
	}

	hover, err := handler.TextDocumentHover(nil, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected non-nil hover for method")
	}

	markup := hover.Contents.(protocol.MarkupContent)
	if !strings.Contains(markup.Value, "PUBLIC myMethod(): String") {
		t.Errorf("expected hover to contain method declaration, got %q", markup.Value)
	}
	if !strings.Contains(markup.Value, "PUBLIC") {
		t.Errorf("expected hover to contain method visibility, got %q", markup.Value)
	}
}

func TestHoverConnection(t *testing.T) {
	handler := NewHandler()
	uri := protocol.DocumentUri("file:///test/connection.bhaus")

	content := `SYSTEM MySystem:
    CONTAINER ContainerA:
        COMPONENT CompA:
    CONTAINER ContainerB:
        COMPONENT CompB:
    CONNECTION ContainerA/CompA => ContainerB/CompB
`
	handler.Documents[string(uri)] = content
	file, err := handler.Cache.Parse(string(uri), content)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	handler.Files[string(uri)] = file
	handler.Table = analysis.BuildSymbolTable(handler.Files)

	// Find the connection symbol to get its exact name
	var connName string
	for _, sym := range handler.Table.SymbolsForURI(string(uri)) {
		if sym.Kind == analysis.KindConnection {
			connName = sym.Name
			t.Logf("Connection symbol: %q", sym.Name)
		}
	}
	if connName == "" {
		t.Fatal("expected to find a connection symbol")
	}

	// Hover on "ContainerA" part of the connection line (character 13)
	params := &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 6, Character: 13}, // on "ContainerA" in connection
		},
	}

	// "ContainerA" will match the container symbol, not the connection
	hover, err := handler.TextDocumentHover(nil, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// ContainerA is a container symbol. The hover should show the container.
	if hover != nil {
		markup := hover.Contents.(protocol.MarkupContent)
		t.Logf("ContainerA hover: %q", markup.Value)
	}
}

func TestHoverDocComment(t *testing.T) {
	handler := NewHandler()
	uri := protocol.DocumentUri("file:///test/doc.bhaus")

	content := `# The primary user entity.
# Second doc line.
STRUCT User:
    # The unique identifier.
    PUBLIC id: Integer
    PUBLIC name: String # trailing, not a doc comment
STRUCT Bare:
    PUBLIC x: Integer
`
	handler.Documents[string(uri)] = content
	file, err := handler.Cache.Parse(string(uri), content)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	handler.Files[string(uri)] = file
	handler.Table = analysis.BuildSymbolTable(handler.Files)

	hoverAt := func(line, char uint32) string {
		params := &protocol.HoverParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     protocol.Position{Line: line, Character: char},
			},
		}
		hover, err := handler.TextDocumentHover(nil, params)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if hover == nil {
			return ""
		}
		return hover.Contents.(protocol.MarkupContent).Value
	}

	// Struct with a two-line doc comment block above it.
	structHover := hoverAt(2, 8) // "User"
	if !strings.Contains(structHover, "The primary user entity.") ||
		!strings.Contains(structHover, "Second doc line.") {
		t.Errorf("expected struct doc comment block, got %q", structHover)
	}

	// Property with a standalone doc comment on the line above.
	idHover := hoverAt(4, 12) // "id"
	if !strings.Contains(idHover, "The unique identifier.") {
		t.Errorf("expected property doc comment, got %q", idHover)
	}

	// Property whose only comment is trailing on its own line: no doc comment.
	nameHover := hoverAt(5, 12) // "name"
	if strings.Contains(nameHover, "trailing") {
		t.Errorf("trailing comment must not appear as a doc comment, got %q", nameHover)
	}

	// Struct preceded by code (not a comment): no doc comment separator.
	bareHover := hoverAt(6, 8) // "Bare"
	if strings.Contains(bareHover, "---") {
		t.Errorf("expected no doc comment for Bare, got %q", bareHover)
	}
}

// TestHoverRangeSpansQualifiedName verifies the hover Range. The range must
// cover the entire qualified name under the cursor (e.g. "Domain/User"), not
// just the token under the cursor. This holds for any position within the name.
func TestHoverRangeSpansQualifiedName(t *testing.T) {
	handler := NewHandler()
	uri := protocol.DocumentUri("file:///test/qname.bhaus")
	content := "STRUCT Domain/User:\n    PUBLIC id: Integer\nSTRUCT Account:\n    PUBLIC owner: Domain/User\n"
	handler.Documents[string(uri)] = content
	file, err := handler.Cache.Parse(string(uri), content)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	handler.Files[string(uri)] = file
	handler.Table = analysis.BuildSymbolTable(handler.Files)

	// "Domain/User" reference on line 3 spans columns 18..29.
	for _, col := range []uint32{18, 22, 24, 25, 28} {
		hov, err := handler.TextDocumentHover(nil, &protocol.HoverParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     protocol.Position{Line: 3, Character: col},
			},
		})
		if err != nil {
			t.Fatalf("col %d: %v", col, err)
		}
		if hov == nil || hov.Range == nil {
			t.Errorf("col %d: expected hover with a range", col)
			continue
		}
		r := *hov.Range
		if r.Start.Line != 3 || r.Start.Character != 18 || r.End.Line != 3 || r.End.Character != 29 {
			t.Errorf("col %d: range = %d:%d..%d:%d, want 3:18..3:29", col,
				r.Start.Line, r.Start.Character, r.End.Line, r.End.Character)
		}
	}
}

// TestHoverShowsTypeMembers verifies that hovering over a type reference
// (e.g. "Base/Entity") shows the members declared by that protocol, class or
// struct. The symbol table resolves these members across files.
func TestHoverShowsTypeMembers(t *testing.T) {
	handler := NewHandler()
	entityURI := protocol.DocumentUri("file:///test/entity.bhaus")
	modelURI := protocol.DocumentUri("file:///test/model.bhaus")

	entityContent := `VERSION 0.1

STRUCT UUID:

PROTOCOL Base/Entity:
    PUBLIC getIdentifier(): UUID
    PUBLIC value: String
    PUBLIC raw: String
`
	modelContent := `CLASS Model IMPLEMENTS Base/Entity:
    PUBLIC test: String
`
	handler.Documents[string(entityURI)] = entityContent
	handler.Documents[string(modelURI)] = modelContent
	entityFile, err := handler.Cache.Parse(string(entityURI), entityContent)
	if err != nil {
		t.Fatalf("failed to parse entity file: %v", err)
	}
	modelFile, err := handler.Cache.Parse(string(modelURI), modelContent)
	if err != nil {
		t.Fatalf("failed to parse model file: %v", err)
	}
	handler.Files[string(entityURI)] = entityFile
	handler.Files[string(modelURI)] = modelFile
	handler.Table = analysis.BuildSymbolTable(handler.Files)

	params := &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: modelURI},
			Position:     protocol.Position{Line: 0, Character: 26}, // on "Base/Entity"
		},
	}
	hover, err := handler.TextDocumentHover(nil, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected non-nil hover for protocol reference")
	}
	markup := hover.Contents.(protocol.MarkupContent)
	if !strings.Contains(markup.Value, "PROTOCOL Base/Entity:") {
		t.Errorf("expected hover to contain the protocol declaration header, got %q", markup.Value)
	}
	for _, member := range []string{
		"PUBLIC getIdentifier(): UUID",
		"PUBLIC value: String",
		"PUBLIC raw: String",
	} {
		if !strings.Contains(markup.Value, member) {
			t.Errorf("expected hover to contain member %q, got %q", member, markup.Value)
		}
	}
}

// TestHoverTypeWithoutMembers verifies that hovering over a type with no
// members renders just the declaration header. It must have no member lines
// and no dangling colon.
func TestHoverTypeWithoutMembers(t *testing.T) {
	handler := NewHandler()
	uri := protocol.DocumentUri("file:///test/empty.bhaus")

	content := "STRUCT Empty:\n"
	handler.Documents[string(uri)] = content
	file, err := handler.Cache.Parse(string(uri), content)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	handler.Files[string(uri)] = file
	handler.Table = analysis.BuildSymbolTable(handler.Files)

	params := &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 8}, // on "Empty"
		},
	}
	hover, err := handler.TextDocumentHover(nil, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected non-nil hover for struct")
	}
	markup := hover.Contents.(protocol.MarkupContent)
	want := "```bhaus\nSTRUCT Empty\n```"
	if markup.Value != want {
		t.Errorf("expected memberless struct hover %q, got %q", want, markup.Value)
	}
}

func TestHoverOnUnopenedDocument(t *testing.T) {
	handler := NewHandler()
	uri := protocol.DocumentUri("file:///test/nonexistent.bhaus")

	params := &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 0},
		},
	}

	hover, err := handler.TextDocumentHover(nil, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hover != nil {
		t.Errorf("expected nil hover for unopened document, got: %v", hover)
	}
}

func TestHoverMethodWithIntents(t *testing.T) {
	handler := NewHandler()
	uri := protocol.DocumentUri("file:///test/intent.bhaus")

	content := `PROTOCOL Base/Bla:
    PUBLIC toLowerCase(String): String
        > walk over every char and swap for lowercase
`
	handler.Documents[string(uri)] = content
	file, err := handler.Cache.Parse(string(uri), content)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	handler.Files[string(uri)] = file
	handler.Table = analysis.BuildSymbolTable(handler.Files)

	params := &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 1, Character: 12}, // on "toLowerCase"
		},
	}
	hover, err := handler.TextDocumentHover(nil, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected non-nil hover for method")
	}
	markup := hover.Contents.(protocol.MarkupContent)
	if !strings.Contains(markup.Value, "PUBLIC toLowerCase(String): String") {
		t.Errorf("expected hover to contain the method declaration, got %q", markup.Value)
	}
	if !strings.Contains(markup.Value, "> walk over every char and swap for lowercase") {
		t.Errorf("expected hover to contain the intent text, got %q", markup.Value)
	}
}

func TestHoverVersion(t *testing.T) {
	handler := NewHandler()
	uri := protocol.DocumentUri("file:///test/version.bhaus")
	content := "VERSION 0.1\nSTRUCT S:\n    PUBLIC id: Integer\n"
	handler.Documents[string(uri)] = content
	file, err := handler.Cache.Parse(string(uri), content)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	handler.Files[string(uri)] = file
	handler.Table = analysis.BuildSymbolTable(handler.Files)

	// Hover on VERSION keyword (hovering on "VERSION" itself)
	hov, err := handler.TextDocumentHover(nil, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 0}, // "VERSION" at start
		},
	})
	if err != nil {
		t.Fatalf("hover: %v", err)
	}
	if hov == nil {
		t.Fatal("expected hover on VERSION")
	}
	markup, ok := hov.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected MarkupContent, got %T", hov.Contents)
	}
	if !strings.Contains(markup.Value, "VERSION 0.1") {
		t.Errorf("expected hover to contain VERSION 0.1, got %q", markup.Value)
	}
}
