package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sjsone/bhaus-util/pkg/ast"
)

const kotlinYAML = `
language: kotlin
types:
  String: String
  Integer: Int
formats:
  array: "List<%s>"
  optional: "%s?"
  union: "%s"
  named: "%s"
  param: "%[1]s: %[2]s"
templates:
  class: |
    class {{ .Name }} {
    {{- range .Methods }}
        fun {{ .Name }}({{ .Params }}){{ if .ReturnType }}: {{ .ReturnType }}{{ end }} {
    {{- range .Intents }}
            // TODO: {{ . }}
    {{- end }}
            TODO("not implemented")
        }
    {{- end }}
    }
`

func TestYAMLScaffolderSatisfiesInterface(t *testing.T) {
	s, err := loadYAMLDef([]byte(kotlinYAML))
	if err != nil {
		t.Fatalf("loadYAMLDef: %v", err)
	}
	var _ Scaffolder = s // must satisfy the interface
	if s.Language() != "kotlin" {
		t.Fatalf("Language(): got %q, want kotlin", s.Language())
	}
}

func TestYAMLScaffolderRendersClass(t *testing.T) {
	s, err := loadYAMLDef([]byte(kotlinYAML))
	if err != nil {
		t.Fatalf("loadYAMLDef: %v", err)
	}
	decl := &ast.ClassDecl{
		Name: qname("Greeter"),
		Members: []ast.ClassMember{
			&ast.MethodMember{
				Name:       &ast.Ident{Name: "greet"},
				Vis:        ast.VisibilityPublic,
				Params:     []*ast.Parameter{{Name: &ast.Ident{Name: "x"}, Type: &ast.SimpleType{Name: "Integer"}}},
				ReturnType: &ast.ArrayType{Elem: &ast.SimpleType{Name: "String"}},
				Intents:    []*ast.FunctionalIntent{{Text: "say hi"}},
			},
		},
	}
	out := render(t, s, decl)
	for _, want := range []string{
		"class Greeter {",
		"fun greet(x: Int): List<String> {",
		"// TODO: say hi",
		`TODO("not implemented")`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n---\n%s", want, out)
		}
	}
}

func TestLoadDirMissingIsEmpty(t *testing.T) {
	got, err := LoadDir(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Fatalf("LoadDir(missing): unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("LoadDir(missing): got %d, want 0", len(got))
	}
}

func TestLoadDirReadsYAMLFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "kotlin.yaml"), []byte(kotlinYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := LoadDir(dir)
	if err != nil {
		t.Fatalf("LoadDir: %v", err)
	}
	if len(got) != 1 || got[0].Language() != "kotlin" {
		t.Fatalf("LoadDir: got %v, want one kotlin scaffolder", got)
	}
}
