package launcher

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/pairlifecycle"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// The attach + quit-cleanup orchestrators behind RunLaunch's in-process restart
// loop (#99 M3, ported from bin/pair-shell's attach branch + cleanup_quit_marker).
// Both stay thin drivers over Runtime effects + the pure marker/config helpers;
// the loop itself (createflow.go) re-decides create-vs-attach each iteration.

// runAttach ports the shell's attach branch (1701-1723): re-attach to a live
// Pair session (`pair resume <tag>`). Like create it exports the tag +
// refreshes the outer-tty/title/cmux/poller, but it re-uses the existing pane's
// agent (no fresh spawn, no arg composition) and blocks on the attach handoff so
// the loop regains control for cleanup + restart. agent is the inferred title
// agent (the on-disk agent-<tag> record, resolved by the caller).
func runAttach(opts LaunchOptions, env Env, rt Runtime, tag, session, agent string) (int, error) {
	if session == "" {
		// Degraded fallback: callers pass the resolved name. Reached only when
		// attach was invoked without one (#130).
		session = legacySessionPrefix + tag
	}
	// Export what the spawned poller inherits (pair-shell exports these globally
	// before the branch; the attach branch itself only re-exports PAIR_TAG).
	rt.SetEnv("PAIR_HOME", opts.PairHome)
	rt.SetEnv("PAIR_DATA_DIR", env.DataDir)
	rt.SetEnv("PAIR_TAG", tag)
	rt.SetEnv("PAIR_SESSION_NAME", session)

	// zellij creates the draft on new-session but not on attach; ensure it.
	paths, err := artifactpath.ResolveScoped(env.DataDir, tag)
	if err != nil {
		return 1, err
	}
	_ = rt.Touch(paths.Draft())
	rt.SetTerminalTitle(session)
	rt.RecordOuterTTY(tag)
	rt.CmuxRename(tag, session)
	// agent is already the on-disk record: attach is reached via `pair resume
	// <tag>` (ParseArgs leaves Agent=="") or a live-session pick (runOnce clears
	// Agent) — either way runOnce sets it via InferAgent(tag), so the poller
	// matches the running pane's agent regardless of any bare-`pair` default.
	rt.SpawnTitlePoller(tag, agent, session)

	return rt.AttachSession(session, filepath.Join(opts.PairHome, "zellij"))
}

// runCleanup ports cleanup_quit_marker (shell 1520-1647): after a blocking
// handoff returns, if the Alt+x quit marker is present, tear the session down —
// delete the zellij record, reap nvim, offer to park the scrollback, remove the
// per-tag sidecars, print the resume hint, kill the title poller, and reset the
// cmux workspace. A detach (Alt+d) leaves no marker, so this is a no-op then.
// Runs after BOTH create and attach handoffs (either can leave a quit marker).
const cleanupTimeout = 10 * time.Second

func runCleanup(env Env, rt Runtime, step launchStep, scopeKey string, parkTimeout int, out io.Writer) {
	ctx, cancel := context.WithTimeout(context.Background(), cleanupTimeout)
	defer cancel()
	_, _ = runCleanupContext(ctx, env, rt, step, scopeKey, parkTimeout, out)
}

