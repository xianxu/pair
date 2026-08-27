package main

import (
	"os"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func main() {
	os.Exit(couchcore.LaunchHelperMain(os.Args[1:], os.Stderr))
}
