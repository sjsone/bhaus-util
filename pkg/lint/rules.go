package lint

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/sjsone/bhaus-util/pkg/analysis"
	"github.com/sjsone/bhaus-util/pkg/ast"
)

// ruleSyntax surfaces tree-sitter parse errors (ERROR / MISSING nodes) that the
// parser already collected on the file.
func ruleSyntax(c *ctx) []Diagnostic {
	var out []Diagnostic
	for _, e := range c.file.Errors {
		sev := Error
		if e.Severity == "Warning" {
			sev = Warning
		}
		msg := e.Message
		if msg == "" {
			msg = "syntax error"
		}
		out = append(out, Diagnostic{
			URI: c.uri, Span: e.Span, Severity: sev,
			Rule: "syntax", Message: msg,
		})
	}
	return out
}

// ruleUnresolved flags reference sites in the target file that do not resolve
// to exactly one declaration. A reference site is a type annotation, an
// EXTENDS/IMPLEMENTS target or a C4 endpoint. It is unresolved if it names
// nothing (undefined) or names several things (ambiguous).
//
// Built-in simple types (String, Integer, Unknown and similar) are
// *ast.SimpleType, not references. They never reach this rule, so built-in
// types never cause false positives.
func ruleUnresolved(c *ctx) []Diagnostic {
	var out []Diagnostic
	for _, r := range c.refs {
		if r.URI != c.uri || r.Sym != nil {
			continue // wrong file or resolved to exactly one symbol
		}
		name, count := c.resolveCount(r)
		var msg string
		if count > 1 {
			msg = fmt.Sprintf("reference to %q is ambiguous (%d declarations)", name, count)
		} else {
			msg = fmt.Sprintf("cannot resolve reference to %q", name)
		}
		out = append(out, Diagnostic{
			URI: c.uri, Span: r.Span, Severity: Error,
			Rule: "unresolved-ref", Message: msg,
		})
	}
	return out
}

// resolveCount mirrors analysis.resolveSite. It uses the table's exported
// lookups. It returns the referenced name and how many declarations match it.
// A count other than 1 explains why the rule left the reference unresolved.
func (c *ctx) resolveCount(r analysis.Reference) (string, int) {
	switch n := r.Site.(type) {
	case *ast.NamedType:
		return c.countName(n.Name)
	case *ast.QualifiedName:
		return c.countName(n)
	case *ast.C4Path:
		path := n.String()
		if _, ok := c.table.LookupC4(path); ok {
			return path, 1
		}
		return path, 0
	}
	return "", 0
}

func (c *ctx) countName(q *ast.QualifiedName) (string, int) {
	path := q.String()
	if q.IsPath() {
		if _, ok := c.table.LookupFull(path); ok {
			return path, 1
		}
		return path, 0
	}
	return path, len(c.table.LookupSimple(path))
}

// ruleDuplicate flags declarations in the target file whose full name appears
// more than once across the whole fileset. For example, Domain/User might be
// defined in two files or twice in one file. VERSION is excluded because each
// file has its own version metadata.
func ruleDuplicate(c *ctx) []Diagnostic {
	var out []Diagnostic
	for _, sym := range c.table.SymbolsForURI(c.uri) {
		// VERSION is per-file metadata; duplicates across files are expected.
		if sym.Kind == analysis.KindVersion {
			continue
		}
		all := c.table.LookupAllFull(sym.FullName)
		if len(all) < 2 {
			continue
		}
		msg := fmt.Sprintf("%s %q is declared %d times%s",
			sym.Kind, sym.FullName, len(all), otherLocations(all, sym))
		out = append(out, Diagnostic{
			URI: c.uri, Span: sym.Span, Severity: Error,
			Rule: "duplicate-decl", Message: msg,
		})
	}
	return out
}

// otherLocations describes where else a duplicated symbol appears. It names
// other files by base name. For a same-file duplicate, it returns a short
// summary instead of a location, because the second diagnostic on the other
// declaration already covers that case.
func otherLocations(all []analysis.Symbol, self analysis.Symbol) string {
	seen := make(map[string]bool)
	var files []string
	sameFile := false
	for _, s := range all {
		if s.Decl == self.Decl {
			continue // this very declaration
		}
		if s.URI == self.URI {
			sameFile = true
			continue
		}
		base := filepath.Base(strings.TrimPrefix(s.URI, "file://"))
		if seen[base] {
			continue
		}
		seen[base] = true
		files = append(files, base)
	}
	if len(files) > 0 {
		return " (also in " + strings.Join(files, ", ") + ")"
	}
	if sameFile {
		return " (declared more than once in this file)"
	}
	return ""
}

