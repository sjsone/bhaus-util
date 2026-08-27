package ast

import "strings"

// Pos is a 0-based position in a source document.
// Column is a byte offset on the line (matching tree-sitter convention).
type Pos struct {
	Line   uint32
	Column uint32
	Offset uint32 // byte offset from start of document
}

// Span is a half-open range [Start, End).
type Span struct {
	Start Pos
	End   Pos
}

// All AST node types implement Node.
type Node interface {
	Pos() Pos
	End() Pos
}

// Every concrete AST node embeds Base. Base carries the node's Span.
type Base struct {
	span Span
}

func (b *Base) Pos() Pos   { return b.span.Start }
func (b *Base) End() Pos   { return b.span.End }
func (b *Base) Span() Span { return b.span }

// At creates a Base spanning [start, end).
func At(start, end Pos) Base {
	return Base{span: Span{Start: start, End: end}}
}

// SpanOf returns the span of a node.
func SpanOf(n Node) Span {
	return Span{Start: n.Pos(), End: n.End()}
}

// Contains reports whether span contains pos (half-open on end row/col).
func Contains(sp Span, pos Pos) bool {
	if pos.Line < sp.Start.Line || pos.Line > sp.End.Line {
		return false
	}
	if pos.Line == sp.Start.Line && pos.Column < sp.Start.Column {
		return false
	}
	if pos.Line == sp.End.Line && pos.Column >= sp.End.Column {
		return false
	}
	return true
}

// Ident is a simple identifier token (name of a system, container, etc.).
type Ident struct {
	Base
	Name string
}

// QualifiedName is a slash-separated contextual name: "Domain/Entity/User".
type QualifiedName struct {
	Base
	Segments []*Ident
}

func (n *QualifiedName) String() string {
	if len(n.Segments) == 0 {
		return ""
	}
	parts := make([]string, len(n.Segments))
	for i, s := range n.Segments {
		parts[i] = s.Name
	}
	return strings.Join(parts, "/")
}

func (n *QualifiedName) Simple() string {
	if len(n.Segments) == 0 {
		return ""
	}
	return n.Segments[len(n.Segments)-1].Name
}

func (n *QualifiedName) IsPath() bool {
	return len(n.Segments) > 1
}

// C4Path is a dot-separated C4 path: "MailSystem.MTA".
type C4Path struct {
	Base
	Segments []*Ident
}

func (n *C4Path) String() string {
	if len(n.Segments) == 0 {
		return ""
	}
	parts := make([]string, len(n.Segments))
	for i, s := range n.Segments {
		parts[i] = s.Name
	}
	return strings.Join(parts, ".")
}
