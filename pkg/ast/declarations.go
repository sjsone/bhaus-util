package ast

// Decl is any top-level declaration in source order.
type Decl interface {
	Node
	declNode()
}

// ── Structural declarations ──

// ClassDecl is a CLASS declaration.
//
//	CLASS Controller/UserController EXTENDS Base/Controller IMPLEMENTS Event/Listener:
//	    PUBLIC index(): String
type ClassDecl struct {
	Base
	Name       *QualifiedName
	Extends    *QualifiedName // nil if absent
	Implements []*QualifiedName
	Members    []ClassMember
}

func (*ClassDecl) declNode() {}

// StructDecl is a STRUCT declaration.
//
//	STRUCT Domain/Entity/User:
//	    PUBLIC name: String
type StructDecl struct {
	Base
	Name    *QualifiedName
	Members []StructMember
}

func (*StructDecl) declNode() {}

// ProtocolDecl is a PROTOCOL declaration.
//
//	PROTOCOL Domain/Repository:
//	    PUBLIC findById(Integer): ?Domain/Entity/User
type ProtocolDecl struct {
	Base
	Name    *QualifiedName
	Members []ProtocolMember
}

func (*ProtocolDecl) declNode() {}

// FunctionDecl is a top-level FUNCTION/FUNC declaration.
//
//	FUNCTION calculateTotal(Array[Integer]): Integer
type FunctionDecl struct {
	Base
	Name    *QualifiedName
	Params  []*Parameter
	Result  TypeRef             // nil = no return type
	Intents []*FunctionalIntent // "> ..." lines describing the body, in source order
}

func (*FunctionDecl) declNode() {}

// ── File-level declarations ──

// VersionDecl is the VERSION directive: VERSION 0.1
type VersionDecl struct {
	Base
	Version string
}

func (*VersionDecl) declNode() {}

// IncludeDecl is an INCLUDE directive: INCLUDE *.bhaus
type IncludeDecl struct {
	Base
	Pattern string // glob pattern as written
}

func (*IncludeDecl) declNode() {}

// ExternDecl is an EXTERN declaration: EXTERN Domain/ExternalType
type ExternDecl struct {
	Base
	Name *QualifiedName
}

func (*ExternDecl) declNode() {}

// Comment is a single-line "# ..." comment.
type Comment struct {
	Base
	Text string
}

// SyntaxError records a tree-sitter parse error (ERROR or MISSING node).
type SyntaxError struct {
	Span     Span
	Message  string
	Severity string // "Error" or "Warning"
}

// File is a parsed BHaus document. It is the root of every AST.
type File struct {
	Base
	URI      string
	Decls    []Decl // all top-level declarations in source order
	Comments []*Comment
	Errors   []SyntaxError
}
