package language

import (
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/sjsone/bhaus-util/pkg/ast"
	"github.com/sjsone/bhaus-util/pkg/scaffold"
)

// goFileName is the path of the single file the Go scaffolder emits.
const goFileName = "scaffold.go"

func init() { scaffold.Register(goScaffolder{}) }

// goScaffolder emits rough Go source. Visibility maps to name casing, because Go
// has no visibility keyword. Unions collapse to any, because Go has no sum types.
type goScaffolder struct{}

func (goScaffolder) Language() string { return "go" }

var goTypes = scaffold.TypeMap{
	Simple: map[string]string{
		"String":          "string",
		"Integer":         "int",
		"Int":             "int",
		"Float":           "float64",
		"Boolean":         "bool",
		"Bool":            "bool",
		"Character":       "rune",
		"Char":            "rune",
		"UnsignedInteger": "uint",
		"UInt":            "uint",
		"UnsignedFloat":   "float64",
		"UFloat":          "float64",
		"Unknown":         "any",
	},
	Array:    "[]%s",
	Optional: "*%s",
	Union:    "any",
	Named:    "%s",
	Bits: func(width int) string {
		switch {
		case width <= 8:
			return "uint8"
		case width <= 16:
			return "uint16"
		case width <= 32:
			return "uint32"
		default:
			return "uint64"
		}
	},
}

func (s goScaffolder) Scaffold(decls []ast.Decl) ([]scaffold.GeneratedFile, error) {
	var files []scaffold.GeneratedFile
	for _, d := range decls {
		var f *scaffold.GeneratedFile
		switch decl := d.(type) {
		case *ast.ClassDecl:
			f = s.classFile(decl.Name.Simple(), decl.Extends, scaffold.Members(decl.Members))
		case *ast.StructDecl:
			f = s.structFile(decl.Name.Simple(), nil, scaffold.Members(decl.Members))
		case *ast.ProtocolDecl:
			f = s.interfaceFile(decl.Name.Simple(), scaffold.Members(decl.Members))
		case *ast.FunctionDecl:
			f = s.functionFile(decl.Name.Simple(), decl.Params, decl.Result, scaffold.Intents(decl.Intents))
		case *ast.ExternDecl:
			f = s.externFile(decl.Name.String())
		}
		// C4, INCLUDE and VERSION declarations produce no Go file.
		if f != nil {
			files = append(files, *f)
		}
	}
	return files, nil
}

// classFile builds a Go file for a CLASS (struct + methods).
func (s goScaffolder) classFile(name string, extends *ast.QualifiedName, ms []ast.ClassMember) *scaffold.GeneratedFile {
	var b strings.Builder
	s.emitStruct(&b, name, extends, ms)
	s.emitMethods(&b, name, ms)
	return &scaffold.GeneratedFile{Path: name + ".go", Content: b.String()}
}

// structFile builds a Go file for a STRUCT (struct + methods).
func (s goScaffolder) structFile(name string, extends *ast.QualifiedName, ms []ast.ClassMember) *scaffold.GeneratedFile {
	var b strings.Builder
	s.emitStruct(&b, name, extends, ms)
	s.emitMethods(&b, name, ms)
	return &scaffold.GeneratedFile{Path: name + ".go", Content: b.String()}
}

// interfaceFile builds a Go file for a PROTOCOL.
func (s goScaffolder) interfaceFile(name string, ms []ast.ClassMember) *scaffold.GeneratedFile {
	var b strings.Builder
	s.emitInterface(&b, name, ms)
	return &scaffold.GeneratedFile{Path: name + ".go", Content: b.String()}
}

// functionFile builds a Go file for a top-level FUNCTION.
func (s goScaffolder) functionFile(name string, params []*ast.Parameter, result ast.TypeRef, ins []string) *scaffold.GeneratedFile {
	var b strings.Builder
	s.emitFunc(&b, name, params, result, ins)
	return &scaffold.GeneratedFile{Path: name + ".go", Content: b.String()}
}

