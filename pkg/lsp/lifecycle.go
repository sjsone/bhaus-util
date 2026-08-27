package lsp

import (
	"github.com/sjsone/bhaus-util/pkg/version"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Initialize handles the LSP initialize request
func (h *Handler) Initialize(context *glsp.Context, params *protocol.InitializeParams) (any, error) {
	logger.Info("initialize: starting BHaus language server")

	// Remember whether the client accepts LocationLink[] from definition.
	// This lets the server send an origin selection range covering the
	// whole qualified name.
	if td := params.Capabilities.TextDocument; td != nil && td.Definition != nil && td.Definition.LinkSupport != nil {
		h.definitionLinkSupport = *td.Definition.LinkSupport
		logger.Debugf("initialize: client definition linkSupport=%t", h.definitionLinkSupport)
	}

	// Store the workspace root URI
	if params.RootURI != nil {
		h.RootURI = string(*params.RootURI)
		logger.Infof("initialize: workspace root %s", h.RootURI)
	} else if params.RootPath != nil {
		// Fallback to rootPath if available
		h.RootURI = "file://" + *params.RootPath
		logger.Infof("initialize: workspace root (from rootPath) %s", h.RootURI)
	}

	// Create server capabilities
	triggerChars := []string{" ", "\n", "."}
	capabilities := protocol.ServerCapabilities{
		TextDocumentSync:        protocol.TextDocumentSyncKindFull, // Full sync
		DefinitionProvider:      true,
		ReferencesProvider:      true,
		DocumentSymbolProvider:  true,
		WorkspaceSymbolProvider: true,
		HoverProvider:           true,
		CompletionProvider: &protocol.CompletionOptions{
			TriggerCharacters: triggerChars,
		},
		DocumentOnTypeFormattingProvider: &protocol.DocumentOnTypeFormattingOptions{
			FirstTriggerCharacter: "\n",
		},
	}

	return protocol.InitializeResult{
		Capabilities: capabilities,
		ServerInfo: &protocol.InitializeResultServerInfo{
			Name:    "BHaus Language Server",
			Version: &version.Version,
		},
	}, nil
}

// Initialized handles the LSP initialized notification
func (h *Handler) Initialized(context *glsp.Context, params *protocol.InitializedParams) error {
	logger.Info("initialized: BHaus language server ready")
	return nil
}

// Shutdown handles the LSP shutdown request
func (h *Handler) Shutdown(context *glsp.Context) error {
	logger.Info("shutdown: BHaus language server stopping")
	return nil
}

// SetTrace handles the $/setTrace notification
func (h *Handler) SetTrace(context *glsp.Context, params *protocol.SetTraceParams) error {
	// Trace configuration is not implemented. The server handles the
	// notification anyway, to avoid errors.
	return nil
}
