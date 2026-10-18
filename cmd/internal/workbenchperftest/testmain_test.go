package workbenchperftest

import (
	"os"
	"testing"

	"github.com/xianxu/pair/cmd/internal/testdisk"
)

func TestMain(m *testing.M) {
	os.Exit(testdisk.RunWithWriteBudget("workbenchperftest", testdisk.WorkbenchFixture, m.Run))
}
