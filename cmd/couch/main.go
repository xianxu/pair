// Command couch supervises agent actors, one per working tree.
//
// It is a separate binary from pair on purpose: pair is the thing the operator
// sits inside, so a supervisor bug must not break the ability to fix it. The
// fallback is always to launch pair the old way.
package main

import (
	"os"

	"github.com/xianxu/pair/cmd/internal/couchcmd"
)

func main() {
	os.Exit(couchcmd.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
