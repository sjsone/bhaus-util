package language

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/sjsone/bhaus-util/pkg/ast"
	"github.com/sjsone/bhaus-util/pkg/scaffold"
)

func init() { scaffold.Register(phpScaffolder{}) }

// phpScaffolder emits PSR-4-conformant PHP: one class, interface or function per
// file, a namespace declaration per file and file paths that mirror the
// fully-qualified name. It derives the namespace and path purely from the BHaus
// qualified name ("Controller/UserController" -> namespace Controller, file
// Controller/UserController.php). Cross-namespace type references become `use`
// statements. Structs and classes both map to a PHP class. Protocols map to an
// interface. Generic arrays collapse to the untyped "array".
type phpScaffolder struct{}

func (phpScaffolder) Language() string { return "php" }

var phpTypes = scaffold.TypeMap{
	Simple: map[string]string{
		"String":          "string",
		"Integer":         "int",
		"Int":             "int",
		"Float":           "float",
		"Boolean":         "bool",
		"Bool":            "bool",
		"Character":       "string",
		"Char":            "string",
		"UnsignedInteger": "int",
		"UInt":            "int",
		"UnsignedFloat":   "float",
		"UFloat":          "float",
		"Unknown":         "mixed",
	},
	Array:    "array",
	Optional: "?%s",
	Union:    "%s|%s",
	Named:    "%s",
	Bits:     func(int) string { return "int" },
}

func (s phpScaffolder) Scaffold(decls []ast.Decl) ([]scaffold.GeneratedFile, error) {
	var files []scaffold.GeneratedFile
	for _, d := range decls {
		var f *scaffold.GeneratedFile
		switch decl := d.(type) {
		case *ast.ClassDecl:
			f = s.classFile(decl.Name, decl.Extends, decl.Implements, scaffold.Members(decl.Members))
		case *ast.StructDecl:
			f = s.classFile(decl.Name, nil, nil, scaffold.Members(decl.Members))
		case *ast.ProtocolDecl:
			f = s.interfaceFile(decl.Name, scaffold.Members(decl.Members))
		case *ast.FunctionDecl:
			f = s.functionFile(decl)
		}
		// C4, INCLUDE, VERSION and EXTERN declarations produce no PHP file.
		if f != nil {
			files = append(files, *f)
		}
	}
	return files, nil
}

// classFile builds the PSR-4 file for a CLASS or STRUCT.
func (s phpScaffolder) classFile(name, extends *ast.QualifiedName, impls []*ast.QualifiedName, ms []ast.ClassMember) *scaffold.GeneratedFile {
	ns, class, path := phpNamespaceAndPath(name)
	set := make(map[string]struct{})
	phpAddQN(set, ns, extends)
	for _, impl := range impls {
		phpAddQN(set, ns, impl)
	}
	phpAddTypeUses(set, ns, memberTypes(ms))
	var b strings.Builder
	b.WriteString(phpHeader(ns, phpSortedUses(set)))
	s.emitClass(&b, class, extends, impls, ms)
	return &scaffold.GeneratedFile{Path: path, Content: b.String()}
}

// interfaceFile builds the PSR-4 file for a PROTOCOL.
func (s phpScaffolder) interfaceFile(name *ast.QualifiedName, ms []ast.ClassMember) *scaffold.GeneratedFile {
	ns, iface, path := phpNamespaceAndPath(name)
	set := make(map[string]struct{})
	phpAddTypeUses(set, ns, memberTypes(ms))
	var b strings.Builder
	b.WriteString(phpHeader(ns, phpSortedUses(set)))
	s.emitInterface(&b, iface, ms)
	return &scaffold.GeneratedFile{Path: path, Content: b.String()}
}

// functionFile builds a file for a top-level FUNCTION. PSR-4 does not autoload
// functions, so the file carries a note that callers must include it manually.
func (s phpScaffolder) functionFile(decl *ast.FunctionDecl) *scaffold.GeneratedFile {
	ns, fn, path := phpNamespaceAndPath(decl.Name)
	types := make([]ast.TypeRef, 0, len(decl.Params)+1)
	for _, p := range decl.Params {
		types = append(types, p.Type)
	}
	types = append(types, decl.Result)
	set := make(map[string]struct{})
	phpAddTypeUses(set, ns, types)

	var b strings.Builder
	b.WriteString(phpHeader(ns, phpSortedUses(set)))
	b.WriteString("// PSR-4 does not autoload functions; include this file manually.\n")
	fmt.Fprintf(&b, "function %s(%s)%s {\n", fn, phpParams(decl.Params), phpResult(decl.Result))
	scaffold.WriteIntents(&b, "\t", "// TODO: ", scaffold.Intents(decl.Intents))
	b.WriteString("\tthrow new \\Exception('not implemented');\n}\n")
	return &scaffold.GeneratedFile{Path: path, Content: b.String()}
}

// phpHeader renders the "<?php", namespace and use lines that open every file.
func phpHeader(ns string, uses []string) string {
	var b strings.Builder
	b.WriteString("<?php\n\n")
	if ns != "" {
		b.WriteString("namespace " + ns + ";\n\n")
	}
	if len(uses) > 0 {
		for _, u := range uses {
			b.WriteString("use " + u + ";\n")
		}
		b.WriteString("\n")
	}
	return b.String()
}

