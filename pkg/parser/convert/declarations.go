package convert

import (
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/sjsone/bhaus-util/pkg/ast"
)

// ── File-level declarations ──────────────────────────────────────────────────

// versionDecl converts "VERSION 0.1".
func (c *Converter) versionDecl(n *tree_sitter.Node) *ast.VersionDecl {
	v := findChildByKind(n, "version")
	if v == nil {
		return nil
	}
	return &ast.VersionDecl{Base: c.spanOf(n), Version: c.text(v)}
}

// includeDecl converts "INCLUDE *.bhaus".
func (c *Converter) includeDecl(n *tree_sitter.Node) *ast.IncludeDecl {
	g := findChildByKind(n, "glob_pattern")
	if g == nil {
		return nil
	}
	return &ast.IncludeDecl{Base: c.spanOf(n), Pattern: c.text(g)}
}

// externDecl converts "EXTERN Domain/ExternalType".
func (c *Converter) externDecl(n *tree_sitter.Node) *ast.ExternDecl {
	cn := findChildByKind(n, "contextual_name")
	if cn == nil {
		return nil
	}
	return &ast.ExternDecl{Base: c.spanOf(n), Name: c.qualifiedName(cn)}
}

// comment converts a "# ..." comment, storing the text without the "#".
func (c *Converter) comment(n *tree_sitter.Node) *ast.Comment {
	text := strings.TrimPrefix(c.text(n), "#")
	return &ast.Comment{Base: c.spanOf(n), Text: strings.TrimSpace(text)}
}

// functionalIntent converts a "> ..." intent line, storing the text without
// the leading ">".
func (c *Converter) functionalIntent(n *tree_sitter.Node) *ast.FunctionalIntent {
	text := strings.TrimPrefix(c.text(n), ">")
	return &ast.FunctionalIntent{Base: c.spanOf(n), Text: strings.TrimSpace(text)}
}

// intents collects every functional_intent child of a member/function node,
// in source order.
func (c *Converter) intents(n *tree_sitter.Node) []*ast.FunctionalIntent {
	var out []*ast.FunctionalIntent
	for i := uint(0); i < n.ChildCount(); i++ {
		if child := n.Child(i); child.Kind() == "functional_intent" {
			out = append(out, c.functionalIntent(child))
		}
	}
	return out
}

// ── Structural declarations ──────────────────────────────────────────────────

// classDecl converts a CLASS declaration. The first contextual_name is the
// class name. EXTENDS takes a single contextual_name. IMPLEMENTS wraps a
// comma-separated implements_list. The implements_list's children are
// contextual_name nodes.
func (c *Converter) classDecl(n *tree_sitter.Node) *ast.ClassDecl {
	d := &ast.ClassDecl{
		Base:       c.spanOf(n),
		Implements: []*ast.QualifiedName{},
		Members:    []ast.ClassMember{},
	}
	pending := "" // "EXTENDS" or "IMPLEMENTS" seen since the last name
	for i := uint(0); i < n.ChildCount(); i++ {
		child := n.Child(i)
		switch child.Kind() {
		case "contextual_name":
			qn := c.qualifiedName(child)
			if d.Name == nil {
				d.Name = qn
				continue
			}
			switch pending {
			case "EXTENDS":
				if d.Extends == nil {
					d.Extends = qn
				}
			case "IMPLEMENTS":
				d.Implements = append(d.Implements, qn)
			}
		case "implements_list":
			for j := uint(0); j < child.ChildCount(); j++ {
				if cn := child.Child(j); cn.Kind() == "contextual_name" {
					d.Implements = append(d.Implements, c.qualifiedName(cn))
				}
			}
		case "EXTENDS":
			pending = "EXTENDS"
		case "IMPLEMENTS":
			pending = "IMPLEMENTS"
		case "class_member":
			if m := c.classMember(child); m != nil {
				d.Members = append(d.Members, m)
			}
		}
	}
	if d.Name == nil {
		return nil
	}
	return d
}

// structDecl converts a STRUCT declaration.
func (c *Converter) structDecl(n *tree_sitter.Node) *ast.StructDecl {
	name := findChildByKind(n, "contextual_name")
	if name == nil {
		return nil
	}
	d := &ast.StructDecl{
		Base:    c.spanOf(n),
		Name:    c.qualifiedName(name),
		Members: []ast.StructMember{},
	}
	for i := uint(0); i < n.ChildCount(); i++ {
		child := n.Child(i)
		if child.Kind() == "struct_member" {
			if m := c.structMember(child); m != nil {
				d.Members = append(d.Members, m)
			}
		}
	}
	return d
}

// protocolDecl converts a PROTOCOL declaration.
func (c *Converter) protocolDecl(n *tree_sitter.Node) *ast.ProtocolDecl {
	name := findChildByKind(n, "contextual_name")
	if name == nil {
		return nil
	}
	d := &ast.ProtocolDecl{
		Base:    c.spanOf(n),
		Name:    c.qualifiedName(name),
		Members: []ast.ProtocolMember{},
	}
	for i := uint(0); i < n.ChildCount(); i++ {
		child := n.Child(i)
		if child.Kind() == "protocol_member" {
			if m := c.protocolMember(child); m != nil {
				d.Members = append(d.Members, m)
			}
		}
	}
	return d
}

// functionDecl converts a top-level FUNCTION/FUNC declaration.
func (c *Converter) functionDecl(n *tree_sitter.Node) *ast.FunctionDecl {
	name := findChildByKind(n, "contextual_name")
	if name == nil {
		return nil
	}
	d := &ast.FunctionDecl{
		Base:   c.spanOf(n),
		Name:   c.qualifiedName(name),
		Params: []*ast.Parameter{},
	}
	for i := uint(0); i < n.ChildCount(); i++ {
		child := n.Child(i)
		switch child.Kind() {
		case "parameter":
			if p := c.parameter(child); p != nil {
				d.Params = append(d.Params, p)
			}
		case "type":
			d.Result = c.typeRef(child)
		}
	}
	d.Intents = c.intents(n)
	return d
}
