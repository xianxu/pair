package launcher

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RunLaunch is the native launcher's in-process driver (#99 M2 create + M3
// attach/restart/quit + M5b compaction/pick/rename/continue): a thin loop over the
// pure deciders (DecideLaunch + createlogic.go + M1's agentargs) that maps each
// shell orchestration step to a Runtime effect. Each iteration runs one create OR
// attach handoff (blocking fork+wait, so control returns here), runs the Alt+x
// quit cleanup, then — if a restart marker is present — re-decides under the
// marker's tag/agent (applying any rename_to move + continue re-seed), replacing
// the shell's `exec $0`. As of M5c the shell is retired — every launch path is
// handled here (compact / reject-in-pane / pick / rename / continue all native);
// user-facing messages are on the writer, the int is the exit code, the returned
// error is always nil.
func RunLaunch(opts LaunchOptions, rt Runtime, stderr io.Writer) (int, error) {
	env := normalizeEnv(opts.Env)

	// Prepend the resolved asset root's bin/ to PATH so zellij (and its panes:
	// pair-wrap, copy_command "copy-on-select", Run "pair-help"/openers, and the
	// nvim viewers) resolve the bundled helpers by bare name. The retired shell
	// bin/pair did this; the Go launcher that replaced it (#99 M5c) dropped it, so
	// a copied/Homebrew install (bin/ not on the user's PATH) couldn't launch (#95).
	// RunLaunch is the sole zellij-spawning path (create/attach/resurrect/restart-
	// loop), so once here covers them all.
	exeDir := ""
	if exe, err := os.Executable(); err == nil {
		exeDir = filepath.Dir(exe)
	}
	rt.SetEnv("PATH", prependBinToPath(opts.PairHome, exeDir, os.Getenv("PATH")))

	// #55 in-session compaction (M5b): `pair continue <slug>` from inside the
	// matching pane parks the scrollback (copy), drops a restart marker carrying
	// the slug, and kills the session — the outer loop below then re-launches
	// fresh, seeded from the slug. First entry only: a restart re-launch is the
	// same outer process, never in a pane.
	if opts.ContinueSlug != "" &&
		compactionDecision(opts.ForceInSession, rt.InZellijPane() || opts.FakeInZellij, opts.PairTag, opts.ZellijSession) {
		return runCompaction(opts, rt, stderr)
	}
	// Otherwise a launch from inside a pane can't proceed (a nested --session
	// would break; the create path's prompt would block) — shell 1064-1067.
	if rt.InZellijPane() {
		fmt.Fprintf(stderr, "pair: already running inside a zellij session.\n")
		fmt.Fprintf(stderr, "      detach first (Alt+d) or run pair from a fresh terminal.\n")
		return 1, nil
	}

	// Startup nvim hygiene (shell 1243): reap embeds whose Pair session is
	// gone (an external kill / reboot leaves no quit marker). Once, up front — a
	// clean restart below leaves nothing new to sweep.
	if sessions, err := rt.Sessions(); err == nil {
		rt.SweepOrphanNvim(liveTagsForSweep(sessions))
	}

	for {
		step, err := runOnce(opts, env, rt, stderr)
		if err != nil {
			return step.code, err // defensive: runOnce messages + returns nil now
		}
		if !step.handedOff {
			return step.code, nil // aborted or errored before the blocking handoff
		}
		runCleanup(env, rt, step, opts.ParkPromptTimeout, stderr)

		m, ok := rt.TakeRestartMarker(step.session)
		if !ok {
			return step.code, nil // no restart pending — done.
		}
		rTag := firstNonEmpty(m.Tag, step.tag)
		rAgent := firstNonEmpty(m.Agent, step.agent)

		// rename_to re-entry (M5b, shell 743-750): move the tag-scoped sidecars
		// old→new FIRST — the session was just killed, so the live-old gate passes
		// — then the config read + relaunch below run under the new tag. A failure
		// keeps the old tag (don't strand the user).
		if m.RenameTo != "" {
			if runRename(rt, LaunchArgs{RenameOld: rTag, RenameNew: m.RenameTo}, env.DataDir, io.Discard, stderr) == 0 {
				rTag = m.RenameTo
			} else {
				fmt.Fprintf(stderr, "pair: rename to '%s' failed; continuing under '%s'.\n", m.RenameTo, rTag)
			}
		}

		configPath := resolveConfigPath(rt, env.DataDir, rTag, rAgent)
		plan := planRestart(m, rTag, rAgent, readSavedConfig(rt, configPath))
		if plan.DropConfig {
			rt.Remove(configPath) // Shift+Alt+N / compaction: drop the config so create mints fresh.
		}
		opts.Args = plan.Args
		opts.SkipConfigPicker = true
		// A #55 compaction re-entry re-seeds the draft from the continuation slug
		// (M5b); every other restart re-entry leaves the draft as-is. The re-entry
		// is the outer process (never in a pane), so ContinueSlug stays cleared —
		// it can't re-trigger compaction, only the ContinueDoc draft re-seed.
		opts.ContinueDoc = ""
		opts.ContinueSlug = ""
		if plan.ContinueSlug != "" {
			if docPath, _, ok := rt.ResolveContinuationDoc(plan.ContinueSlug); ok {
				opts.ContinueDoc = docPath
			}
		}
	}
}

