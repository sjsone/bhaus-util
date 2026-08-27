// Package convert translates a tree-sitter concrete syntax tree (CST) into
// the typed AST in pkg/ast. Conversion is a pure function of the CST and the
// source text. Semantic analysis happens later, in pkg/analysis.
//
// There is one converter method per grammar rule, grouped by rule kind across
// files: declarations.go (file-level and structural declarations), members.go
// (STRUCT/CLASS/PROTOCOL members), type.go (type expressions), c4.go (C4
// model), errors.go (syntax-error reporting) and names.go (contextual names).
package convert

import (
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/sjsone/bhaus-util/pkg/ast"
)

// Converter holds state during a single CST→AST conversion.
type Converter struct {
	uri     string
	content string   // original source text
	src     []byte   // source bytes for Utf8Text
	lineOff []uint32 // byte offset of each line's start, for Pos.Offset
}

// New creates a Converter for one document.
func New(uri, content string, src []byte) *Converter {
	c := &Converter{uri: uri, content: content, src: src}
	c.buildLineOffsets()
	return c
}

func (c *Converter) buildLineOffsets() {
	c.lineOff = []uint32{0}
	for i, b := range c.src {
		if b == '\n' {
			c.lineOff = append(c.lineOff, uint32(i+1))
		}
	}
}

// findChildByKind returns the first direct child of node with the given kind.
func findChildByKind(node *tree_sitter.Node, kind string) *tree_sitter.Node {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == kind {
			return child
		}
	}
	return nil
}

// isMethodMember checks if a class/struct/protocol member is a method (has parentheses).
func isMethodMember(member *tree_sitter.Node) bool {
	for i := uint(0); i < member.ChildCount(); i++ {
		child := member.Child(i)
		if child.Kind() == "(" {
			return true
		}
	}
	return false
}

// pos converts a tree-sitter Point to ast.Pos.
func (c *Converter) pos(p tree_sitter.Point) ast.Pos {
	var offset uint32
	if int(p.Row) < len(c.lineOff) {
		offset = c.lineOff[p.Row] + uint32(p.Column)
	}
	return ast.Pos{
		Line:   uint32(p.Row),
		Column: uint32(p.Column),
		Offset: offset,
	}
}

// text extracts source text for a node.
func (c *Converter) text(n *tree_sitter.Node) string {
	return n.Utf8Text(c.src)
}

// spanOf converts a node's range into an ast.Base.
func (c *Converter) spanOf(n *tree_sitter.Node) ast.Base {
	return ast.At(c.pos(n.StartPosition()), c.pos(n.EndPosition()))
}

// ident builds an ast.Ident from an identifier node.
func (c *Converter) ident(n *tree_sitter.Node) *ast.Ident {
	if n == nil {
		return nil
	}
	return &ast.Ident{Base: c.spanOf(n), Name: c.text(n)}
}

// File is the entry point for CST→AST conversion. It walks the root
// tree-sitter node. It produces the *ast.File for the document.
//
// Top-level children are dispatched by kind to the individual declaration
// converters. File collects comments and ERROR/MISSING nodes recursively
// over the whole tree. This captures them wherever they nest.
func (c *Converter) File(root *tree_sitter.Node) *ast.File {
	f := &ast.File{
		URI:      c.uri,
		Decls:    []ast.Decl{},
		Comments: []*ast.Comment{},
		Errors:   []ast.SyntaxError{},
	}
	f.Base = c.spanOf(root)

	c.collectSyntaxErrors(root, &f.Errors)
	c.collectComments(root, &f.Comments)

	for i := uint(0); i < root.ChildCount(); i++ {
		child := root.Child(i)
		var decl ast.Decl
		switch child.Kind() {
		case "version_declaration":
			decl = c.versionDecl(child)
		case "include_declaration":
			decl = c.includeDecl(child)
		case "extern":
			decl = c.externDecl(child)
		case "class_declaration":
			decl = c.classDecl(child)
		case "struct_declaration":
			decl = c.structDecl(child)
		case "protocol_declaration":
			decl = c.protocolDecl(child)
		case "function_declaration":
			decl = c.functionDecl(child)
		case "system_declaration":
			decl = c.systemDecl(child)
		case "container_declaration":
			decl = c.containerDecl(child)
		case "component_declaration":
			decl = c.componentDecl(child)
		case "connection_declaration":
			decl = c.connectionDecl(child)
		}
		if decl != nil {
			f.Decls = append(f.Decls, decl)
		}
	}
	return f
}

// collectSyntaxErrors recursively records ERROR and MISSING nodes as
// ast.SyntaxError entries, in source order (pre-order traversal).
func (c *Converter) collectSyntaxErrors(n *tree_sitter.Node, out *[]ast.SyntaxError) {
	if n.IsError() || n.IsMissing() {
		*out = append(*out, c.syntaxError(n))
	}
	for i := uint(0); i < n.ChildCount(); i++ {
		c.collectSyntaxErrors(n.Child(i), out)
	}
}

// collectComments recursively records every "# ..." comment node in source
// order (pre-order traversal), regardless of nesting depth. A comment may
// appear at the top level or nested inside a STRUCT/CLASS/PROTOCOL body or
// inside a C4 SYSTEM/CONTAINER/COMPONENT block. collectComments gathers all
// of them into File.Comments. Comment nodes have no comment descendants.
// Recursion therefore stops at each one.
func (c *Converter) collectComments(n *tree_sitter.Node, out *[]*ast.Comment) {
	if n.Kind() == "comment" {
		*out = append(*out, c.comment(n))
		return
	}
	for i := uint(0); i < n.ChildCount(); i++ {
		c.collectComments(n.Child(i), out)
	}
}
