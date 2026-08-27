package lsp

import (
	"testing"

	"github.com/sjsone/bhaus-util/pkg/version"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// TestInitializeReportsBuildVersion guards against the ServerInfo version
// drifting from the real build: it must come from pkg/version, not a literal.
func TestInitializeReportsBuildVersion(t *testing.T) {
	h := NewHandler()

	res, err := h.Initialize(nil, &protocol.InitializeParams{})
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}

	result, ok := res.(protocol.InitializeResult)
	if !ok {
		t.Fatalf("Initialize returned %T, want protocol.InitializeResult", res)
	}
	if result.ServerInfo == nil || result.ServerInfo.Version == nil {
		t.Fatal("ServerInfo.Version is nil")
	}
	if got := *result.ServerInfo.Version; got != version.Version {
		t.Fatalf("ServerInfo.Version = %q, want %q", got, version.Version)
	}
}
