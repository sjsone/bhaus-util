package lsp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

func openModelWithInclude(t *testing.T) (*Handler, string) {
	t.Helper()
	dir := t.TempDir()
	base := "VERSION 0.1\nPROTOCOL Base/Entity:\n    PUBLIC raw: String\nPROTOCOL Base/Bla:\n    PUBLIC rec: Base/Entity\n"
	model := "VERSION 0.1\nINCLUDE *\n\nCLASS Model IMPLEMENTS Base/Entity:\n    PUBLIC test: Base/Bla\n"
	os.WriteFile(filepath.Join(dir, "base.bhaus"), []byte(base), 0o644)
	mp := filepath.Join(dir, "model.bhaus")
	os.WriteFile(mp, []byte(model), 0o644)
	uri := "file://" + mp
	h := NewHandler()
	if err := h.TextDocumentDidOpen(nil, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{URI: protocol.DocumentUri(uri), Text: model},
	}); err != nil {
		t.Fatal(err)
	}
	return h, uri
}

func defAt(t *testing.T, h *Handler, uri string, line, col uint32) any {
	t.Helper()
	res, err := h.TextDocumentDefinition(nil, &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: protocol.DocumentUri(uri)},
			Position:     protocol.Position{Line: line, Character: col},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return res
}

// With linkSupport, definition returns LocationLink[] whose origin spans the
// whole "Base/Bla" (cols 17..25) from any position inside the name.
func TestDefinitionLinkOriginSpansQualifiedName(t *testing.T) {
	h, uri := openModelWithInclude(t)
	h.definitionLinkSupport = true

	for _, col := range []uint32{17, 20, 22, 24} {
		links, ok := defAt(t, h, uri, 4, col).([]protocol.LocationLink)
		if !ok || len(links) == 0 {
			t.Fatalf("col %d: expected LocationLink[], got %T", col, defAt(t, h, uri, 4, col))
		}
		o := links[0].OriginSelectionRange
		if o == nil || o.Start.Line != 4 || o.Start.Character != 17 || o.End.Character != 25 {
			t.Errorf("col %d: origin = %v, want 4:17..4:25", col, o)
		}
		if !strings.HasSuffix(string(links[0].TargetURI), "base.bhaus") {
			t.Errorf("col %d: target uri = %s, want base.bhaus", col, links[0].TargetURI)
		}
		// Target selection range is the whole "Base/Bla" at the declaration (3:9..3:17).
		if links[0].TargetSelectionRange.Start.Line != 3 || links[0].TargetSelectionRange.Start.Character != 9 {
			t.Errorf("col %d: target range start = %v, want 3:9", col, links[0].TargetSelectionRange.Start)
		}
	}
}

// Without linkSupport, definition falls back to Location[].
func TestDefinitionFallsBackToLocation(t *testing.T) {
	h, uri := openModelWithInclude(t)
	h.definitionLinkSupport = false

	locs, ok := defAt(t, h, uri, 4, 23).([]protocol.Location)
	if !ok || len(locs) == 0 {
		t.Fatalf("expected Location[], got %T", defAt(t, h, uri, 4, 23))
	}
	if !strings.HasSuffix(string(locs[0].URI), "base.bhaus") {
		t.Errorf("target uri = %s, want base.bhaus", locs[0].URI)
	}
}
