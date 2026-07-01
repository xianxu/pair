package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/xianxu/pair/cmd/internal/runtimebundlegen"
)

func main() {
	repo := flag.String("repo", ".", "repository root")
	out := flag.String("out", "", "output root")
	flag.Parse()
	if _, err := runtimebundlegen.Generate(runtimebundlegen.GenerateOptions{RepoRoot: *repo, OutRoot: *out}); err != nil {
		fmt.Fprintf(os.Stderr, "runtimebundle-generate: %v\n", err)
		os.Exit(1)
	}
}
