package lsp

import (
	"testing"

	"github.com/sjsone/bhaus-util/pkg/analysis"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// didChangeWhole simulates a full-sync textDocument/didChange the way glsp
// delivers it: a single TextDocumentContentChangeEventWhole value.
func (h *Handler) didChangeWhole(t *testing.T, uri, text string) {
	t.Helper()
	err := h.TextDocumentDidChange(nil, &protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: protocol.DocumentUri(uri)},
		},
		ContentChanges: []any{
			protocol.TextDocumentContentChangeEventWhole{Text: text},
		},
	})
	if err != nil {
		t.Fatalf("didChange: %v", err)
	}
}

// TestDidChangeReparsesAndResolvesDefinition is the end-to-end guard for the
// reported bug: after an edit, go-to-definition must resolve at the new cursor
// position. It exercises the real content-change extraction path.
func TestDidChangeReparsesAndResolvesDefinition(t *testing.T) {
	h := NewHandler()
	uri := "file:///m.bhaus"

	v1 := "STRUCT User:\n    PUBLIC id: Integer\nSTRUCT Account:\n    PUBLIC owner: User\n"
	if err := h.TextDocumentDidOpen(nil, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{URI: protocol.DocumentUri(uri), Text: v1},
	}); err != nil {
		t.Fatalf("didOpen: %v", err)
	}

	// Edit the file: prepend a comment line. This shifts the whole file down by one line.
	v2 := "# a new comment line\n" + v1
	h.didChangeWhole(t, uri, v2)

	// The Files/Documents maps must reflect v2, not the opened v1.
	if h.Documents[uri] != v2 {
		t.Fatalf("Documents not updated after didChange")
	}

	// cmd+click on the "User" reference (now on line 4) resolves to the decl.
	res, err := h.TextDocumentDefinition(nil, &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: protocol.DocumentUri(uri)},
			Position:     protocol.Position{Line: 4, Character: 19},
		},
	})
	if err != nil {
		t.Fatalf("definition: %v", err)
	}
	locs, _ := res.([]protocol.Location)
	if len(locs) == 0 {
		t.Fatalf("no definition resolved after edit (the reported bug)")
	}
	if locs[0].Range.Start.Line != 1 {
		t.Errorf("definition at line %d, want 1 (User decl in v2)", locs[0].Range.Start.Line)
	}

	// The test clears the file. This drops its symbols (guard-removal behavior).
	h.didChangeWhole(t, uri, "")
	if got := len(h.Table.SymbolsForURI(uri)); got != 0 {
		t.Errorf("expected 0 symbols after clearing, got %d", got)
	}
	_ = analysis.Kind(0)
}
