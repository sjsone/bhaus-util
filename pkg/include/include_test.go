package include

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestPatterns(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    []string
	}{
		{"none", "VERSION 0.1\nSTRUCT X:\n", nil},
		{"literal", "INCLUDE types.bhaus\n", []string{"types.bhaus"}},
		{"wildcard gets suffix", "INCLUDE types/*\n", []string{"types/*.bhaus"}},
		{"wildcard keeps suffix", "INCLUDE types/*.bhaus\n", []string{"types/*.bhaus"}},
		{"multiple", "INCLUDE a.bhaus\nINCLUDE b/*\n", []string{"a.bhaus", "b/*.bhaus"}},
		{"leading whitespace", "   INCLUDE  a.bhaus  \n", []string{"a.bhaus"}},
		{"empty pattern ignored", "INCLUDE \n", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Patterns(tc.content)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Patterns() = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestMatchGlobsRelativeToFileDir(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.bhaus", "b.bhaus", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("VERSION 0.1\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	uri := "file://" + filepath.Join(dir, "root.bhaus")
	got := Match(uri, "INCLUDE *.bhaus\n")
	sort.Strings(got)

	want := []string{filepath.Join(dir, "a.bhaus"), filepath.Join(dir, "b.bhaus")}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Match() = %v, want %v", got, want)
	}
}

func TestMatchDedupesAcrossPatterns(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.bhaus"), []byte("VERSION 0.1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := "file://" + filepath.Join(dir, "root.bhaus")
	// Two patterns that both match a.bhaus.
	got := Match(uri, "INCLUDE a.bhaus\nINCLUDE *.bhaus\n")
	if len(got) != 1 {
		t.Fatalf("Match() = %v, want a single deduped path", got)
	}
}

func TestMatchNoIncludes(t *testing.T) {
	if got := Match("file:///x.bhaus", "VERSION 0.1\n"); got != nil {
		t.Fatalf("Match() = %v, want nil", got)
	}
}
