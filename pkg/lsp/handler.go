package lsp

import (
	"github.com/sjsone/bhaus-util/pkg/analysis"
	"github.com/sjsone/bhaus-util/pkg/ast"
	"github.com/sjsone/bhaus-util/pkg/parser"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/tliron/glsp/server"
)

// Handler is the LSP request handler that implements all LSP protocols
type Handler struct {
	Server    *server.Server
	RootURI   string
	Documents map[string]string     // uri → raw content (for text-based handlers)
	Files     map[string]*ast.File  // uri → parsed AST
	Cache     *parser.Cache         // tree-sitter trees (handler never imports tree-sitter)
	Table     *analysis.SymbolTable // immutable symbol table, rebuilt on change

	// definitionLinkSupport records whether the client accepts LocationLink[]
	// (with an origin selection range) from textDocument/definition. Set from
	// the client capabilities during Initialize.
	definitionLinkSupport bool
}

// NewHandler creates a new LSP handler
func NewHandler() *Handler {
	handler := &Handler{
		Documents: make(map[string]string),
		Files:     make(map[string]*ast.File),
		Cache:     parser.NewCache(),
	}

	protocolHandler := protocol.Handler{
		Initialize:                     handler.Initialize,
		Initialized:                    handler.Initialized,
		Shutdown:                       handler.Shutdown,
		TextDocumentDidOpen:            handler.TextDocumentDidOpen,
		TextDocumentDidChange:          handler.TextDocumentDidChange,
		TextDocumentDidClose:           handler.TextDocumentDidClose,
		TextDocumentDidSave:            handler.TextDocumentDidSave,
		TextDocumentDefinition:         handler.TextDocumentDefinition,
		TextDocumentReferences:         handler.TextDocumentReferences,
		TextDocumentDocumentSymbol:     handler.TextDocumentDocumentSymbol,
		WorkspaceSymbol:                handler.WorkspaceSymbol,
		WorkspaceDidChangeWatchedFiles: handler.WorkspaceDidChangeWatchedFiles,
		TextDocumentCompletion:         handler.TextDocumentCompletion,
		TextDocumentHover:              handler.TextDocumentHover,
		TextDocumentOnTypeFormatting:   handler.TextDocumentOnTypeFormatting,
		SetTrace:                       handler.SetTrace,
	}

	srv := server.NewServer(&protocolHandler, "BHaus Language Server", false)
	handler.Server = srv

	return handler
}
