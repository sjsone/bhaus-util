// Package ast defines the abstract syntax tree for BHaus documents.
package ast

// TypeRef is a type annotation in the syntax tree.
// All concrete type nodes implement this interface.
type TypeRef interface {
	Node
	typeRefNode()
}

// SimpleType is a built-in type: String, Integer, Boolean, etc.
type SimpleType struct {
	Base
	Name string
}

func (*SimpleType) typeRefNode() {}

// NamedType is a reference to a user-defined type by contextual name.
// In the AST, every user type reference is a NamedType that wraps a QualifiedName.
// This differs from a QualifiedName in declaration position. The difference
// makes precise reference finding possible.
type NamedType struct {
	Base
	Name *QualifiedName
}

func (*NamedType) typeRefNode() {}

// ArrayType is Array[ElementType].
type ArrayType struct {
	Base
	Elem TypeRef
}

func (*ArrayType) typeRefNode() {}

// BitWidth is the numeric literal argument of a Bits<N> type, e.g. the 8 in Bits<8>.
type BitWidth struct {
	Base
	Value int
}

// BitsType is Bits<Width>, a fixed-width sequence of bits.
type BitsType struct {
	Base
	Width *BitWidth
}

func (*BitsType) typeRefNode() {}

// OptionalType is ?InnerType.
type OptionalType struct {
	Base
	Inner TypeRef
}

func (*OptionalType) typeRefNode() {}

// UnionType is Left | Right (grammar allows exactly two branches).
type UnionType struct {
	Base
	Left  TypeRef
	Right TypeRef
}

func (*UnionType) typeRefNode() {}
