// pair-review-readiness — print the review-start readiness case (JSON) or perform
// the deterministic --prepare git effects. Thin entry over
// reviewcmd.RunReadinessCLI (#93 M3, ported from bin/pair-review-readiness).
package main

import (
	"os"

	"github.com/xianxu/pair/cmd/internal/reviewcmd"
)

func main() {
	os.Exit(reviewcmd.RunReadinessCLI(os.Args[1:], os.Getenv, os.Stdout, os.Stderr))
}
