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

// ResolveInvocation classifies an invocation and returns the args the
// dispatcher should see. It is ClassifyInvocation plus busybox arg-rewrite: when
// pair is invoked under a helper's base name (a symlink, e.g. the external
// pair-slug Stop hook), the resolved subcommand is prepended to args so the
// dispatcher runs it. For the `pair` launcher and the launch handoff, args pass
// through unchanged.
func ResolveInvocation(executable string, args []string, dispatchNames []string) (EntrypointMode, []string) {
	mode := ClassifyInvocation(executable, args, dispatchNames)
	if mode == ModeDispatch && filepath.Base(executable) != "pair" {
		if sub, ok := busyboxSubcommand(filepath.Base(executable), dispatchNames); ok {
			return ModeDispatch, append([]string{sub}, args...)
		}
	}
	return mode, args
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}
