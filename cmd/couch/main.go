// Command couch supervises agent actors, one per working tree.
//
// It is a separate binary from pair on purpose: pair is the thing the operator
// sits inside, so a supervisor bug must not break the ability to fix it. The
// fallback is always to launch pair the old way.
package main

import (
	"os"

	"github.com/xianxu/pair/cmd/internal/couchcmd"
	"github.com/xianxu/pair/cmd/internal/crashreport"
)

func main() {
	code := couchcmd.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	// Only a normal return reaches this line, which is the point: a panic
	// must leave its crash file behind (#397).
	_ = crashreport.Finish()
	os.Exit(code)
}