// launchStep is one iteration's outcome: the exit code, the session handed off
// (so cleanup + the restart marker key off it), the resolved tag/agent (the
// restart plan's current-run defaults), and whether the blocking handoff ran
// (only then do cleanup + restart apply).
type launchStep struct {
	code      int
	session   string
	tag       string
	agent     string
	handedOff bool
}

// runOnce runs one decision → create-or-attach handoff. Every outcome is handled
// natively (#99 M5c — no shell to fall back to): a handled abort/error returns
// handedOff=false with the exit code already messaged on stderr.
func runOnce(opts LaunchOptions, env Env, rt Runtime, stderr io.Writer) (launchStep, error) {
	agent := opts.Args.Agent
	// The agent the forwarded `-- <args>` were typed against. If a pick below
	// resolves a DIFFERENT agent (resume-by-name inference), those args are
	// agent-specific and invalid for it — the guard after inference drops them.
	requestedAgent := agent
	scopeRoot := envScopeRoot(env)
	base := DefaultTag(scopeRoot)
	cutoff := env.Now.Add(-time.Duration(env.HistoryD) * 24 * time.Hour)
	sessions, err := rt.Sessions()
	if err != nil {
		fmt.Fprintf(stderr, "pair: failed to query zellij sessions: %v\n", err)
		return launchStep{code: 1}, nil
	}
	historical, err := rt.ScanHistory(base, cutoff)
	if err != nil {
		fmt.Fprintf(stderr, "pair: failed to scan session history: %v\n", err)
		return launchStep{code: 1}, nil
	}
	scopedSessions, sessionNames, sessionNameEntries, ok := assignLaunchSessionNames(rt, sessions, scopeRoot, opts.GlobalDataDir, opts.Args, base, stderr)
	if !ok {
		return launchStep{code: 1}, nil
	}
	snap := SessionSnapshot{BaseTag: base, Sessions: scopedSessions, Historical: historical, SessionNames: sessionNames}
	decision, err := DecideLaunch(opts.Args, snap)
	if err != nil {
		fmt.Fprintf(stderr, "pair: %v\n", err)
		return launchStep{code: 1}, nil
	}

	// Native fzf session pick (#99 M5a): resolve the pick to a concrete
	// attach/create decision. A pick over an existing tag is resume-by-name, so
	// its agent must be inferred from disk (below), not the bare-`pair` claude
	// default — only the "+ new" pick (PromptName) keeps the default agent.
	if decision.Action == ActionPick {
		d, aborted := resolvePick(rt, snap, base, env.Now.Unix())
		if aborted {
			return launchStep{code: 0}, nil // fzf ESC / empty pick → exit 0 (shell 1478/1489)
		}
		decision = d
		if d.Action == ActionAttach || !d.PromptName {
			agent = "" // existing-tag pick → infer the paired agent below
		}
	}

	// `pair resume <tag>` / an existing-tag pick leaves the agent unset: infer
	// what the tag was last paired with from disk, defaulting to claude — so a
	// single tag fully restarts regardless of the original agent (shell 993-1007).
	if agent == "" {
		agent = rt.InferAgent(decision.Tag)
		if agent == "" {
			agent = "claude"
		}
	}

	// Agent-scoped CLI-args guard (#107): forwarded `-- <args>` belong to the
	// agent the user named (requestedAgent). When a pick resolves a different
	// agent — e.g. `pair codex -- --sandbox …` picks an existing claude tag, so
	// the agent re-infers to claude — those args (codex's `--sandbox`) would ride
	// onto and crash the wrong agent. Drop them; the resumed tag uses its own
	// saved config. requestedAgent=="" is `pair resume <tag>` (no args typed).
	if requestedAgent != "" && agent != requestedAgent {
		opts.Args.AgentArgs = nil
	}

	switch decision.Action {
	case ActionAttach:
		code, err := runAttach(opts, env, rt, decision.Tag, decision.SessionName, agent)
		if err != nil {
			fmt.Fprintf(stderr, "pair: failed to attach session '%s': %v\n", decision.SessionName, err)
			return launchStep{code: 1}, nil
		}
		return launchStep{code: code, session: decision.SessionName, tag: decision.Tag, agent: agent, handedOff: true}, nil
	case ActionCreate:
		return runCreate(opts, env, rt, sessions, decision, base, agent, sessionNameEntries[decision.Tag], stderr)
	default: // ActionPick is resolved above — unreachable; a defensive guard.
		fmt.Fprintf(stderr, "pair: internal error: unresolved launch decision (%s)\n", decision.Action)
		return launchStep{code: 1}, nil
	}
}

