package lsp

import "github.com/tliron/commonlog"

// logger is the shared commonlog logger for the LSP layer. The process
// configures commonlog's output destination (see cmd/bhaus-util/main.go).
// The --log-verbosity flag controls the log level:
//
//	Info:  one line per request, with the target and a short result summary.
//	Debug: per-request detail, such as the cursor word, resolved symbols and counts.
//
// High-frequency handlers (completion, on-type formatting) log only at Debug.
// This avoids flooding the Info stream.
var logger = commonlog.GetLogger("bhaus.lsp")
