package lsp

import (
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// WorkspaceSymbol handles the workspace/symbol request.
func (h *Handler) WorkspaceSymbol(context *glsp.Context, params *protocol.WorkspaceSymbolParams) ([]protocol.SymbolInformation, error) {
	query := params.Query
	logger.Infof("workspaceSymbol: query=%q", query)
	if query == "" {
		return []protocol.SymbolInformation{}, nil
	}

	symbols := h.Table.Search(query)
	logger.Infof("workspaceSymbol: query=%q -> %d symbol(s)", query, len(symbols))
	result := make([]protocol.SymbolInformation, 0, len(symbols))
	for _, sym := range symbols {
		result = append(result, protocol.SymbolInformation{
			Name: sym.Name,
			Kind: toLSPKind(sym.Kind),
			Location: protocol.Location{
				URI:   sym.URI,
				Range: spanToRange(sym.Span),
			},
		})
	}
	return result, nil
}
