package lsp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

// TestIncludeResolvesCrossFileSymbols verifies that opening a file with an
// INCLUDE directive loads the referenced files. Definition and hover then
// resolve types defined there. This is a regression test for a "no symbol"
// error on Base/Bla in a file that includes its definitions.
func TestIncludeResolvesCrossFileSymbols(t *testing.T) {
	dir := t.TempDir()
	base := "VERSION 0.1\nPROTOCOL Base/Entity:\n    PUBLIC raw: String\nPROTOCOL Base/Bla:\n    PUBLIC rec: Base/Entity\n"
	model := "VERSION 0.1\nINCLUDE *\n\nCLASS Model IMPLEMENTS Base/Entity:\n    PUBLIC test: Base/Bla\n"
	if err := os.WriteFile(filepath.Join(dir, "base.bhaus"), []byte(base), 0o644); err != nil {
		t.Fatal(err)
	}
	modelPath := filepath.Join(dir, "model.bhaus")
	if err := os.WriteFile(modelPath, []byte(model), 0o644); err != nil {
		t.Fatal(err)
	}
	modelURI := "file://" + modelPath

	h := NewHandler()
	if err := h.TextDocumentDidOpen(nil, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{URI: protocol.DocumentUri(modelURI), Text: model},
	}); err != nil {
		t.Fatalf("didOpen: %v", err)
	}

	// INCLUDE must auto-load base.bhaus.
	if _, ok := h.Files["file://"+filepath.Join(dir, "base.bhaus")]; !ok {
		t.Fatalf("base.bhaus was not loaded via INCLUDE; files loaded: %d", len(h.Files))
	}

	// Hover on Base/Bla (line 4, inside "Bla").
	hov, err := h.TextDocumentHover(nil, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: protocol.DocumentUri(modelURI)},
		Position:     protocol.Position{Line: 4, Character: 23},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if hov == nil {
		t.Fatalf("hover on Base/Bla returned nil (cross-file symbol not resolved)")
	}
	if v := hov.Contents.(protocol.MarkupContent).Value; !strings.Contains(v, "Base/Bla") {
		t.Errorf("hover value = %q, want it to mention Base/Bla", v)
	}

	// Definition on Base/Entity in the IMPLEMENTS clause (line 3).
	def, _ := h.TextDocumentDefinition(nil, &protocol.DefinitionParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: protocol.DocumentUri(modelURI)},
		Position:     protocol.Position{Line: 3, Character: 27},
	}})
	if locs, _ := def.([]protocol.Location); len(locs) == 0 {
		t.Errorf("definition on Base/Entity returned no locations")
	}
}
