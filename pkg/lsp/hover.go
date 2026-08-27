package lsp

import (
	"fmt"
	"strings"

	"github.com/sjsone/bhaus-util/pkg/analysis"
	"github.com/sjsone/bhaus-util/pkg/ast"
	"github.com/sjsone/bhaus-util/pkg/util"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// TextDocumentHover handles the textDocument/hover request.
func (h *Handler) TextDocumentHover(context *glsp.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	uri := string(params.TextDocument.URI)
	position := params.Position
	logger.Infof("hover: %s @ %d:%d", uri, position.Line, position.Character)

	content, ok := h.Documents[uri]
	if !ok {
		logger.Debugf("hover: no open document for %s", uri)
		return nil, nil
	}

	word := util.GetWordAtPosition(content, int(position.Line), int(position.Character))
	logger.Debugf("hover: word=%q", word)
	if word == "" {
		return nil, nil
	}

	// Check built-in types first
	if IsBuiltInType(word) {
		logger.Debugf("hover: %q is a built-in type", word)
		return &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: fmt.Sprintf("**type** %s", word),
			},
		}, nil
	}

	file, ok := h.Files[uri]
	if !ok {
		logger.Debugf("hover: no parsed file for %s", uri)
		return nil, nil
	}

	sym, span, found := analysis.HoverInfo(file, uri,
		uint32(position.Line), uint32(position.Character),
		h.Table)
	if !found {
		logger.Debugf("hover: no symbol at %d:%d (word=%q)", position.Line, position.Character, word)
		return nil, nil
	}
	logger.Infof("hover: %s %q at %d:%d", sym.Kind, sym.Name, position.Line, position.Character)

	value := hoverMarkdown(sym)

	// Append the symbol's doc comment: the standalone comment block directly
	// above its declaration. The function looks it up in the file that
	// declares the symbol. That file may differ from the file being hovered.
	if doc := analysis.DocComment(h.Files[sym.URI], h.Documents[sym.URI], sym); doc != "" {
		value += "\n\n---\n\n" + doc
	}

	rng := spanToRange(span)
	return &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: value,
		},
		Range: &rng,
	}, nil
}

// hoverMarkdown renders one symbol as a fenced bhaus code block containing
// the symbol's declaration. Every hover reads like the source it came from.
// Types append their members indented under the header line.
func hoverMarkdown(sym analysis.Symbol) string {
	switch sym.Kind {
	case analysis.KindClass:
		var s strings.Builder
		s.WriteString("```bhaus\nCLASS " + sym.FullName)
		if cd, ok := sym.Decl.(*ast.ClassDecl); ok {
			if cd.Extends != nil {
				s.WriteString(" EXTENDS " + cd.Extends.String())
			}
			for _, imp := range cd.Implements {
				s.WriteString(" IMPLEMENTS " + imp.String())
			}
		}
		if ml := memberLines(sym); ml != "" {
			s.WriteString(":" + ml)
		}
		return s.String() + "\n```"
	case analysis.KindStruct:
		s := "```bhaus\nSTRUCT " + sym.FullName
		if ml := memberLines(sym); ml != "" {
			s += ":" + ml
		}
		return s + "\n```"
	case analysis.KindProtocol:
		s := "```bhaus\nPROTOCOL " + sym.FullName
		if ml := memberLines(sym); ml != "" {
			s += ":" + ml
		}
		return s + "\n```"
	case analysis.KindMethod:
		var s strings.Builder
		s.WriteString("```bhaus\n")
		if mm, ok := sym.Decl.(*ast.MethodMember); ok {
			if mm.Override {
				s.WriteString("OVERRIDE ")
			}
			s.WriteString(visibilityString(mm.Vis) + " " + mm.Name.Name + "(")
			for i, p := range mm.Params {
				if i > 0 {
					s.WriteString(", ")
				}
				if p.Name != nil {
					s.WriteString(p.Name.Name + " " + typeToString(p.Type))
				} else {
					s.WriteString(typeToString(p.Type))
				}
			}
			s.WriteString(")")
			if mm.ReturnType != nil {
				s.WriteString(": " + typeToString(mm.ReturnType))
			}
			for _, it := range mm.Intents {
				s.WriteString("\n    > " + it.Text)
			}
		}
		return s.String() + "\n```"
	case analysis.KindProperty:
		s := "```bhaus\n"
		if pm, ok := sym.Decl.(*ast.PropertyMember); ok {
			s += visibilityString(pm.Vis) + " " + pm.Name.Name + ": " + typeToString(pm.Type)
		}
		return s + "\n```"
	case analysis.KindFunction:
		var s strings.Builder
		s.WriteString("```bhaus\nFUNCTION " + sym.FullName)
		if fd, ok := sym.Decl.(*ast.FunctionDecl); ok {
			s.WriteString("(")
			for i, p := range fd.Params {
				if i > 0 {
					s.WriteString(", ")
				}
				if p.Name != nil {
					s.WriteString(p.Name.Name + " " + typeToString(p.Type))
				} else {
					s.WriteString(typeToString(p.Type))
				}
			}
			s.WriteString(")")
			if fd.Result != nil {
				s.WriteString(": " + typeToString(fd.Result))
			}
			for _, it := range fd.Intents {
				s.WriteString("\n    > " + it.Text)
			}
		}
		return s.String() + "\n```"
	case analysis.KindSystem:
		s := "```bhaus\nSYSTEM " + sym.FullName
		if sd, ok := sym.Decl.(*ast.SystemDecl); ok && sd.Description != "" {
			s += " \"" + sd.Description + "\""
		}
		return s + "\n```"
	case analysis.KindContainer:
		s := "```bhaus\nCONTAINER " + sym.FullName
		if cd, ok := sym.Decl.(*ast.ContainerDecl); ok && cd.Description != "" {
			s += " \"" + cd.Description + "\""
		}
		return s + "\n```"
	case analysis.KindComponent:
		s := "```bhaus\nCOMPONENT " + sym.FullName
		if cd, ok := sym.Decl.(*ast.ComponentDecl); ok && cd.Description != "" {
			s += " \"" + cd.Description + "\""
		}
		return s + "\n```"
	case analysis.KindConnection:
		return "```bhaus\nCONNECTION " + sym.Name + "\n```"
	case analysis.KindExtern:
		return "```bhaus\nEXTERN " + sym.FullName + "\n```"
	case analysis.KindVersion:
		if vd, ok := sym.Decl.(*ast.VersionDecl); ok {
			return "```bhaus\nVERSION " + vd.Version + "\n```"
		}
		return "```bhaus\nVERSION\n```"
	}
	return "```bhaus\n" + strings.ToUpper(sym.Kind.String()) + " " + sym.Name + "\n```"
}

