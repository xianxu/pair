package couchcmd

import (
	"fmt"
	"os"
	"testing"

	"github.com/xianxu/pair/cmd/internal/testdisk"
	"github.com/xianxu/pair/cmd/internal/testpair"
)

func TestMain(m *testing.M) {
	run := func() int {
		code := testpair.Run(m)
		if err := scheduleCouchCache.Close(); err != nil {
			fmt.Fprintln(os.Stderr, "remove suite-owned scheduler Couch:", err)
			code = 1
		}
		return code
	}
	if os.Getenv("PAIR_CAPTURE_CHILD_HELPER") == "1" || os.Getenv(continuationPublicationFixtureEnv) != "" || os.Getenv("PAIR_PROVISION_SIGNAL_HELPER") == "1" {
		os.Exit(run())
	}
	os.Exit(testdisk.RunWithWriteBudget("couchcmd", testdisk.CouchCmd, run))
}