// phpNamespaceAndPath splits a qualified name into its PHP namespace (segments
// before the last, joined with "\"), class name (the last segment) and PSR-4 file
// path (segment directories + "Name.php").
func phpNamespaceAndPath(qn *ast.QualifiedName) (ns, class, path string) {
	segs := qn.Segments
	class = segs[len(segs)-1].Name
	dirs := make([]string, 0, len(segs)-1)
	for _, s := range segs[:len(segs)-1] {
		dirs = append(dirs, s.Name)
	}
	ns = strings.Join(dirs, "\\")
	if len(dirs) > 0 {
		path = strings.Join(dirs, "/") + "/" + class + ".php"
	} else {
		path = class + ".php"
	}
	return ns, class, path
}

// memberTypes gathers every type referenced by a member set (property types,
// method parameter types, method return types).
func memberTypes(ms []ast.ClassMember) []ast.TypeRef {
	var types []ast.TypeRef
	for _, m := range ms {
		switch member := m.(type) {
		case *ast.PropertyMember:
			types = append(types, member.Type)
		case *ast.MethodMember:
			for _, p := range member.Params {
				types = append(types, p.Type)
			}
			types = append(types, member.ReturnType)
		}
	}
	return types
}

// phpAddTypeUses adds the imports needed for each type in types to set.
func phpAddTypeUses(set map[string]struct{}, currentNS string, types []ast.TypeRef) {
	for _, t := range types {
		for _, qn := range scaffold.NamedTypesIn(t) {
			phpAddQN(set, currentNS, qn)
		}
	}
}

// phpAddQN records qn as an import in set when it is a qualified name in a
// namespace other than currentNS. Single-segment (global) and same-namespace names
// need no import.
func phpAddQN(set map[string]struct{}, currentNS string, qn *ast.QualifiedName) {
	if qn == nil || !qn.IsPath() || phpNamespaceOf(qn) == currentNS {
		return
	}
	set[phpFQN(qn)] = struct{}{}
}

// phpSortedUses returns the deduped import set as a sorted slice.
func phpSortedUses(set map[string]struct{}) []string {
	uses := make([]string, 0, len(set))
	for u := range set {
		uses = append(uses, u)
	}
	sort.Strings(uses)
	return uses
}

// phpNamespaceOf is the "\"-joined namespace of a qualified name (all but last).
func phpNamespaceOf(qn *ast.QualifiedName) string {
	names := make([]string, 0, len(qn.Segments)-1)
	for _, s := range qn.Segments[:len(qn.Segments)-1] {
		names = append(names, s.Name)
	}
	return strings.Join(names, "\\")
}

// phpFQN is the "\"-joined fully-qualified name of a qualified name.
func phpFQN(qn *ast.QualifiedName) string {
	names := make([]string, len(qn.Segments))
	for i, s := range qn.Segments {
		names[i] = s.Name
	}
	return strings.Join(names, "\\")
}

// emitClass writes a PHP class body with optional extends/implements clauses.
func (phpScaffolder) emitClass(w io.Writer, name string, extends *ast.QualifiedName, impls []*ast.QualifiedName, ms []ast.ClassMember) {
	header := "class " + name
	if extends != nil {
		header += " extends " + extends.Simple()
	}
	if len(impls) > 0 {
		names := make([]string, len(impls))
		for i, impl := range impls {
			names[i] = impl.Simple()
		}
		header += " implements " + strings.Join(names, ", ")
	}
	fmt.Fprintf(w, "%s {\n", header)
	for _, m := range ms {
		switch member := m.(type) {
		case *ast.PropertyMember:
			fmt.Fprintf(w, "\t%s %s $%s;\n", scaffold.VisLower(member.Vis), phpTypes.Render(member.Type), member.Name.Name)
		case *ast.MethodMember:
			fmt.Fprintf(w, "\t%s function %s(%s)%s {\n", scaffold.VisLower(member.Vis), member.Name.Name,
				phpParams(member.Params), phpResult(member.ReturnType))
			scaffold.WriteIntents(w, "\t\t", "// TODO: ", scaffold.Intents(member.Intents))
			fmt.Fprint(w, "\t\tthrow new \\Exception('not implemented');\n\t}\n")
		}
	}
	fmt.Fprint(w, "}\n")
}

// emitInterface writes a PHP interface body (public method signatures, no bodies).
func (phpScaffolder) emitInterface(w io.Writer, name string, ms []ast.ClassMember) {
	fmt.Fprintf(w, "interface %s {\n", name)
	for _, m := range ms {
		if method, ok := m.(*ast.MethodMember); ok {
			fmt.Fprintf(w, "\tpublic function %s(%s)%s;\n", method.Name.Name,
				phpParams(method.Params), phpResult(method.ReturnType))
		}
	}
	fmt.Fprint(w, "}\n")
}

// phpParams renders a comma-separated "type $name" list, naming anonymous params
// positionally ($arg0, $arg1, ...).
func phpParams(params []*ast.Parameter) string {
	parts := make([]string, len(params))
	for i, p := range params {
		parts[i] = phpTypes.Render(p.Type) + " $" + scaffold.ParamName(p, i)
	}
	return strings.Join(parts, ", ")
}

// phpResult renders a return type as ": T" or "" when there is none.
func phpResult(t ast.TypeRef) string {
	if t == nil {
		return ""
	}
	return ": " + phpTypes.Render(t)
}
