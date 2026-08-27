// Package scaffold turns a parsed BHaus AST into rough target-language code.
//
// The output is intentionally not perfect or compilable.
// It is a skeleton that a later refinement pass fills in.
// The scaffolder preserves functional intents ("> ..." lines) as "// TODO:" markers.
// That refinement pass then knows what each body must do.
package scaffold

import (
	"fmt"
	"sort"

	"github.com/sjsone/bhaus-util/pkg/ast"
)

// GeneratedFile is one output file: a relative path and its content. Single-file
// languages (Go, TypeScript) return one GeneratedFile. PSR-4 PHP returns one per
// declaration.
type GeneratedFile struct {
	Path    string // relative, e.g. "Controller/UserController.php" or "scaffold.go"
	Content string
}

// Scaffolder renders a set of declarations as one target language's source files.
type Scaffolder interface {
	// Language is the lookup key for this scaffolder ("go", "typescript", "php").
	Language() string
	// Scaffold returns rough target-language files for decls.
	Scaffold(decls []ast.Decl) ([]GeneratedFile, error)
}

// registry maps a language name to its Scaffolder.
type registry struct {
	byLang map[string]Scaffolder
}

func newRegistry() *registry {
	return &registry{byLang: make(map[string]Scaffolder)}
}

func (r *registry) register(s Scaffolder) {
	r.byLang[s.Language()] = s
}

func (r *registry) get(lang string) (Scaffolder, error) {
	if s, ok := r.byLang[lang]; ok {
		return s, nil
	}
	return nil, fmt.Errorf("unknown language %q; available: %v", lang, r.available())
}

func (r *registry) available() []string {
	langs := make([]string, 0, len(r.byLang))
	for l := range r.byLang {
		langs = append(langs, l)
	}
	sort.Strings(langs)
	return langs
}

// defaultRegistry holds the compiled-in scaffolders (populated by init() in each
// language file) plus any external YAML definitions registered at CLI startup.
var defaultRegistry = newRegistry()

// Register adds a scaffolder to the default registry. Compiled-in languages call
// this from init(). The CLI calls it for each loaded YAML definition.
func Register(s Scaffolder) { defaultRegistry.register(s) }

// Get resolves a language name to its scaffolder or errors listing what is
// available.
func Get(lang string) (Scaffolder, error) { return defaultRegistry.get(lang) }

// Available returns the sorted list of registered language names.
func Available() []string { return defaultRegistry.available() }

// FilterByName returns the declarations whose declared name matches name exactly.
// Structural declarations match on their slash-qualified name ("Domain/User").
// C4 declarations match on their simple name. An empty name returns decls unchanged.
func FilterByName(decls []ast.Decl, name string) []ast.Decl {
	if name == "" {
		return decls
	}
	var out []ast.Decl
	for _, d := range decls {
		if declName(d) == name {
			out = append(out, d)
		}
	}
	return out
}

// declName returns the declared name of a declaration or "" if it has none.
func declName(d ast.Decl) string {
	switch decl := d.(type) {
	case *ast.ClassDecl:
		return decl.Name.String()
	case *ast.StructDecl:
		return decl.Name.String()
	case *ast.ProtocolDecl:
		return decl.Name.String()
	case *ast.FunctionDecl:
		return decl.Name.String()
	case *ast.ExternDecl:
		return decl.Name.String()
	case *ast.SystemDecl:
		return decl.Name.Name
	case *ast.ContainerDecl:
		return decl.Name.Name
	case *ast.ComponentDecl:
		return decl.Name.Name
	}
	return ""
}
