package cli

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/sjsone/bhaus-util/pkg/version"
)

// captureStdout runs f with os.Stdout redirected to a pipe and returns what was
// written. Unlike withStdout, it keeps the output. This lets tests assert on it.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		done <- buf.String()
	}()

	f()

	os.Stdout = orig
	w.Close()
	out := <-done
	r.Close()
	return out
}

// captureStderr is captureStdout's counterpart, for the error paths.
func captureStderr(t *testing.T, f func()) string {
	t.Helper()
	orig := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		done <- buf.String()
	}()

	f()

	os.Stderr = orig
	w.Close()
	out := <-done
	r.Close()
	return out
}

func TestVersionSubcommandPrintsVersion(t *testing.T) {
	out := captureStdout(t, func() {
		if code := Run([]string{"bhaus-util", "version"}); code != 0 {
			t.Errorf("exit = %d, want 0", code)
		}
	})
	if !strings.Contains(out, version.Version) {
		t.Fatalf("output %q does not contain version %q", out, version.Version)
	}
}

func TestVersionFlagPrintsVersion(t *testing.T) {
	for _, flag := range []string{"--version", "-v"} {
		t.Run(flag, func(t *testing.T) {
			out := captureStdout(t, func() {
				if code := Run([]string{"bhaus-util", flag}); code != 0 {
					t.Errorf("exit = %d, want 0", code)
				}
			})
			if !strings.Contains(out, version.Version) {
				t.Fatalf("%s output %q does not contain version %q", flag, out, version.Version)
			}
		})
	}
}
