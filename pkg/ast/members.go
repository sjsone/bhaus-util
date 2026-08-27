package ast

// Visibility of a member declaration.
type Visibility uint8

const (
	VisibilityPublic Visibility = iota
	VisibilityPrivate
	VisibilityProtected
)

func (v Visibility) String() string {
	switch v {
	case VisibilityPublic:
		return "PUBLIC"
	case VisibilityPrivate:
		return "PRIVATE"
	case VisibilityProtected:
		return "PROTECTED"
	}
	return "UNKNOWN"
}

// ── Member interfaces ──

// ClassMember is implemented by PropertyMember and MethodMember.
// Use a type switch to discriminate:
//
//	switch m := member.(type) {
//	case *PropertyMember: ...
//	case *MethodMember: ...
//	}
type ClassMember interface {
	Node
	Visibility() Visibility
	classMember()
}

// StructMember is implemented by PropertyMember and MethodMember.
type StructMember interface {
	Node
	Visibility() Visibility
	structMember()
}

// ProtocolMember is implemented by PropertyMember and MethodMember.
type ProtocolMember interface {
	Node
	Visibility() Visibility
	protocolMember()
}

// ── Concrete member types ──

// PropertyMember is a field/property declaration in a CLASS, STRUCT or PROTOCOL.
//
//	PUBLIC name: String
type PropertyMember struct {
	Base
	Name *Ident
	Vis  Visibility
	Type TypeRef
}

func (m *PropertyMember) Visibility() Visibility { return m.Vis }
func (*PropertyMember) classMember()             {}
func (*PropertyMember) structMember()            {}
func (*PropertyMember) protocolMember()          {}

// MethodMember is a method declaration in a CLASS, STRUCT or PROTOCOL.
//
//	OVERRIDE PUBLIC validate(x String): Boolean
type MethodMember struct {
	Base
	Name       *Ident
	Vis        Visibility
	Override   bool // OVERRIDE keyword present (only meaningful in CLASS)
	Params     []*Parameter
	ReturnType TypeRef             // nil = no return type (void procedure)
	Intents    []*FunctionalIntent // "> ..." lines describing the body, in source order
}

func (m *MethodMember) Visibility() Visibility { return m.Vis }
func (*MethodMember) classMember()             {}
func (*MethodMember) structMember()            {}
func (*MethodMember) protocolMember()          {}

// Parameter is a formal parameter in a method or function.
//
//	name Type
type Parameter struct {
	Base
	Name *Ident
	Type TypeRef
}

// FunctionalIntent is a "> ..." line. It states what a function or method
// body must do. It is free text, with the leading ">" stripped. It attaches
// to a MethodMember or a FunctionDecl. It carries no visibility and declares
// no symbol.
type FunctionalIntent struct {
	Base
	Text string
}
