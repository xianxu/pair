// pair-slug — propose an orientation slug for a pair tab. Thin shim over
// slugcmd.Run; the logic + docs live in cmd/internal/slugcmd (shared with the
// `pair slug` dispatcher route). Spawned (backgrounded) by pair-wrap at
// turn-end; env-driven, no stdin, exits 0 on any failure.
package main

import (
	"os"

	"github.com/xianxu/pair/cmd/internal/slugcmd"
)

func main() { os.Exit(slugcmd.Run()) }