func runCleanupContext(ctx context.Context, env Env, rt Runtime, step launchStep, scopeKey string, parkTimeout int, out io.Writer) (pairlifecycle.CleanupResult, bool) {
	intent, present, err := takeQuitIntent(rt, step.session)
	if err != nil {
		return cleanupSetupFailure(err), true
	}
	if !present {
		return pairlifecycle.CleanupResult{}, false
	}
	dataDir := env.DataDir
	paths, err := artifactpath.ResolveScoped(dataDir, step.tag)
	if err != nil {
		return cleanupSetupFailure(err), true
	}
	// Resolve the agent this tag was paired with BEFORE the agent-<tag> record is
	// removed below, so the park path + resume hint name the right binary.
	quitAgent := rt.InferAgent(step.tag)
	if quitAgent == "" {
		quitAgent = step.agent
	}
	scrollback, err := paths.ScrollbackArtifacts(quitAgent)
	if err != nil {
		return cleanupSetupFailure(err), true
	}
	panePath, err := paths.PaneChecked(quitAgent)
	if err != nil {
		return cleanupSetupFailure(err), true
	}
	ops := &launcherCleanupOps{
		rt: rt, env: env, step: step, scopeKey: scopeKey, parkTimeout: parkTimeout,
		out: out, quitAgent: quitAgent, now: time.Now, scrollback: scrollback,
		panePath: panePath,
		editorPaths: lifecycleEditorPaths{
			draft: paths.Draft(), scrollbackPrefix: paths.ScrollbackPrefix(),
			pids: []string{paths.NvimPID("draft"), paths.NvimPID("scrollback")},
		},
		outerTTY: paths.OuterTTY(), agentPath: paths.Agent(), agentOutput: paths.AgentOutput(),
		pairWrapPID: paths.PairWrapPID(), adaptLog: paths.AdaptLog(), imageCapture: paths.ImageCapture(),
		imageCaptureDone: paths.ImageCaptureDone(), titlePID: paths.TitlePID(),
	}
	if intent.Kind == QuitIntentDirect {
		return pairlifecycle.RunCleanup(ctx, pairlifecycle.CleanupDirect, ops), true
	}
	ref := intent.Request
	if ref == nil || ref.Tag != step.tag || ref.RepoScope != scopeKey {
		return cleanupSetupFailure(errors.New("Couch quit reference does not match active Pair thread")), true
	}
	address, err := artifactpath.Resolve(artifactpath.Address{DataDir: ref.DataDir, RepoScope: ref.RepoScope, Tag: ref.Tag})
	if err != nil {
		return cleanupSetupFailure(err), true
	}
	lifecyclePaths, err := address.Lifecycle(ref.Nonce)
	if err != nil {
		return cleanupSetupFailure(err), true
	}
	var cleanupResult pairlifecycle.CleanupResult
	cleanupResult, err = ConsumeCouchAttempt(ctx,
		pairlifecycle.Store{Runtime: pairlifecycle.OSRuntime{}}, lifecyclePaths, *ref, step.session, ops)
	if err != nil {
		return cleanupSetupFailure(err), true
	}
	return cleanupResult, true
}

// ConsumeCouchAttempt is Pair's production request-authority/cleanup/completion
// boundary. The outer launcher resolves local artifacts and builds the effect
// adapter; conformance drivers can invoke this same durable seam directly.
func ConsumeCouchAttempt(ctx context.Context, store pairlifecycle.Store, paths artifactpath.LifecyclePaths, ref QuitRequestReference, session string, ops pairlifecycle.QuitLifecycleOps) (pairlifecycle.CleanupResult, error) {
	var cleanupResult pairlifecycle.CleanupResult
	completion, err := store.ConsumeAttempt(ctx, paths, ref.Attempt, func(callbackContext context.Context, _ *pairlifecycle.LockedAttempt, request pairlifecycle.QuitRequest) pairlifecycle.CleanupResult {
		if request.Identity.Nonce != ref.Nonce || request.Identity.RepoScope != ref.RepoScope || request.Identity.Tag != ref.Tag || request.Attempt != ref.Attempt || request.Session != session {
			return cleanupSetupFailure(errors.New("committed Couch request does not match active Pair session"))
		}
		cleanupResult = pairlifecycle.RunCleanup(callbackContext, pairlifecycle.CleanupCouch, ops)
		return cleanupResult
	})
	if err != nil {
		return cleanupResult, err
	}
	if cleanupResult.CompletedAt.IsZero() {
		cleanupResult = pairlifecycle.CleanupResult{Outcome: completion.Outcome, CompletedAt: completion.CompletedAt}
		if completion.Outcome == pairlifecycle.CompletionFailure {
			cleanupResult.Failures = []pairlifecycle.StageFailure{{Stage: pairlifecycle.StageSessionQuiescence, Code: completion.FailureCode, Err: errors.New("previous cleanup attempt failed")}}
		}
	}
	return cleanupResult, nil
}

type typedQuitIntentRuntime interface {
	TakeQuitIntent(string) (QuitIntent, bool, error)
	WriteQuitIntent(string, QuitIntent) error
}

