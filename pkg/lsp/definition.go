package lsp

import (
	"github.com/sjsone/bhaus-util/pkg/analysis"
	"github.com/sjsone/bhaus-util/pkg/util"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// TextDocumentDefinition handles the textDocument/definition request.
func (h *Handler) TextDocumentDefinition(context *glsp.Context, params *protocol.DefinitionParams) (any, error) {
	uri := string(params.TextDocument.URI)
	line, col := params.Position.Line, params.Position.Character
	logger.Infof("definition: %s @ %d:%d", uri, line, col)

	file, ok := h.Files[uri]
	if !ok {
		logger.Debugf("definition: no parsed file for %s", uri)
		return nil, nil
	}

	// Quick check: is the word a built-in type? Skip if so
	if content, ok := h.Documents[uri]; ok {
		word := util.GetWordAtPosition(content, int(line), int(col))
		logger.Debugf("definition: word=%q", word)
		if word == "" || IsBuiltInType(word) {
			logger.Debugf("definition: skipping (empty word or built-in type)")
			return nil, nil
		}
	}

	syms, originSpan, found := analysis.DefinitionSitesAt(file, uri, uint32(line), uint32(col), h.Table)
	if !found {
		logger.Infof("definition: %s @ %d:%d -> 0 location(s)", uri, line, col)
		return nil, nil
	}
	for _, sym := range syms {
		logger.Debugf("definition: resolved %s %q in %s", sym.Kind, sym.Name, sym.URI)
	}

	// When the client supports it, return LocationLink[] with an origin
	// selection range covering the whole name under the cursor (e.g. the
	// entire "Base/Bla"). This makes cmd+click highlight the full name, not
	// just one token.
	if h.definitionLinkSupport {
		origin := spanToRange(originSpan)
		links := make([]protocol.LocationLink, 0, len(syms))
		for _, sym := range syms {
			target := spanToRange(sym.Span)
			links = append(links, protocol.LocationLink{
				OriginSelectionRange: &origin,
				TargetURI:            sym.URI,
				TargetRange:          target,
				TargetSelectionRange: target,
			})
		}
		logger.Infof("definition: %s @ %d:%d -> %d link(s)", uri, line, col, len(links))
		return links, nil
	}

	locations := make([]protocol.Location, 0, len(syms))
	for _, sym := range syms {
		locations = append(locations, symbolToLocation(sym))
	}
	logger.Infof("definition: %s @ %d:%d -> %d location(s)", uri, line, col, len(locations))
	return locations, nil
}
