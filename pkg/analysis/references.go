package analysis

import "github.com/sjsone/bhaus-util/pkg/ast"

// CollectReferences performs a full workspace reference walk. It collects
// every type usage, extends/implements/extern reference and C4 connection
// reference. It resolves each one against the symbol table.
func CollectReferences(files map[string]*ast.File, table *SymbolTable) []Reference {
	var refs []Reference

	for uri, file := range files {
		if file == nil {
			continue
		}
		walkFile(file, uri, nil,
			func(rs refSite) {
				ref := Reference{
					URI:  uri,
					Span: ast.SpanOf(rs.node),
					Kind: rs.kind,
					Site: rs.node,
				}
				syms := resolveSite(rs, table)
				if len(syms) == 1 {
					ref.Sym = &syms[0]
				}
				refs = append(refs, ref)
			},
		)
	}

	return refs
}

// resolveSite resolves a reference site to its target symbols.
func resolveSite(rs refSite, table *SymbolTable) []Symbol {
	switch n := rs.node.(type) {
	case *ast.NamedType:
		path := n.Name.String()
		if n.Name.IsPath() {
			if s, ok := table.LookupFull(path); ok {
				return []Symbol{s}
			}
			return nil
		}
		return table.LookupSimple(path)

	case *ast.QualifiedName:
		path := n.String()
		if n.IsPath() {
			if s, ok := table.LookupFull(path); ok {
				return []Symbol{s}
			}
			return nil
		}
		return table.LookupSimple(path)

	case *ast.C4Path:
		path := n.String()
		if s, ok := table.LookupC4(path); ok {
			return []Symbol{s}
		}
		return nil
	}
	return nil
}

// ReferencesTo returns all references to sym. It matches references by
// declaration pointer identity. It excludes definition sites.
func ReferencesTo(sym Symbol, refs []Reference) []Reference {
	var result []Reference
	for _, r := range refs {
		if r.Sym != nil && r.Sym.Decl == sym.Decl && !isDefinitionSite(r, sym) {
			result = append(result, r)
		}
	}
	return result
}

// isDefinitionSite reports whether ref is the defining occurrence of sym.
// This is true when the reference site node is the declaring node's own
// name node.
//
// Reference sites (type annotations, extends/implements, extern names, C4
// paths) and declaration names are distinct nodes in the AST. This makes
// pointer identity exact: only a site that IS the declaration's name counts
// as the definition.
func isDefinitionSite(ref Reference, sym Symbol) bool {
	if ref.URI != sym.URI {
		return false
	}
	switch d := sym.Decl.(type) {
	case *ast.ClassDecl:
		return ref.Site == d.Name
	case *ast.StructDecl:
		return ref.Site == d.Name
	case *ast.ProtocolDecl:
		return ref.Site == d.Name
	case *ast.FunctionDecl:
		return ref.Site == d.Name
	case *ast.PropertyMember:
		return ref.Site == d.Name
	case *ast.MethodMember:
		return ref.Site == d.Name
	case *ast.ExternDecl:
		return ref.Site == d.Name
	case *ast.SystemDecl:
		return ref.Site == d.Name
	case *ast.ContainerDecl:
		return ref.Site == d.Name
	case *ast.ComponentDecl:
		return ref.Site == d.Name
	case *ast.ConnectionDecl:
		return ref.Site == d.Source || ref.Site == d.Target
	}
	return false
}
