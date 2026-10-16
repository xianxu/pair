package launcher

import (
	"errors"
	"os"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// nativeInventory applies the same native transcript namespace to selection,
// ledger identity and recovery validation. It performs no startup/config loads.
// Empty cwd is unknown historical startup context; relative homes must refuse.
func (r OSRuntime) nativeInventory(agent, cwd string) (sessioninventory.OSRuntime, error) {
	native := sessioninventory.NewOSRuntime(os.Getenv("HOME"), r.DataDir)
	if agent != string(sessioninventory.AgentCodex) {
		return native, nil
	}
	home, err := sessioninventory.ResolveCodexHome(cwd, os.Getenv)
	if err != nil {
		return sessioninventory.OSRuntime{}, errors.New("Codex native home unavailable")
	}
	native, err = native.WithCodexHome(home)
	if err != nil {
		return sessioninventory.OSRuntime{}, errors.New("Codex native home unavailable")
	}
	return native, nil
}