// externFile builds a Go file for an EXTERN declaration.
func (s goScaffolder) externFile(name string) *scaffold.GeneratedFile {
	var b strings.Builder
	fmt.Fprintf(&b, "// EXTERN %s (defined elsewhere)\n\n", name)
	return &scaffold.GeneratedFile{Path: name + ".go", Content: b.String()}
}

// emitStruct writes a struct type with an optional embedded parent and its fields.
func (goScaffolder) emitStruct(w io.Writer, name string, extends *ast.QualifiedName, ms []ast.ClassMember) {
	fmt.Fprintf(w, "type %s struct {\n", name)
	if extends != nil {
		fmt.Fprintf(w, "\t%s // EXTENDS\n", extends.Simple())
	}
	for _, m := range ms {
		if p, ok := m.(*ast.PropertyMember); ok {
			fmt.Fprintf(w, "\t%s %s\n", goName(p.Name, p.Vis), goTypes.Render(p.Type))
		}
	}
	fmt.Fprint(w, "}\n\n")
}

// emitMethods writes each method member as a func with a pointer receiver.
func (s goScaffolder) emitMethods(w io.Writer, recv string, ms []ast.ClassMember) {
	for _, m := range ms {
		method, ok := m.(*ast.MethodMember)
		if !ok {
			continue
		}
		sig := fmt.Sprintf("func (r *%s) %s(%s)%s", recv, goName(method.Name, method.Vis),
			goParams(method.Params), goResult(method.ReturnType))
		fmt.Fprintf(w, "%s {\n", sig)
		scaffold.WriteIntents(w, "\t", "// TODO: ", scaffold.Intents(method.Intents))
		fmt.Fprint(w, "\tpanic(\"not implemented\")\n}\n\n")
	}
}

// emitInterface writes a Go interface with one signature per method member.
func (goScaffolder) emitInterface(w io.Writer, name string, ms []ast.ClassMember) {
	fmt.Fprintf(w, "type %s interface {\n", name)
	for _, m := range ms {
		if method, ok := m.(*ast.MethodMember); ok {
			fmt.Fprintf(w, "\t%s(%s)%s\n", goName(method.Name, method.Vis),
				goParams(method.Params), goResult(method.ReturnType))
		}
	}
	fmt.Fprint(w, "}\n\n")
}

// emitFunc writes a top-level function.
func (goScaffolder) emitFunc(w io.Writer, name string, params []*ast.Parameter, result ast.TypeRef, ins []string) {
	fmt.Fprintf(w, "func %s(%s)%s {\n", name, goParams(params), goResult(result))
	scaffold.WriteIntents(w, "\t", "// TODO: ", ins)
	fmt.Fprint(w, "\tpanic(\"not implemented\")\n}\n\n")
}

// goParams renders a comma-separated parameter list. Nameless params (common in
// protocols) render as their type alone.
func goParams(params []*ast.Parameter) string {
	parts := make([]string, 0, len(params))
	for _, p := range params {
		typ := goTypes.Render(p.Type)
		if p.Name != nil && p.Name.Name != "" {
			parts = append(parts, p.Name.Name+" "+typ)
		} else {
			parts = append(parts, typ)
		}
	}
	return strings.Join(parts, ", ")
}

// goResult renders a return type as " T" or "" when there is none.
func goResult(t ast.TypeRef) string {
	if t == nil {
		return ""
	}
	return " " + goTypes.Render(t)
}

// goName applies Go visibility-by-casing: public → exported, otherwise unexported.
func goName(id *ast.Ident, vis ast.Visibility) string {
	if id == nil {
		return ""
	}
	if vis == ast.VisibilityPublic {
		return exportCase(id.Name, true)
	}
	return exportCase(id.Name, false)
}

// exportCase forces the first rune of name to upper (exported) or lower case.
func exportCase(name string, exported bool) string {
	if name == "" {
		return name
	}
	r, size := utf8.DecodeRuneInString(name)
	if exported {
		r = unicode.ToUpper(r)
	} else {
		r = unicode.ToLower(r)
	}
	return string(r) + name[size:]
}

func firstLower(s string) string {
	r, size := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError && size <= 1 {
		return s
	}
	lc := unicode.ToLower(r)
	if r == lc {
		return s
	}
	return string(lc)
}
