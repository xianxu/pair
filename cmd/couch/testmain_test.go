package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/xianxu/pair/cmd/internal/testpair"
)

func TestMain(m *testing.M) {
	code := testpair.Run(m)
	if err := installedBinaryCache.Close(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		code = 1
	}
	os.Exit(code)
}
