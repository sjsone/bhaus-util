package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sjsone/bhaus-util/pkg/ast"
	"github.com/sjsone/bhaus-util/pkg/lint"
)

func sampleDiag() lint.Diagnostic {
	return lint.Diagnostic{
		URI:      "file:///dir/user.bhaus",
		Span:     ast.Span{Start: ast.Pos{Line: 2, Column: 7}}, // 0-based
		Severity: lint.Error,
		Rule:     "unresolved-ref",
		Message:  `cannot resolve reference to "Missing"`,
	}
}

func TestRenderTextIsOneBased(t *testing.T) {
	var buf bytes.Buffer
	renderText(&buf, []lint.Diagnostic{sampleDiag()})
	out := buf.String()

	// 0-based (2,7) must render as 1-based (3,8).
	if !strings.Contains(out, "/dir/user.bhaus:3:8: error:") {
		t.Fatalf("text output not 1-based / missing prefix:\n%s", out)
	}
	if !strings.Contains(out, "[unresolved-ref]") {
		t.Fatalf("text output missing rule tag:\n%s", out)
	}
	if !strings.Contains(out, "1 error, 0 warnings") {
		t.Fatalf("text output missing summary:\n%s", out)
	}
}

func TestRenderTextCleanPrintsOk(t *testing.T) {
	var buf bytes.Buffer
	renderText(&buf, nil)
	if strings.TrimSpace(buf.String()) != "ok" {
		t.Fatalf("clean render = %q, want ok", buf.String())
	}
}

func TestRenderJSONShape(t *testing.T) {
	var buf bytes.Buffer
	if err := renderJSON(&buf, []lint.Diagnostic{sampleDiag()}); err != nil {
		t.Fatal(err)
	}
	var got []jsonDiagnostic
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	if len(got) != 1 {
		t.Fatalf("got %d entries, want 1", len(got))
	}
	d := got[0]
	if d.Line != 3 || d.Column != 8 || d.Severity != "error" || d.Rule != "unresolved-ref" {
		t.Fatalf("unexpected JSON diagnostic: %+v", d)
	}
	if d.File != "/dir/user.bhaus" {
		t.Fatalf("File = %q, want stripped path", d.File)
	}
}

func TestRenderJSONEmptyIsArray(t *testing.T) {
	var buf bytes.Buffer
	if err := renderJSON(&buf, nil); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(buf.String()) != "[]" {
		t.Fatalf("empty render = %q, want []", buf.String())
	}
}

// withStdout redirects os.Stdout for the duration of f.
// This stops exit-code tests from polluting test output.
func withStdout(t *testing.T, f func()) {
	t.Helper()
	orig := os.Stdout
	devnull, _ := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	os.Stdout = devnull
	defer func() { os.Stdout = orig; devnull.Close() }()
	f()
}

func TestHandleLintExitCodes(t *testing.T) {
	dir := t.TempDir()
	clean := filepath.Join(dir, "clean.bhaus")
	if err := os.WriteFile(clean, []byte("VERSION 0.1\nSTRUCT S:\n    PUBLIC id: Integer\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	broken := filepath.Join(dir, "broken.bhaus")
	if err := os.WriteFile(broken, []byte("VERSION 0.1\nSTRUCT S:\n    PUBLIC id: Missing/Type\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		args []string
		want int
	}{
		{"clean", []string{clean}, 0},
		{"error", []string{broken}, 1},
		{"clean json", []string{"--format", "json", clean}, 0},
		{"bad format", []string{"--format", "xml", clean}, 1},
		{"missing file", []string{filepath.Join(dir, "nope.bhaus")}, 1},
		{"no args", []string{}, 1},
		{"help", []string{"--help"}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var code int
			withStdout(t, func() { code = handleLint(tc.args) })
			if code != tc.want {
				t.Fatalf("handleLint(%v) = %d, want %d", tc.args, code, tc.want)
			}
		})
	}
}
