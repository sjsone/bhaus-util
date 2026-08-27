package lsp

import (
	"github.com/sjsone/bhaus-util/pkg/analysis"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// TextDocumentReferences handles the textDocument/references request.
func (h *Handler) TextDocumentReferences(context *glsp.Context, params *protocol.ReferenceParams) ([]protocol.Location, error) {
	uri := string(params.TextDocument.URI)
	line, col := params.Position.Line, params.Position.Character
	logger.Infof("references: %s @ %d:%d", uri, line, col)

	file, ok := h.Files[uri]
	if !ok {
		logger.Debugf("references: no parsed file for %s", uri)
		return []protocol.Location{}, nil
	}

	// Find the reference at the cursor using AST position
	ref, found := analysis.ReferenceAt(
		file, uri,
		uint32(line), uint32(col),
		h.Table,
	)
	if !found || ref.Sym == nil {
		logger.Debugf("references: no resolved symbol at %d:%d", line, col)
		return []protocol.Location{}, nil
	}

	// Collect ALL references to the resolved symbol across workspace
	allRefs := analysis.CollectReferences(h.Files, h.Table)
	refs := analysis.ReferencesTo(*ref.Sym, allRefs)

	includeDecl := params.Context.IncludeDeclaration
	logger.Debugf("references: symbol %s %q -> %d usage(s), includeDeclaration=%t",
		ref.Sym.Kind, ref.Sym.Name, len(refs), includeDecl)

	locations := make([]protocol.Location, 0, len(refs)+1)
	if includeDecl {
		locations = append(locations, symbolToLocation(*ref.Sym))
	}
	for _, r := range refs {
		locations = append(locations, referenceToLocation(r))
	}
	logger.Infof("references: %s %q -> %d location(s)", ref.Sym.Kind, ref.Sym.Name, len(locations))
	return locations, nil
}

func rangeForLocationFromSymbol(sym analysis.Symbol) protocol.Range {
	switch sym.Kind {
	// case analysis.KindClass:
	default:
		return protocol.Range{
			Start: protocol.Position{Line: sym.Span.Start.Line, Character: sym.Span.Start.Column},
			End:   protocol.Position{Line: sym.Span.End.Line, Character: sym.Span.End.Column},
		}
	}
}

func symbolToLocation(sym analysis.Symbol) protocol.Location {
	rangeForLocation := rangeForLocationFromSymbol(sym)
	return protocol.Location{
		URI:   sym.URI,
		Range: rangeForLocation,
	}
}

func referenceToLocation(ref analysis.Reference) protocol.Location {
	return protocol.Location{
		URI: ref.URI,
		Range: protocol.Range{
			Start: protocol.Position{Line: ref.Span.Start.Line, Character: ref.Span.Start.Column},
			End:   protocol.Position{Line: ref.Span.End.Line, Character: ref.Span.End.Column},
		},
	}
}
