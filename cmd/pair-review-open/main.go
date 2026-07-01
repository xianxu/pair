// pair-review-open — spawn the full-screen floating nvim review pane. Thin entry
// over reviewcmd.RunOpenCLI (#93 M3, ported from bin/pair-review-open).
package main

import (
	"os"

	"github.com/xianxu/pair/cmd/internal/reviewcmd"
)

func main() {
	os.Exit(reviewcmd.RunOpenCLI(os.Args[1:], os.Getenv, os.Stderr))
}
