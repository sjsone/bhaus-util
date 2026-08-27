package lsp

import (
	"testing"

	"github.com/sjsone/bhaus-util/pkg/analysis"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestTextDocumentCompletion(t *testing.T) {
	handler := NewHandler()
	uri := protocol.DocumentUri("file:///test/example.bhaus")

	// Layout:
	// Line 0: CLASS TestClass:
	// Line 1:     PUBLIC test(): String
	// Line 2:     PRIVATE count: Integer
	// Line 3: (empty, inside CLASS body)
	// Line 4: STRUCT TestStruct:
	// Line 5: (empty, inside STRUCT body)
	// Line 6: PROTOCOL TestProtocol:
	// Line 7:     PUBLIC handle(Event)
	testContent := `CLASS TestClass:
    PUBLIC test(): String
    PRIVATE count: Integer

STRUCT TestStruct:

PROTOCOL TestProtocol:
    PUBLIC handle(Event)
`

	handler.Documents[string(uri)] = testContent
	file, err := handler.Cache.Parse(string(uri), testContent)
	if err != nil {
		t.Fatalf("failed to parse test content: %v", err)
	}
	handler.Files[string(uri)] = file
	handler.Table = analysis.BuildSymbolTable(handler.Files)

	tests := []struct {
		name         string
		line         uint32
		character    uint32
		expectPrefix string
		expectLabel  string
	}{
		{
			name:         "CL prefix at start should show CLASS",
			line:         0,
			character:    2, // "CL" in "CLASS"
			expectPrefix: "CL",
			expectLabel:  "CLASS",
		},
		{
			name:         "ST prefix should show STRUCT",
			line:         4,
			character:    2, // "ST" in "STRUCT"
			expectPrefix: "ST",
			expectLabel:  "STRUCT",
		},
		{
			name:         "Test prefix should find TestClass symbol",
			line:         0,
			character:    11, // "TestClass" in "CLASS TestClass"
			expectPrefix: "TestClass",
			expectLabel:  "TestClass",
		},
		{
			name:         "empty line inside class body shows member-level completions",
			line:         3,
			character:    0,
			expectPrefix: "",
			expectLabel:  "PUBLIC",
		},
		{
			name:         "indented line with P prefix",
			line:         1,
			character:    5, // "P" after indentation in "PUBLIC"
			expectPrefix: "P",
			expectLabel:  "PUBLIC",
		},
		{
			name:         "indented PU should show PUBLIC with indent preserved",
			line:         1,
			character:    6, // "U" in "PUBLIC"
			expectPrefix: "PU",
			expectLabel:  "PUBLIC",
		},
		{
			name:         "PRO prefix at top level should find PROTOCOL",
			line:         6,
			character:    3, // "PRO" in "PROTOCOL"
			expectPrefix: "PRO",
			expectLabel:  "PROTOCOL",
		},
		{
			name:         "PR prefix inside class shows PRIVATE not PROTOCOL",
			line:         2,
			character:    6, // "PR" in "PRIVATE". The cursor sits after 'R'.
			expectPrefix: "PR",
			expectLabel:  "PRIVATE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := &protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: tt.line, Character: tt.character},
				},
				Context: &protocol.CompletionContext{
					TriggerKind: protocol.CompletionTriggerKindInvoked,
				},
			}

			result, err := handler.TextDocumentCompletion(nil, params)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var items []protocol.CompletionItem
			if list, ok := result.(protocol.CompletionList); ok {
				items = list.Items
			} else if itemList, ok := result.([]protocol.CompletionItem); ok {
				items = itemList
			}

			if tt.expectLabel != "" {
				found := false
				for _, item := range items {
					if item.Label == tt.expectLabel {
						found = true

						// Verify indentation preservation.
						if tt.name == "indented PU should show PUBLIC with indent preserved" && item.InsertText != nil {
							insertText := *item.InsertText
							if insertText != "    PUBLIC" {
								t.Errorf("expected insert text '    PUBLIC' with indentation, got %q", insertText)
							} else {
								t.Logf("✓ Indentation preserved correctly: %q", insertText)
							}
						}
						break
					}
				}
				if !found {
					t.Errorf("expected to find completion with label %q for prefix %q, got labels: %v",
						tt.expectLabel, tt.expectPrefix, getLabels(items))
				}
			}

			t.Logf("Test '%s': prefix=%q, got %d completions: %v",
				tt.name, tt.expectPrefix, len(items), getLabels(items))
		})
	}
}

