package ast

// Visitor visits nodes in source order. Walk calls Visit for each node.
// Walk skips a node's children when Visit returns nil.
type Visitor interface {
	Visit(node Node) Visitor
}

// Walk traverses node and its children in source order.
func Walk(v Visitor, node Node) {
	if v = v.Visit(node); v == nil {
		return
	}
	for _, c := range children(node) {
		Walk(v, c)
	}
}

type inspector func(Node) bool

func (f inspector) Visit(node Node) Visitor {
	if f(node) {
		return f
	}
	return nil
}

// Inspect traverses node and its children and calls f for each node.
// Inspect skips a node's children when f returns false.
func Inspect(node Node, f func(Node) bool) {
	Walk(inspector(f), node)
}

// NodeAt returns the deepest AST node at the given position. The position
// uses 0-based line and column numbers. NodeAt returns nil if no node
// contains the position.
//
// This is the main way LSP queries map a cursor position to a node.
func NodeAt(root Node, line, column uint32) Node {
	target := Pos{Line: line, Column: column}
	var deepest Node
	var deepestLen uint64

	Inspect(root, func(n Node) bool {
		sp := SpanOf(n)
		if !Contains(sp, target) {
			return false // prune: position not in this subtree
		}
		spLen := uint64(sp.End.Offset - sp.Start.Offset)
		if deepest == nil || (spLen > 0 && spLen < deepestLen) {
			deepest = n
			deepestLen = spLen
		}
		return true // keep searching children
	})

	return deepest
}

// PathTo returns the ancestor chain from root to the deepest node at pos.
// Index 0 is the root. The last index is the deepest node. PathTo returns
// nil if pos is outside root.
//
// LSP handlers use PathTo to recover context. For example, a handler can
// check if an Ident sits inside an Extends clause. It can also check if a
// C4Path is a shorthand target.
func PathTo(root Node, line, column uint32) []Node {
	target := Pos{Line: line, Column: column}
	if !Contains(SpanOf(root), target) {
		return nil
	}
	return pathTo(root, target, nil)
}

func pathTo(n Node, target Pos, path []Node) []Node {
	if !Contains(SpanOf(n), target) {
		return nil
	}
	path = append(path, n)
	for _, c := range children(n) {
		if p := pathTo(c, target, path); p != nil {
			return p
		}
	}
	return path
}

// children returns the structural children of a node in source order.
// Update this function when you add a new node type.
func children(n Node) []Node {
	switch n := n.(type) {
	case *File:
		out := make([]Node, 0, len(n.Decls)+len(n.Comments))
		for _, d := range n.Decls {
			out = append(out, d)
		}
		for _, c := range n.Comments {
			out = append(out, c)
		}
		return out

	case *ClassDecl:
		out := []Node{n.Name}
		if n.Extends != nil {
			out = append(out, n.Extends)
		}
		for _, imp := range n.Implements {
			out = append(out, imp)
		}
		for _, m := range n.Members {
			out = append(out, m)
		}
		return out

	case *StructDecl:
		out := []Node{n.Name}
		for _, m := range n.Members {
			out = append(out, m)
		}
		return out

	case *ProtocolDecl:
		out := []Node{n.Name}
		for _, m := range n.Members {
			out = append(out, m)
		}
		return out

	case *FunctionDecl:
		out := []Node{n.Name}
		for _, p := range n.Params {
			out = append(out, p)
		}
		if n.Result != nil {
			out = append(out, n.Result)
		}
		for _, it := range n.Intents {
			out = append(out, it)
		}
		return out

	case *PropertyMember:
		return []Node{n.Name, n.Type}

	case *MethodMember:
		out := []Node{n.Name}
		for _, p := range n.Params {
			out = append(out, p)
		}
		if n.ReturnType != nil {
			out = append(out, n.ReturnType)
		}
		for _, it := range n.Intents {
			out = append(out, it)
		}
		return out

	case *Parameter:
		if n.Name != nil {
			return []Node{n.Name, n.Type}
		}
		return []Node{n.Type}

	case *SystemDecl:
		out := []Node{n.Name}
		for _, ct := range n.Containers {
			out = append(out, ct)
		}
		for _, cs := range n.Connections {
			out = append(out, cs)
		}
		return out

	case *ContainerDecl:
		out := []Node{n.Name}
		for _, cp := range n.Components {
			out = append(out, cp)
		}
		for _, cs := range n.Connections {
			out = append(out, cs)
		}
		return out

	case *ComponentDecl:
		out := []Node{n.Name}
		for _, cs := range n.Connections {
			out = append(out, cs)
		}
		return out

	case *ConnectionDecl:
		return []Node{n.Source, n.Target}

	case *ConnectionShorthand:
		return []Node{n.Target}

	case *ExternDecl:
		return []Node{n.Name}

	case *QualifiedName:
		out := make([]Node, 0, len(n.Segments))
		for _, s := range n.Segments {
			out = append(out, s)
		}
		return out

	case *C4Path:
		out := make([]Node, 0, len(n.Segments))
		for _, s := range n.Segments {
			out = append(out, s)
		}
		return out

	case *NamedType:
		return []Node{n.Name}

	case *ArrayType:
		return []Node{n.Elem}

	case *BitsType:
		return []Node{n.Width}

	case *OptionalType:
		return []Node{n.Inner}

	case *UnionType:
		return []Node{n.Left, n.Right}

	// Leaf nodes: no children
	case *Ident, *SimpleType, *BitWidth, *VersionDecl, *IncludeDecl, *Comment, *FunctionalIntent:
		return nil

	default:
		return nil
	}
}
