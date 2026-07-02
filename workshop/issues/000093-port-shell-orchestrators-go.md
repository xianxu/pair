---
id: 000093
status: working
deps: [000092]
github_issue:
created: 2026-07-01
updated: 2026-07-01
estimate_hours: 17.4
started: 2026-07-01T14:39:06-07:00
---

# port stateful shell orchestrators to Go

Tracking: #91 (native single binary) — step 3 of 5. Depends on #92.

## Problem

The runtime's stateful orchestration still lives in shell. `bin/pair-shell`
(the launcher) owns the zellij lifecycle, prompt UI, restart/quit cleanup, cmux
ownership, dev rebuild, continuation, rename, config/session migration, and the
title poller; `bin/pair-title.sh`, `bin/pair-scrollback-open`, the
`bin/pair-review-*` helpers, and the clipboard helpers
(`clipboard-to-pane.sh`, `copy-on-select.sh`, `flash-pane.sh`) are all shell
orchestrators. Until this logic moves into Go, the binary can never stop
extracting a shell tree (#94), so this is the load-bearing step toward a native
single binary. `atlas/go-migration-inventory.md` already flags each surface with
a migration priority (P0 launcher, P1 title/scrollback, P2 review helpers).

## Spec

Port the stateful shell orchestrators into Go **one at a time**, each behind a
merge-safe compatibility shim, following the #78 precedent (session watcher
ported to `cmd/pair-session-watch` with `bin/pair-session-watch.sh` retained as
a shim). Reuse the fakeable pure decision core from #75 (`cmd/internal/launcher`)
for the launcher work.

- Each surface is an independent, merge-safe milestone: land it, keep the old
  shell name working as a shim, verify the flow, then move to the next. Pair
  stays usable throughout.
