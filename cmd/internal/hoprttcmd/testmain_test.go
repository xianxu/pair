package hoprttcmd

import (
	"os"
	"testing"

	"github.com/xianxu/pair/cmd/internal/testpair"
)

func TestMain(m *testing.M) {
	os.Exit(testpair.Run(m))
}
