// Command couch supervises agent actors, one per working tree.
//
// It is a separate binary from pair on purpose: pair is the thing the operator
// sits inside, so a supervisor bug must not break the ability to fix it. The
// fallback is always to launch pair the old way.
package main

import (
	"fmt"
	"io"
	"os"
	"runtime"

	"github.com/xianxu/pair/cmd/internal/couchcmd"
	"github.com/xianxu/pair/cmd/internal/crashreport"
	"github.com/xianxu/pair/cmd/internal/launcher"
)

func main() {
	code := runWithPinnedSocketRoot(func() int {
		return couchcmd.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	}, os.Stderr)
	// Only a normal return reaches this line, which is the point: a panic
	// must leave its crash file behind (#397).
	_ = crashreport.Finish()
	os.Exit(code)
}

// runWithPinnedSocketRoot pins the native namespace before Couch starts any
// workers or commands. Slot temp overlays must not redirect native discovery.
func runWithPinnedSocketRoot(run func() int, stderr io.Writer) int {
	root, err := launcher.ResolveZellijSocketRoot(os.Getenv("ZELLIJ_SOCKET_DIR"), os.TempDir(), os.Getenv("XDG_RUNTIME_DIR"), runtime.GOOS, os.Getuid())
	if err == nil {
		err = os.Setenv("ZELLIJ_SOCKET_DIR", root)
	}
	if err != nil {
		fmt.Fprintf(stderr, "couch: Zellij socket namespace: %v\n", err)
		return 1
	}
	return run()
}