func assignLaunchSessionNames(rt Runtime, live []Session, repoRoot, globalDataDir string, args LaunchArgs, base string, stderr io.Writer) ([]Session, map[string]string, map[string]SessionNameEntry, bool) {
	scope, err := ResolveRepoScope(repoRoot)
	if err != nil {
		return live, nil, nil, true
	}
	index, err := rt.ReadSessionNameIndex()
	if err != nil {
		index = SessionNameIndex{}
	}
	scopedLive := SessionsForScope(live, index, scope)
	scopedLive = append(scopedLive, legacyLiveSessionsForScope(rt, live, index, scope, globalDataDir)...)
	tags := launchNameTags(args, base)
	if len(tags) == 0 {
		return scopedLive, nil, nil, true
	}
	names := map[string]string{}
	newEntries := map[string]SessionNameEntry{}
	for _, tag := range tags {
		if tag == "" {
			continue
		}
		name, updated, err := AssignSessionName(index, live, scope, tag, func(session string) bool {
			return rt.ProbeSessionName(session) == nil
		})
		if err != nil {
			fmt.Fprintf(stderr, "pair: %v\n", err)
			return nil, nil, nil, false
		}
		if len(updated.Entries) > len(index.Entries) {
			entry := updated.Entries[len(updated.Entries)-1]
			newEntries[tag] = entry
		}
		index = updated
		names[tag] = name
	}
	return scopedLive, names, newEntries, true
}

func launchNameTags(args LaunchArgs, base string) []string {
	switch {
	case args.SelectedTag != "":
		return []string{args.SelectedTag}
	case args.ForcedTag != "":
		return []string{args.ForcedTag}
	default:
		if base == "" {
			base = "pair"
		}
		return []string{base}
	}
}