// TestCompletionAfterOverride verifies that after OVERRIDE the completions
// list only visibility keywords. It also verifies that PROTOCOL (a top-level
// keyword) does not appear inside a container body.
func TestCompletionAfterOverride(t *testing.T) {
	handler := NewHandler()
	uri := protocol.DocumentUri("file:///test/override.bhaus")

	// Line 0: CLASS MyService:
	// Line 1:     OVERRIDE PR          ← cursor at end of "PR"
	// Line 2:     PUBLIC doWork()
	content := `CLASS MyService:
    OVERRIDE PR
    PUBLIC doWork(): Boolean
`

	handler.Documents[string(uri)] = content
	file, err := handler.Cache.Parse(string(uri), content)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	handler.Files[string(uri)] = file
	handler.Table = analysis.BuildSymbolTable(handler.Files)

	// The cursor is at the end of "OVERRIDE PR", past 'R' in "PR".
	params := &protocol.CompletionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 1, Character: 15}, // after "    OVERRIDE PR" (col is length of line)
		},
		Context: &protocol.CompletionContext{
			TriggerKind: protocol.CompletionTriggerKindInvoked,
		},
	}

	result2, err := handler.TextDocumentCompletion(nil, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var items []protocol.CompletionItem
	if list, ok := result2.(protocol.CompletionList); ok {
		items = list.Items
	} else if itemList, ok := result2.([]protocol.CompletionItem); ok {
		items = itemList
	}

	labels := getLabels(items)
	t.Logf("After OVERRIDE PR: got completions: %v", labels)

	// Should include PRIVATE and PROTECTED (visibility matching "PR").
	hasPrivate := false
	hasProtected := false
	for _, l := range labels {
		if l == "PRIVATE" {
			hasPrivate = true
		}
		if l == "PROTECTED" {
			hasProtected = true
		}
	}
	if !hasPrivate {
		t.Errorf("expected PRIVATE in completions after OVERRIDE PR, got: %v", labels)
	}
	if !hasProtected {
		t.Errorf("expected PROTECTED in completions after OVERRIDE PR, got: %v", labels)
	}

	// Must NOT include PROTOCOL (top-level keyword inside container body).
	for _, l := range labels {
		if l == "PROTOCOL" {
			t.Errorf("PROTOCOL should NOT appear inside container body, got: %v", labels)
		}
	}
	// Must NOT include PUBLIC (doesn't match "PR" prefix).
	for _, l := range labels {
		if l == "PUBLIC" {
			t.Errorf("PUBLIC should NOT appear when prefix is 'PR', got: %v", labels)
		}
	}
}

func getLabels(items []protocol.CompletionItem) []string {
	labels := make([]string, len(items))
	for i, item := range items {
		labels[i] = item.Label
	}
	return labels
}

func TestCompletionSuppressedInFunctionalIntent(t *testing.T) {
	handler := NewHandler()
	uri := protocol.DocumentUri("file:///test/intent-complete.bhaus")

	// Line 2 is a functional intent; line 3 is a normal member line.
	content := `PROTOCOL P:
    PUBLIC toLowerCase(String): String
        > walk over every char and
    PUBLIC other(): String
`
	handler.Documents[string(uri)] = content
	file, err := handler.Cache.Parse(string(uri), content)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	handler.Files[string(uri)] = file
	handler.Table = analysis.BuildSymbolTable(handler.Files)

	complete := func(line, char uint32) int {
		res, err := handler.TextDocumentCompletion(nil, &protocol.CompletionParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     protocol.Position{Line: line, Character: char},
			},
		})
		if err != nil {
			t.Fatalf("completion: %v", err)
		}
		items, ok := res.([]protocol.CompletionItem)
		if !ok {
			t.Fatalf("unexpected completion result type %T", res)
		}
		return len(items)
	}

	// Cursor at the end of the intent prose → no completions.
	if n := complete(2, 34); n != 0 {
		t.Errorf("expected 0 completions inside functional intent, got %d", n)
	}
	// Sanity: a real member line still yields completions.
	if n := complete(3, 10); n == 0 {
		t.Error("expected completions on a normal member line, got 0")
	}
}
