package entrypoint

import "path/filepath"

type EntrypointMode int

const (
	ModeDispatch EntrypointMode = iota
	ModePublicPair
	ModePairGoLaunch
)

func ClassifyInvocation(executable string, args []string) EntrypointMode {
	if filepath.Base(executable) == "pair" {
		return ModePublicPair
	}
	if len(args) > 0 && args[0] == "launch" {
		return ModePairGoLaunch
	}
	return ModeDispatch
}
