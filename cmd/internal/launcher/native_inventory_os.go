package launcher

import (
	"os"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// nativeInventory applies the same native transcript namespace to selection,
// ledger identity and recovery validation. It performs no startup/config loads.
// Empty cwd is unknown historical startup context; relative homes must refuse.
func (r OSRuntime) nativeInventory(agent, cwd string) (sessioninventory.OSRuntime, error) {
	return sessioninventory.NewNativeTranscriptRuntime(os.Getenv("HOME"), r.DataDir, sessioninventory.Agent(agent), cwd, os.Getenv)
}