- Keep native assets native: `nvim/*.lua` and `zellij/*.kdl` are NOT ported to
  Go (that boundary is owned by #95). This issue audits their shell-outs but
  leaves them as native assets.
- `ARCH-PURE`: extract pure decision logic (lifecycle state, poller cadence,
  target resolution) into unit-tested packages; keep zellij/nvim/filesystem/cmux
  interaction in thin, process-tested seams.
- `ARCH-DRY`: reuse the existing internal packages (`launcher`, `entrypoint`,
  `sessionwatch`, dispatcher runners) rather than reimplementing.

Ordering rationale: the launcher is the last and largest surface because it owns
the most state; the leaf orchestrators (title poller, scrollback/review/clipboard
openers) are ported first to shrink `bin/pair-shell`'s dependency set before it
is itself replaced.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module design=0.4 impl=1.6
item: smaller-go-module design=0.3 impl=1.2
item: smaller-go-module design=0.4 impl=1.6
item: smaller-go-module design=0.2 impl=0.9
item: smaller-go-module design=1.0 impl=6.0
item: milestone-review design=0.0 impl=1.5
item: atlas-docs design=0.2 impl=0.6
total: 17.4
```

Whole-issue estimate across M1–M5 (the five `smaller-go-module` items, in
milestone order; `milestone-review` covers the five boundary reviews;
design hours are weighted 1.6× per the model). The **M5 launcher** item
(design=1.0 impl=6.0 → ~7.6h) is the dominant uncertainty — `bin/pair-shell`
is ~2200 lines; per the Plan's granularity note it may split into its own
ticket, which would re-scope this estimate. M1–M4 (the leaf orchestrators,
~7.4h) are well-understood ports on the verified #78 sessionwatch template.
Durable plan:
`workshop/plans/000093-port-shell-orchestrators-plan.md` (M1 detailed; M2–M5
milestone-level, detailed as reached).

## Done when

- [ ] Each listed shell orchestrator has a Go owner; the shell name survives only
      as a thin re-exec shim (or is removed where no caller needs it).
- [ ] Pure lifecycle/poller/target decision logic is unit-tested; zellij/nvim/
      cmux/filesystem interaction is behind process-tested seams.
- [ ] `nvim/*.lua` / `zellij/*.kdl` remain native assets; their shell-outs are
      audited and repointed to Go owners where applicable.
- [ ] `pair`, `pair-dev`, keybindings, scrollback, changelog, continuation,
      restart/quit, rename, and review flows work after each milestone.

## Plan

Each `Mx` is a merge-safe review boundary closed on its own (`sdlc
milestone-close`); the surfaces are independent enough to port and verify one at
a time.

- [x] M1 — title poller: port `bin/pair-title.sh` to Go (the explicit #78
      next-candidate), keep the `.sh` name as a shim.
- [x] M2 — scrollback/changelog openers: port `bin/pair-scrollback-open` (and the
      changelog opener) to Go orchestration; `nvim/*.lua` viewers stay native.
- [x] M3 — review helpers: port `bin/pair-review-target` / `pair-review-open` /
      `pair-review-readiness` orchestration to Go.
- [x] M4 — clipboard helpers: port `clipboard-to-pane.sh`, `copy-on-select.sh`,
      `flash-pane.sh` to Go (or fold behind the dispatcher).
- [ ] M5 — launcher / session lifecycle: port `bin/pair-shell`'s orchestration to
      Go on the `cmd/internal/launcher` core, retaining a compatibility shim;
      zellij/nvim stay external.

## Log

### 2026-07-01
- 2026-07-01: closed M4 — Full make test green (exit 0) incl test-copy-on-select driving the Go copy-on-select binary (PASS: in_nvim keys on terminal_command, parley.nvim agent pane still hands off); clipcmd + zellijpane Go unit + fake-Runtime tests green; runtimebundle drift-check clean with the 3 Go binaries + .sh shims bundled; git ls-files bin/ lists only the 3 .sh shims (Go binaries gitignored); review verdict: FIX-THEN-SHIP

**M4 review follow-ups (FIX-THEN-SHIP → SHIP).** No Critical/Important — only 3
Minor + plan bookkeeping. Fixed: (1) the clipboard-debug log grew unbounded
(`O_APPEND`) where the source truncated per-run — now truncated once at the
copy-on-select pipeline head (`LogFresh`), appended thereafter, so it holds one
selection's chain AND keeps copy-on-select's lines (the source's mid-chain
truncate in clipboard-to-pane clobbered them); (3) trimmed the stale "nvim
flows/tests" consumer from `flash-pane`'s inventory row (only copy-on-select +
the shell test invoke it). Finding (2) — `opener.firstAgentPaneID` still
open-codes the walk `zellijpane.Parse` now owns — deferred as a tracked
follow-up (the `Title` field makes it a pure swap; not retrofitted to avoid
touching tested opener). Plan `## Revisions` records the `Exec`→`RunSubprocess`
/`ExecReplace` seam split, the log-truncation delta, and the opener follow-up.

**M4 change-code:** both judges INFO (branch created, no block). Estimate-quality
re-raised the standing M5-optimism + light-`milestone-review` advisories (same as
M2) — deferred to M5's own design pass/ticket per plan `## Revisions`; whole-issue
estimate stays 17.4. Plan-quality raised 3 non-blocking sharpenings, folded into
the M4 build: (1) add a `Title` field to `zellijpane.Pane` now so opener's later
adoption is a pure swap (its `isAgentPane` keys off title); (2) split the `Exec`
seam into two named modes — flash is call-and-return (subprocess), the
clipboard-to-pane hand-off is a process-replacing `exec` (syscall.Exec); (3) the
flash `set+reset` fake test asserts the reset is *scheduled* (detached), not run
synchronously, pinning the "don't block the caller" contract.

**M3 review follow-ups (FIX-THEN-SHIP → SHIP).** No Critical. Fixed both
Important: (1) `test-review` gained the 3 review-binary prereqs (they're built Go
binaries now, so a fresh-tree `make test` must build them first — same class as
M2's `test-changelog` fix); (2) added `--prepare` `track` + `resume` fake-Runtime
tests (`track` is the consequential path — add→commit→ls-files→status→checkout-b
+ mark-ready; `resume` asserts branch kept, no checkout). Minor: made the target
JSON write atomic (`WriteAtomic`, since nvim's Alt+c re-reads it) and strengthened
the JSON test to assert `scoped_file`/`file_matches`; updated `readiness.lua`'s
stale "gathering lives in bin/pair-review-readiness" comment to point at
`cmd/internal/reviewcmd`. Plan `## Revisions` records the seam deltas
(`codexsid` dropped the unused `home` param; path-resolution + `prepare()` are IO
seam, not pure) and the follow-up: the codex walk is triplicated
(`codexsid`+`slug`+`sessionwatch`) — `codexsid` is the canonical home;
slug/sessionwatch adopt it later.

- 2026-07-01: closed M3 — Three review orchestrators ported to cmd/pair-review-{target,open,readiness} on shared cmd/internal/reviewcmd: pure slugify/JSON/action-mapping unit-tested; fake-Runtime tests cover target session-priority (env/config/codex fallback), readiness JSON + --prepare (new branch-create/track-add-commit/resume/stop/interact + mark-ready + xx-fix ack), open single-pane replace+spawn; the existing pair-review-target / review-readiness-cli / review-window shell tests drive the Go binaries UNCHANGED (real git + real nvim --headless classify, incl the quoted-path/branch robustness case); the 4-case readiness decision stays single-source in nvim/review/readiness.lua; osfs + codexsid extracted (opener+titlepoller retrofitted to embed osfs.FS, their tests still green); Go binaries replace the shell scripts (no shim, 3 .gitignore negations dropped, git ls-files bin/ clean); full make test green (sandbox-off, real git/nvim); runtimebundle drift-check clean with the 3 review binaries bundled.; review verdict: FIX-THEN-SHIP
- 2026-07-01: closed M2 — Both openers ported to cmd/pair-scrollback-open + cmd/pair-changelog-open on a shared cmd/internal/opener package: pure viewport scorer (high-confidence/sub-threshold/top-clamp/empty) + session keying + distiller argv unit-tested; fake-Runtime loop tests cover render→viewport→viewer wiring, re-entrancy defer, distiller singleton no-double-spawn, config-keyed base; the existing changelog-open e2e + session-key shell tests now drive the Go binary UNCHANGED + a new scrollback-open smoke; the Go binaries replace the same-named shell scripts (zellij invokes by PATH, no shim; two .gitignore negations dropped, git ls-files bin/ confirms untracked); full make test green (sandbox-off, real setsid/nvim); runtimebundle drift-check clean with both openers bundled.; review verdict: FIX-THEN-SHIP
- 2026-07-01: closed M1 — bin/pair-title.sh ported to cmd/pair-title Go poller: titlepoller unit tests cover all 8 old shell-harness cases (identity guard incl 21-vs-211 collision; frame-meter count/no-count/cwd-fallback/unchanged-skip) + loop tests (single-instance defer, stale-pidfile reclaim, session-miss-threshold exit); shared procutil backs both OSRuntimes; context count reused in-process via contextcmd (no subprocess); bin/pair-title.sh is a thin re-exec shim preserving the argv the single-instance guard matches; full make test green (sandbox-off, real ps/kill); runtimebundle drift-check clean with bin/pair-title + shim bundled.; review verdict: FIX-THEN-SHIP

**M1 review follow-ups (FIX-THEN-SHIP → SHIP).** No Critical/Important-blocking
correctness bugs. Addressed the one Important finding — the loop body was
untested — by adding `TestRunRendersFrameAndCmuxTitles` (claim path renders
frame + cmux through `Run`), `TestRunDefersCmuxToLiveForeignOwner` (defer path),
and direct `updateWorkspaceTitle` reclaim/unchanged-bucket + `activityMTime`
tests. Recorded two deliberate refinements (not faithful copies): (1) the
activity-transcript resolution moved from the shell's `$PWD` to paneCwd (via the
shared `contextcmd.TranscriptPath`) — a no-op for the primary pane; (2) a brief
double-poller window is possible on the very first spawn after upgrading from a
still-running pre-port `bash pair-title.sh` process (the new `pair-title <tag> `
argv guard doesn't recognize the old `.sh` argv) — self-heals when that session
ends. Plan `## Revisions` records the dropped `Log`/adapt seam (faithful: the
shell poller never emitted adapt) and the `latest`→`activityMTime` naming.

**M2 change-code:** plan-quality CLEAN (INFO). Estimate-quality raised advisories
(M5's `smaller-go-module`/6.0h is optimistic for a 2287-line launcher;
`milestone-review` 1.5h is light given M1's fix-then-ship tail) — both are
observations, not fabrications (the judge confirmed the derivation "maps
item-for-item"). Left the whole-issue estimate at 17.4 (unchanged since M1, and
M5's uncertainty is already disclosed in the `## Estimate` block with an explicit
split-into-own-ticket escape); proceeded via `--no-judge` since plan-quality had
already passed this run. Re-visit the total if M5 stays one milestone.

**M2 review follow-ups (FIX-THEN-SHIP → SHIP).** No Critical. Fixed the one
Important — added `$(BIN_DIR)/pair` to `test-changelog`'s prereqs so the
detached-distiller e2e can't silently SKIP (it was only building via a sibling
target's order). Minor fixes: made the `.viewport` write atomic (temp+rename,
matching the shell's `> .tmp && mv -f`, since a live viewer's `G` refresh may
re-read it); restored the two error paths' second explanatory line;
documented `firstAgentPaneID`'s map-order non-determinism as moot under the
two-pane invariant. Plan `## Revisions` records the seam-name deltas
(`AgentPaneID`/`StartDetached`/`FileSize`+`Executable`+`Touch`) and the reviewer's
forward note: extract a shared `osfs`/`osseam` before M3–M5 add a 4th–6th
`OSRuntime` copy of the trivial fs primitives.

Created as step 3 of the native-single-binary tracker (#91) — the load-bearing
port. Surfaces and priorities drawn from `atlas/go-migration-inventory.md`;
milestone ordering puts the leaf orchestrators before the launcher so
`bin/pair-shell` shrinks before it is replaced. Per the tracker's granularity
decision this stays one ticket with per-surface milestones (M1–M5); a milestone
can be split into its own ticket later if its scope grows.
