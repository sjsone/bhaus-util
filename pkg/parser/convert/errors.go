package convert

import (
	"fmt"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/sjsone/bhaus-util/pkg/ast"
)

// syntaxError converts an ERROR or MISSING node into an ast.SyntaxError with a
// human-readable message. It reports the specific unexpected token or the
// missing token, plus the surrounding context. It does not dump the raw
// error text. tree-sitter reports parse state 0 for these nodes. The
// lookahead iterator therefore cannot narrow down an "expected" set. The
// structural context (the error's first child and its parent) gives the
// reliable signal instead.
func (c *Converter) syntaxError(n *tree_sitter.Node) ast.SyntaxError {
	if n.IsMissing() {
		return c.missingError(n)
	}
	return c.unexpectedError(n)
}

// connectionShorthandHint returns the enclosing connection_declaration and an
// explanatory message. It does this when n is the missing identifier of a
// top-level shorthand CONNECTION (=> target). At the top level, no enclosing
// SYSTEM, CONTAINER or COMPONENT exists to supply the implicit source. The
// grammar therefore recovers a connection_declaration with an empty leading
// c4_path. connectionShorthandHint returns (nil, "") otherwise. A malformed
// nested shorthand parses as connection_shorthand instead. This code never
// sees that case.
func (c *Converter) connectionShorthandHint(n *tree_sitter.Node) (*tree_sitter.Node, string) {
	path := n.Parent()
	if path == nil || path.Kind() != "c4_path" || path.ChildCount() != 1 {
		return nil, ""
	}
	decl := path.Parent()
	if decl == nil || decl.Kind() != "connection_declaration" {
		return nil, ""
	}
	arrow := findChildByKind(decl, "connection_arrow")
	if arrow == nil || c.text(arrow) != "=>" {
		return nil, ""
	}
	// The empty path must end before the arrow. It is the missing source, not
	// a missing target. A valid CONNECTION A => B has a source path with real
	// segments and no MISSING node. This code therefore never sees that case.
	pe, as := path.EndPosition(), arrow.StartPosition()
	if pe.Row > as.Row || (pe.Row == as.Row && pe.Column > as.Column) {
		return nil, ""
	}
	return decl, "shorthand CONNECTION (=>) needs an enclosing SYSTEM, CONTAINER or COMPONENT to supply the implicit source — use CONNECTION source -> target at the top level or move the line inside the element it connects from"
}

// missingError handles a MISSING node. This is a token that tree-sitter
// expected but did not find (for example, the ":" that closes a CLASS
// header). The node has zero width at the insertion point. A domain-specific
// hint replaces the generic message when the missing token is the empty
// source path of a top-level CONNECTION shorthand.
func (c *Converter) missingError(n *tree_sitter.Node) ast.SyntaxError {
	if decl, msg := c.connectionShorthandHint(n); msg != "" {
		base := c.spanOf(decl)
		return ast.SyntaxError{
			Span:     base.Span(),
			Severity: "Error",
			Message:  msg,
		}
	}
	msg := "expected " + describeExpected(n.Kind()) + describeContext(n.Parent())
	return ast.SyntaxError{
		Span:     ast.Span{Start: c.pos(n.StartPosition()), End: c.pos(n.EndPosition())},
		Severity: "Error",
		Message:  msg,
	}
}

// unexpectedError handles an ERROR node. This is input that tree-sitter
// could not fit into the grammar. The offending construct is the error's
// first child. The span points at that token. This keeps the squiggle
// precise instead of covering the whole error region. That region can span
// multiple lines.
func (c *Converter) unexpectedError(n *tree_sitter.Node) ast.SyntaxError {
	tok := n
	if n.ChildCount() > 0 {
		tok = n.Child(0)
	}
	span := ast.Span{Start: c.pos(tok.StartPosition()), End: c.pos(tok.EndPosition())}

	// Domain-specific hint: a functional intent is a valid token, just misplaced.
	if tok.Kind() == "functional_intent" {
		return ast.SyntaxError{
			Span:     span,
			Severity: "Error",
			Message: fmt.Sprintf(
				"unexpected functional intent %q — an intent line (\"> ...\") must be inside a function or method body",
				c.text(tok)),
		}
	}

	msg := "unexpected " + describeNode(tok, c.text(tok)) + describeContext(n.Parent())
	return ast.SyntaxError{Span: span, Severity: "Error", Message: msg}
}

// describeNode returns a friendly noun phrase for an unexpected token. Named
// nodes (identifier, visibility and so on) get a description. Anonymous
// literal tokens are punctuation and keywords, such as "(" or "PUBLIC".
// describeNode quotes these verbatim.
func describeNode(n *tree_sitter.Node, text string) string {
	if text == "" {
		return "input"
	}
	if !n.IsNamed() {
		return fmt.Sprintf("%q", text)
	}
	switch n.Kind() {
	case "identifier":
		return fmt.Sprintf("identifier %q", text)
	case "visibility":
		return fmt.Sprintf("visibility keyword %q", text)
	case "type", "simple_type", "contextual_name", "array_type", "union_type", "optional_type":
		return fmt.Sprintf("type %q", text)
	case "comment":
		return "comment"
	default:
		return fmt.Sprintf("%s %q", n.Kind(), text)
	}
}

// describeExpected returns a friendly name for a MISSING token kind.
// describeExpected quotes anonymous literals (":", ")"). It gives named
// kinds a description instead.
func describeExpected(kind string) string {
	switch kind {
	case "identifier":
		return "an identifier"
	case "type", "simple_type", "contextual_name":
		return "a type"
	case "visibility":
		return "a visibility keyword (PUBLIC, PRIVATE or PROTECTED)"
	default:
		return fmt.Sprintf("%q", kind)
	}
}

// describeContext returns a trailing phrase naming the construct that contains
// the error, e.g. " in a CLASS declaration". Returns "" when the parent gives no
// useful context.
func describeContext(parent *tree_sitter.Node) string {
	if parent == nil {
		return ""
	}
	switch parent.Kind() {
	case "source_file":
		return " at the top level"
	case "protocol_member", "struct_member", "class_member":
		return " in a member declaration"
	case "class_declaration":
		return " in a CLASS declaration"
	case "struct_declaration":
		return " in a STRUCT declaration"
	case "protocol_declaration":
		return " in a PROTOCOL declaration"
	case "function_declaration":
		return " in a FUNCTION declaration"
	case "system_declaration", "container_declaration", "component_declaration":
		return " in a C4 declaration"
	case "connection_declaration", "connection_shorthand":
		return " in a CONNECTION"
	case "parameter":
		return " in a parameter"
	case "type", "array_type", "union_type", "optional_type":
		return " in a type"
	default:
		return ""
	}
}
