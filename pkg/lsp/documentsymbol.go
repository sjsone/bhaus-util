package lsp

import (
	"strings"

	"github.com/sjsone/bhaus-util/pkg/analysis"
	"github.com/sjsone/bhaus-util/pkg/ast"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// TextDocumentDocumentSymbol handles the textDocument/documentSymbol request.
func (h *Handler) TextDocumentDocumentSymbol(context *glsp.Context, params *protocol.DocumentSymbolParams) (any, error) {
	uri := string(params.TextDocument.URI)
	logger.Infof("documentSymbol: %s", uri)

	symbols := h.Table.SymbolsForURI(uri)
	if len(symbols) == 0 {
		logger.Debugf("documentSymbol: no symbols for %s", uri)
		return []protocol.DocumentSymbol{}, nil
	}

	result := buildDocumentSymbols(symbols)
	logger.Infof("documentSymbol: %s -> %d top-level symbol(s)", uri, len(result))
	return result, nil
}

func buildDocumentSymbols(symbols []analysis.Symbol) []protocol.DocumentSymbol {
	var result []protocol.DocumentSymbol

	for _, sym := range symbols {
		if sym.Kind == analysis.KindMethod || sym.Kind == analysis.KindProperty {
			continue // children only
		}
		ds := symbolToDocumentSymbol(sym, symbols)
		result = append(result, ds)
	}
	return result
}

func symbolToDocumentSymbol(sym analysis.Symbol, allSymbols []analysis.Symbol) protocol.DocumentSymbol {
	kind := toLSPKind(sym.Kind)
	children := findMemberSymbols(sym, allSymbols)

	var childrenDS []protocol.DocumentSymbol
	for _, c := range children {
		childrenDS = append(childrenDS, protocol.DocumentSymbol{
			Name:           c.Name,
			Kind:           toLSPKind(c.Kind),
			Range:          spanToRange(c.Span),
			SelectionRange: spanToRange(c.Span),
		})
	}

	return protocol.DocumentSymbol{
		Name:           sym.Name,
		Kind:           kind,
		Range:          spanToRange(sym.Span),
		SelectionRange: spanToRange(sym.Span),
		Children:       childrenDS,
	}
}

func findMemberSymbols(sym analysis.Symbol, allSymbols []analysis.Symbol) []analysis.Symbol {
	var result []analysis.Symbol
	prefix := sym.FullName + "/"
	for _, s := range allSymbols {
		if s.FullName != "" && strings.HasPrefix(s.FullName, prefix) {
			rest := s.FullName[len(prefix):]
			if !strings.Contains(rest, "/") {
				result = append(result, s)
			}
		}
	}
	return result
}

func spanToRange(sp ast.Span) protocol.Range {
	return protocol.Range{
		Start: protocol.Position{Line: sp.Start.Line, Character: sp.Start.Column},
		End:   protocol.Position{Line: sp.End.Line, Character: sp.End.Column},
	}
}

func toLSPKind(k analysis.Kind) protocol.SymbolKind {
	switch k {
	case analysis.KindClass:
		return protocol.SymbolKindClass
	case analysis.KindStruct:
		return protocol.SymbolKindStruct
	case analysis.KindProtocol:
		return protocol.SymbolKindInterface
	case analysis.KindMethod:
		return protocol.SymbolKindMethod
	case analysis.KindFunction:
		return protocol.SymbolKindFunction
	case analysis.KindProperty:
		return protocol.SymbolKindProperty
	case analysis.KindSystem:
		return protocol.SymbolKindNamespace
	case analysis.KindContainer:
		return protocol.SymbolKindConstant
	case analysis.KindComponent:
		return protocol.SymbolKindProperty
	case analysis.KindConnection:
		return protocol.SymbolKindOperator
	}
	return protocol.SymbolKindVariable
}
