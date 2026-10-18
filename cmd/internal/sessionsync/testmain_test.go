package sessionsync

import (
	"os"
	"testing"

	"github.com/xianxu/pair/cmd/internal/testdisk"
)

func TestMain(m *testing.M) {
	if os.Getenv("SESSIONSYNC_RACE_FILE") != "" {
		os.Exit(m.Run())
	}
	os.Exit(testdisk.RunWithWriteBudget("sessionsync", testdisk.SessionSync, m.Run))
}
