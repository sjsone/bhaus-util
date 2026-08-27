package analysis

import (
	"testing"

	"github.com/sjsone/bhaus-util/pkg/ast"
)

// lineComment builds a parsed comment node starting on the given 0-based line.
func lineComment(line uint32, text string) *ast.Comment {
	start := ast.Pos{Line: line}
	end := ast.Pos{Line: line, Column: 100}
	return &ast.Comment{Base: ast.At(start, end), Text: text}
}

// declAt builds a throwaway declaration node. Its span starts on the given
// 0-based line. It stands in for the symbol being documented.
func declAt(line uint32) ast.Node {
	start := ast.Pos{Line: line}
	return &ast.StructDecl{Base: ast.At(start, ast.Pos{Line: line, Column: 20})}
}

func TestDocComment(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		comments []*ast.Comment
		declLine uint32
		want     string
	}{
		{
			name:     "single line above",
			source:   "# the user entity\nSTRUCT User:\n",
			comments: []*ast.Comment{lineComment(0, "the user entity")},
			declLine: 1,
			want:     "the user entity",
		},
		{
			name:     "contiguous block joined in source order",
			source:   "# line one\n# line two\nSTRUCT User:\n",
			comments: []*ast.Comment{lineComment(0, "line one"), lineComment(1, "line two")},
			declLine: 2,
			want:     "line one\nline two",
		},
		{
			name:     "blank line breaks the block",
			source:   "# far away\n\nSTRUCT User:\n",
			comments: []*ast.Comment{lineComment(0, "far away")},
			declLine: 2,
			want:     "",
		},
		{
			name:     "only the contiguous run is kept",
			source:   "# detached\n\n# attached\nSTRUCT User:\n",
			comments: []*ast.Comment{lineComment(0, "detached"), lineComment(2, "attached")},
			declLine: 3,
			want:     "attached",
		},
		{
			name:     "trailing comment on code line is not a doc comment",
			source:   "PUBLIC id: Integer # trailing\nSTRUCT Next:\n",
			comments: []*ast.Comment{lineComment(0, "trailing")},
			declLine: 1,
			want:     "",
		},
		{
			name:     "no comments at all",
			source:   "STRUCT User:\n",
			comments: nil,
			declLine: 0,
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := &ast.File{Comments: tt.comments}
			sym := Symbol{Decl: declAt(tt.declLine)}
			got := DocComment(file, tt.source, sym)
			if got != tt.want {
				t.Errorf("DocComment() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDocCommentNilSafety(t *testing.T) {
	sym := Symbol{Decl: declAt(1)}
	if got := DocComment(nil, "whatever", sym); got != "" {
		t.Errorf("expected empty for nil file, got %q", got)
	}
	file := &ast.File{Comments: []*ast.Comment{lineComment(0, "doc")}}
	if got := DocComment(file, "# doc\nSTRUCT U:\n", Symbol{}); got != "" {
		t.Errorf("expected empty for symbol with nil Decl, got %q", got)
	}
}
