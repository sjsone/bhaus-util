package scaffold

import (
	"fmt"
	"strings"

	"github.com/sjsone/bhaus-util/pkg/ast"
)

// TypeMap describes how to render BHaus types for one target language. Simple maps
// built-in type names to their target spelling. Unknown names pass through unchanged.
// The format strings shape the compound types. Render applies each one recursively:
//
//	Array    one %s   e.g. "[]%s"      (verb-less "array" drops the element)
//	Optional one %s   e.g. "*%s"
//	Union    two %s   e.g. "%s | %s"   (verb-less "any" drops both operands)
//	Named    one %s   e.g. "%s"        (rendered from the type's simple name)
//
// A verb-less format string (no "%") returns literally. This lets a language without
// a given construct collapse it (Go has no unions; PHP has no generic arrays).
//
// Bits<N> is a bit width, not a nested type. It does not fit the %s-substitution
// shape, so it renders through the Bits function instead. That function receives N
// and picks the target spelling. For example, Go picks uint8/uint16/uint32/uint64 by
// width. Languages without sized integers can ignore N and return a constant type.
type TypeMap struct {
	Simple   map[string]string
	Array    string
	Optional string
	Union    string
	Named    string
	Bits     func(width int) string
}

// Render turns a TypeRef into its target-language spelling, recursing through
// nested array/optional/union nodes. A nil TypeRef renders as "".
func (m TypeMap) Render(t ast.TypeRef) string {
	switch typ := t.(type) {
	case nil:
		return ""
	case *ast.SimpleType:
		if mapped, ok := m.Simple[typ.Name]; ok {
			return mapped
		}
		return typ.Name
	case *ast.NamedType:
		return apply(m.Named, typ.Name.Simple())
	case *ast.ArrayType:
		return apply(m.Array, m.Render(typ.Elem))
	case *ast.BitsType:
		if m.Bits == nil || typ.Width == nil {
			return ""
		}
		return m.Bits(typ.Width.Value)
	case *ast.OptionalType:
		return apply(m.Optional, m.Render(typ.Inner))
	case *ast.UnionType:
		return apply(m.Union, m.Render(typ.Left), m.Render(typ.Right))
	}
	return ""
}

// apply substitutes args into a format string. It returns the format unchanged
// when the format contains no verb. This lets a language collapse a construct it
// lacks (e.g. Union: "any") without fmt appending %!(EXTRA ...) noise.
func apply(format string, args ...string) string {
	if !strings.Contains(format, "%") {
		return format
	}
	anys := make([]any, len(args))
	for i, a := range args {
		anys[i] = a
	}
	return fmt.Sprintf(format, anys...)
}

// ── View models (consumed by the YAML template engine) ──

// ClassView is the template data for a CLASS (or STRUCT rendered as a class).
type ClassView struct {
	Name       string
	Extends    string
	Implements []string
	Properties []PropertyView
	Methods    []MethodView
}

// InterfaceView is the template data for a PROTOCOL (or STRUCT rendered as an
// interface).
type InterfaceView struct {
	Name       string
	Properties []PropertyView
	Methods    []MethodView
}

// FunctionView is the template data for a top-level FUNCTION.
type FunctionView struct {
	Name       string
	Params     string // pre-joined parameter list
	ParamList  []ParamView
	ReturnType string
	Intents    []string
}

// MethodView is the template data for a method member.
type MethodView struct {
	Name       string
	Visibility string
	Params     string // pre-joined parameter list
	ParamList  []ParamView
	ReturnType string
	Intents    []string
}

// PropertyView is the template data for a property member.
type PropertyView struct {
	Name       string
	Visibility string
	Type       string
}

// ParamView is one rendered parameter (name + target type).
type ParamView struct {
	Name string
	Type string
}
