---
type: continuation
slug: launcher-retired
agent: claude
created: 2026-07-03T10:48:19
supersedes: launcher-m5
branch: main
issues: [000099]
---

# Continuation: launcher-retired

## NEXT ACTION

**#99 is DONE (closed + merged) — this session finished it.** The immediate next
step is to **close #93** (`sdlc close --issue 93`): it is `working` 4/5 with
M1–M4 done and **M5 explicitly "extracted to #99"** (the launcher), which now
landed — so its only open milestone is delivered and `sdlc state` already
drift-flags it "looks done." Verify its Done-when is satisfied by #99's Go
launcher owner, then close it (compute `--actual` via `sdlc actual --issue 93`,
never hand-typed). That formally lands the #91 native-single-binary roadmap's
launcher step.

Then pick the next native-binary step with the user (see Open questions):
**#94** (stop extracting a shell tree — the remaining bundled `*.sh` shims),
**#95** (native nvim/zellij startup assets), or the small **`pair restart`
subcommand** follow-up (#99 deferred it).

## State of play

- **#99 `done`, merged, archived** to `workshop/history/`. `bin/pair-shell`
  (2287 lines) is **removed**; `pair` (`cmd/pair-go` → `cmd/internal/launcher`)
  is a single Go launcher end-to-end (create/attach/fzf-pick/list/rename/continue/
  compaction/in-process-restart-loop/--help). No shell fallback — `ErrFallbackToShell`,
  `PAIR_LEGACY_LAUNCH`, the `Exec` seam all deleted. Shipped M5a/M5b/M5c as PRs
  **#61/#62/#63**; est 17.7h / **actual 11.57h** (1.5×, trusted-window). Run
  `git log --oneline` / read `workshop/history/000099-*` for the record.
- **#93 `working` 4/5 — closeable now.** M5 = launcher, extracted to #99 (done).
  `sdlc state` flags it "looks done." → the NEXT ACTION.
- **#94 `open` 0/4 — unblocked** by #99. But `bin/pair-shell` was only *one*
  shell file; `bin/pair-restart.sh`/`pair-quit.sh`/`copy-on-select.sh`/etc. still
  ship in the runtime bundle, so #94 is its own effort (not auto-done by #99).
- **#95 `open`** (native nvim/zellij assets); **#91** is the parent roadmap
  (launcher was step 3, now complete).
- **Deferred follow-up (needs an issue):** convert `bin/pair-restart.sh`'s nvim
  keybind marker-WRITE to an in-process `pair restart [--new-session] [--rename-to]`
  subcommand — reuses `serializeRestartMarker`/`WriteRestartMarker`/`TouchQuitMarker`/
  `ExecKillSession` (all exist) + repoint `nvim/init.lua:~3288`. Re-scoped out of
  #99's Done-when (it's an nvim marker-writer, independent of the launcher).

## Thread arc & user model

Unbroken relentless-forward-through-the-#91-roadmap cadence: terse "continue"s,
trusts recommendations, steps away for long stretches and returns with status
pulses ("where are we?", "not normal for make test to take this long?") to
sanity-check rather than redirect. This multi-session arc methodically ported the
last, largest shell orchestrator (`bin/pair-shell`) to Go one review-boundary at a
time — M2 create → M3 attach/restart → M4 cutover → **M5a pick+list → M5b
compaction+continue/rename → M5c retirement** (this session did all of M5). Each
milestone got the FULL SDLC treatment and a fresh-context continuation seeding the
next; this one closes the loop (the launcher is done, the arc's goal reached).

The user's values, reinforced every milestone and honored this session:
**dogfood-real verification** (stub-zellij smokes + full `make test`, not
fakes-only); **surfacing-and-deciding contradictions loudly** rather than papering
over them (the M4 shim→loop crux; the M5c remove-vs-shim decision, which the user
confirmed "remove outright, proceed autonomously"); **honoring the sdlc gates** (not
manual git; the gate errors are next-action specs); and **proceeding on best
judgment when away** (the M5-split + remove-outright calls were made autonomously
during an away stretch and later confirmed).

## Open questions

On resume, resolve these open questions with the user before continuing with the
NEXT ACTION.

1. **After closing #93, which native-binary step next?** Candidates: **#94**
   (retire the remaining bundled shell shims — the bigger, roadmap-central one),
   **#95** (native nvim/zellij startup assets), or the small **`pair restart`
   subcommand** follow-up. My leaning: the `pair restart` follow-up first (small,
   self-contained, finishes the launcher's marker story), then #94. But the user
   drives roadmap order — ask.
2. **Does #94 want a fresh survey first?** Its scope (which `*.sh` still ship, which
   have Go owners with shims vs. need porting) shifted as #93/#99 landed — a
   `git ls-files bin/` + bundle-manifest survey should precede its plan.

## Artifact map

Read-first ordering (issues are NOT auto-loaded; CLAUDE.md is):
- **`workshop/history/000099-port-launcher-to-go.md`** + **`...-plan.md`** — the
  archived #99 issue + durable plan (record of truth). The plan's `## Revisions`
  carry the per-milestone corrected scope (M5→M5a/b/c split, M5c retirement design
  incl. the `pair-restart.sh` deferral + the asset-root-marker rationale). The
  issue `## Log` has per-milestone actuals + FIX-THEN-SHIP verdicts.
- **`workshop/lessons.md`** — 5 lessons added this session (the durable flush):
  shell-seam-hazards-when-porting-a-flow-native (PAIR_TEST_CALL/PAIR_DEBUG_* lose
  their shell fallback when a flow goes native); `| tail` hides a running suite +
  `sdlc milestone-close --dry-run` mutates; **the sandbox blocks `ps`** → breaks
  `InZellijPane()` (run launcher ancestry tests sandbox-off); an asset-root validity
  marker must exist in every layout (why it's `zellij/layouts/main.kdl`, not the
  gitignored `bin/pair-wrap`).
- **`cmd/internal/launcher/`** — the Go launcher (the whole flow now). Key M5 files:
  `pick.go`/`list.go` (M5a), `compaction.go`/`continuation.go`/`rename.go` (M5b),
  `help.go` + the `runcli.go`/`createflow.go` fallback-arm removal (M5c). `cmd/pair-go/main.go`
  is the entry (`runLegacyLaunch` → `LaunchNative`, no shell). `cmd/internal/entrypoint/asset_root.go`
  has `ValidRootMarker`.
- **Memory updated:** `sdlc_gate_gotchas_pair_checkout` (sdlc actual works now;
  merge plan-judge gone #160; --dry-run mutates). MEMORY.md index refreshed.
- **On `main`, no active branch.** #99's branch was merged + deleted.

## Live deliberations

- **The `pair restart` follow-up vs #94 ordering** (Open question 1). Leaning: the
  restart subcommand first — it's small, finishes the launcher's marker-write story
  (the launcher already reads/writes markers; only the nvim-keybind WRITE stays
  shell), and needs a new tiny issue. #94 is the larger "retire the shell tree"
  effort and deserves its own survey+plan. Not decided — the user sets roadmap order.
- **Whether #94 subsumes the `pair restart` follow-up.** `pair-restart.sh`/`pair-quit.sh`
  are in the runtime bundle, so retiring them *is* part of #94's "stop extracting a
  shell tree." So the `pair restart` subcommand could be #94's first milestone
  rather than a separate issue. Worth deciding when planning #94.

## Decisions & dead ends

- **M5c: remove `bin/pair-shell` outright, not a thin shim.** User-confirmed (#94's
  point is to stop shipping a shell tree; the caller check showed the only runtime
  caller was the now-deleted fallback arm).
- **Asset-root validity marker → `zellij/layouts/main.kdl`**, NOT `bin/pair-wrap`.
  pair-wrap is a built, gitignored binary → absent in a fresh checkout pre-`make build`;
  keying the marker on it would reject un-built source roots. main.kdl is tracked +
  bundled + the file the launch reads. (Now a lesson.)
- **`pair-restart.sh` markers-in-process DEFERRED** (not done in M5c) — it's an nvim
  keybind marker-writer, independent of the launcher removal; a separable follow-up.
- **M5 split into M5a/b/c** (read-only surfaces / lifecycle writes / retirement) —
  three distinct risk profiles, retirement last (ARCH-PURPOSE: nothing can fall back
  before removal). Lettered tags recognized by sdlc's `M\d+[a-z]?` regex.
- **`--no-judge` the milestone/issue reviews** (ariadne#162 auto-window bug persists)
  — ran each review manually via `sdlc judge milestone-review --base <merge-base>`,
  put the REAL verdict in the `Review-Verdict:` trailer.

## Lessons learned

All flushed to **`workshop/lessons.md`** this session (5 entries — see Artifact
map). The meta-pattern worth carrying: **when porting a shell flow to Go
one-boundary-at-a-time, each boundary silently removes the fallback that shell-only
*test/debug seams* depended on** — at every boundary, ask "which seam reached the
shell only via *this* flow's fallback?" and route it explicitly (or retire its test
when the shell goes). This bit M5a (`PAIR_TEST_CALL`), M5b (`PAIR_DEBUG_ARGS`), and
M5c (all of them, retired). Also: in this agent-inside-pair sandbox, **`ps` is
blocked** and **`| tail` hides a running suite** — two "it's hung!" false alarms
that were pure environment; check those before suspecting the code.
