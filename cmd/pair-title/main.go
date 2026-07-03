// pair-title — the per-tag title poller (zellij frame meter + cmux workspace
// heat-ramp). Thin entry over titlepoller.RunCLI; the logic lives in
// cmd/internal/titlepoller (#93 M1, ported from bin/pair-title.sh). Spawned in
// the background by the launcher on create + attach (directly since #94 M2).
package main

import (
	"os"

	"github.com/xianxu/pair/cmd/internal/titlepoller"
)

func main() {
	os.Exit(titlepoller.RunCLI(os.Args[1:], os.Getenv, os.Stderr))
}
