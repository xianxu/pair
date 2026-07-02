---
id: 000091
status: open
deps: []
github_issue:
created: 2026-07-01
updated: 2026-07-01
estimate_hours:
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

- [ ] Sub-tickets exist for each remaining merge-safe phase (#92–#96), deps-chained.
- [ ] Each sub-ticket states that Pair must remain usable after its merge.
- [ ] The true native single binary is reached: `pair` provisions its runtime
      without extracting a shell tree, and only native `nvim/`/`zellij/` assets
      (if any) plus system platform tools remain.
- [ ] `atlas/go-migration-inventory.md` reflects the native-single-binary end
      state when the sequence completes.

## Plan

Tracking checklist — ticks as each sub-ticket closes:

- [x] Step 1 — embed + extract runtime tree — #90 (done)
- [x] Step 2 — route internal calls through the Go dispatcher — #92 (done)
- [x] Step 2b — route pair-wrap + pair-scribe PTY proxies — #96 (done)
- [ ] Step 3 — port stateful shell orchestrators to Go — #93 (in progress: M1 title poller, M2 scrollback/changelog openers, M3 review helpers, M4 clipboard helpers done; M5 launcher remains — likely its own ticket)
- [ ] Step 4 — stop extracting shell scripts — #94
- [ ] Step 5 — native nvim/zellij startup assets — #95

This is a tracking umbrella, not a coding issue: it holds no code milestones of
its own and stays `open` as a live tracker until the sequence completes. The
actual work + reviews happen in the sub-tickets.

## Log

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
