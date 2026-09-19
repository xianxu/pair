package launcher

import (
	"encoding/json"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/checkpoint"
	"io"
	"strings"
)

// In-session continuation publishes a validated handoff before teardown. Couch
// owns hosted replacement; standalone sessions publish an exact snapshot marker
// for their existing outer launcher. Neither path resolves a slug after shutdown.

// compactionDecision decides whether an in-pane `continue` compacts (shell
// 1035-1043). PAIR_FORCE_IN_SESSION forces it (bypassing both halves);
// otherwise it needs the ancestry/fake half AND a tag-match — the guard against
// cmux leaking ZELLIJ_SESSION_NAME to sibling non-pair panes (env-only detection
// would park+kill the wrong session).
func compactionDecision(forceInSession, inPaneOrFake bool, pairTag, zellijSession, pairSession string) bool {
	if forceInSession {
		return true
	}
	return inPaneOrFake && pairTag != "" && sessionMatchesTag(zellijSession, pairTag, pairSession)
}

func sessionMatchesTag(session, tag, pairSession string) bool {
	if session == "" || tag == "" {
		return false
	}
	if pairSession != "" {
		return session == pairSession
	}
	// Legacy shape, reached only when PAIR_SESSION_NAME is unset. The 📁 scheme
	// is not derivable from a tag, so there is nothing better to compare against
	// here — the pairSession branch above is the real answer (#130).
	return session == legacySessionPrefix+tag
}

// serializeRestartMarker renders a RestartMarker as the `key=value` text
// pair-restart.sh's format expects — the inverse of parseRestartMarker, so a
// marker written here round-trips through ReadRestartMarker. Only non-empty
// fields are emitted (matching the shell's compaction write, 1052-1057).
func serializeRestartMarker(m RestartMarker) string {
	if m.Version != 0 {
		raw, _ := json.Marshal(m)
		return string(raw) + "\n"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "tag=%s\n", m.Tag)
	fmt.Fprintf(&b, "agent=%s\n", m.Agent)
	if m.NewSession {
		b.WriteString("new_session=1\n")
	}
	if m.SessionID != "" {
		fmt.Fprintf(&b, "session_id=%s\n", m.SessionID)
	}
	if m.RenameTo != "" {
		fmt.Fprintf(&b, "rename_to=%s\n", m.RenameTo)
	}
	if m.Continue != "" {
		fmt.Fprintf(&b, "continue=%s\n", m.Continue)
	}
	return b.String()
}

// runCompaction delegates hosted ownership to Couch or durably publishes the
// standalone snapshot, checks quit intent, and stops the exact source session.
func runCompaction(opts LaunchOptions, rt Runtime, stderr io.Writer) (int, error) {
	tag := opts.PairTag
	if tag == "" {
		fmt.Fprintf(stderr, "pair: compaction needs PAIR_TAG\n") // shell 1046
		return 1, nil
	}
	agent := firstNonEmpty(opts.PairAgent, opts.Args.Agent, "claude")
	session := opts.ZellijSession
	if session == "" {
		// PAIR_SESSION_NAME is authoritative; the legacy spelling is a last-resort
		// fallback for a pre-#130 session whose env is missing it.
		session = firstNonEmpty(opts.PairSession, legacySessionPrefix+tag)
	}
	if opts.ContinueCheckpoint.Version == 0 {
		fmt.Fprintln(stderr, "pair: continuation checkpoint was not validated; source kept running")
		return 1, nil
	}
	if err := opts.ContinueCheckpoint.Validate(); err != nil {
		fmt.Fprintf(stderr, "pair: %v\n", err)
		return 1, nil
	}
	if opts.Env.CouchHosted() {
		if opts.Env.CouchThreadScope == "" || opts.Env.CouchThreadTag != tag {
			fmt.Fprintln(stderr, "pair: Couch continuation address does not match this pane")
			return 1, nil
		}
		rt.SetEnv(checkpoint.DigestEnv, opts.ContinueCheckpoint.Digest)
		if err := rt.RequestCouchContinuation(opts.ContinueCheckpoint.SourcePath); err != nil {
			fmt.Fprintf(stderr, "pair: continuation request failed; checkpoint kept at %s: %v\n", opts.ContinueCheckpoint.SourcePath, err)
			return 1, nil
		}
		fmt.Fprintln(stderr, "pair: continuation accepted by Couch; the thread owner will restart from the saved checkpoint")
		return 0, nil
	}
	// A session Couch presents without having created it has no thread address
	// to route to, and its client would refuse the marker below (#284). Every
	// other arm here names what it retained and how to resume; so does this one,
	// because the checkpoint is already written and validated by now, and
	// Couch's relaunch keeps the conversation rather than consuming it.
	if err := couchRestartGate(rt, opts.Env.CouchHosted(), tag); err != nil {
		fmt.Fprintf(stderr, "pair: compaction: %v; checkpoint kept at %s — compact from the relaunched thread, or pair continue --retry %s once this session is gone\n",
			err, opts.ContinueCheckpoint.SourcePath, tag)
		return 1, nil
	}
	fmt.Fprintf(stderr, "pair: compacting %s — parking scrollback, restarting from continuation…\n", session)
	attempt := rt.MintLaunchNonce()
	if attempt == "" {
		fmt.Fprintln(stderr, "pair: cannot allocate continuation restart attempt")
		return 1, nil
	}
	saved := readSavedConfig(rt, resolveConfigPath(rt, opts.Env.DataDir, tag, agent))
	marker := RestartMarker{AgentArgs: FreshAgentArgs(saved.Args), Version: 1, Attempt: attempt, Tag: tag, Agent: agent, NewSession: true, Continue: opts.ContinueSlug, Checkpoint: opts.ContinueCheckpoint}
	if err := rt.WriteRestartMarker(session, marker); err != nil {
		fmt.Fprintf(stderr, "pair: write continuation restart intent: %v\n", err)
		return 1, nil
	}
	if err := writeQuitIntent(rt, session, QuitIntent{Version: QuitIntentVersion, Kind: QuitIntentDirect}); err != nil {
		fmt.Fprintf(stderr, "pair: write continuation quit intent: %v; retry with pair continue --retry %s\n", err, tag)
		return 1, nil
	}
	rt.ParkScrollback(tag, agent, false)
	if err := rt.ExecKillSession(session); err != nil {
		fmt.Fprintf(stderr, "pair: source stop failed: %v; checkpoint retained, inspect source before pair continue --retry %s\n", err, tag)
		return 1, nil
	}
	return 0, nil
}
