package analysis

import (
	"slices"

	"github.com/sjsone/bhaus-util/pkg/ast"
)

// ReferenceAt returns the resolved reference at a cursor position, if any.
func ReferenceAt(file *ast.File, uri string, line, column uint32, table *SymbolTable) (*Reference, bool) {
	path := ast.PathTo(file, line, column)
	if len(path) < 1 {
		return nil, false
	}

	for i := range slices.Backward(path) {
		if rs, ok := refSiteFromPath(path, i); ok {
			syms := resolveSite(rs, table)
			ref := &Reference{
				URI:  uri,
				Span: ast.SpanOf(rs.node),
				Kind: rs.kind,
				Site: rs.node,
			}
			if len(syms) == 1 {
				ref.Sym = &syms[0]
			}
			return ref, true
		}
	}
	return nil, false
}

// refSiteFromPath classifies path[i] by inspecting it and its parent.
func refSiteFromPath(path []ast.Node, i int) (refSite, bool) {
	n := path[i]

	switch n := n.(type) {
	case *ast.NamedType:
		return refSite{node: n, kind: RefType}, true
	case *ast.QualifiedName:
		if i == 0 {
			return refSite{}, false
		}
		parent := path[i-1]
		switch parent := parent.(type) {
		case *ast.NamedType:
			return refSite{node: parent, kind: RefType}, true
		case *ast.ClassDecl:
			if parent.Extends == n {
				return refSite{node: n, kind: RefExtends}, true
			}
			if slices.Contains(parent.Implements, n) {
				return refSite{node: n, kind: RefImplements}, true
			}
		case *ast.ExternDecl:
			if parent.Name == n {
				return refSite{node: n, kind: RefExtern}, true
			}
		}
		return refSite{}, false
	case *ast.C4Path:
		if i == 0 {
			return refSite{}, false
		}
		parent := path[i-1]
		switch parent := parent.(type) {
		case *ast.ConnectionDecl:
			if parent.Source == n {
				return refSite{node: n, kind: RefC4Source}, true
			}
			if parent.Target == n {
				return refSite{node: n, kind: RefC4Target}, true
			}
		case *ast.ConnectionShorthand:
			return refSite{node: n, kind: RefC4Target}, true
		}
	}
	return refSite{}, false
}

// DefinitionSitesAt returns the definition target symbol(s) at the cursor,
// together with the source span of the name under the cursor. This span
// covers the whole qualified name (e.g. "Base/Bla"). It suits use as a
// LocationLink origin, so the editor highlights the entire name rather than
// a single token.
func DefinitionSitesAt(file *ast.File, uri string, line, column uint32, table *SymbolTable) ([]Symbol, ast.Span, bool) {
	// A reference site (type reference, EXTENDS/IMPLEMENTS, etc.): its span
	// covers the full name as written at the use site.
	if ref, ok := ReferenceAt(file, uri, line, column, table); ok && ref.Sym != nil {
		return []Symbol{*ref.Sym}, ref.Span, true
	}

	// Otherwise a declaration name at the cursor; span it from the name node.
	path := ast.PathTo(file, line, column)
	for i, p := range slices.Backward(path) {
		if sym, ok := declAtPath(path, i, uri, table); ok {
			return []Symbol{sym}, ast.SpanOf(p), true
		}
	}
	return nil, ast.Span{}, false
}

// DefinitionsAt returns the symbol(s) the cursor position refers to.
func DefinitionsAt(file *ast.File, uri string, line, column uint32, table *SymbolTable) []Symbol {
	syms, _, _ := DefinitionSitesAt(file, uri, line, column, table)
	return syms
}

func declAtPath(path []ast.Node, i int, uri string, table *SymbolTable) (Symbol, bool) {
	n := path[i]
	switch n := n.(type) {
	case *ast.VersionDecl:
		// VERSION declarations are matched directly by their node
		syms := table.LookupSimple("VERSION")
		for _, s := range syms {
			if s.URI == uri && ast.Contains(s.Span, n.Pos()) {
				return s, true
			}
		}
		return Symbol{}, false
	case *ast.QualifiedName:
		if i == 0 {
			return Symbol{}, false
		}
		switch path[i-1].(type) {
		case *ast.ClassDecl, *ast.StructDecl, *ast.ProtocolDecl,
			*ast.FunctionDecl, *ast.ExternDecl:
			full := n.String()
			if s, ok := table.LookupFull(full); ok {
				return s, true
			}
			syms := table.LookupSimple(n.Simple())
			for _, s := range syms {
				if s.URI == uri {
					return s, true
				}
			}
		}
	case *ast.Ident:
		if i == 0 {
			return Symbol{}, false
		}
		switch path[i-1].(type) {
		case *ast.PropertyMember, *ast.MethodMember,
			*ast.SystemDecl, *ast.ContainerDecl, *ast.ComponentDecl:
			name := n.Name
			for _, s := range table.LookupSimple(name) {
				if s.URI == uri && ast.Contains(s.Span, n.Pos()) {
					return s, true
				}
			}
		}
	}
	return Symbol{}, false
}

// HoverSymbol returns the symbol the cursor position refers to.
func HoverSymbol(file *ast.File, uri string, line, column uint32, table *SymbolTable) (Symbol, bool) {
	sym, _, ok := HoverInfo(file, uri, line, column, table)
	return sym, ok
}

// HoverInfo returns the symbol at the cursor, together with the source span
// of the name under the cursor. This span is the reference site (e.g. an
// entire "Base/Bla" type reference) or the declaration name. It lets the
// hover highlight the whole qualified name, rather than just the token
// beneath the cursor.
func HoverInfo(file *ast.File, uri string, line, column uint32, table *SymbolTable) (Symbol, ast.Span, bool) {
	syms, span, ok := DefinitionSitesAt(file, uri, line, column, table)
	if !ok || len(syms) == 0 {
		return Symbol{}, ast.Span{}, false
	}
	return syms[0], span, true
}
