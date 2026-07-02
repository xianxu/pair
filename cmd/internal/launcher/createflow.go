package launcher

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"
)

// RunLaunch is the native create-flow orchestrator (#99 M2): a thin driver over
// the pure deciders (DecideLaunch + createlogic.go + M1's agentargs) that maps
// each shell create-branch step to a Runtime effect. It owns ONLY the create
// path — a decision that resolves to attach/pick (or a launch from inside an
// existing zellij pane) returns ErrFallbackToShell so the cmd/pair-go gate defers
// to bin/pair-shell until M3 lands those flows. Any other return is handled:
// RunLaunch has already written user-facing messages to stderr, and the int is
// the process exit code (the blocking zellij child's, or an abort/error code).
//
// It stops at the blocking zellij handoff. The post-handoff cleanup_quit_marker
// + restart-marker re-entry loop are M3 (LaunchSession is fork+wait precisely so
// the Go launcher regains control to run them).
func RunLaunch(opts LaunchOptions, rt Runtime, stderr io.Writer) (int, error) {
	env := normalizeEnv(opts.Env)
	agent := opts.Args.Agent

	// Launches from inside an existing zellij pane are not fresh-terminal
	// creates — they are either the "already inside zellij" error or the #55
	// in-session compaction, both M3. Defer the whole in-pane surface to the
	// shell for now.
	if rt.InZellijPane() {
		return 0, ErrFallbackToShell
	}

	base := DefaultTag(env.Cwd)
	cutoff := env.Now.Add(-time.Duration(env.HistoryD) * 24 * time.Hour)
	sessions, err := rt.Sessions()
	if err != nil {
		return 0, ErrFallbackToShell
	}
	historical, err := rt.ScanHistory(base, cutoff)
	if err != nil {
		return 0, ErrFallbackToShell
	}
	decision, err := DecideLaunch(opts.Args, SessionSnapshot{BaseTag: base, Sessions: sessions, Historical: historical})
	if err != nil {
		return 0, ErrFallbackToShell
	}
	if decision.Action != ActionCreate {
		// attach / pick — M3.
		return 0, ErrFallbackToShell
	}
	// The shell also sweeps orphan nvim --embed processes here (sweep_orphan_nvim,
	// shell 1243); that reaping is grouped with M3's reap_nvim_for_tag (both nvim
	// concerns) and deferred — it's best-effort hygiene, not create-path-load-bearing.

	// `pair resume <tag>` leaves the agent unset (the parse defers it): infer
	// what the tag was last paired with from disk, defaulting to claude — so a
	// single tag fully restarts regardless of the original agent (shell 993-1007).
	if agent == "" {
		agent = rt.InferAgent(decision.Tag)
		if agent == "" {
			agent = "claude"
		}
	}

	// Validate the agent here (deferred past the decision so attach paths, once
	// native, still work when AGENT isn't a real binary — shell 1728).
	if !rt.CommandExists(agent) {
		fmt.Fprintf(stderr, "pair: agent '%s' not found on PATH.\n", agent)
		fmt.Fprintf(stderr, "      install it first, then re-run.\n")
		return 1, nil
	}

	chosenTag := decision.Tag
	if decision.PromptName {
		tag, code, ok := promptForTag(rt, decision.Tag, base, stderr)
		if !ok {
			return code, nil
		}
		chosenTag = tag
	}

	session := "pair-" + chosenTag
	dataDir := env.DataDir
	configPath := resolveConfigPath(rt, dataDir, chosenTag, agent)

	agentArgs := append([]string(nil), opts.Args.AgentArgs...)

	// Tag-restart config picker (#000016): a saved config for this (tag, agent)
	// offers to reuse its args / resume its session, unless an explicit resume
	// token on argv already made the choice.
	if code, ok := runConfigPicker(rt, configPath, agent, chosenTag, &agentArgs, env.Cwd, stderr); !ok {
		return code, nil
	}

	// Env exports every child (watcher, poller, zellij + its panes) inherits.
	rt.SetEnv("PAIR_HOME", opts.PairHome)
	rt.SetEnv("PAIR_DATA_DIR", dataDir)
	rt.SetEnv("PAIR_TAG", chosenTag)
	rt.SetEnv("PAIR_AGENT", agent)

	draft := filepath.Join(dataDir, "draft-"+chosenTag+".md")
	_ = rt.Touch(draft)
	if opts.ContinueDoc != "" {
		_ = rt.WriteAtomic(draft, fmt.Sprintf("Read workshop/continuation/%s and continue from its NEXT ACTION.\n", filepath.Base(opts.ContinueDoc)))
	}

	// Record the agent for `pair list` / the title poller (survives detach).
	_ = rt.WriteAtomic(filepath.Join(dataDir, "agent-"+chosenTag), agent+"\n")

	// Pre-capture the session id an explicit --resume/--conversation/`resume`
	// pinned: the watcher only catches NEW jsonl files, so an explicit resume
	// needs the config written synchronously (shell 2053-2110).
	explicitResume := extractExplicitResume(agent, agentArgs)
	if explicitResume != "" {
		writeConfig(rt, configPath, agent, persistedConfigArgs(agentArgs), explicitResume)
	}

	// Claude: mint a deterministic --session-id (uuidgen + collision retry) so
	// two tags in one cwd can't race for the same new jsonl (#20). Codex/agy
	// have no such flag — the watcher discovers their id async.
	newSid := ""
	if shouldMintClaudeSessionID(agent, explicitResume, agentArgs) {
		for i := 0; i < 5; i++ {
			cand := rt.MintUUID()
			if cand != "" && !rt.AgentSessionExists("claude", cand, env.Cwd) {
				newSid = cand
				break
			}
		}
		if newSid != "" {
			agentArgs = append(agentArgs, "--session-id", newSid)
			writeConfig(rt, configPath, agent, persistedConfigArgs(agentArgs), newSid)
		}
	}

	// Codex: force inline mode so the conversation flows through zellij's
	// scrollback (idempotent; opt-out via PAIR_CODEX_ALT_SCREEN=1).
	if agent == "codex" {
		agentArgs = codexAltScreenArgs(agentArgs, opts.CodexAltScreenOptOut)
	}

	rt.SetEnv("PAIR_AGENT_ARGS", strings.Join(agentArgs, " "))
	rt.SetEnv("PAIR_SESSION_ID", firstNonEmpty(explicitResume, newSid))
	rt.SetEnv("PAIR_PANE_TITLE", PaneTitle(agent, env.Cwd, env.Home))
	rt.SetEnv("PAIR_PANE_CWD", TildeAbbrev(env.Cwd, env.Home))

	// Truncate the adaptation flight recorder once, before any appender starts.
	_ = rt.WriteAtomic(filepath.Join(dataDir, "adapt-"+chosenTag+".jsonl"), "")

	// Spawn the (already-Go) sidecars + set the frame title. agentArgs is the
	// final resolved vector (post mint / codex / resume compose).
	rt.SpawnSessionWatcher(agent, chosenTag, env.Cwd, agentArgs)
	rt.SetTerminalTitle(session)
	rt.RecordOuterTTY(chosenTag)
	rt.CmuxRename(chosenTag, session)
	rt.SpawnTitlePoller(chosenTag, agent)
	rt.DevRebuild(opts.PairHome)

	// A too-long tag makes zellij reject the session name (#54); probe its own
	// validator and translate the rejection.
	if err := rt.ProbeSessionName(session); err != nil {
		fmt.Fprintf(stderr, "pair: tag '%s' makes zellij's session name too long for this\n", chosenTag)
		fmt.Fprintf(stderr, "      machine's socket path (%s). Pick a shorter tag.\n", session)
		return 1, nil
	}

	// Free the name (clear a stale EXITED resurrect record) and guard against a
	// live session unexpectedly occupying it before the blocking handoff.
	if rt.SessionBlocksReuse(session) {
		fmt.Fprintf(stderr, "pair: session '%s' already exists.\n", session)
		return 1, nil
	}

	configDir := filepath.Join(opts.PairHome, "zellij")
	layout := filepath.Join(opts.PairHome, "zellij", "layouts", "main.kdl")
	code, err := rt.LaunchSession(session, configDir, layout)
	if err != nil {
		fmt.Fprintf(stderr, "pair: failed to launch zellij session '%s': %v\n", session, err)
		return 1, nil
	}
	// M3: cleanup_quit_marker + handle_restart_marker re-entry loop go here.
	return code, nil
}

