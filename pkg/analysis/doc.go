package analysis

import (
	"strings"

	"github.com/sjsone/bhaus-util/pkg/ast"
)

// DocComment returns the doc comment for a symbol. This is the contiguous
// block of standalone "# ..." comment lines directly above the symbol's
// declaration, with no blank line between the block and the declaration.
//
// The returned text lists the comments in source order, joined by newlines.
// Each comment already has its "#" and surrounding whitespace stripped, in
// ast.Comment.Text.
//
// A trailing comment that shares its line with code (e.g. "PUBLIC id:
// Integer # note") is never a doc comment. Only lines whose first
// non-whitespace character is "#" qualify. Returns "" when the symbol has no
// doc comment.
//
// source is the full text of the file that declares sym (sym.URI); file is its
// parsed AST. Both must describe the same document revision.
func DocComment(file *ast.File, source string, sym Symbol) string {
	if file == nil || sym.Decl == nil || len(file.Comments) == 0 {
		return ""
	}

	// Index parsed comments by the line they start on. A comment runs to the
	// end of its line, so at most one comment begins on any given line.
	byLine := make(map[uint32]*ast.Comment, len(file.Comments))
	for _, c := range file.Comments {
		byLine[c.Pos().Line] = c
	}

	lines := strings.Split(source, "\n")
	declLine := sym.Decl.Pos().Line

	// Walk upward from the line above the declaration. Collect a contiguous
	// run of standalone comment lines. Stop at the first blank line, code
	// line or trailing comment.
	var block []string
	for l := int(declLine) - 1; l >= 0; l-- {
		c, ok := byLine[uint32(l)]
		if !ok || !standaloneComment(lines, l) {
			break
		}
		block = append(block, c.Text)
	}
	if len(block) == 0 {
		return ""
	}

	// block was gathered bottom-up; reverse it into source order.
	for i, j := 0, len(block)-1; i < j; i, j = i+1, j-1 {
		block[i], block[j] = block[j], block[i]
	}
	return strings.Join(block, "\n")
}

// standaloneComment reports whether line l (0-based) exists. It also checks
// that the line holds only a comment, i.e. its first non-whitespace
// character is "#". This distinguishes a doc comment from a trailing comment
// that shares a line with code.
func standaloneComment(lines []string, l int) bool {
	if l < 0 || l >= len(lines) {
		return false
	}
	return strings.HasPrefix(strings.TrimSpace(lines[l]), "#")
}