// runCreate ports the shell's create branch: prompt/validate the tag, run the
// tag-restart config picker, compose the per-agent launch args, spawn the
// sidecars, then hand off to the blocking zellij create.
func runCreate(opts LaunchOptions, env Env, rt Runtime, live []Session, decision LaunchDecision, base, agent string, sessionEntry SessionNameEntry, stderr io.Writer) (launchStep, error) {
	// Validate the agent here (create-only; attach re-uses an existing pane's
	// agent, so shell 1728 defers this past the attach branch).
	if !rt.CommandExists(agent) {
		fmt.Fprintf(stderr, "pair: agent '%s' not found on PATH.\n", agent)
		fmt.Fprintf(stderr, "      install it first, then re-run.\n")
		return launchStep{code: 1}, nil
	}

	chosenTag := decision.Tag
	if decision.PromptName {
		tag, code, ok := promptForTag(rt, decision.Tag, base, stderr)
		if !ok {
			return launchStep{code: code}, nil
		}
		chosenTag = tag
	}

	session := decision.SessionName
	if session == "" || decision.PromptName || chosenTag != decision.Tag {
		name, entry, ok := assignSingleSessionName(rt, live, envScopeRoot(env), chosenTag, stderr)
		if !ok {
			return launchStep{code: 1}, nil
		}
		session = name
		sessionEntry = entry
	}
	// A too-long tag makes zellij reject the session name (#54); probe its own
	// validator and translate the rejection before writing sidecars or spawning
	// helpers.
	if err := rt.ProbeSessionName(session); err != nil {
		fmt.Fprintf(stderr, "pair: tag '%s' makes zellij's session name too long for this\n", chosenTag)
		fmt.Fprintf(stderr, "      machine's socket path (%s). Pick a shorter tag.\n", session)
		return launchStep{code: 1}, nil
	}
	// Free the name (clear a stale EXITED resurrect record) and guard against a
	// live session unexpectedly occupying it before any source-of-truth writes.
	if rt.SessionBlocksReuse(session) {
		fmt.Fprintf(stderr, "pair: session '%s' already exists.\n", session)
		return launchStep{code: 1}, nil
	}
	dataDir := env.DataDir
	legacyImported := false
	if decision.LegacyImport {
		legacyImported = importLegacyFlatTag(rt, chosenTag, opts.GlobalDataDir, dataDir)
	}
	configPath := resolveConfigPath(rt, dataDir, chosenTag, agent)
	savedForPicker := readSavedConfigForTag(rt, configPath, chosenTag, agent)

	agentArgs := append([]string(nil), opts.Args.AgentArgs...)

	// Tag-restart config picker (#000016): a saved config for this (tag, agent)
	// offers to reuse its args / resume its session, unless an explicit resume
	// token on argv already made the choice.
	if !opts.SkipConfigPicker {
		if code, ok := runConfigPicker(rt, configPath, savedForPicker, agent, chosenTag, &agentArgs, env.Cwd, stderr); !ok {
			return launchStep{code: code}, nil
		}
	}

	// Pre-capture the session id an explicit --resume/--conversation/`resume`
	// pinned: the watcher only catches NEW jsonl files, so an explicit resume
	// needs the config written synchronously (shell 2053-2110).
	explicitResume := extractExplicitResume(agent, agentArgs)

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
		}
	}

	// Codex: force inline mode so the conversation flows through zellij's
	// scrollback (idempotent; opt-out via PAIR_CODEX_ALT_SCREEN=1).
	if agent == "codex" {
		agentArgs = codexAltScreenArgs(agentArgs, opts.CodexAltScreenOptOut)
	}

	sessionID := firstNonEmpty(explicitResume, newSid)
	persistedArgs := persistedConfigArgs(agentArgs)
	repoRoot := envScopeRoot(env)
	repoName := DefaultTag(repoRoot)
	if sessionEntry.SessionName != "" {
		if err := rt.AppendSessionNameIndex(sessionEntry); err != nil {
			fmt.Fprintf(stderr, "pair: failed to append session-name index for '%s': %v\n", sessionEntry.SessionName, err)
			return launchStep{code: 1}, nil
		}
	}
	if err := rt.AppendLedger(chosenTag, LedgerEntry{
		Agent:        agent,
		Args:         persistedArgs,
		SessionID:    sessionID,
		Started:      env.Now,
		LastActive:   env.Now,
		RepoRoot:     repoRoot,
		RepoName:     repoName,
		LegacyImport: legacyImported,
	}); err != nil {
		fmt.Fprintf(stderr, "pair: failed to append ledger for tag '%s': %v\n", chosenTag, err)
		return launchStep{code: 1}, nil
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
	if sessionID != "" {
		writeConfig(rt, configPath, agent, persistedArgs, sessionID)
	}

	rt.SetEnv("PAIR_AGENT_ARGS", strings.Join(agentArgs, " "))
	rt.SetEnv("PAIR_SESSION_ID", sessionID)
	rt.SetEnv("PAIR_PANE_TITLE", PaneTitle(agent, env.Cwd, env.Home))
	rt.SetEnv("PAIR_PANE_CWD", TildeAbbrev(env.Cwd, env.Home))

	// Truncate the adaptation flight recorder once, before any appender starts.
	_ = rt.WriteAtomic(filepath.Join(dataDir, "adapt-"+chosenTag+".jsonl"), "")

	// Spawn the (already-Go) sidecars + set the frame title. agentArgs is the
	// final resolved vector (post mint / codex / resume compose).
	rt.SpawnSessionWatcher(agent, chosenTag, env.Cwd, repoRoot, repoName, agentArgs)
	rt.SetTerminalTitle(session)
	rt.RecordOuterTTY(chosenTag)
	rt.CmuxRename(chosenTag, session)
	rt.SpawnTitlePoller(chosenTag, agent)
	rt.DevRebuild(opts.PairHome)

	configDir := filepath.Join(opts.PairHome, "zellij")
	layout := filepath.Join(opts.PairHome, "zellij", "layouts", "main.kdl")
	code, err := rt.LaunchSession(session, configDir, layout)
	if err != nil {
		fmt.Fprintf(stderr, "pair: failed to launch zellij session '%s': %v\n", session, err)
		return launchStep{code: 1}, nil
	}
	return launchStep{code: code, session: session, tag: chosenTag, agent: agent, handedOff: true}, nil
}

