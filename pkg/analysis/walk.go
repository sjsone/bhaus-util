package analysis

import "github.com/sjsone/bhaus-util/pkg/ast"

// refSite is an unresolved reference encountered during the walk.
type refSite struct {
	node ast.Node // *ast.NamedType, *ast.QualifiedName (extends/implements) or *ast.C4Path
	kind RefKind
	uri  string
}

// walkFile performs a single semantic traversal of a file. It emits
// declared symbols and reference sites. Either callback may be nil.
func walkFile(
	file *ast.File,
	uri string,
	onSymbol func(Symbol),
	onRef func(refSite),
) {
	if onSymbol == nil {
		onSymbol = func(Symbol) {}
	}
	if onRef == nil {
		onRef = func(refSite) {}
	}
	for _, d := range file.Decls {
		walkDecl(d, uri, "", onSymbol, onRef)
	}
}

func walkDecl(d ast.Decl, uri, c4Parent string, onSymbol func(Symbol), onRef func(refSite)) {
	switch d := d.(type) {
	case *ast.ClassDecl:
		full := d.Name.String()
		onSymbol(Symbol{
			Kind: KindClass, URI: uri,
			Name: d.Name.Simple(), FullName: full,
			Span: ast.SpanOf(d.Name), Decl: d,
		})
		if d.Extends != nil {
			onRef(refSite{d.Extends, RefExtends, uri})
		}
		for _, imp := range d.Implements {
			onRef(refSite{imp, RefImplements, uri})
		}
		for _, m := range d.Members {
			walkMember(m, full, uri, onSymbol, onRef)
		}

	case *ast.StructDecl:
		full := d.Name.String()
		onSymbol(Symbol{
			Kind: KindStruct, URI: uri,
			Name: d.Name.Simple(), FullName: full,
			Span: ast.SpanOf(d.Name), Decl: d,
		})
		for _, m := range d.Members {
			walkMember(m, full, uri, onSymbol, onRef)
		}

	case *ast.ProtocolDecl:
		full := d.Name.String()
		onSymbol(Symbol{
			Kind: KindProtocol, URI: uri,
			Name: d.Name.Simple(), FullName: full,
			Span: ast.SpanOf(d.Name), Decl: d,
		})
		for _, m := range d.Members {
			walkMember(m, full, uri, onSymbol, onRef)
		}

	case *ast.FunctionDecl:
		full := d.Name.String()
		onSymbol(Symbol{
			Kind: KindFunction, URI: uri,
			Name: d.Name.Simple(), FullName: full,
			Span: ast.SpanOf(d.Name), Decl: d,
		})
		for _, p := range d.Params {
			walkTypeRef(p.Type, uri, onRef)
		}
		if d.Result != nil {
			walkTypeRef(d.Result, uri, onRef)
		}

	case *ast.SystemDecl:
		c4Path := d.Name.Name
		onSymbol(Symbol{
			Kind: KindSystem, URI: uri,
			Name: d.Name.Name, FullName: c4Path,
			Span: ast.SpanOf(d.Name), Decl: d,
		})
		for _, ct := range d.Containers {
			walkContainer(ct, c4Path, uri, onSymbol, onRef)
		}
		for _, cs := range d.Connections {
			onRef(refSite{cs.Target, RefC4Shorthand, uri})
		}

	case *ast.ConnectionDecl:
		name := d.String()
		onSymbol(Symbol{
			Kind: KindConnection, URI: uri,
			Name: name, FullName: name,
			Span: ast.SpanOf(d), Decl: d,
		})
		onRef(refSite{d.Source, RefC4Source, uri})
		onRef(refSite{d.Target, RefC4Target, uri})

	case *ast.ExternDecl:
		full := d.Name.String()
		onSymbol(Symbol{
			Kind: KindExtern, URI: uri,
			Name: d.Name.Simple(), FullName: full,
			Span: ast.SpanOf(d.Name), Decl: d,
		})
		onRef(refSite{d.Name, RefExtern, uri})

	case *ast.VersionDecl:
		// Emit a symbol for VERSION so it can be hovered and validated.
		onSymbol(Symbol{
			Kind:     KindVersion,
			URI:      uri,
			Name:     "VERSION",
			FullName: "VERSION",
			Span:     ast.SpanOf(d),
			Decl:     d,
		})

		// IncludeDecl, Comment: no symbols, no references
	}
}

func walkContainer(ct *ast.ContainerDecl, parentC4, uri string, onSymbol func(Symbol), onRef func(refSite)) {
	c4Path := parentC4 + "." + ct.Name.Name
	onSymbol(Symbol{
		Kind: KindContainer, URI: uri,
		Name: ct.Name.Name, FullName: c4Path,
		Span: ast.SpanOf(ct.Name), Decl: ct,
	})
	for _, cp := range ct.Components {
		walkComponent(cp, c4Path, uri, onSymbol, onRef)
	}
	for _, cs := range ct.Connections {
		onRef(refSite{cs.Target, RefC4Shorthand, uri})
	}
}

func walkComponent(cp *ast.ComponentDecl, parentC4, uri string, onSymbol func(Symbol), onRef func(refSite)) {
	c4Path := parentC4 + "." + cp.Name.Name
	onSymbol(Symbol{
		Kind: KindComponent, URI: uri,
		Name: cp.Name.Name, FullName: c4Path,
		Span: ast.SpanOf(cp.Name), Decl: cp,
	})
	for _, cs := range cp.Connections {
		onRef(refSite{cs.Target, RefC4Shorthand, uri})
	}
}

func walkMember(m any, containerFullName, uri string, onSymbol func(Symbol), onRef func(refSite)) {
	switch m := m.(type) {
	case *ast.PropertyMember:
		full := containerFullName + "/" + m.Name.Name
		onSymbol(Symbol{
			Kind: KindProperty, URI: uri,
			Name: m.Name.Name, FullName: full,
			Span: ast.SpanOf(m.Name), Decl: m,
		})
		walkTypeRef(m.Type, uri, onRef)

	case *ast.MethodMember:
		full := containerFullName + "/" + m.Name.Name
		onSymbol(Symbol{
			Kind: KindMethod, URI: uri,
			Name: m.Name.Name, FullName: full,
			Span: ast.SpanOf(m.Name), Decl: m,
		})
		for _, p := range m.Params {
			walkTypeRef(p.Type, uri, onRef)
		}
		if m.ReturnType != nil {
			walkTypeRef(m.ReturnType, uri, onRef)
		}
	}
}

func walkTypeRef(tr ast.TypeRef, uri string, onRef func(refSite)) {
	if tr == nil {
		return
	}
	switch tr := tr.(type) {
	case *ast.NamedType:
		onRef(refSite{tr, RefType, uri})
	case *ast.ArrayType:
		walkTypeRef(tr.Elem, uri, onRef)
	case *ast.OptionalType:
		walkTypeRef(tr.Inner, uri, onRef)
	case *ast.UnionType:
		if tr.Left != nil {
			walkTypeRef(tr.Left, uri, onRef)
		}
		if tr.Right != nil {
			walkTypeRef(tr.Right, uri, onRef)
		}
	}
}
