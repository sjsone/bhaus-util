package version

import (
	"runtime"
	"strings"
	"testing"
)

// TestDefaultVersionIsNotEmpty documents that an un-stamped build still reports
// something ("dev") rather than an empty string. An empty string would surface
// as a blank LSP ServerInfo version in the client.
func TestDefaultVersionIsNotEmpty(t *testing.T) {
	if Version == "" {
		t.Fatal("Version must never be empty")
	}
}

func TestStringIncludesVersionAndPlatform(t *testing.T) {
	got := String()
	for _, want := range []string{Version, runtime.GOOS, runtime.GOARCH, runtime.Version()} {
		if !strings.Contains(got, want) {
			t.Errorf("String() = %q, missing %q", got, want)
		}
	}
}

// TestStringOmitsEmptyCommit keeps unstamped output free of an empty "()".
func TestStringOmitsEmptyCommit(t *testing.T) {
	orig := Commit
	t.Cleanup(func() { Commit = orig })

	Commit = ""
	if got := String(); strings.Contains(got, "()") {
		t.Errorf("String() = %q, want no empty parentheses", got)
	}

	Commit = "abc1234"
	if got := String(); !strings.Contains(got, "(abc1234)") {
		t.Errorf("String() = %q, want the commit in parentheses", got)
	}
}
