package convert

import (
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/sjsone/bhaus-util/pkg/ast"
)

// qualifiedName converts a contextual_name node (identifier or path) into
// a slash-separated ast.QualifiedName.
func (c *Converter) qualifiedName(n *tree_sitter.Node) *ast.QualifiedName {
	if n == nil {
		return nil
	}
	qn := &ast.QualifiedName{Base: c.spanOf(n), Segments: []*ast.Ident{}}
	for i := uint(0); i < n.ChildCount(); i++ {
		child := n.Child(i)
		switch child.Kind() {
		case "identifier":
			qn.Segments = append(qn.Segments, c.ident(child))
		case "path":
			for j := uint(0); j < child.ChildCount(); j++ {
				if seg := child.Child(j); seg.Kind() == "identifier" {
					qn.Segments = append(qn.Segments, c.ident(seg))
				}
			}
		}
	}
	return qn
}
