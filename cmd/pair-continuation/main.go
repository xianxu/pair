// pair-continuation — write and commit a continuation datatype instance. Thin
// shim over continuationcmd.Run; logic lives in cmd/internal/continuationcmd
// (shared with the `pair continuation` dispatcher route). Reads the body from
// --body-file (or stdin).
package main

import (
	"os"
	"time"

	"github.com/xianxu/pair/cmd/internal/continuationcmd"
)

func main() {
	os.Exit(continuationcmd.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, time.Now))
}
