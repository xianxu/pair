package entrypoint

import "path/filepath"

type EntrypointMode int

const (
	ModeDispatch EntrypointMode = iota
	ModePublicPair
	ModePairGoLaunch
)

// ClassifyInvocation decides how an invocation is handled. The public `pair`
// binary is the session launcher — EXCEPT it peels off the reserved dispatcher
// subcommands (dispatchNames, sourced from dispatcher.DispatchNames()) so
// `pair slug` dispatches while `pair claude` / `pair resume` still launch. The
// reserved set is passed in rather than imported to keep entrypoint free of a
// dispatcher dependency.
func ClassifyInvocation(executable string, args []string, dispatchNames []string) EntrypointMode {
	if filepath.Base(executable) == "pair" {
		if len(args) > 0 && contains(dispatchNames, args[0]) {
			return ModeDispatch
		}
		return ModePublicPair
	}
	if len(args) > 0 && args[0] == "launch" {
		return ModePairGoLaunch
	}
	return ModeDispatch
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}
