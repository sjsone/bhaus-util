package scaffold

import (
	"strings"
	"testing"

	"github.com/sjsone/bhaus-util/pkg/ast"
)

// render is a test helper. It runs a scaffolder over decls. It returns the
// concatenated content of all generated files.
func render(t *testing.T, s Scaffolder, decls ...ast.Decl) string {
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

// fakeScaffolder is a test double that exercises the registry.
type fakeScaffolder struct{ lang string }

func (f fakeScaffolder) Language() string                           { return f.lang }
func (fakeScaffolder) Scaffold([]ast.Decl) ([]GeneratedFile, error) { return nil, nil }

func TestRegisterAndGet(t *testing.T) {
	reg := newRegistry()
	reg.register(fakeScaffolder{lang: "fake"})

	got, err := reg.get("fake")
	if err != nil {
		t.Fatalf("get(fake): unexpected error: %v", err)
	}
	if got.Language() != "fake" {
		t.Fatalf("get(fake): got language %q, want %q", got.Language(), "fake")
	}
}

func TestGetUnknownErrors(t *testing.T) {
	reg := newRegistry()
	reg.register(fakeScaffolder{lang: "fake"})

	if _, err := reg.get("missing"); err == nil {
		t.Fatal("get(missing): expected error, got nil")
	}
}

func TestAvailableSorted(t *testing.T) {
	reg := newRegistry()
	reg.register(fakeScaffolder{lang: "php"})
	reg.register(fakeScaffolder{lang: "go"})

	avail := reg.available()
	if len(avail) != 2 || avail[0] != "go" || avail[1] != "php" {
		t.Fatalf("available(): got %v, want [go php]", avail)
	}
}

func TestFilterByName(t *testing.T) {
	user := &ast.ClassDecl{Name: qname("Domain", "User")}
	order := &ast.ClassDecl{Name: qname("Domain", "Order")}
	decls := []ast.Decl{user, order}

	got := FilterByName(decls, "Domain/User")
	if len(got) != 1 || got[0] != user {
		t.Fatalf("FilterByName: got %v, want [Domain/User]", got)
	}
}

func TestFilterByNameEmptyReturnsAll(t *testing.T) {
	decls := []ast.Decl{
		&ast.ClassDecl{Name: qname("Domain", "User")},
	}
	if got := FilterByName(decls, ""); len(got) != 1 {
		t.Fatalf("FilterByName(empty): got %d decls, want 1", len(got))
	}
}

// qname builds a QualifiedName from segments (test helper).
func qname(segs ...string) *ast.QualifiedName {
	idents := make([]*ast.Ident, len(segs))
	for i, s := range segs {
		idents[i] = &ast.Ident{Name: s}
	}
	return &ast.QualifiedName{Segments: idents}
}
