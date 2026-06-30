// scrollback-render — replay a pair-wrap raw capture through a VT100 emulator.
package main

import (
	"os"

	"github.com/xianxu/pair/cmd/internal/scrollbackcmd"
)

func main() {
	os.Exit(scrollbackcmd.Run(os.Args[1:], os.Stdout, os.Stderr))
}