// promptForTag runs the editable name prompt, normalizing + collision-checking
// the result. ok=false means the caller should exit with the returned code
// (0 on user abort, 1 on invalid name / collision).
func promptForTag(rt Runtime, prefill, base string, stderr io.Writer) (tag string, code int, ok bool) {
	rt.ShowFamilyExisting("pair-" + base)
	value, entered := rt.PromptSessionName(prefill)
	if !entered {
		return "", 0, false // user aborted (ESC / EOF)
	}
	if value == "" {
		value = prefill
	}
	tag, err := NormalizeTag(value)
	if err != nil {
		fmt.Fprintf(stderr, "pair: invalid name '%s' (allowed: letters, digits, dash, underscore)\n", value)
		return "", 1, false
	}
	if rt.SessionBlocksReuse("pair-" + tag) {
		fmt.Fprintf(stderr, "pair: session 'pair-%s' already exists.\n", tag)
		return "", 1, false
	}
	return tag, 0, true
}

// runConfigPicker drives the tag-restart config picker, mutating *agentArgs to
// the resolved launch vector. ok=false means abort with the returned exit code.
// When no saved config applies (absent, or an explicit resume already chose),
// it is a no-op that returns ok=true.
func runConfigPicker(rt Runtime, configPath, agent, chosenTag string, agentArgs *[]string, cwd string, stderr io.Writer) (code int, ok bool) {
	if extractExplicitResume(agent, *agentArgs) != "" {
		return 0, true // argv already pinned a resume — nothing to offer.
	}
	if _, exists := rt.FileSize(configPath); !exists {
		return 0, true
	}
	raw, err := rt.ReadFile(configPath)
	if err != nil {
		return 0, true
	}
	cfg, err := parseConfig(raw)
	if err != nil {
		return 0, true // unusable config — proceed as if none.
	}

	savedArgsClean := persistedConfigArgs(cfg.Args)
	hasResumable := rt.AgentSessionExists(agent, cfg.SessionID, cwd)
	choices := buildConfigChoices(hasResumable, savedArgsClean, *agentArgs, cfg.SessionID)

	labels := make([]string, len(choices))
	for i, c := range choices {
		labels[i] = c.Label
	}
	header := fmt.Sprintf("saved config for tag '%s' (%s)", chosenTag, agent)
	sel := rt.PickFromList(header, labels, len(choices)*3+4)
	if sel == "" {
		fmt.Fprintf(stderr, "pair: aborted.\n")
		return 1, false
	}
	action := selectAction(choices, sel)
	if action == "new" {
		rt.Remove(configPath) // clean overwrite — the watcher writes a fresh one.
	}
	*agentArgs = composeTagRestartArgs(action, agent, savedArgsClean, *agentArgs, cfg.SessionID)
	return 0, true
}