func assignSingleSessionName(rt Runtime, live []Session, cwd, tag string, stderr io.Writer) (string, SessionNameEntry, bool) {
	scope, err := ResolveRepoScope(cwd)
	if err != nil {
		return sessionName(tag), SessionNameEntry{}, true
	}
	index, err := rt.ReadSessionNameIndex()
	if err != nil {
		index = SessionNameIndex{}
	}
	name, updated, err := AssignSessionName(index, live, scope, tag, func(session string) bool {
		return rt.ProbeSessionName(session) == nil
	})
	if err != nil {
		fmt.Fprintf(stderr, "pair: %v\n", err)
		return "", SessionNameEntry{}, false
	}
	if len(updated.Entries) > len(index.Entries) {
		return name, updated.Entries[len(updated.Entries)-1], true
	}
	return name, SessionNameEntry{}, true
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
	return tag, 0, true
}

// runConfigPicker drives the tag-restart config picker, mutating *agentArgs to
// the resolved launch vector. ok=false means abort with the returned exit code.
// When no saved config applies (absent, or an explicit resume already chose),
// it is a no-op that returns ok=true.
func runConfigPicker(rt Runtime, configPath string, saved savedConfig, agent, chosenTag string, agentArgs *[]string, cwd string, stderr io.Writer) (code int, ok bool) {
	if extractExplicitResume(agent, *agentArgs) != "" {
		return 0, true // argv already pinned a resume — nothing to offer.
	}
	if saved.Agent == "" {
		return 0, true // unusable config — proceed as if none.
	}

	savedArgsClean := persistedConfigArgs(saved.Args)
	hasResumable := rt.AgentSessionExists(agent, saved.SessionID, cwd)
	choices := buildConfigChoices(hasResumable, savedArgsClean, *agentArgs, saved.SessionID)

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
	*agentArgs = composeTagRestartArgs(action, agent, savedArgsClean, *agentArgs, saved.SessionID)
	return 0, true
}

func readSavedConfigForTag(rt Runtime, configPath, tag, agent string) savedConfig {
	if raw, err := rt.ReadFile(configPath); err == nil {
		if cfg, err := parseConfig(raw); err == nil {
			return cfg
		}
	}
	entries, err := rt.ReadLedger(tag)
	if err != nil {
		return savedConfig{}
	}
	if latest, ok := LatestLedgerEntryForAgent(entries, agent); ok {
		return savedConfig{Agent: latest.Agent, Args: latest.Args, SessionID: latest.SessionID}
	}
	return savedConfig{}
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
	if env.RepoRoot == "" {
		env.RepoRoot = env.Cwd
	}
	if env.HistoryD == 0 {
		env.HistoryD = 14
	}
	if env.Now.IsZero() {
		env.Now = time.Now()
	}
	return env
}

func envScopeRoot(env Env) string {
	if env.RepoRoot != "" {
		return env.RepoRoot
	}
	return env.Cwd
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