func takeQuitIntent(rt Runtime, session string) (QuitIntent, bool, error) {
	if typed, ok := rt.(typedQuitIntentRuntime); ok {
		return typed.TakeQuitIntent(session)
	}
	if rt.TakeQuitMarker(session) {
		return QuitIntent{Version: QuitIntentVersion, Kind: QuitIntentDirect}, true, nil
	}
	return QuitIntent{}, false, nil
}

func writeQuitIntent(rt Runtime, session string, intent QuitIntent) error {
	if typed, ok := rt.(typedQuitIntentRuntime); ok {
		return typed.WriteQuitIntent(session, intent)
	}
	if intent.Kind != QuitIntentDirect {
		return errors.New("runtime does not support Couch quit intent")
	}
	rt.TouchQuitMarker(session)
	return nil
}

type contextualCleanupRuntime interface {
	DeleteSessionContext(context.Context, string) error
	ReapNvimContext(context.Context, lifecycleEditorPaths) error
	KillTitlePollerContext(context.Context, string) error
	CleanupCmuxContext(context.Context, string, string) error
}

type lifecycleEditorPaths struct {
	draft            string
	scrollbackPrefix string
	pids             []string
}

type launcherCleanupOps struct {
	rt                                                      Runtime
	env                                                     Env
	step                                                    launchStep
	scopeKey                                                string
	parkTimeout                                             int
	out                                                     io.Writer
	quitAgent                                               string
	parked                                                  bool
	now                                                     func() time.Time
	scrollback                                              artifactpath.ScrollbackArtifactSet
	panePath                                                string
	editorPaths                                             lifecycleEditorPaths
	outerTTY, agentPath, agentOutput, pairWrapPID, adaptLog string
	imageCapture, imageCaptureDone, titlePID                string
}

func (o *launcherCleanupOps) QuiesceSession(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if runtime, ok := o.rt.(contextualCleanupRuntime); ok {
		return runtime.DeleteSessionContext(ctx, o.step.session)
	}
	return o.rt.DeleteSession(o.step.session)
}

func (o *launcherCleanupOps) ReapEditors(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if runtime, ok := o.rt.(contextualCleanupRuntime); ok {
		return runtime.ReapNvimContext(ctx, o.editorPaths)
	}
	o.rt.ReapNvim(o.step.tag)
	return nil
}

func (o *launcherCleanupOps) PreserveScrollback(ctx context.Context, intent pairlifecycle.CleanupIntent) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	size, exists := o.rt.FileSize(o.scrollback.Raw)
	if !exists || size <= 0 {
		return nil
	}
	preserve := intent == pairlifecycle.CleanupCouch
	if intent == pairlifecycle.CleanupDirect && o.rt.IsTTY() && !o.rt.RestartMarkerPresent(o.step.session) {
		preserve = o.rt.ConfirmParkNudge(o.step.session, o.parkTimeout)
	}
	if !preserve {
		return nil
	}
	base, ok := o.rt.ParkScrollback(o.step.tag, o.quitAgent, true)
	if !ok {
		return errors.New("preserve scrollback failed")
	}
	o.parked = true
	fmt.Fprintf(o.out, "pair: scrollback preserved at\n        %s.raw\n      open a session and \"park %s\" to distill it into a continuation.\n", base, o.step.session)
	return nil
}

