package convert

import (
	"strconv"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/sjsone/bhaus-util/pkg/ast"
)

// typeRef unwraps a "type" wrapper node and dispatches to typeRefInner.
func (c *Converter) typeRef(n *tree_sitter.Node) ast.TypeRef {
	if n == nil {
		return nil
	}
	for i := uint(0); i < n.ChildCount(); i++ {
		child := n.Child(i)
		if child.IsNamed() {
			return c.typeRefInner(child)
		}
	}
	return nil
}

// typeRefInner converts the concrete type node inside a "type" wrapper.
func (c *Converter) typeRefInner(n *tree_sitter.Node) ast.TypeRef {
	switch n.Kind() {
	case "simple_type":
		return &ast.SimpleType{Base: c.spanOf(n), Name: c.text(n)}
	case "contextual_name":
		return &ast.NamedType{Base: c.spanOf(n), Name: c.qualifiedName(n)}
	case "array_type":
		return &ast.ArrayType{Base: c.spanOf(n), Elem: c.typeRef(findChildByKind(n, "type"))}
	case "bits_type":
		return &ast.BitsType{Base: c.spanOf(n), Width: c.bitWidth(findChildByKind(n, "bit_width"))}
	case "optional_type":
		return &ast.OptionalType{Base: c.spanOf(n), Inner: c.typeRef(findChildByKind(n, "type"))}
	case "union_type":
		var left, right ast.TypeRef
		for i := uint(0); i < n.ChildCount(); i++ {
			child := n.Child(i)
			if child.Kind() == "|" || !child.IsNamed() {
				continue
			}
			t := c.typeRefInner(child)
			if t == nil {
				continue
			}
			if left == nil {
				left = t
			} else if right == nil {
				right = t
			}
		}
		if left == nil || right == nil {
			return left // degraded union: return the recovered branch
		}
		return &ast.UnionType{Base: c.spanOf(n), Left: left, Right: right}
	}
	return nil
}

// bitWidth converts a "bit_width" node (a bare integer literal) into an
// ast.BitWidth. The grammar's /\d+/ pattern guarantees valid digits. A parse
// error therefore cannot happen for a well-formed node.
func (c *Converter) bitWidth(n *tree_sitter.Node) *ast.BitWidth {
	if n == nil {
		return nil
	}
	value, _ := strconv.Atoi(c.text(n))
	return &ast.BitWidth{Base: c.spanOf(n), Value: value}
}
