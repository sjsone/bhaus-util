package cli

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/sjsone/bhaus-util/pkg/lsp"
)

// handleLS starts the LSP server over stdio. args are the arguments following
// the "ls" subcommand (i.e. os.Args[2:]). RunStdio blocks until the client
// disconnects. Under normal operation, this function returns 0 only after
// shutdown.
func handleLS(args []string) int {
	fs := flag.NewFlagSet("ls", flag.ExitOnError)
	fs.Usage = func() { lsUsage(os.Stderr) }
	logFile := fs.String("log-file", "", "log file path (default: <user cache dir>/bhaus/bhaus-util.log)")
	logVerbosity := fs.Int("log-verbosity", 1, "log verbosity: 0=Notice, 1=Info, 2=Debug, -4=off")
	fs.Parse(args)

	logPath, err := setupLogging(*logFile, *logVerbosity)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bhaus-util: failed to set up file logging: %v\n", err)
	}
	log.Printf("Starting BHaus language server (pid %d, log file: %s)", os.Getpid(), logPath)

	handler := lsp.NewHandler()
	handler.Server.RunStdio()
	return 0
}

func lsUsage(w io.Writer) {
	fmt.Fprint(w, `bhaus-util ls — run the BHaus language server over stdio

Usage:
  bhaus-util ls [flags]

Speaks the Language Server Protocol on stdin/stdout. This is meant to be launched
by an editor (VS Code, Zed, Neovim, ...), not run by hand — on its own it will just
wait for JSON-RPC on stdin. Because stdout carries the protocol, all server logs
go to a file instead.

Flags:
  --log-file <path>       Log file path.
                          Default: <user cache dir>/bhaus/bhaus-util.log.
  --log-verbosity <int>   0=Notice, 1=Info (default), 2=Debug, -4=off.

Examples:
  # Configure your editor to run the server with debug logging
  bhaus-util ls --log-verbosity 2

  # Follow the log while the editor is connected
  tail -f "$(getconf DARWIN_USER_CACHE_DIR 2>/dev/null || echo ~/.cache)/bhaus/bhaus-util.log"
`)
}
