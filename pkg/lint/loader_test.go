package lint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadResolvesIncludedTypes(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "types.bhaus"),
		"VERSION 0.1\nSTRUCT Domain/Account:\n    PUBLIC balance: Integer\n")
	main := filepath.Join(dir, "main.bhaus")
	write(t, main,
		"VERSION 0.1\nINCLUDE types.bhaus\nSTRUCT Domain/User:\n    PUBLIC account: Domain/Account\n")

	files, rootURI, err := Load(main)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("loaded %d files, want 2 (root + include)", len(files))
	}

	diags := Check(files, rootURI)
	if len(diags) != 0 {
		t.Fatalf("expected clean file, got %v", diags)
	}
}

func TestLoadDetectsCrossFileDuplicate(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "types.bhaus"),
		"VERSION 0.1\nSTRUCT Domain/User:\n    PUBLIC id: Integer\n")
	main := filepath.Join(dir, "main.bhaus")
	write(t, main,
		"VERSION 0.1\nINCLUDE types.bhaus\nSTRUCT Domain/User:\n    PUBLIC name: String\n")

	files, rootURI, err := Load(main)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	d, ok := firstOfRule(Check(files, rootURI), "duplicate-decl")
	if !ok {
		t.Fatal("expected a duplicate-decl diagnostic across the include boundary")
	}
	// The other declaration lives in types.bhaus. The message must name it.
	if !strings.Contains(d.Message, "types.bhaus") {
		t.Fatalf("message %q should mention types.bhaus", d.Message)
	}
}

func TestLoadMissingRootFileErrors(t *testing.T) {
	if _, _, err := Load(filepath.Join(t.TempDir(), "nope.bhaus")); err == nil {
		t.Fatal("expected an error loading a missing root file")
	}
}
