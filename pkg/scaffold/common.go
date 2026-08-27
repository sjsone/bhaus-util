package scaffold

import (
	"fmt"
	"io"

	"github.com/sjsone/bhaus-util/pkg/ast"
)

// ParamName returns a parameters name or a positional fallback for anonymous parameters.
// Anonymous parameters are common in PROTOCOL method signatures.
func ParamName(p *ast.Parameter, index int) string {
	if p.Name != nil && p.Name.Name != "" {
		return p.Name.Name
	}
	return fmt.Sprintf("arg%d", index)
}

// Members normalizes a CLASS/STRUCT/PROTOCOL member slice into []ast.ClassMember.
// PropertyMember and MethodMember implement all three member interfaces.
// The concrete nodes carry over unchanged. Only the static element type differs.
func Members[T ast.Node](ms []T) []ast.ClassMember {
	out := make([]ast.ClassMember, 0, len(ms))
	for _, m := range ms {
		if cm, ok := any(m).(ast.ClassMember); ok {
			out = append(out, cm)
		}
	}
	return out
}

// NamedTypesIn returns every NamedType's qualified name reachable in t.
// It recurses through Array/Optional/Union wrappers.
// Callers use it to discover cross-namespace type references (for example for PHP `use` statements).
func NamedTypesIn(t ast.TypeRef) []*ast.QualifiedName {
	switch typ := t.(type) {
	case *ast.NamedType:
		return []*ast.QualifiedName{typ.Name}
	case *ast.ArrayType:
		return NamedTypesIn(typ.Elem)
	case *ast.OptionalType:
		return NamedTypesIn(typ.Inner)
	case *ast.UnionType:
		return append(NamedTypesIn(typ.Left), NamedTypesIn(typ.Right)...)
	}
	return nil
}

// Intents extracts the plain text of each functional intent in source order.
func Intents(ins []*ast.FunctionalIntent) []string {
	out := make([]string, 0, len(ins))
	for _, in := range ins {
		out = append(out, in.Text)
	}
	return out
}

// WriteIntents writes each intent as a comment line: indent + prefix + text.
func WriteIntents(w io.Writer, indent, prefix string, ins []string) {
	for _, in := range ins {
		io.WriteString(w, indent+prefix+in+"\n")
	}
}

// VisLower renders a visibility as its lowercase keyword.
func VisLower(v ast.Visibility) string {
	switch v {
	case ast.VisibilityPrivate:
		return "private"
	case ast.VisibilityProtected:
		return "protected"
	default:
		return "public"
	}
}
