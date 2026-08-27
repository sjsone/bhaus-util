package convert

import (
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/sjsone/bhaus-util/pkg/ast"
)

// classMember converts a class_member node into a ClassMember.
func (c *Converter) classMember(n *tree_sitter.Node) ast.ClassMember {
	switch m := c.buildMember(n, true).(type) {
	case *ast.PropertyMember:
		return m
	case *ast.MethodMember:
		return m
	}
	return nil
}

// structMember converts a struct_member node into a StructMember.
func (c *Converter) structMember(n *tree_sitter.Node) ast.StructMember {
	switch m := c.buildMember(n, false).(type) {
	case *ast.PropertyMember:
		return m
	case *ast.MethodMember:
		return m
	}
	return nil
}

// protocolMember converts a protocol_member node into a ProtocolMember.
func (c *Converter) protocolMember(n *tree_sitter.Node) ast.ProtocolMember {
	switch m := c.buildMember(n, false).(type) {
	case *ast.PropertyMember:
		return m
	case *ast.MethodMember:
		return m
	}
	return nil
}

// keywordIdentifiers lists words that tree-sitter may emit as identifier
// tokens during error recovery. These words are keywords, not member names.
var keywordIdentifiers = map[string]bool{
	"FUNCTION":   true,
	"FUNC":       true,
	"OVERRIDE":   true,
	"CLASS":      true,
	"STRUCT":     true,
	"PROTOCOL":   true,
	"SYSTEM":     true,
	"CONTAINER":  true,
	"COMPONENT":  true,
	"CONNECTION": true,
	"PUBLIC":     true,
	"PRIVATE":    true,
	"PROTECTED":  true,
}

// buildMember converts a member node into a *ast.PropertyMember or a
// *ast.MethodMember. The presence of a "(" child distinguishes the two.
// buildMember returns nil if the member is malformed beyond recovery.
// allowOverride controls whether buildMember honors the OVERRIDE keyword.
// Only CLASS members allow OVERRIDE.
func (c *Converter) buildMember(n *tree_sitter.Node, allowOverride bool) any {
	visNode := findChildByKind(n, "visibility")
	if visNode == nil {
		return nil // no visibility: malformed member, reported as a syntax error
	}

	// Name is the first non-keyword identifier directly under the member.
	// Parameter identifiers do not qualify. They nest inside parameter
	// nodes, not directly under the member.
	var name *tree_sitter.Node
	for i := uint(0); i < n.ChildCount(); i++ {
		child := n.Child(i)
		if child.Kind() == "identifier" && !keywordIdentifiers[c.text(child)] {
			name = child
			break
		}
	}
	if name == nil {
		return nil
	}

	vis := ast.VisibilityPublic
	switch c.text(visNode) {
	case "PRIVATE":
		vis = ast.VisibilityPrivate
	case "PROTECTED":
		vis = ast.VisibilityProtected
	}

	if isMethodMember(n) {
		m := &ast.MethodMember{
			Base:   c.spanOf(n),
			Name:   c.ident(name),
			Vis:    vis,
			Params: []*ast.Parameter{},
		}
		if allowOverride {
			for i := uint(0); i < n.ChildCount(); i++ {
				if n.Child(i).Kind() == "OVERRIDE" {
					m.Override = true
					break
				}
			}
		}
		for i := uint(0); i < n.ChildCount(); i++ {
			child := n.Child(i)
			switch child.Kind() {
			case "parameter":
				if p := c.parameter(child); p != nil {
					m.Params = append(m.Params, p)
				}
			case "type":
				m.ReturnType = c.typeRef(child)
			}
		}
		m.Intents = c.intents(n)
		return m
	}

	prop := &ast.PropertyMember{
		Base: c.spanOf(n),
		Name: c.ident(name),
		Vis:  vis,
	}
	t := findChildByKind(n, "type")
	if t == nil {
		return nil // property without a type cannot be represented
	}
	prop.Type = c.typeRef(t)
	if prop.Type == nil {
		return nil
	}
	return prop
}

// parameter converts a parameter node. A parameter node is either an
// unnamed type ("Request") or a named parameter ("name: Type"). parameter
// returns nil if the type is missing.
func (c *Converter) parameter(n *tree_sitter.Node) *ast.Parameter {
	t := findChildByKind(n, "type")
	if t == nil {
		return nil
	}
	p := &ast.Parameter{
		Base: c.spanOf(n),
		Type: c.typeRef(t),
	}
	if ident := findChildByKind(n, "identifier"); ident != nil {
		p.Name = c.ident(ident)
	}
	return p
}
