package entrypoint

// LegacyLaunchRequest describes the current compatibility handoff from the Go
// entrypoint to the shell-owned pair launcher.
type LegacyLaunchRequest struct {
	Path string
	Argv []string
}

func ResolveLegacyLaunch(root AssetRoot, launchArgs []string) LegacyLaunchRequest {
	argv := make([]string, 0, len(launchArgs)+1)
	argv = append(argv, "pair")
	argv = append(argv, launchArgs...)
	return LegacyLaunchRequest{
		Path: root.ShellPath,
		Argv: argv,
	}
}
