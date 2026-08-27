package language

import (
	"strings"
	"testing"

	"github.com/sjsone/bhaus-util/pkg/ast"
	"github.com/sjsone/bhaus-util/pkg/scaffold"
)

// qname builds a QualifiedName from segments (test helper).
func qname(segs ...string) *ast.QualifiedName {
	names := make([]*ast.Ident, len(segs))
	for i, s := range segs {
		names[i] = &ast.Ident{Name: s}
	}
	return &ast.QualifiedName{Segments: names}
}

// render is a test helper. It runs a scaffolder over decls. It returns the
// concatenated content of all generated files.
func render(t *testing.T, s scaffold.Scaffolder, decls ...ast.Decl) string {
	t.Helper()
	files, err := s.Scaffold(decls)
	if err != nil {
		t.Fatalf("Scaffold: unexpected error: %v", err)
	}
	var b strings.Builder
	for _, f := range files {
		b.WriteString(f.Content)
	}
	return b.String()
}

func TestGoScaffolderRegistered(t *testing.T) {
	s, err := scaffold.Get("go")
	if err != nil {
		t.Fatalf("Get(go): %v", err)
	}
	if s.Language() != "go" {
		t.Fatalf("Language(): got %q, want go", s.Language())
	}
}

func TestGoClassWithMethod(t *testing.T) {
	decl := &ast.ClassDecl{
		Name: qname("Controller", "UserController"),
		Members: []ast.ClassMember{
			&ast.MethodMember{
				Name:       &ast.Ident{Name: "index"},
				Vis:        ast.VisibilityPublic,
				ReturnType: &ast.SimpleType{Name: "String"},
				Intents:    []*ast.FunctionalIntent{{Text: "return the index page"}},
			},
		},
	}
	out := render(t, mustGet(t, "go"), decl)

	for _, want := range []string{
		"type UserController struct {",
		"func (r *UserController) Index() string {",
		"// TODO: return the index page",
		`panic("not implemented")`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n---\n%s", want, out)
		}
	}
}

func TestGoProtocol(t *testing.T) {
	decl := &ast.ProtocolDecl{
		Name: qname("Domain", "Repository"),
		Members: []ast.ProtocolMember{
			&ast.MethodMember{
				Name:       &ast.Ident{Name: "findById"},
				Vis:        ast.VisibilityPublic,
				Params:     []*ast.Parameter{{Type: &ast.SimpleType{Name: "Integer"}}},
				ReturnType: &ast.OptionalType{Inner: &ast.NamedType{Name: qname("Domain", "User")}},
			},
		},
	}
	out := render(t, mustGet(t, "go"), decl)
	if !strings.Contains(out, "type Repository interface {") {
		t.Errorf("missing interface decl\n%s", out)
	}
	if !strings.Contains(out, "FindById(int) *User") {
		t.Errorf("missing method signature\n%s", out)
	}
}

func TestGoStructWithProperty(t *testing.T) {
	decl := &ast.StructDecl{
		Name: qname("Domain", "User"),
		Members: []ast.StructMember{
			&ast.PropertyMember{
				Name: &ast.Ident{Name: "name"},
				Vis:  ast.VisibilityPrivate,
				Type: &ast.SimpleType{Name: "String"},
			},
		},
	}
	out := render(t, mustGet(t, "go"), decl)
	if !strings.Contains(out, "type User struct {") || !strings.Contains(out, "name string") {
		t.Errorf("missing struct field\n%s", out)
	}
}

func TestGoFunctionAndArrayType(t *testing.T) {
	decl := &ast.FunctionDecl{
		Name:   qname("calculateTotal"),
		Params: []*ast.Parameter{{Name: &ast.Ident{Name: "xs"}, Type: &ast.ArrayType{Elem: &ast.SimpleType{Name: "Integer"}}}},
		Result: &ast.SimpleType{Name: "Integer"},
	}
	out := render(t, mustGet(t, "go"), decl)
	if !strings.Contains(out, "func calculateTotal(xs []int) int {") {
		t.Errorf("missing function signature\n%s", out)
	}
}

func mustGet(t *testing.T, lang string) scaffold.Scaffolder {
	t.Helper()
	s, err := scaffold.Get(lang)
	if err != nil {
		t.Fatalf("Get(%s): %v", lang, err)
	}
	return s
}
