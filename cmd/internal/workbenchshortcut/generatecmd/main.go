package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
)

func main() {
	out := flag.String("out", "", "path to generated Lua file")
	flag.Parse()
	if *out == "" {
		fmt.Fprintln(os.Stderr, "generatecmd: --out is required")
		os.Exit(2)
	}
	if err := os.WriteFile(*out, []byte(workbenchshortcut.RenderLuaGlobalMaps()), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "generatecmd: %v\n", err)
		os.Exit(1)
	}
}
