package lsp

import (
	"github.com/sjsone/bhaus-util/pkg/lint"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// publishDiagnostics runs the shared lint engine over the current fileset.
// It publishes the diagnostics for uri. The CLI `lint` command uses the same
// engine. This keeps the editor and the command line in agreement about
// what counts as a problem. For a clean file, the function publishes an
// empty diagnostics array. This clears any diagnostics the client is still
// showing.
func (h *Handler) publishDiagnostics(context *glsp.Context, uri string) {
	if context == nil {
		return // no client to notify (e.g. in tests)
	}

	diags := lint.Check(h.Files, uri)
	out := make([]protocol.Diagnostic, 0, len(diags))
	source := "bhaus"
	for _, d := range diags {
		severity := protocol.DiagnosticSeverityError
		if d.Severity == lint.Warning {
			severity = protocol.DiagnosticSeverityWarning
		}

		// Ensure a visible, non-empty range even for zero-width spans (e.g. a
		// "missing VERSION" diagnostic anchored at the start of the document).
		start := protocol.Position{Line: d.Span.Start.Line, Character: d.Span.Start.Column}
		end := protocol.Position{Line: d.Span.End.Line, Character: d.Span.End.Column}
		if end.Line < start.Line || (end.Line == start.Line && end.Character <= start.Character) {
			end = protocol.Position{Line: start.Line, Character: start.Character + 1}
		}

		rule := d.Rule
		out = append(out, protocol.Diagnostic{
			Range:    protocol.Range{Start: start, End: end},
			Severity: &severity,
			Source:   &source,
			Code:     &protocol.IntegerOrString{Value: rule},
			Message:  d.Message,
		})
	}

	context.Notify(protocol.ServerTextDocumentPublishDiagnostics, protocol.PublishDiagnosticsParams{
		URI:         protocol.DocumentUri(uri),
		Diagnostics: out,
	})
}
