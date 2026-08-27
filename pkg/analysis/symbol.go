package analysis

import (
	"github.com/sjsone/bhaus-util/pkg/ast"
)

// Kind is the semantic kind of a symbol.
type Kind uint8

const (
	KindClass Kind = iota
	KindStruct
	KindProtocol
	KindFunction
	KindMethod
	KindProperty
	KindSystem
	KindContainer
	KindComponent
	KindConnection
	KindExtern
	KindVersion
)

func (k Kind) String() string {
	switch k {
	case KindClass:
		return "class"
	case KindStruct:
		return "struct"
	case KindProtocol:
		return "protocol"
	case KindFunction:
		return "function"
	case KindMethod:
		return "method"
	case KindProperty:
		return "property"
	case KindSystem:
		return "system"
	case KindContainer:
		return "container"
	case KindComponent:
		return "component"
	case KindConnection:
		return "connection"
	case KindExtern:
		return "extern"
	case KindVersion:
		return "version"
	}
	return "unknown"
}

// Symbol is a declared symbol in the workspace.
type Symbol struct {
	Kind     Kind
	URI      string
	Name     string   // simple name
	FullName string   // e.g. "Domain/Entity/User", "MailSystem.MTA"
	Span     ast.Span // span of the declared name
	Decl     ast.Node // pointer to the declaring AST node
}

// RefKind identifies the syntactic position of a reference.
type RefKind uint8

const (
	RefType        RefKind = iota // type annotation in member, param, return type, etc.
	RefExtends                    // CLASS Foo EXTENDS Bar
	RefImplements                 // CLASS Foo IMPLEMENTS Bar
	RefExtern                     // EXTERN Domain/Repo
	RefC4Source                   // CONNECTION A -> B (left side)
	RefC4Target                   // CONNECTION A -> B (right side)
	RefC4Shorthand                // CONNECTION => Target (shorthand, source implicit)
)

// Reference is a resolved usage of a symbol at a specific source location.
type Reference struct {
	URI  string
	Span ast.Span // exact span of the reference text
	Kind RefKind
	Sym  *Symbol  // resolved target; nil = unresolved
	Site ast.Node // the AST node that IS the reference
}
