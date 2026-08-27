package analysis

import (
	"strings"

	"github.com/sjsone/bhaus-util/pkg/ast"
)

// SymbolTable is a flat, immutable index of all declared symbols.
type SymbolTable struct {
	all      []Symbol
	bySimple map[string][]Symbol
	byFull   map[string]Symbol
	byC4     map[string]Symbol
	byURI    map[string][]Symbol
}

// BuildSymbolTable constructs a SymbolTable from parsed AST files.
func BuildSymbolTable(files map[string]*ast.File) *SymbolTable {
	t := &SymbolTable{
		bySimple: make(map[string][]Symbol),
		byFull:   make(map[string]Symbol),
		byC4:     make(map[string]Symbol),
		byURI:    make(map[string][]Symbol),
	}

	for uri, file := range files {
		if file == nil {
			continue
		}
		walkFile(file, uri,
			func(sym Symbol) {
				t.all = append(t.all, sym)
				t.bySimple[sym.Name] = append(t.bySimple[sym.Name], sym)
				t.byFull[sym.FullName] = sym
				if sym.Kind >= KindSystem && sym.Kind <= KindComponent {
					t.byC4[sym.FullName] = sym
				}
				t.byURI[uri] = append(t.byURI[uri], sym)
			},
			nil, // no ref-collection during table build
		)
	}
	return t
}

// Lookup returns symbols matching name. Full name match first, then simple.
func (t *SymbolTable) Lookup(name string) []Symbol {
	if s, ok := t.byFull[name]; ok {
		return []Symbol{s}
	}
	return t.bySimple[name]
}

// LookupSimple returns all symbols with the given simple name.
func (t *SymbolTable) LookupSimple(name string) []Symbol {
	return t.bySimple[name]
}

// LookupFull returns the symbol with the exact full name, if any.
//
// byFull is keyed by full name, so it collapses duplicate declarations
// (last write wins). Use LookupAllFull to detect duplicates.
func (t *SymbolTable) LookupFull(fullName string) (Symbol, bool) {
	s, ok := t.byFull[fullName]
	return s, ok
}

// LookupAllFull returns every symbol declared with the exact full name,
// across all files, in declaration order. Unlike LookupFull, it does not
// collapse duplicates. A result with len > 1 means the name has more than
// one declaration.
func (t *SymbolTable) LookupAllFull(fullName string) []Symbol {
	var out []Symbol
	for _, s := range t.all {
		if s.FullName == fullName {
			out = append(out, s)
		}
	}
	return out
}

// LookupC4 returns the symbol for a C4 path, if any.
func (t *SymbolTable) LookupC4(path string) (Symbol, bool) {
	s, ok := t.byC4[path]
	return s, ok
}

// SymbolsForURI returns all symbols declared in a specific file.
func (t *SymbolTable) SymbolsForURI(uri string) []Symbol {
	return t.byURI[uri]
}

// All returns every symbol in the table, in declaration order.
func (t *SymbolTable) All() []Symbol {
	return t.all
}

// Search returns symbols whose name or full name contains query
// (case-insensitive substring).
func (t *SymbolTable) Search(query string) []Symbol {
	if query == "" {
		return nil
	}
	queryLower := strings.ToLower(query)
	seen := make(map[string]bool)
	var results []Symbol
	for _, sym := range t.all {
		key := sym.URI + sym.FullName
		if seen[key] {
			continue
		}
		if strings.Contains(strings.ToLower(sym.Name), queryLower) ||
			strings.Contains(strings.ToLower(sym.FullName), queryLower) {
			results = append(results, sym)
			seen[key] = true
		}
	}
	return results
}
