package lint

import (
	"os"
	"path/filepath"

	"github.com/sjsone/bhaus-util/pkg/ast"
	"github.com/sjsone/bhaus-util/pkg/include"
	"github.com/sjsone/bhaus-util/pkg/parser"
)

// Load reads and parses rootPath together with every file it pulls in through
// INCLUDE. It resolves each INCLUDE recursively, relative to that file's own
// directory. It returns the parsed fileset keyed by URI, plus the root file's
// URI. Pass both to Check.
//
// Load treats included files as best-effort. If it cannot read or parse an
// included file, it skips that file rather than failing the whole load,
// because only the root file gets linted. Load returns an error only when it
// cannot read or parse the root file itself.
func Load(rootPath string) (files map[string]*ast.File, rootURI string, err error) {
	abs, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, "", err
	}
	rootURI = "file://" + abs

	files = make(map[string]*ast.File)
	content, err := os.ReadFile(abs)
	if err != nil {
		return nil, "", err
	}
	root, err := parser.Parse(rootURI, string(content))
	if err != nil {
		return nil, "", err
	}
	files[rootURI] = root

	loadIncludes(rootURI, string(content), files)
	return files, rootURI, nil
}

// loadIncludes recursively loads the files included by the document at uri.
// It stores each parsed file in files. It skips URIs already loaded and it
// skips includes it cannot read or parse.
func loadIncludes(uri, content string, files map[string]*ast.File) {
	for _, path := range include.Match(uri, content) {
		matchURI := "file://" + path
		if _, ok := files[matchURI]; ok {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		file, err := parser.Parse(matchURI, string(data))
		if err != nil {
			continue
		}
		files[matchURI] = file
		loadIncludes(matchURI, string(data), files)
	}
}
