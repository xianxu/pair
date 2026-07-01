// pair-review-target — write the session-scoped review-target seam. Thin entry
// over reviewcmd.RunTargetCLI (#93 M3, ported from bin/pair-review-target).
package main

import (
	"os"

	"github.com/xianxu/pair/cmd/internal/reviewcmd"
)

func main() {
	os.Exit(reviewcmd.RunTargetCLI(os.Args[1:], os.Getenv, os.Stdout, os.Stderr))
}
