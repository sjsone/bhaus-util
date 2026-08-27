package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/sjsone/bhaus-util/pkg/lint"
)

// handleLint lints a single .bhaus file. It also loads the files it includes,
// for reference resolution, but does not lint them. It reports diagnostics
// as text or JSON. It exits non-zero when it finds any error-severity
// diagnostic. Warnings alone do not fail the run.
func handleLint(args []string) int {
	if len(args) >= 1 && isHelpFlag(args[0]) {
		lintUsage(os.Stdout)
		return 0
	}

	fs := flag.NewFlagSet("lint", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() { lintUsage(os.Stderr) }
	format := fs.String("format", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	rest := fs.Args()
	if len(rest) != 1 {
		lintUsage(os.Stderr)
		return 1
	}
	if *format != "text" && *format != "json" {
		fmt.Fprintf(os.Stderr, "bhaus-util: unknown --format %q (want \"text\" or \"json\")\n", *format)
		return 1
	}

	files, rootURI, err := lint.Load(rest[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "bhaus-util: %v\n", err)
		return 1
	}
	diags := lint.Check(files, rootURI)

	if *format == "json" {
		if err := renderJSON(os.Stdout, diags); err != nil {
			fmt.Fprintf(os.Stderr, "bhaus-util: %v\n", err)
			return 1
		}
	} else {
		renderText(os.Stdout, diags)
	}

	if lint.HasError(diags) {
		return 1
	}
	return 0
}

// renderText prints diagnostics as "path:line:col: severity: message [rule]"
// with 1-based positions, followed by a summary. A clean file prints "ok".
func renderText(w io.Writer, diags []lint.Diagnostic) {
	if len(diags) == 0 {
		fmt.Fprintln(w, "ok")
		return
	}
	var errs, warns int
	for _, d := range diags {
		if d.Severity == lint.Error {
			errs++
		} else {
			warns++
		}
		fmt.Fprintf(w, "%s:%d:%d: %s: %s [%s]\n",
			displayPath(d.URI),
			d.Span.Start.Line+1, d.Span.Start.Column+1,
			d.Severity, d.Message, d.Rule)
	}
	fmt.Fprintf(w, "\n%s, %s\n", plural(errs, "error"), plural(warns, "warning"))
}

// jsonDiagnostic is the stable JSON shape for machine consumers. Positions are
// 1-based to match the text output and common linter conventions.
type jsonDiagnostic struct {
	File     string `json:"file"`
	Line     uint32 `json:"line"`
	Column   uint32 `json:"column"`
	Severity string `json:"severity"`
	Rule     string `json:"rule"`
	Message  string `json:"message"`
}

// renderJSON prints diagnostics as a JSON array (always an array, even when
// empty) so consumers can parse a stable shape.
func renderJSON(w io.Writer, diags []lint.Diagnostic) error {
	out := make([]jsonDiagnostic, 0, len(diags))
	for _, d := range diags {
		out = append(out, jsonDiagnostic{
			File:     displayPath(d.URI),
			Line:     d.Span.Start.Line + 1,
			Column:   d.Span.Start.Column + 1,
			Severity: d.Severity.String(),
			Rule:     d.Rule,
			Message:  d.Message,
		})
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// displayPath turns a file:// URI into a filesystem path for display.
func displayPath(uri string) string {
	return strings.TrimPrefix(uri, "file://")
}

func plural(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

func lintUsage(w io.Writer) {
	fmt.Fprint(w, `bhaus-util lint — check a .bhaus file for problems

Usage:
  bhaus-util lint [--format text|json] <file>

Arguments:
  <file>    Path to a single .bhaus file to check. Files it INCLUDEs are loaded
            too (for cross-file reference resolution) but are not themselves
            linted.

Flags:
  --format text|json   Output format. Default: text.

Checks:
  syntax           parse errors (malformed declarations)
  unresolved-ref   a type / EXTENDS / IMPLEMENTS / C4 endpoint that names nothing
  duplicate-decl   a declaration whose full name is declared more than once
  naming           type names should be Uppercase, function/method names lowercase
  structure        VERSION must be present and be the first declaration
  unknown-type     a type annotation uses the built-in Unknown type

Exit code:
  0   no error-severity diagnostics (warnings alone still exit 0)
  1   at least one error-severity diagnostic or the file could not be read

Examples:
  bhaus-util lint domain/user.bhaus
  bhaus-util lint --format json domain/user.bhaus
`)
}
