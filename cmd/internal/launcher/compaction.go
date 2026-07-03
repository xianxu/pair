package launcher

import (
	"fmt"
	"io"
	"strings"
)

// In-session compaction (#55, #99 M5b, ported from bin/pair-shell 1019-1062).
// `pair continue <slug>` run from INSIDE the matching live pane must not
// fresh-start (a nested --session would break, and the create path's name prompt
// would block). Instead: park the scrollback (copy — pair-wrap is still appending
// to .raw), drop a restart marker carrying the slug, kill the session; the outer
// RunLaunch loop then re-launches fresh under the same tag. The decision + marker
// serialization are pure; park/marker-write/kill are Runtime effects.

// compactionDecision decides whether an in-pane `continue` compacts (shell
// 1035-1043). PAIR_FORCE_IN_SESSION forces it (bypassing both halves);
// otherwise it needs the ancestry/fake half AND a tag-match — the guard against
// cmux leaking ZELLIJ_SESSION_NAME to sibling non-pair panes (env-only detection
// would park+kill the wrong session).
func compactionDecision(forceInSession, inPaneOrFake bool, pairTag, zellijSession string) bool {
	if forceInSession {
		return true
	}
	return inPaneOrFake && pairTag != "" && zellijSession == "pair-"+pairTag
}

// serializeRestartMarker renders a RestartMarker as the `key=value` text
// pair-restart.sh's format expects — the inverse of parseRestartMarker, so a
// marker written here round-trips through TakeRestartMarker. Only non-empty
// fields are emitted (matching the shell's compaction write, 1052-1057).
func serializeRestartMarker(m RestartMarker) string {
	var b strings.Builder
	fmt.Fprintf(&b, "tag=%s\n", m.Tag)
	fmt.Fprintf(&b, "agent=%s\n", m.Agent)
	if m.NewSession {
		b.WriteString("new_session=1\n")
	}
	if m.RenameTo != "" {
		fmt.Fprintf(&b, "rename_to=%s\n", m.RenameTo)
	}
	if m.Continue != "" {
		fmt.Fprintf(&b, "continue=%s\n", m.Continue)
	}
	return b.String()
}

// runCompaction executes the in-pane compaction (shell 1045-1060): park the
// scrollback (copy), write the restart marker (new_session + continue slug),
// touch the quit marker, then exec kill-session. ExecKillSession is terminal
// (replaces the process), so the return is unreachable on the real runtime — it
// exists so the fake-Runtime loop test can observe the sequence.
func runCompaction(opts LaunchOptions, rt Runtime, stderr io.Writer) (int, error) {
	tag := opts.PairTag
	if tag == "" {
		fmt.Fprintf(stderr, "pair: compaction needs PAIR_TAG\n") // shell 1046
		return 1, nil
	}
	agent := firstNonEmpty(opts.PairAgent, opts.Args.Agent, "claude")
	fmt.Fprintf(stderr, "pair: compacting pair-%s — parking scrollback, restarting from continuation…\n", tag)
	session := "pair-" + tag
	rt.ParkScrollback(tag, agent, false) // copy: the live .raw is still being appended
	rt.WriteRestartMarker(session, RestartMarker{Tag: tag, Agent: agent, NewSession: true, Continue: opts.ContinueSlug})
	rt.TouchQuitMarker(session)
	rt.ExecKillSession(session)
	return 0, nil
}