// ruleNaming enforces the casing conventions: type names start uppercase,
// function and method names start lowercase. Names beginning with an
// underscore or a non-letter are not classifiable. The rule leaves them
// alone. Warnings only.
func ruleNaming(c *ctx) []Diagnostic {
	var out []Diagnostic
	upper := func(name string, span ast.Span, kind string) {
		if d := wantCase(name, span, kind, true); d != nil {
			d.URI = c.uri
			out = append(out, *d)
		}
	}
	lower := func(name string, span ast.Span, kind string) {
		if d := wantCase(name, span, kind, false); d != nil {
			d.URI = c.uri
			out = append(out, *d)
		}
	}

	for _, d := range c.file.Decls {
		switch d := d.(type) {
		case *ast.ClassDecl:
			upper(d.Name.Simple(), ast.SpanOf(d.Name), "class")
			for _, m := range d.Members {
				if mm, ok := m.(*ast.MethodMember); ok {
					lower(mm.Name.Name, ast.SpanOf(mm.Name), "method")
				}
			}
		case *ast.StructDecl:
			upper(d.Name.Simple(), ast.SpanOf(d.Name), "struct")
			for _, m := range d.Members {
				if mm, ok := m.(*ast.MethodMember); ok {
					lower(mm.Name.Name, ast.SpanOf(mm.Name), "method")
				}
			}
		case *ast.ProtocolDecl:
			upper(d.Name.Simple(), ast.SpanOf(d.Name), "protocol")
			for _, m := range d.Members {
				if mm, ok := m.(*ast.MethodMember); ok {
					lower(mm.Name.Name, ast.SpanOf(mm.Name), "method")
				}
			}
		case *ast.FunctionDecl:
			lower(d.Name.Simple(), ast.SpanOf(d.Name), "function")
		case *ast.SystemDecl:
			upper(d.Name.Name, ast.SpanOf(d.Name), "system")
			for _, ct := range d.Containers {
				upper(ct.Name.Name, ast.SpanOf(ct.Name), "container")
				for _, cp := range ct.Components {
					upper(cp.Name.Name, ast.SpanOf(cp.Name), "component")
				}
			}
		}
	}
	return out
}

// wantCase returns a naming diagnostic if name's first letter has the wrong
// case or nil if it is fine or not classifiable.
func wantCase(name string, span ast.Span, kind string, wantUpper bool) *Diagnostic {
	if name == "" {
		return nil
	}
	first := []rune(name)[0]
	if !unicode.IsLetter(first) {
		return nil // e.g., a leading underscore: valid, with no case to check
	}
	if wantUpper && !unicode.IsUpper(first) {
		return &Diagnostic{
			URI: "", Span: span, Severity: Warning, Rule: "naming",
			Message: fmt.Sprintf("%s name %q should start with an uppercase letter", kind, name),
		}
	}
	if !wantUpper && !unicode.IsLower(first) {
		return &Diagnostic{
			URI: "", Span: span, Severity: Warning, Rule: "naming",
			Message: fmt.Sprintf("%s name %q should start with a lowercase letter", kind, name),
		}
	}
	return nil
}

// ruleUnknownType flags every use of the built-in Unknown type. This type
// signals that a type annotation is unspecified rather than deliberately
// chosen. The rule walks the whole target file, so it catches occurrences
// nested inside Array[], ?Optional and unions, not just top-level type
// annotations. Warnings only.
func ruleUnknownType(c *ctx) []Diagnostic {
	var out []Diagnostic
	ast.Inspect(c.file, func(n ast.Node) bool {
		if st, ok := n.(*ast.SimpleType); ok && st.Name == "Unknown" {
			out = append(out, Diagnostic{
				URI: c.uri, Span: ast.SpanOf(st), Severity: Warning, Rule: "unknown-type",
				Message: `type "Unknown" should be avoided; consider a specific type`,
			})
		}
		return true
	})
	return out
}

// ruleStructure checks document-level shape: a VERSION declaration must be
// present and must be the first declaration. Warnings only.
func ruleStructure(c *ctx) []Diagnostic {
	versionIdx := -1
	for i, d := range c.file.Decls {
		if _, ok := d.(*ast.VersionDecl); ok {
			versionIdx = i
			break
		}
	}

	if versionIdx == -1 {
		span := ast.Span{}
		if len(c.file.Decls) > 0 {
			span = ast.SpanOf(c.file.Decls[0])
		}
		return []Diagnostic{{
			URI: c.uri, Span: span, Severity: Warning, Rule: "structure",
			Message: "missing VERSION declaration",
		}}
	}
	if versionIdx != 0 {
		return []Diagnostic{{
			URI: c.uri, Span: ast.SpanOf(c.file.Decls[versionIdx]), Severity: Warning, Rule: "structure",
			Message: "VERSION should be the first declaration",
		}}
	}
	return nil
}
