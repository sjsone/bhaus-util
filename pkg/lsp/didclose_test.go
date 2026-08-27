package lsp

import (
	"os"
	"path/filepath"
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

func didOpen(t *testing.T, h *Handler, uri, text string) {
	t.Helper()
	if err := h.TextDocumentDidOpen(nil, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{URI: protocol.DocumentUri(uri), Text: text},
	}); err != nil {
		t.Fatalf("didOpen %s: %v", uri, err)
	}
}

func didClose(t *testing.T, h *Handler, uri string) {
	t.Helper()
	if err := h.TextDocumentDidClose(nil, &protocol.DidCloseTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: protocol.DocumentUri(uri)},
	}); err != nil {
		t.Fatalf("didClose %s: %v", uri, err)
	}
}

// TestDidCloseReleasesDocument is the leak guard. Without a didClose handler,
// the server kept the content, AST and tree-sitter tree of every opened file
// for the lifetime of the session.
func TestDidCloseReleasesDocument(t *testing.T) {
	h := NewHandler()
	kept := "file:///kept.bhaus"
	closed := "file:///closed.bhaus"

	didOpen(t, h, kept, "VERSION 0.1\nSTRUCT Kept:\n    PUBLIC id: Integer\n")
	didOpen(t, h, closed, "VERSION 0.1\nSTRUCT Closed:\n    PUBLIC id: Integer\n")

	didClose(t, h, closed)

	if _, ok := h.Documents[closed]; ok {
		t.Error("Documents still holds the closed file")
	}
	if _, ok := h.Files[closed]; ok {
		t.Error("Files still holds the closed file's AST")
	}
	if got := len(h.Table.SymbolsForURI(closed)); got != 0 {
		t.Errorf("closed file still contributes %d symbols to the table", got)
	}

	// The still-open document must be untouched.
	if _, ok := h.Files[kept]; !ok {
		t.Error("closing one document dropped another")
	}
	if got := len(h.Table.SymbolsForURI(kept)); got == 0 {
		t.Error("open document lost its symbols after an unrelated close")
	}
}

// TestDidCloseIsIdempotent covers two cases: a close for a file the server
// never had and a double close. Clients send both cases.
func TestDidCloseIsIdempotent(t *testing.T) {
	h := NewHandler()
	uri := "file:///m.bhaus"
	didOpen(t, h, uri, "VERSION 0.1\nSTRUCT S:\n    PUBLIC id: Integer\n")

	didClose(t, h, uri)
	didClose(t, h, uri)
	didClose(t, h, "file:///never-opened.bhaus")
}

// TestDidCloseKeepsIncludeTargetResolvable checks a correctness constraint
// beyond a simple delete. base.bhaus is loaded as an INCLUDE target of
// model.bhaus. Opening and then closing base.bhaus must not break go-to-
// definition in model.bhaus. model.bhaus stays open.
func TestDidCloseKeepsIncludeTargetResolvable(t *testing.T) {
	dir := t.TempDir()
	base := "VERSION 0.1\nPROTOCOL Base/Entity:\n    PUBLIC raw: String\n"
	model := "VERSION 0.1\nINCLUDE *\n\nCLASS Model IMPLEMENTS Base/Entity:\n    PUBLIC id: Integer\n"

	basePath := filepath.Join(dir, "base.bhaus")
	modelPath := filepath.Join(dir, "model.bhaus")
	if err := os.WriteFile(basePath, []byte(base), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(modelPath, []byte(model), 0o644); err != nil {
		t.Fatal(err)
	}
	baseURI, modelURI := "file://"+basePath, "file://"+modelPath

	h := NewHandler()
	didOpen(t, h, modelURI, model) // pulls base.bhaus in via INCLUDE
	didOpen(t, h, baseURI, base)   // user then opens base.bhaus directly

	didClose(t, h, baseURI) // ...and closes it again

	// model.bhaus is still open. Base/Entity must still resolve.
	def, err := h.TextDocumentDefinition(nil, &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: protocol.DocumentUri(modelURI)},
			Position:     protocol.Position{Line: 3, Character: 27},
		},
	})
	if err != nil {
		t.Fatalf("definition: %v", err)
	}
	if locs, _ := def.([]protocol.Location); len(locs) == 0 {
		t.Fatal("Base/Entity no longer resolves after its file was closed")
	}
}
