package launcher

import (
	"errors"
	"os"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// nativeInventory applies the same native transcript namespace to selection,
// ledger identity and recovery validation. It performs no startup/config loads.
// Empty cwd means the process startup directory for callers without a cwd.
func (r OSRuntime) nativeInventory(agent, cwd string) (sessioninventory.OSRuntime, error) {
	native := sessioninventory.NewOSRuntime(os.Getenv("HOME"), r.DataDir)
	if agent != string(sessioninventory.AgentCodex) {
		return native, nil
	}
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return sessioninventory.OSRuntime{}, errors.New("Codex native home unavailable")
		}
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
