package language

import (
	"strings"
	"testing"

	"github.com/sjsone/bhaus-util/pkg/ast"
)

func TestTSClassWithMethodAndImplements(t *testing.T) {
	decl := &ast.ClassDecl{
		Name:       qname("UserController"),
		Implements: []*ast.QualifiedName{qname("Http", "Handler")},
		Members: []ast.ClassMember{
			&ast.MethodMember{
				Name:       &ast.Ident{Name: "index"},
				Vis:        ast.VisibilityPublic,
				ReturnType: &ast.SimpleType{Name: "String"},
				Intents:    []*ast.FunctionalIntent{{Text: "render index"}},
			},
		},
	}
	out := render(t, mustGet(t, "typescript"), decl)
	for _, want := range []string{
		"class UserController implements Handler {",
		"public index(): string {",
		"// TODO: render index",
		`throw new Error("not implemented")`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n---\n%s", want, out)
		}
	}
}

func TestTSInterfaceAndOptionalType(t *testing.T) {
	decl := &ast.ProtocolDecl{
		Name: qname("Repository"),
		Members: []ast.ProtocolMember{
			&ast.MethodMember{
				Name:       &ast.Ident{Name: "findById"},
				Vis:        ast.VisibilityPublic,
				Params:     []*ast.Parameter{{Type: &ast.SimpleType{Name: "Integer"}}},
				ReturnType: &ast.OptionalType{Inner: &ast.NamedType{Name: qname("Domain", "User")}},
			},
		},
	}
	out := render(t, mustGet(t, "typescript"), decl)
	if !strings.Contains(out, "interface Repository {") {
		t.Errorf("missing interface\n%s", out)
	}
	if !strings.Contains(out, "findById(arg0: number): User | null;") {
		t.Errorf("missing method signature\n%s", out)
	}
}

func TestTSStructAsInterface(t *testing.T) {
	decl := &ast.StructDecl{
		Name: qname("User"),
		Members: []ast.StructMember{
			&ast.PropertyMember{Name: &ast.Ident{Name: "name"}, Vis: ast.VisibilityPublic, Type: &ast.SimpleType{Name: "String"}},
		},
	}
	out := render(t, mustGet(t, "typescript"), decl)
	if !strings.Contains(out, "interface User {") || !strings.Contains(out, "name: string;") {
		t.Errorf("missing interface property\n%s", out)
	}
}

func TestTSUnionAndArray(t *testing.T) {
	decl := &ast.FunctionDecl{
		Name:   qname("pick"),
		Params: []*ast.Parameter{{Name: &ast.Ident{Name: "xs"}, Type: &ast.ArrayType{Elem: &ast.SimpleType{Name: "String"}}}},
		Result: &ast.UnionType{Left: &ast.SimpleType{Name: "String"}, Right: &ast.SimpleType{Name: "Integer"}},
	}
	out := render(t, mustGet(t, "typescript"), decl)
	if !strings.Contains(out, "function pick(xs: string[]): string | number {") {
		t.Errorf("missing function signature\n%s", out)
	}
}
