package entrypoint

import "path/filepath"

// LegacyLaunchRequest describes the current compatibility handoff from
// pair-go launch to the shell-owned pair launcher.
type LegacyLaunchRequest struct {
	Path string
	Argv []string
}

func ResolveLegacyLaunch(executable string, launchArgs []string) LegacyLaunchRequest {
	argv := make([]string, 0, len(launchArgs)+1)
	argv = append(argv, "pair")
	argv = append(argv, launchArgs...)
	return LegacyLaunchRequest{
		Path: filepath.Join(filepath.Dir(executable), "pair"),
		Argv: argv,
	}
}
