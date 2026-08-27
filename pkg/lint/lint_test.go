package lint

import (
	"strings"
	"testing"

	"github.com/sjsone/bhaus-util/pkg/ast"
	"github.com/sjsone/bhaus-util/pkg/parser"
)

const testURI = "file:///t.bhaus"

// checkSrc parses src as the sole file and runs the engine over it.
func checkSrc(t *testing.T, src string) []Diagnostic {
	t.Helper()
	f, err := parser.Parse(testURI, src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return Check(map[string]*ast.File{testURI: f}, f.URI)
}

// rulesOf returns the set of rule ids present, for order-independent assertions.
func rulesOf(diags []Diagnostic) map[string]int {
	m := make(map[string]int)
	for _, d := range diags {
		m[d.Rule]++
	}
	return m
}

func firstOfRule(diags []Diagnostic, rule string) (Diagnostic, bool) {
	for _, d := range diags {
		if d.Rule == rule {
			return d, true
		}
	}
	return Diagnostic{}, false
}

func TestCleanFileHasNoDiagnostics(t *testing.T) {
	src := "VERSION 0.1\n" +
		"STRUCT Domain/User:\n" +
		"    PUBLIC id: Integer\n" +
		"    PUBLIC name: String\n"
	diags := checkSrc(t, src)
	if len(diags) != 0 {
		t.Fatalf("expected no diagnostics, got %v", diags)
	}
}

func TestBuiltinTypesNeverUnresolved(t *testing.T) {
	src := "VERSION 0.1\n" +
		"STRUCT S:\n" +
		"    PUBLIC a: String\n" +
		"    PUBLIC b: Array[Integer]\n" +
		"    PUBLIC c: ?Bool\n" +
		"    PUBLIC d: Unknown\n"
	if r := rulesOf(checkSrc(t, src))["unresolved-ref"]; r != 0 {
		t.Fatalf("built-in types produced %d unresolved-ref diagnostics", r)
	}
}

func TestUnresolvedReference(t *testing.T) {
	src := "VERSION 0.1\n" +
		"STRUCT S:\n" +
		"    PUBLIC x: Missing/Type\n"
	d, ok := firstOfRule(checkSrc(t, src), "unresolved-ref")
	if !ok {
		t.Fatal("expected an unresolved-ref diagnostic")
	}
	if d.Severity != Error {
		t.Fatalf("unresolved-ref severity = %v, want error", d.Severity)
	}
	if !strings.Contains(d.Message, "Missing/Type") {
		t.Fatalf("message %q should name the missing type", d.Message)
	}
}

func TestResolvedReferenceAcrossDecls(t *testing.T) {
	src := "VERSION 0.1\n" +
		"STRUCT Account:\n" +
		"    PUBLIC balance: Integer\n" +
		"STRUCT User:\n" +
		"    PUBLIC account: Account\n"
	if r := rulesOf(checkSrc(t, src))["unresolved-ref"]; r != 0 {
		t.Fatalf("a resolvable reference was reported unresolved (%d)", r)
	}
}

func TestDuplicateDeclaration(t *testing.T) {
	src := "VERSION 0.1\n" +
		"STRUCT Domain/User:\n" +
		"    PUBLIC id: Integer\n" +
		"STRUCT Domain/User:\n" +
		"    PUBLIC name: String\n"
	diags := checkSrc(t, src)
	if got := rulesOf(diags)["duplicate-decl"]; got != 2 {
		t.Fatalf("duplicate-decl count = %d, want 2 (one per declaration)", got)
	}
	d, _ := firstOfRule(diags, "duplicate-decl")
	if d.Severity != Error || !strings.Contains(d.Message, "Domain/User") {
		t.Fatalf("unexpected duplicate-decl diagnostic: %+v", d)
	}
}

func TestNamingConventions(t *testing.T) {
	src := "VERSION 0.1\n" +
		"STRUCT user:\n" + // should be uppercase
		"    PUBLIC id: Integer\n" +
		"FUNCTION Compute():\n" // should be lowercase
	diags := checkSrc(t, src)
	if got := rulesOf(diags)["naming"]; got != 2 {
		t.Fatalf("naming count = %d, want 2; diags=%v", got, diags)
	}
	for _, d := range diags {
		if d.Rule == "naming" && d.Severity != Warning {
			t.Fatalf("naming severity = %v, want warning", d.Severity)
		}
	}
}

func TestNamingIgnoresUnderscoreLeadingNames(t *testing.T) {
	src := "VERSION 0.1\n" +
		"STRUCT _Internal:\n" +
		"    PUBLIC id: Integer\n"
	if got := rulesOf(checkSrc(t, src))["naming"]; got != 0 {
		t.Fatalf("underscore-leading name produced %d naming diagnostics", got)
	}
}

func TestUnknownTypeWarns(t *testing.T) {
	src := "VERSION 0.1\n" +
		"STRUCT S:\n" +
		"    PUBLIC a: Unknown\n" +
		"    PUBLIC b: Array[Unknown]\n" +
		"    PUBLIC c: String\n"
	diags := checkSrc(t, src)
	if got := rulesOf(diags)["unknown-type"]; got != 2 {
		t.Fatalf("unknown-type count = %d, want 2 (top-level and nested in Array); diags=%v", got, diags)
	}
	d, ok := firstOfRule(diags, "unknown-type")
	if !ok || d.Severity != Warning || !strings.Contains(d.Message, "Unknown") {
		t.Fatalf("expected an 'Unknown' warning, got %+v (ok=%v)", d, ok)
	}
}

func TestSyntaxError(t *testing.T) {
	src := "VERSION 0.1\nCLASS @@@ bad!!!\n"
	d, ok := firstOfRule(checkSrc(t, src), "syntax")
	if !ok {
		t.Fatal("expected a syntax diagnostic for malformed input")
	}
	if d.Severity != Error {
		t.Fatalf("syntax severity = %v, want error", d.Severity)
	}
}

func TestStructureMissingVersion(t *testing.T) {
	src := "STRUCT S:\n    PUBLIC id: Integer\n"
	d, ok := firstOfRule(checkSrc(t, src), "structure")
	if !ok || d.Severity != Warning || !strings.Contains(d.Message, "missing VERSION") {
		t.Fatalf("expected 'missing VERSION' warning, got %+v (ok=%v)", d, ok)
	}
}

func TestStructureVersionNotFirst(t *testing.T) {
	src := "STRUCT S:\n    PUBLIC id: Integer\nVERSION 0.1\n"
	d, ok := firstOfRule(checkSrc(t, src), "structure")
	if !ok || !strings.Contains(d.Message, "first declaration") {
		t.Fatalf("expected 'first declaration' warning, got %+v (ok=%v)", d, ok)
	}
}

func TestDiagnosticsSortedByPosition(t *testing.T) {
	src := "VERSION 0.1\n" +
		"STRUCT S:\n" +
		"    PUBLIC a: MissingA\n" +
		"    PUBLIC b: MissingB\n"
	diags := checkSrc(t, src)
	for i := 1; i < len(diags); i++ {
		prev, cur := diags[i-1].Span.Start, diags[i].Span.Start
		if cur.Line < prev.Line || (cur.Line == prev.Line && cur.Column < prev.Column) {
			t.Fatalf("diagnostics not sorted by position: %v", diags)
		}
	}
}

func TestHasError(t *testing.T) {
	if HasError([]Diagnostic{{Severity: Warning}}) {
		t.Fatal("HasError true for warning-only set")
	}
	if !HasError([]Diagnostic{{Severity: Warning}, {Severity: Error}}) {
		t.Fatal("HasError false despite an error")
	}
}

func TestCheckUnknownTargetReturnsNil(t *testing.T) {
	if diags := Check(map[string]*ast.File{}, "file:///nope.bhaus"); diags != nil {
		t.Fatalf("expected nil for unknown target, got %v", diags)
	}
}
