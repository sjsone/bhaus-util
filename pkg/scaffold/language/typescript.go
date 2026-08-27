package language

import (
	"fmt"
	"io"
	"strings"

	"github.com/sjsone/bhaus-util/pkg/ast"
	"github.com/sjsone/bhaus-util/pkg/scaffold"
)

func init() { scaffold.Register(tsScaffolder{}) }

// tsScaffolder emits rough TypeScript source. Visibility maps to the public,
// private and protected keywords. Nameless parameters (common in protocols) get
// positional names arg0, arg1, ... because TypeScript requires parameter names.
type tsScaffolder struct{}

func (tsScaffolder) Language() string { return "typescript" }

var tsTypes = scaffold.TypeMap{
	Simple: map[string]string{
		"String":          "string",
		"Integer":         "number",
		"Int":             "number",
		"Float":           "number",
		"Boolean":         "boolean",
		"Bool":            "boolean",
		"Character":       "string",
		"Char":            "string",
		"UnsignedInteger": "number",
		"UInt":            "number",
		"UnsignedFloat":   "number",
		"UFloat":          "number",
		"Unknown":         "unknown",
	},
	Array:    "%s[]",
	Optional: "%s | null",
	Union:    "%s | %s",
	Named:    "%s",
	Bits:     func(int) string { return "number" },
}

func (s tsScaffolder) Scaffold(decls []ast.Decl) ([]scaffold.GeneratedFile, error) {
	var files []scaffold.GeneratedFile
	for _, d := range decls {
		var f *scaffold.GeneratedFile
		switch decl := d.(type) {
		case *ast.ClassDecl:
			f = s.classFile(decl)
		case *ast.StructDecl:
			f = s.interfaceFile(decl.Name.Simple(), scaffold.Members(decl.Members))
		case *ast.ProtocolDecl:
			f = s.interfaceFile(decl.Name.Simple(), scaffold.Members(decl.Members))
		case *ast.FunctionDecl:
			f = s.functionFile(decl.Name.Simple(), decl.Params, decl.Result, scaffold.Intents(decl.Intents))
		case *ast.ExternDecl:
			f = s.externFile(decl.Name.String())
		}
		// C4, INCLUDE and VERSION declarations produce no TypeScript file.
		if f != nil {
			files = append(files, *f)
		}
	}
	return files, nil
}

// classFile builds a TypeScript file for a CLASS.
func (s tsScaffolder) classFile(decl *ast.ClassDecl) *scaffold.GeneratedFile {
	var b strings.Builder
	s.emitClass(&b, decl)
	return &scaffold.GeneratedFile{Path: decl.Name.Simple() + ".ts", Content: b.String()}
}

// interfaceFile builds a TypeScript file for a STRUCT or PROTOCOL.
func (s tsScaffolder) interfaceFile(name string, ms []ast.ClassMember) *scaffold.GeneratedFile {
	var b strings.Builder
	s.emitInterface(&b, name, ms)
	return &scaffold.GeneratedFile{Path: name + ".ts", Content: b.String()}
}

// functionFile builds a TypeScript file for a top-level FUNCTION.
func (s tsScaffolder) functionFile(name string, params []*ast.Parameter, result ast.TypeRef, ins []string) *scaffold.GeneratedFile {
	var b strings.Builder
	fmt.Fprintf(&b, "function %s(%s)%s {\n", name, tsParams(params), tsResult(result))
	scaffold.WriteIntents(&b, "\t", "// TODO: ", ins)
	fmt.Fprint(&b, "\tthrow new Error(\"not implemented\");\n}\n\n")
	return &scaffold.GeneratedFile{Path: name + ".ts", Content: b.String()}
}

// externFile builds a TypeScript file for an EXTERN declaration.
func (s tsScaffolder) externFile(name string) *scaffold.GeneratedFile {
	var b strings.Builder
	fmt.Fprintf(&b, "// EXTERN %s (defined elsewhere)\n\n", name)
	return &scaffold.GeneratedFile{Path: name + ".ts", Content: b.String()}
}

// emitClass writes a class with optional extends/implements clauses.
func (tsScaffolder) emitClass(w io.Writer, decl *ast.ClassDecl) {
	header := "class " + decl.Name.Simple()
	if decl.Extends != nil {
		header += " extends " + decl.Extends.Simple()
	}
	if len(decl.Implements) > 0 {
		names := make([]string, len(decl.Implements))
		for i, impl := range decl.Implements {
			names[i] = impl.Simple()
		}
		header += " implements " + strings.Join(names, ", ")
	}
	fmt.Fprintf(w, "%s {\n", header)
	for _, m := range scaffold.Members(decl.Members) {
		switch member := m.(type) {
		case *ast.PropertyMember:
			fmt.Fprintf(w, "\t%s %s: %s;\n", scaffold.VisLower(member.Vis), member.Name.Name, tsTypes.Render(member.Type))
		case *ast.MethodMember:
			fmt.Fprintf(w, "\t%s %s(%s)%s {\n", scaffold.VisLower(member.Vis), member.Name.Name,
				tsParams(member.Params), tsResult(member.ReturnType))
			scaffold.WriteIntents(w, "\t\t", "// TODO: ", scaffold.Intents(member.Intents))
			fmt.Fprint(w, "\t\tthrow new Error(\"not implemented\");\n\t}\n")
		}
	}
	fmt.Fprint(w, "}\n\n")
}

// emitInterface writes an interface for a PROTOCOL or STRUCT (property signatures
// and method signatures, no bodies, no visibility keywords).
func (tsScaffolder) emitInterface(w io.Writer, name string, ms []ast.ClassMember) {
	fmt.Fprintf(w, "interface %s {\n", name)
	for _, m := range ms {
		switch member := m.(type) {
		case *ast.PropertyMember:
			fmt.Fprintf(w, "\t%s: %s;\n", member.Name.Name, tsTypes.Render(member.Type))
		case *ast.MethodMember:
			fmt.Fprintf(w, "\t%s(%s)%s;\n", member.Name.Name, tsParams(member.Params), tsResult(member.ReturnType))
		}
	}
	fmt.Fprint(w, "}\n\n")
}

// tsParams renders a comma-separated "name: type" list, naming anonymous params
// positionally (arg0, arg1, ...).
func tsParams(params []*ast.Parameter) string {
	parts := make([]string, len(params))
	for i, p := range params {
		name := scaffold.ParamName(p, i)
		parts[i] = name + ": " + tsTypes.Render(p.Type)
	}
	return strings.Join(parts, ", ")
}

// tsResult renders a return type as ": T" or "" when there is none.
func tsResult(t ast.TypeRef) string {
	if t == nil {
		return ""
	}
	return ": " + tsTypes.Render(t)
}