// memberLines renders the declared members of a protocol, class or struct
// as source lines indented under the declaration header. It returns "" when
// the declaration has no members. Members carry their visibility keyword.
// This makes the block read like the declaration body itself.
func memberLines(sym analysis.Symbol) string {
	var lines []string
	switch d := sym.Decl.(type) {
	case *ast.ProtocolDecl:
		for _, m := range d.Members {
			lines = append(lines, memberLine(m))
		}
	case *ast.ClassDecl:
		for _, m := range d.Members {
			lines = append(lines, memberLine(m))
		}
	case *ast.StructDecl:
		for _, m := range d.Members {
			lines = append(lines, memberLine(m))
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return "\n    " + strings.Join(lines, "\n    ")
}

// memberLine renders one member as a declaration line, e.g.
// "PUBLIC getIdentifier(): UUID".
func memberLine(m ast.Node) string {
	switch m := m.(type) {
	case *ast.PropertyMember:
		return visibilityString(m.Vis) + " " + m.Name.Name + ": " + typeToString(m.Type)
	case *ast.MethodMember:
		var line strings.Builder
		if m.Override {
			line.WriteString("OVERRIDE ")
		}
		line.WriteString(visibilityString(m.Vis) + " " + m.Name.Name + "(")
		for i, p := range m.Params {
			if i > 0 {
				line.WriteString(", ")
			}
			if p.Name != nil {
				line.WriteString(p.Name.Name + " " + typeToString(p.Type))
			} else {
				line.WriteString(typeToString(p.Type))
			}
		}
		line.WriteString(")")
		if m.ReturnType != nil {
			line.WriteString(": " + typeToString(m.ReturnType))
		}
		return line.String()
	}
	return ""
}

func typeToString(tr ast.TypeRef) string {
	if tr == nil {
		return "?"
	}
	switch t := tr.(type) {
	case *ast.SimpleType:
		return t.Name
	case *ast.NamedType:
		return t.Name.String()
	case *ast.ArrayType:
		return "Array[" + typeToString(t.Elem) + "]"
	case *ast.BitsType:
		return fmt.Sprintf("Bits<%d>", t.Width.Value)
	case *ast.OptionalType:
		return "?" + typeToString(t.Inner)
	case *ast.UnionType:
		return typeToString(t.Left) + "|" + typeToString(t.Right)
	}
	return "?"
}

func visibilityString(v ast.Visibility) string {
	switch v {
	case ast.VisibilityPublic:
		return "PUBLIC"
	case ast.VisibilityPrivate:
		return "PRIVATE"
	case ast.VisibilityProtected:
		return "PROTECTED"
	}
	return ""
}
