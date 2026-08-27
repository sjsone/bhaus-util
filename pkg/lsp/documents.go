package lsp

import (
	"maps"
	"os"

	"github.com/sjsone/bhaus-util/pkg/analysis"
	"github.com/sjsone/bhaus-util/pkg/include"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// TextDocumentDidOpen handles the textDocument/didOpen notification
func (h *Handler) TextDocumentDidOpen(context *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
	uri := string(params.TextDocument.URI)
	content := params.TextDocument.Text
	logger.Infof("didOpen: %s (%d bytes)", uri, len(content))
	return h.updateFile(context, uri, content)
}

// TextDocumentDidChange handles the textDocument/didChange notification
func (h *Handler) TextDocumentDidChange(context *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
	uri := string(params.TextDocument.URI)

	// glsp decodes each content change into a concrete struct value, never a
	// map. Full-document sync (what this server registers) produces
	// TextDocumentContentChangeEventWhole. Ranged edits produce
	// TextDocumentContentChangeEvent. The code must type-switch on those
	// types. A type assertion to map[string]any would silently match
	// nothing. Then the document would never be reparsed after an edit.
	var content string
	var found bool
	for _, change := range params.ContentChanges {
		switch c := change.(type) {
		case protocol.TextDocumentContentChangeEventWhole:
			content, found = c.Text, true
		case protocol.TextDocumentContentChangeEvent:
			content, found = c.Text, true
		}
	}
	if !found {
		// No usable change payload. Leave the current state untouched instead
		// of wiping the document to empty. (An intentionally cleared file
		// still arrives as a Whole change with Text == "". In that case
		// found is true.)
		logger.Debugf("didChange: %s had no usable content change; ignoring", uri)
		return nil
	}
	logger.Infof("didChange: %s (%d bytes)", uri, len(content))
	return h.updateFile(context, uri, content)
}

// updateFile parses the given content. It caches the resulting AST. It
// rebuilds the symbol table. It publishes diagnostics for any syntax errors.
func (h *Handler) updateFile(context *glsp.Context, uri, content string) error {
	file, err := h.Cache.Parse(uri, content)
	if err != nil {
		return err
	}

	h.Documents[uri] = content
	h.Files[uri] = file

	// Load files pulled in via INCLUDE. This makes their declarations
	// available for cross-file resolution (definition/hover of types like
	// Base/Bla). Already loaded files are skipped. This keeps the operation
	// cheap after the first parse.
	h.loadIncludedFiles(uri, content)

	// Rebuild the symbol table (now including any newly loaded include targets).
	h.Table = analysis.BuildSymbolTable(h.Files)

	// Publish diagnostics from the shared lint engine (syntax + semantic).
	h.publishDiagnostics(context, uri)

	logger.Infof("parsed %s: %d decls, %d comments, %d error(s)",
		uri, len(file.Decls), len(file.Comments), len(file.Errors))

	return nil
}

// TextDocumentDidClose handles the textDocument/didClose notification. The
// client has given up ownership of the buffer. The server releases the
// content, the AST and the cached tree-sitter tree. Without this release,
// the three maps grow for every file ever opened. The trees also keep
// holding C memory for the lifetime of the session.
//
// Eviction alone would be wrong. The closed file may still be an INCLUDE
// target of a file that is still open. To handle this, the server
// re-resolves the includes of the remaining files afterwards. This reloads
// the closed file from disk when needed. Cross-file definition and hover
// keep working in the files the user is still editing.
func (h *Handler) TextDocumentDidClose(context *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
	uri := string(params.TextDocument.URI)

	if _, tracked := h.Files[uri]; !tracked {
		logger.Debugf("didClose: %s was not tracked, nothing to release", uri)
		return nil
	}

	logger.Infof("didClose: releasing %s", uri)
	delete(h.Documents, uri)
	delete(h.Files, uri)
	h.Cache.Remove(uri)

	// Snapshot before recursing. loadIncludedFiles writes into h.Documents.
	// Iterating a map while it is being written panics.
	remaining := make(map[string]string, len(h.Documents))
	maps.Copy(remaining, h.Documents)
	for openURI, content := range remaining {
		h.loadIncludedFiles(openURI, content)
	}

	h.Table = analysis.BuildSymbolTable(h.Files)

	// lint.Check returns nothing for a URI it no longer holds. This publishes
	// an empty array. That clears the diagnostics the client is still
	// showing.
	h.publishDiagnostics(context, uri)

	return nil
}

// TextDocumentDidSave handles the textDocument/didSave notification
func (h *Handler) TextDocumentDidSave(context *glsp.Context, params *protocol.DidSaveTextDocumentParams) error {
	// The document is already up to date from didChange. There is nothing to do.
	return nil
}

// WorkspaceDidChangeWatchedFiles handles the workspace/didChangeWatchedFiles notification
func (h *Handler) WorkspaceDidChangeWatchedFiles(context *glsp.Context, params *protocol.DidChangeWatchedFilesParams) error {
	for _, change := range params.Changes {
		uri := string(change.URI)
		if change.Type == protocol.FileChangeTypeDeleted {
			delete(h.Documents, uri)
			delete(h.Files, uri)
			h.Cache.Remove(uri)
		}
	}
	h.Table = analysis.BuildSymbolTable(h.Files)
	return nil
}

// loadIncludedFiles parses INCLUDE directives in the given content. It loads
// the matching files. Pattern extraction and globbing live in pkg/include,
// shared with the CLI linter. This method owns only the LSP-specific
// recursion and Handler-backed storage. It skips already-loaded files,
// caches the parse and recurses.
func (h *Handler) loadIncludedFiles(uri, content string) {
	for _, match := range include.Match(uri, content) {
		matchURI := "file://" + match
		if matchURI == uri {
			continue // skip the current file itself
		}
		if _, exists := h.Documents[matchURI]; exists {
			continue // already loaded
		}

		logger.Debugf("includes: loading %s", match)
		fileContent, err := loadFileContent(match)
		if err != nil {
			logger.Errorf("includes: cannot read %s: %v", match, err)
			continue
		}

		h.Documents[matchURI] = fileContent
		file, err := h.Cache.Parse(matchURI, fileContent)
		if err != nil {
			logger.Errorf("includes: cannot parse %s: %v", match, err)
			continue
		}

		h.Files[matchURI] = file
		logger.Infof("includes: loaded %s (%d decls)", match, len(file.Decls))

		// Recursively load any includes in this file.
		h.loadIncludedFiles(matchURI, fileContent)
	}
}

// loadFileContent reads a file from disk
func loadFileContent(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}
