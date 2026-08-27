package version

import (
	"fmt"
	"runtime"
)

// Version is the release version. Unstamped builds report "dev" rather than
// claiming a release number.
var Version = "dev"

// Commit is the git revision used to build the binary. It is empty when unstamped.
var Commit = ""

// String returns a one-line build identifier suitable for bug reports.
func String() string {
	v := Version
	if Commit != "" {
		v += " (" + Commit + ")"
	}
	return fmt.Sprintf("%s %s/%s %s", v, runtime.GOOS, runtime.GOARCH, runtime.Version())
}
