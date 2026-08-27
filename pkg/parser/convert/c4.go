package convert

import (
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/sjsone/bhaus-util/pkg/ast"
)

// systemDecl converts a SYSTEM declaration.
func (c *Converter) systemDecl(n *tree_sitter.Node) *ast.SystemDecl {
	ident := findChildByKind(n, "identifier")
	if ident == nil {
		return nil
	}
	d := &ast.SystemDecl{
		Base:        c.spanOf(n),
		Name:        c.ident(ident),
		Description: c.description(n),
		Containers:  []*ast.ContainerDecl{},
		Connections: []*ast.ConnectionShorthand{},
	}
	for i := uint(0); i < n.ChildCount(); i++ {
		child := n.Child(i)
		switch child.Kind() {
		case "container_declaration":
			if ct := c.containerDecl(child); ct != nil {
				d.Containers = append(d.Containers, ct)
			}
		case "connection_shorthand":
			if cs := c.connectionShorthand(child); cs != nil {
				d.Connections = append(d.Connections, cs)
			}
		}
	}
	return d
}

// containerDecl converts a CONTAINER declaration.
func (c *Converter) containerDecl(n *tree_sitter.Node) *ast.ContainerDecl {
	ident := findChildByKind(n, "identifier")
	if ident == nil {
		return nil
	}
	d := &ast.ContainerDecl{
		Base:        c.spanOf(n),
		Name:        c.ident(ident),
		Description: c.description(n),
		Components:  []*ast.ComponentDecl{},
		Connections: []*ast.ConnectionShorthand{},
	}
	for i := uint(0); i < n.ChildCount(); i++ {
		child := n.Child(i)
		switch child.Kind() {
		case "component_declaration":
			if cp := c.componentDecl(child); cp != nil {
				d.Components = append(d.Components, cp)
			}
		case "connection_shorthand":
			if cs := c.connectionShorthand(child); cs != nil {
				d.Connections = append(d.Connections, cs)
			}
		}
	}
	return d
}

// componentDecl converts a COMPONENT declaration.
func (c *Converter) componentDecl(n *tree_sitter.Node) *ast.ComponentDecl {
	ident := findChildByKind(n, "identifier")
	if ident == nil {
		return nil
	}
	d := &ast.ComponentDecl{
		Base:        c.spanOf(n),
		Name:        c.ident(ident),
		Description: c.description(n),
		Connections: []*ast.ConnectionShorthand{},
	}
	for i := uint(0); i < n.ChildCount(); i++ {
		child := n.Child(i)
		if child.Kind() == "connection_shorthand" {
			if cs := c.connectionShorthand(child); cs != nil {
				d.Connections = append(d.Connections, cs)
			}
		}
	}
	return d
}

// connectionDecl converts "CONNECTION A.B -> C.D".
func (c *Converter) connectionDecl(n *tree_sitter.Node) *ast.ConnectionDecl {
	var source, target *tree_sitter.Node
	arrow := ""
	for i := uint(0); i < n.ChildCount(); i++ {
		child := n.Child(i)
		switch child.Kind() {
		case "c4_path":
			if source == nil {
				source = child
			} else if target == nil {
				target = child
			}
		case "connection_arrow":
			arrow = c.text(child)
		}
	}
	if source == nil || target == nil || arrow == "" {
		return nil
	}
	return &ast.ConnectionDecl{
		Base:   c.spanOf(n),
		Source: c.c4Path(source),
		Arrow:  arrowFromString(arrow),
		Target: c.c4Path(target),
	}
}

// connectionShorthand converts "CONNECTION => A.B" inside a system,
// container or component body.
func (c *Converter) connectionShorthand(n *tree_sitter.Node) *ast.ConnectionShorthand {
	target := findChildByKind(n, "c4_path")
	if target == nil {
		return nil
	}
	return &ast.ConnectionShorthand{
		Base:   c.spanOf(n),
		Target: c.c4Path(target),
	}
}

// c4Path converts a dot-separated C4 path ("MailSystem.MTA").
func (c *Converter) c4Path(n *tree_sitter.Node) *ast.C4Path {
	path := &ast.C4Path{Base: c.spanOf(n), Segments: []*ast.Ident{}}
	for i := uint(0); i < n.ChildCount(); i++ {
		child := n.Child(i)
		if child.Kind() == "identifier" {
			path.Segments = append(path.Segments, c.ident(child))
		}
	}
	return path
}

// arrowFromString maps a connection arrow's source text to its AST value.
func arrowFromString(s string) ast.ConnectionArrow {
	switch s {
	case "<->":
		return ast.ArrowBidirectional
	case "=>":
		return ast.ArrowShorthand
	default:
		return ast.ArrowUnidirectional
	}
}

// description extracts the optional quoted label of a C4 declaration
// (for example, "SYSTEM MailSystem \"Mail Backend\":") from its
// c4_description node. The node text includes the surrounding quotes. Its
// content cannot itself contain a quote, because the grammar defines it as
// /[^"]+/. Trimming the quotes therefore yields the label. description
// returns "" when there is no description.
func (c *Converter) description(n *tree_sitter.Node) string {
	d := findChildByKind(n, "c4_description")
	if d == nil {
		return ""
	}
	return strings.Trim(c.text(d), "\"")
}
