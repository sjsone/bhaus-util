// Package lint is the BHaus diagnostics engine. It is the single source of
// truth for "what is a problem in a .bhaus document". It runs a set of rules
// over an already-loaded set of parsed files. It returns diagnostics anchored
// in one target file. Both the CLI (bhaus-util lint) and the language server
// consume this package. This way, the editor and the command line never
// disagree about what is wrong.
//
// The package depends only on pkg/ast, pkg/parser (via loader) and
// pkg/analysis. It never imports glsp or pkg/lsp. It does not render output.
// Spans stay 0-based. Consumers format them: the CLI converts to 1-based
// text/JSON and the LSP converts to protocol ranges.
package lint

import (
	"sort"

	"github.com/sjsone/bhaus-util/pkg/analysis"
	"github.com/sjsone/bhaus-util/pkg/ast"
)

// Severity classifies a diagnostic. Errors fail a lint run; warnings do not.
type Severity int

const (
	Error Severity = iota
	Warning
)

func (s Severity) String() string {
	if s == Warning {
		return "warning"
	}
	return "error"
}

// Diagnostic is a single problem found in a document. Span is 0-based; this
// matches the rest of the AST. Consumers convert it for display.
type Diagnostic struct {
	URI      string
	Span     ast.Span
	Severity Severity
	Rule     string // stable id: "syntax", "unresolved-ref", "duplicate-decl", "naming", "structure", "unknown-type"
	Message  string
}

// ctx is the shared state each rule reads: the target file plus the semantic
// analysis (symbol table + resolved references) built over the whole fileset.
type ctx struct {
	uri   string
	file  *ast.File
	table *analysis.SymbolTable
	refs  []analysis.Reference
}

// rules is the ordered rule set. Order here does not affect output order,
// because Check sorts diagnostics by position. It only affects the order in
// which rules run.
var rules = []func(*ctx) []Diagnostic{
	ruleSyntax,
	ruleUnresolved,
	ruleDuplicate,
	ruleNaming,
	ruleStructure,
	ruleUnknownType,
}

// Check runs every rule over files. It returns the diagnostics anchored in
// targetURI, sorted by (line, column). References resolve against the entire
// fileset, so a type declared in an included file still resolves. Check
// reports only problems located in targetURI; included files serve as
// context, not as targets.
//
// If targetURI is not present in files, Check returns nil.
func Check(files map[string]*ast.File, targetURI string) []Diagnostic {
	target := files[targetURI]
	if target == nil {
		return nil
	}

	table := analysis.BuildSymbolTable(files)
	refs := analysis.CollectReferences(files, table)
	c := &ctx{uri: targetURI, file: target, table: table, refs: refs}

	var diags []Diagnostic
	for _, rule := range rules {
		diags = append(diags, rule(c)...)
	}

	sort.SliceStable(diags, func(i, j int) bool {
		a, b := diags[i].Span.Start, diags[j].Span.Start
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Column < b.Column
	})
	return diags
}

// HasError reports whether any diagnostic is error-severity. Consumers use it
// to decide process exit codes.
func HasError(diags []Diagnostic) bool {
	for _, d := range diags {
		if d.Severity == Error {
			return true
		}
	}
	return false
}
