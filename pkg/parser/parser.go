package parser

import (
	"fmt"

	bhaus "github.com/sjsone/bhaus-tree-sitter/bindings/go"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/sjsone/bhaus-util/pkg/ast"
	"github.com/sjsone/bhaus-util/pkg/parser/convert"
)

// Cache owns tree-sitter parser instances. It also owns the most recent
// syntax tree per document. Cache keeps the old tree only to call Close() on
// it, either at the next parse or on Remove. The LSP handler never touches
// tree-sitter types directly. All mutations happen from document-sync
// handlers (single goroutine).
type Cache struct {
	trees  map[string]*tree_sitter.Tree
	parser *tree_sitter.Parser
}

// NewCache creates a Cache with a fresh tree-sitter parser.
func NewCache() *Cache {
	return &Cache{
		trees:  make(map[string]*tree_sitter.Tree),
		parser: tree_sitter.NewParser(),
	}
}

// Parse parses content as a BHaus document from scratch. It returns a
// freshly built *ast.File.
//
// Parse deliberately does not feed the previous syntax tree to the
// incremental parser. Under TextDocumentSyncKindFull, the client resends the
// whole document with no edit ranges. Tree-sitter incremental parsing only
// works if the old tree first receives every edit through Tree.Edit. An
// un-edited old tree makes tree-sitter reuse subtrees at their stale
// positions. After any edit, the AST spans then drift out of sync with the
// text. Position-based features (definition, hover) then silently stop
// resolving. A full reparse is both correct and cheap for documents this
// size.
func (c *Cache) Parse(uri, content string) (*ast.File, error) {
	language := tree_sitter.NewLanguage(bhaus.Language())
	if err := c.parser.SetLanguage(language); err != nil {
		return nil, fmt.Errorf("failed to set tree-sitter language: %w", err)
	}

	src := []byte(content)
	oldTree := c.trees[uri]
	tree := c.parser.Parse(src, nil)
	if tree == nil {
		return nil, fmt.Errorf("tree-sitter parser returned nil tree for %s", uri)
	}

	// Close the old tree, if any.
	if oldTree != nil {
		oldTree.Close()
	}
	// Cache the new tree.
	c.trees[uri] = tree

	// Build the AST from the CST
	conv := convert.New(uri, content, src)
	file := conv.File(tree.RootNode())

	return file, nil
}

// Remove drops the cached tree for uri (file deleted/closed).
func (c *Cache) Remove(uri string) {
	if t, ok := c.trees[uri]; ok {
		t.Close()
		delete(c.trees, uri)
	}
}

// Close releases all cached trees and the parser.
func (c *Cache) Close() {
	for _, t := range c.trees {
		t.Close()
	}
	c.trees = nil
	c.parser.Close()
}

// Parse is a one-shot parse for CLI usage (no caching).
func Parse(uri, content string) (*ast.File, error) {
	cache := NewCache()
	defer cache.Close()
	return cache.Parse(uri, content)
}
