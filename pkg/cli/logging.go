package cli

import (
	"log"
	"os"
	"path/filepath"

	"github.com/tliron/commonlog"
	"github.com/tliron/commonlog/simple"
)

// setupLogging routes commonlog (used internally by glsp) and the standard
// library log package into a single log file. An editor usually launches the
// server over stdio. In that setup, stdout carries the JSON-RPC channel and
// stderr stays hidden. A file is therefore the only practical place to see
// server logs.
//
// path is the --log-file value. When empty, it defaults to
// <user cache dir>/bhaus/bhaus-util.log. verbosity is the --log-verbosity
// value (1 = Info; 0 = Notice; 2 = Debug; -4 disables). Writes are unbuffered,
// so entries show up immediately under `tail -f`.
func setupLogging(path string, verbosity int) (string, error) {
	if path == "" {
		dir, err := os.UserCacheDir()
		if err != nil {
			dir = os.TempDir()
		}
		path = filepath.Join(dir, "bhaus", "bhaus-util.log")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return path, err
	}

	backend := simple.NewBackend()
	backend.Buffered = false // Flush every entry. The log file is for live debugging.
	backend.Configure(verbosity, &path)
	commonlog.SetBackend(backend)

	// Send stdlib log output (used throughout the handlers) to the same file.
	log.SetOutput(commonlog.GetWriter())
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	return path, nil
}
