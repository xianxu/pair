package threadactivity

import (
	"os"
	"testing"

	"github.com/xianxu/pair/cmd/internal/testdisk"
)

func TestMain(m *testing.M) {
	os.Exit(testdisk.RunWithWriteBudget("threadactivity", testdisk.ThreadActivity, m.Run))
}