func (o *launcherCleanupOps) CleanupSidecars(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// Remove the per-tag sidecars (shell 1583-1591). pane-<tag>-<quitAgent>.json
	// (written by the agent pane in both main-{2,3}.kdl layouts) was historically
	// omitted here — the leak behind #97: a surviving twin misled the frame
	// poller when the tag was later paired with a different agent. Cleaning it on
	// quit stops new twins at the source (the poller also filters defensively).
	for _, path := range []string{
		o.outerTTY, o.agentPath, o.agentOutput, o.pairWrapPID,
		o.adaptLog, o.imageCapture, o.imageCaptureDone, o.panePath,
	} {
		if path != "" {
			o.rt.Remove(path)
		}
	}
	o.rt.Remove(o.scrollback.ANSI)
	// Remove the raw capture only when it wasn't parked (preserved above).
	if !o.parked {
		o.rt.Remove(o.scrollback.Raw)
		o.rt.Remove(o.scrollback.Events)
	}

	// Resume hint: a saved config for this (tag, agent) means the resume path
	// will work next time — surface the repo-local tag, not the public zellij
	// session name.
	if _, err := o.rt.ReadFile(resolveConfigPath(o.rt, o.env.DataDir, o.step.tag, o.quitAgent)); err == nil {
		fmt.Fprintf(o.out, "pair: saved session config for tag \"%s\" (%s).\n", o.step.tag, o.quitAgent)
		fmt.Fprintf(o.out, "      resume with: pair resume %s\n", o.step.tag)
		if sid, status := o.rt.EstablishedSessionID(o.scopeKey, o.step.tag, o.quitAgent); status == sessioninventory.BindingEstablished && sid != "" {
			fmt.Fprintf(o.out, "      session id:  %s\n", sid)
		}
	}
	return nil
}

func (o *launcherCleanupOps) CleanupPoller(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if runtime, ok := o.rt.(contextualCleanupRuntime); ok {
		return runtime.KillTitlePollerContext(ctx, o.titlePID)
	}
	o.rt.KillTitlePoller(o.step.tag)
	return nil
}

func (o *launcherCleanupOps) CleanupCmux(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// Reset the cmux workspace title to the shell cwd — this pair is dead — but
	// only when we own it, then release ownership so a remaining/next pair can
	// claim (shell 1640-1646). On a restart the follow-up create immediately
	// re-renames, so the cwd flash is invisible.
	if o.rt.PairOwnsCmuxWorkspace(o.step.tag) {
		reset := filepath.Base(o.env.Cwd)
		if reset == "" || reset == "." || reset == string(filepath.Separator) {
			reset = "shell"
		}
		if runtime, ok := o.rt.(contextualCleanupRuntime); ok {
			return runtime.CleanupCmuxContext(ctx, o.step.tag, reset)
		}
		o.rt.CmuxRename(o.step.tag, reset)
		o.rt.ClearCmuxOwner()
	}
	return nil
}

func (o *launcherCleanupOps) Now() time.Time { return o.now() }

func cleanupSetupFailure(err error) pairlifecycle.CleanupResult {
	return pairlifecycle.CleanupResult{
		Outcome: pairlifecycle.CompletionFailure,
		Failures: []pairlifecycle.StageFailure{{
			Stage: pairlifecycle.StageSessionQuiescence, Code: pairlifecycle.FailureCleanupFailed, Err: err,
		}},
		CompletedAt: time.Now(),
	}
}

// readSavedConfig loads config-<tag>-<agent>.json for the restart plan; a
// missing/unusable file yields the zero savedConfig (no resume, no saved args).
func readSavedConfig(rt Runtime, configPath string) savedConfig {
	raw, err := rt.ReadFile(configPath)
	if err != nil {
		return savedConfig{}
	}
	cfg, _ := parseConfig(raw)
	return cfg
}

// liveTagsForSweep projects live public session names to repo-local tags for
// SweepOrphanNvim. Indexed scoped names only count for the current repo scope;
// legacy unindexed pair-<tag> names still count as their bare tag.
func liveTagsForSweep(sessions []Session, index SessionNameIndex, scopeKey string) []string {
	tags := make([]string, 0, len(sessions))
	for _, s := range sessions {
		if entry, ok := index.ownerOf(s.Name); ok {
			if scopeKey == "" || entry.ScopeKey == scopeKey {
				tags = append(tags, entry.Tag)
			}
			continue
		}
		// legacySessionPrefix only, for the same reason as legacy_live.go: this
		// strip is the legacy scheme's inverse, and the 📁 scheme has none. Every
		// 📁 session is indexed, so it is served by the ownerOf branch above.
		if strings.HasPrefix(s.Name, legacySessionPrefix) {
			tags = append(tags, strings.TrimPrefix(s.Name, legacySessionPrefix))
		}
	}
	return tags
}
