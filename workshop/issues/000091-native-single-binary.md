---
id: 000091
status: codecomplete
deps: []
github_issue:
created: 2026-07-01
updated: 2026-07-04
estimate_hours:
actual_hours: N/A
---

# native single binary pair

## Problem

After #79 (`pair` is Go-owned) and #90 (a copied `pair` binary embeds and
extracts the Pair-owned runtime), Pair can be deployed as one artifact — but the
artifact is not yet a *true* native single binary. At runtime it still extracts a
mixed shell/Go/Lua/zellij tree to `$PAIR_DATA_DIR/runtime/<digest>/pair-home`
and execs `bin/pair-shell`; the shell lifecycle, the Lua/zellij assets, and the
several legacy helper binaries are all still live.

#90's Spec (lines 49–65) documents the remaining execution path toward the true
native single binary, and `atlas/go-migration-inventory.md` classifies every
remaining shell/asset surface with a migration priority. But that path is
**un-ticketed**: the prior tracking roadmap #72 deliberately closed at #79, and
#90 was created ad hoc afterward without spawning successors. This issue is the
umbrella tracker that carries the remaining phases so they don't stay buried in
#90's Spec and the atlas.

## Spec

**Target architecture.** One Pair executable that owns CLI dispatch, session
lifecycle, and runtime asset provisioning without extracting a shell tree to
disk. External *platform* tools stay external and are not bundled: `zellij`,
`nvim`, `fzf`, `jq`, clipboard tools, and agent CLIs come from the system
(unchanged from #90).

**Merge-safe staging invariant** (inherited from #72): after each sub-ticket
lands, the public `pair` command, `pair-dev`, keybindings, scrollback,
changelog, continuation, restart/quit, and review flows must all still work.
Pair is usable at every intermediate state; no single big-bang rewrite.

**Sub-ticket sequence** (mirrors #90's 5-step execution path; each step is one
sub-ticket, deps-chained so the order is enforced):

1. **Embed + extract the runtime tree — #90 (done).** Single Pair artifact via
   embed/extract while retaining the tested shell/nvim/zellij contracts.
2. **Route internal calls through the Go dispatcher — #92 (+ #96).** `pair slug`,
   `pair changelog`, `pair continuation`, `pair session-watch` resolve through the
   Go dispatcher (`context`/`scrollback-render` already done in #76); legacy
   binary names survive only as thin shims. The interactive PTY proxies
   `pair-wrap`/`pair-scribe` are carved out to **#96** (deps #92, reuses its
   streaming dispatch seam).
3. **Port stateful shell orchestrators to Go — #93.** Launcher/session
   lifecycle, scrollback/changelog openers, title poller, review helpers,
   clipboard helpers — ported one at a time behind merge-safe shims.
4. **Stop extracting shell scripts — #94.** Once shell ownership is gone, drop
   shell scripts from the runtime bundle; the bundle carries only native assets
   (`nvim/`, `zellij/`).
5. **Native nvim/zellij startup assets — #95.** Revisit whether `nvim/` and
   `zellij/` remain extracted native assets or move to generated temp
   files / API-driven startup. Endpoint: one Pair executable, platform tools
   still supplied by the system.

Architecture principles (from #72/#90, single-sourced via `sdlc
arch-principles`):

- `ARCH-PURPOSE` — each sub-ticket is valid only if it moves the repo toward the
  single-native-binary target while preserving current behavior; not a token
  port for its own sake.
- `ARCH-DRY` — reuse existing Go implementations behind dispatch rather than
  copying logic into parallel binaries; the runtime manifest stays the single
  packaging source.
- `ARCH-PURE` — extract pure decision logic from IO-heavy shell behavior; keep
  subprocess/zellij/filesystem interaction in thin, process-tested seams.

## Done when

- [x] Sub-tickets exist for each remaining merge-safe phase (#92–#96), deps-chained.
- [x] Each sub-ticket states that Pair must remain usable after its merge.
- [x] The true native single binary is reached: `pair` provisions its runtime
      without extracting a shell tree, and only native `nvim/`/`zellij/` assets
      (if any) plus system platform tools remain. (Reached **as documented**: no
      shell orchestrator is extracted (#93/#99/#94); the residual `nvim/`+`zellij/`
      config tree is a content-addressed *cache*, physically unavoidable with
      external nvim/zellij — #95 records this as the accepted final gap.)
- [x] `atlas/go-migration-inventory.md` reflects the native-single-binary end
      state when the sequence completes.

## Plan

Tracking checklist — ticks as each sub-ticket closes:

- [x] Step 1 — embed + extract runtime tree — #90 (done)
- [x] Step 2 — route internal calls through the Go dispatcher — #92 (done)
- [x] Step 2b — route pair-wrap + pair-scribe PTY proxies — #96 (done)
- [x] Step 3 — port stateful shell orchestrators to Go — **#93 done** (leaf ports
      M1–M4) + the launcher **#99 done** (extracted from #93 M5 — `bin/pair-shell`
      removed outright, `cmd/internal/launcher` owns create/attach/restart/quit/
      pick/list/continue/rename/compaction end-to-end; est 17.7h / actual 11.57h;
      PRs #61/#62/#63). Both merged + archived; no shell orchestrator remains.
- [x] Step 4 — stop extracting shell scripts — **#94 done** (all seven orchestrator
      shell shims removed from the runtime bundle: M1 ported `pair-restart.sh`/
      `pair-quit.sh` → in-process `pair restart`/`pair quit`; M2 repointed the five
      exec-shim callers to their Go binaries then deleted them. est 4.4h / actual
      2.5h; PR #64. Endpoint: **shell-reduced** — six non-orchestrator utilities
      (`lib/*.sh`, `pair-help`, `pair-notify`, `doctor/*.sh`) remain, out of scope).
- [x] Step 5 — native nvim/zellij startup assets — **#95 done** (decision: keep the
      digest extraction, reframed as a content-addressed runtime cache; true zero-tree
      is unreachable with external nvim/zellij — documented residual gap. Also
      restored the #99-M5c PATH-prepend regression so a copied/Homebrew `pair`
      resolves zellij's bare-name helpers. est 2.4h / actual N/A; PR #65).

This is a tracking umbrella, not a coding issue: it holds no code milestones of
its own and stays `open` as a live tracker until the sequence completes. The
actual work + reviews happen in the sub-tickets. **Sequence complete (2026-07-04).**

## Log

### 2026-07-04 — roadmap complete
- 2026-07-04: closed — Native-single-binary roadmap umbrella: all 5 steps landed + merged (#90 embed/extract; #92/#96 dispatcher+PTY; #93/#99 shell orchestrators+launcher→Go; #94 orchestrator shims dropped from bundle; #95 nvim/zellij provisioning decided + PATH regression fixed). Endpoint reached as documented: pair is one Go executable, no shell tree extracted; the residual nvim/zellij config is a content-addressed cache (zero-tree physically unreachable with external nvim/zellij — #95 accepted gap). --no-actual: tracking umbrella, no code of its own (work measured in sub-tickets). --no-judge: no new code in this close; every sub-ticket got its own boundary review. atlas end-state delivered by #95.; review verdict: not-run
- **All 5 steps landed.** #90 (embed+extract) → #92/#96 (dispatcher + PTY proxies)
  → #93/#99 (shell orchestrators + launcher ported to Go) → #94 (orchestrator shims
  dropped from the bundle) → #95 (nvim/zellij provisioning decided + PATH regression
  fixed). **Endpoint reached as documented:** `pair` is one Go executable that
  provisions its runtime with no shell tree; the only extracted assets are the
  `nvim/`+`zellij/` config, kept as a content-addressed cache because external
  nvim/zellij read config from real filesystem paths (true zero-tree is physically
  unreachable — #95's accepted, documented residual gap). Platform tools
  (zellij/nvim/fzf/jq/agent CLIs) stay external. Closing this umbrella.

### 2026-07-03 (later)
- **Step 4 complete.** #94 merged (PR #64, done + archived) — all seven orchestrator
  shell shims are gone from the runtime bundle (2 ported in M1, 5 repointed-and-dropped
  in M2). est 4.4h / actual 2.5h. The deployed footprint no longer extracts any
  orchestrator shell; the bundle is **shell-reduced** (six non-orchestrator utilities
  remain — `bin/lib/*.sh`, `pair-help`, `pair-notify`, `doctor/*.sh` — never in #93's
  scope). **Only Step 5 (#95, native nvim/zellij startup assets) remains** to reach
  the true native single binary.

### 2026-07-03
- **Step 3 complete.** #99 (the launcher) landed done + merged (PRs #61/#62/#63;
  `bin/pair-shell` retired, `cmd/internal/launcher` owns the full flow), and #93
  (the shell-orchestrator umbrella) closed done as a rollup — actual 5.41h. No
  shell orchestrator remains; the runtime still *bundles* the leaf shims + the
  nvim/zellij assets, which Steps 4 (#94) and 5 (#95) retire. Next: Step 4 (#94).

### 2026-07-02
- Launcher (#93 M5, `bin/pair-shell`) extracted into its own ticket **#99** — P0,
  ~900 lines of new IO orchestration onto the `cmd/internal/launcher` core, phased
  M1–M5, ~17.7h. #93 stays open until #99 lands.

### 2026-07-01
- Steps 2 & 2b closed and merged (#92 dispatcher routing; #96 pair-wrap/pair-scribe
  PTY proxies). Step 3 (#93) in progress: M1–M4 landed (title poller → `cmd/pair-title`;
  scrollback/changelog openers → `cmd/internal/opener`; review helpers →
  `cmd/internal/reviewcmd`; clipboard helpers → `cmd/internal/clipcmd` +
  `cmd/internal/zellijpane`, all behind tracked shims). Only M5 (launcher /
  `bin/pair-shell`, ~2287 lines) remains — very likely its own ticket per the
  #93 granularity note.

Created as the umbrella tracker for the remaining native-single-binary work.
Rationale: #72 (the prior Go-migration roadmap) closed at #79, and #90's Spec
carried the remaining 5-step path forward without spawning successor issues — so
steps 2–5 were un-ticketed and discoverable only by re-reading #90 or the atlas.
This issue re-establishes the tracking umbrella. #90 is step 1 (done); #92–#95
are the remaining steps, deps-chained #90→#92→#93→#94→#95 to enforce the
merge-safe order.

Design of #92 carved the two interactive PTY proxies (`pair-wrap`, `pair-scribe`)
out to a new step-2b sub-ticket **#96** (deps #92): they are session entrypoints
needing the streaming dispatch route, not finite internal calls, so they belong
on their own review boundary. #92 keeps the internal-call helpers
(slug/changelog/continuation/session-watch) + call-site repointing.