// resolveConfigPath returns config-<tag>-<agent>.json, migrating a legacy
// config-<tag>-codex-codex.json to the canonical name first when applicable
// (#67 — the pure decision is ShouldMigrateLegacyCodex).
func resolveConfigPath(rt Runtime, dataDir, tag, agent string) string {
	canonical := CanonicalConfigPath(dataDir, tag, agent)
	if agent != "codex" {
		return canonical
	}
	_, canonicalExists := rt.FileSize(canonical)
	legacy := LegacyCodexConfigPath(dataDir, tag)
	_, legacyExists := rt.FileSize(legacy)
	legacyAgent := ""
	if legacyExists {
		if raw, err := rt.ReadFile(legacy); err == nil {
			if cfg, err := parseConfig(raw); err == nil {
				legacyAgent = cfg.Agent
			}
		}
	}
	if ShouldMigrateLegacyCodex(canonicalExists, agent, legacyExists, legacyAgent) {
		if raw, err := rt.ReadFile(legacy); err == nil {
			if rt.WriteAtomic(canonical, raw) == nil {
				rt.Remove(legacy)
			}
		}
	}
	return canonical
}

// writeConfig persists {agent, args, session_id}; a serialization failure leaves
// the prior config untouched (best-effort, mirroring the shell's mktemp||rm).
func writeConfig(rt Runtime, configPath, agent string, args []string, sid string) {
	if data, err := buildConfigJSON(agent, args, sid); err == nil {
		_ = rt.WriteAtomic(configPath, data)
	}
}

func normalizeEnv(env Env) Env {
	if env.DataDir == "" {
		env.DataDir = ResolveDataDir(env.Home, env.XDGData)
	}
	if env.HistoryD == 0 {
		env.HistoryD = 14
	}
	if env.Now.IsZero() {
		env.Now = time.Now()
	}
	return env
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
