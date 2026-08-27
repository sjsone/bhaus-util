package lsp

import (
	"testing"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// capturePublish returns a glsp.Context whose Notify records the diagnostics of
// the last textDocument/publishDiagnostics notification into *out.
func capturePublish(out *[]protocol.Diagnostic) *glsp.Context {
	return &glsp.Context{
		Notify: func(method string, params any) {
			if method != protocol.ServerTextDocumentPublishDiagnostics {
				return
			}
			if p, ok := params.(protocol.PublishDiagnosticsParams); ok {
				*out = p.Diagnostics
			}
		},
	}
}

func TestDidOpenPublishesUnresolvedRef(t *testing.T) {
	h := NewHandler()
	var got []protocol.Diagnostic
	ctx := capturePublish(&got)

	uri := "file:///m.bhaus"
	src := "VERSION 0.1\nSTRUCT S:\n    PUBLIC x: Missing/Type\n"
	if err := h.TextDocumentDidOpen(ctx, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{URI: protocol.DocumentUri(uri), Text: src},
	}); err != nil {
		t.Fatalf("didOpen: %v", err)
	}

	var found bool
	for _, d := range got {
		if d.Code != nil && d.Code.Value == "unresolved-ref" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected an unresolved-ref diagnostic, got %+v", got)
	}
}

func TestDidOpenPublishesUnknownTypeWarning(t *testing.T) {
	h := NewHandler()
	var got []protocol.Diagnostic
	ctx := capturePublish(&got)

	uri := "file:///m.bhaus"
	src := "VERSION 0.1\nSTRUCT S:\n    PUBLIC x: Unknown\n"
	if err := h.TextDocumentDidOpen(ctx, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{URI: protocol.DocumentUri(uri), Text: src},
	}); err != nil {
		t.Fatalf("didOpen: %v", err)
	}

	var found *protocol.Diagnostic
	for i, d := range got {
		if d.Code != nil && d.Code.Value == "unknown-type" {
			found = &got[i]
		}
	}
	if found == nil {
		t.Fatalf("expected an unknown-type diagnostic, got %+v", got)
	}
	if found.Severity == nil || *found.Severity != protocol.DiagnosticSeverityWarning {
		t.Fatalf("unknown-type severity = %v, want warning", found.Severity)
	}
}

func TestCleanFileClearsDiagnostics(t *testing.T) {
	h := NewHandler()
	var got []protocol.Diagnostic
	ctx := capturePublish(&got)

	uri := "file:///m.bhaus"
	// Start broken so there is something to clear.
	broken := "VERSION 0.1\nSTRUCT S:\n    PUBLIC x: Missing/Type\n"
	if err := h.TextDocumentDidOpen(ctx, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{URI: protocol.DocumentUri(uri), Text: broken},
	}); err != nil {
		t.Fatalf("didOpen: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("expected diagnostics for the broken file")
	}

	// Fix it. To clear diagnostics, the handler must publish an empty (but non-nil) array.
	clean := "VERSION 0.1\nSTRUCT S:\n    PUBLIC x: Integer\n"
	if err := h.TextDocumentDidChange(ctx, &protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: protocol.DocumentUri(uri)},
		},
		ContentChanges: []any{protocol.TextDocumentContentChangeEventWhole{Text: clean}},
	}); err != nil {
		t.Fatalf("didChange: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected diagnostics cleared, got %+v", got)
	}
}
