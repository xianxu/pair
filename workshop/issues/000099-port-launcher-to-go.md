---
id: 000099
status: working
deps: []
github_issue:
created: 2026-07-02
updated: 2026-07-02
estimate_hours: 17.7
started: 2026-07-02T11:20:01-07:00
---

# port the pair-shell launcher to Go

Tracking: #91 (native single binary) — the launcher half of step 3. Extracted
from #93 M5 (the leaf orchestrators M1–M4 landed there; this is the launcher,
carved out for scope). No formal `deps:` on #93 because the prerequisite leaf
ports are already merged — a formal dep would be a self-block, since #93 stays
open until this lands (its Done-when includes a Go owner for the launcher).

## Problem

`bin/pair-shell` (2287 lines) is the last and largest shell orchestrator: it owns
the zellij session lifecycle, the create/attach/pick decision, three UIs (fzf
session picker, fzf config/tag-restart picker, zsh `vared` name-prompt), the
restart/quit marker lifecycle, cmux ownership, config/session migration, per-agent
launch-arg composition, nvim orphan reaping, the `list`/`rename`/`continue`
subcommands, and the spawns of the (already-Go) title poller + session watcher.
Until it moves into Go, `pair` can't stop `syscall.Exec`ing a shell launcher and
#94 (stop extracting a shell tree) can't proceed. It's P0 in
`atlas/go-migration-inventory.md`.

## Spec

Port `bin/pair-shell`'s orchestration onto the existing `cmd/internal/launcher`
pure decision core (from #75), behind a new `launcher.Runtime` effect seam, on the
M1–M4 template: pure decisions unit-tested directly; all IO (zellij exec/query,
fzf/prompt, markers, cmux, config read/write, nvim reap, spawns, tty, env) behind
the `Runtime` seam, fake-tested; a compatibility shim retained during transition.
zellij/nvim stay native (#95 boundary). Detailed design + the four-phase plan:
`workshop/plans/000099-port-launcher-to-go-plan.md`.

Key facts (survey 2026-07-02): the decision core (`ParseArgs`/`DecideLaunch`/
`ZellijSource`/`HistorySource`) already exists but is **bypassed** — `cmd/pair-go`
`syscall.Exec`s `bin/pair-shell`. ~900 lines of stateful IO orchestration have no
Go home; that's the work. Two child-spawns (title poller, session watcher) are
already Go binaries — wire, don't re-port; the `$0` self-re-exec (restart /
in-session compaction) becomes an in-process loop.

## Done when

- [ ] The Go `pair` binary runs the launcher **in-process** (no `syscall.Exec` of
      `bin/pair-shell`); `bin/pair-shell` survives only as a thin re-exec shim (or
      is removed once no caller needs it).
- [ ] Pure launch decisions (parse, tag/name derivation, decision, resume-token +
      config-migration + per-agent-arg rules, rename plan) are unit-tested; all
      zellij/nvim/cmux/fzf/fs interaction is behind a process-tested `Runtime` seam.
- [ ] Every lifecycle flow works natively: create, attach, picker, name-prompt,
      tag-restart config picker, restart-marker re-entry, in-session compaction,
      quit cleanup, and the `list`/`rename`/`continue` subcommands.
- [ ] The `bin/pair-restart.sh` marker handshake is in-process; the shell launcher
      + its markers are retired, unblocking #94.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module design=0.5 impl=1.5
item: greenfield-go-module design=0.9 impl=2.7
item: greenfield-go-module design=0.6 impl=1.8
item: greenfield-go-module design=1.0 impl=2.5
item: greenfield-go-module design=0.7 impl=2.5
item: milestone-review design=0.0 impl=1.5
item: atlas-docs design=0.2 impl=0.7
design-buffer: 0.15
total: 17.7
```

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.* The 5 `greenfield-go-module` items map 1:1 to
the 5 plan milestones (the dominant work is split into two boundaries since the
closed vocabulary tops out at single-concern `greenfield-go-module`, below
charon-scale `greenfield-service`): M1 pure-logic completion; **M2 Runtime seam +
create-flow and M3 attach/restart/quit/compaction — the dominant items + dominant
uncertainty**; M4 in-process cutover + e2e; M5 subcommands + shell retirement; plus
`milestone-review` (the five boundary reviews) and `atlas-docs` (the sweep).
Reconciles: design Σ3.9 × 1.15 (thorough-plan-doc buffer) + impl Σ13.2 × 1.0 =
17.69.

**Honest uncertainty:** this is interaction-heavy lifecycle work (blocking zellij
handoffs, restart re-exec, TTY handling, quit cleanup) — the exact class the M4
estimate-quality judge warned the model under-weights ("6.0h for a 2287-line
launcher is optimistic"). L2/L3 could each run high; the honest band is ~13–22h.
That the launcher alone ≈ #93's original whole-issue 17.4h is the point: the old
6.0h M5 placeholder was the under-scope, now corrected by extracting this ticket.

## Plan

Each `Mx` is a merge-safe review boundary closed on its own (`sdlc
milestone-close`). Independently mergeable; the shell launcher stays the default
until M4 flips it, so pair stays usable throughout.

- [x] M1 — pure-logic completion: port the remaining pure pieces into
      `cmd/internal/launcher` (resume-token strip/compose — one helper for the 4
      duplicated shell loops; config-migration decision rules; per-agent launch-arg
      composition — claude `--session-id` mint/skip, codex `--no-alt-screen`
      idempotence; title/`format_age`/`age_color`). Unit-tested, not yet wired —
      zero behavior change. **Scoped:** the `rename` plan-build + full-`ParseArgs`
      (`continue`/`rename`/`list`) parse deferred to their consuming milestones
      (M5 / M2) to avoid unwired M5-only code + a risky change to the live
      `pair-go launch` parser — the create/restart-flow pure logic M2/M3 need is
      what M1 front-loads.
- [x] M2 — Runtime seam + create-flow orchestration: define `launcher.Runtime`
      (zellij exec/query, fzf/prompt, markers, cmux, config read/write, nvim reap,
      spawns, tty, env); build `RunLaunch` for the **create** path (native create
      behind `PAIR_NATIVE_LAUNCH`; shell stays default). `RunLaunch` stays a thin
      orchestrator over pure deciders — no business logic inline. Fake-`Runtime`
      tests for create / name-prompt / tag-restart config picker.
- [x] M3 — attach / restart / quit orchestration: native attach, the restart-marker
      re-entry (in-process loop, not `exec $0`: Alt+n resume / Shift+Alt+N fresh),
      quit cleanup (`cleanup_quit_marker`), and nvim reap/sweep. Fake-`Runtime` loop
      tests for each + real-OSRuntime file-IO tests. **Scoped:** in-session
      compaction, the `continue`/`rename` restart re-entries, and the fzf session
      **pick** deferred to M5 (they couple to M5's picker + `continue` parsing);
      all resolve to `ErrFallbackToShell` → shell until then.
- [ ] M4 — in-process cutover: flip `cmd/pair-go` to run the native launcher
      in-process under `PAIR_NATIVE_LAUNCH`; convert `bin/pair-shell` to a thin
      shim → `pair-go launch`. Full e2e vs the shell, then flip the default.
- [ ] M5 — subcommands + retirement: port `list`/`rename`/`continue`; retire the
      shell fallback + `bin/pair-restart.sh` markers → in-process; drop the flag;
      resolve `bin/pair-shell` shim-vs-remove via an explicit `git ls-files bin/` +
      caller check.

## Log

### 2026-07-02
- **M3 implemented (attach + quit-cleanup + in-process restart loop).** New file
  `lifecycle.go` (`runAttach` — the shell attach branch; `runCleanup` — the ~130-line
  `cleanup_quit_marker` port; pure `liveTagsForSweep`/`tagFromEmbedArgv`). `runtime.go`
  gains the `LifecycleOps` sub-interface (attach + marker read-clear/peek +
  DeleteSession + ReapNvim + SweepOrphanNvim + ParkScrollback + ConfirmParkNudge +
  IsTTY + KillTitlePoller + cmux ownership). `createflow.go`'s `RunLaunch` is now the
  **in-process restart loop** (replacing `exec $0`): `runOnce` dispatches attach vs
  create, `runCreate` is the extracted M2 body; the in-pane guard + `SweepOrphanNvim`
  are **first-entry only**. `osruntime.go` wires the concrete seams (fork+wait attach
  sharing `runBlockingHandoff` with `LaunchSession` — ARCH-DRY; markers under
  `~/.cache/pair`; park move/copy via `transferFile`; ps-scan orphan sweep). Restart
  decisions stay pure (`planRestart`): Alt+n resumes, Shift+Alt+N drops the config.
  **Scoped to M5:** in-session compaction, the `continue`/`rename` restart re-entries,
  and the fzf **pick** → `ErrFallbackToShell` (plan Revision 2026-07-02). Known M5
  gap: a `continue`/`rename` restart under `PAIR_NATIVE_LAUNCH` runs the native quit
  cleanup (faithful to shell's `cleanup → handle_restart` order) then falls back to a
  fresh shell launch under the original tag — draft+config survive cleanup; the
  rename/continue-seeding is M5's job. **M3's actual is the real re-price signal** the
  M2/change-code judges asked for (impl=1.8h flagged light for the trickiest lifecycle
  logic) — measured at close.
- **M3 boundary review: FIX-THEN-SHIP → SHIP** (via `sdlc judge milestone-review
  --base <merge-base>` — the auto-window bug workaround, ariadne#162). No Critical.
  Two Importants, both doc/verification (not code correctness), fixed: (I-1) the
  plan/issue M3 bullet over-claimed in-session compaction as M3 — narrowed via the
  plan `## Revisions` entry + this Log + the ticked M3 checkbox; (I-2) the exec-only
  seams (attach / delete-session / ps-sweep) had no committed test and the
  `osruntime_test.go` comment referenced an ephemeral smoke — reworded the comment to
  be honest and RECORDED the boundary smoke here (see VERIFICATION). Minors taken:
  `runAttach` now pins the "agent is the inferred title agent" invariant in a comment;
  `ConfirmParkNudge` documents its benign quit-time reader-goroutine. Minors left
  (equal-or-safer / cosmetic): `quitAgent` uses `InferAgent`'s config-glob superset;
  cleanup output routes to the single stderr writer (shell splits stdout + /dev/tty).
- **VERIFICATION (M3):** `go test ./cmd/internal/launcher` green + `-race` clean —
  fake-`Runtime` loop tests (attach / quit-full-teardown-with-park / detach-no-op /
  park-skip-on-restart / Alt+n resume [launchCount==2, `--resume`] / Shift+Alt+N fresh
  [config dropped, no resume] / rename+continue → `ErrFallbackToShell` after one
  cleanup / sweep-once) + pure helper tests + **real-OSRuntime file-IO tests**
  (`osruntime_test.go`: marker peek-vs-take, park move/copy/empty, cmux ownership,
  pidfile reaping against temp dirs — discharges the M2 "don't ship OSRuntime IO
  untested" lesson). Full `make test` green; `go vet`/`go build ./...` clean;
  runtimebundle-drift-check clean. **Real-OSRuntime end-to-end smoke** (stub zellij,
  temp HOME/XDG, scrubbed env): scenario A quit-only → attach handoff → cleanup
  deletes session + removes sidecars + raw-removed(park-skip no-tty) + resume hint,
  no re-create; scenario B restart → attach → cleanup delete → in-process re-create,
  **ATTACH→DELETE→CREATE** order, both markers consumed. PASS.
- 2026-07-02: closed M2 — ATLAS updated in the M2 window at commit 440998c (atlas/architecture.md launcher section + atlas/go-migration-inventory.md launcher row → #99 M2 native create preview); --no-atlas only because milestone-close computed an empty d5b3aa8..HEAD window that misses it (window bug, cf. the M1 milestone-review far-back-base bug). VERIFICATION: go test ./cmd/internal/launcher green (pure createlogic + zellijparse table tests + fake-Runtime loop tests: create/name-prompt/tag-restart-picker/codex/explicit-resume/agent-inference/probe-too-long/pre-handoff-collision/fallbacks); full make test green; go vet + go build ./... clean; runtimebundle-drift-check clean; real-OSRuntime end-to-end create smoke (stub zellij) PASS. Boundary review FIX-THEN-SHIP → SHIP (Important test-gap fixed; Review-Verdict trailer on commit d5b3aa8). Shell stays default; nothing user-facing flips.; review verdict: not-run
- **M2 implemented (Runtime seam + create-flow orchestration).** New files in
  `cmd/internal/launcher`: `runtime.go` (the `Runtime` effect seam, composed from
  sub-interfaces — `ZellijOps`/`SnapshotOps`/`UIOps`/`ProcOps`/`EnvOps`/`IDOps`/
  `FSOps` — per the change-code ISP INFO), `createlogic.go` (pure create-flow
  helpers: explicit-resume extract, config JSON build/parse, tag-restart choice
  build + selection-map + arg compose — all reusing M1's agentargs), `createflow.go`
  (`RunLaunch` — the thin orchestrator + `promptForTag`/`runConfigPicker`/
  `resolveConfigPath` sub-drivers), `osruntime.go` (concrete `OSRuntime`: 5s-timeout
  zellij queries, fork+wait `LaunchSession`, zsh-vared prompt, fzf `--read0` picker,
  cmux/tty/spawn/uuid), `runcli.go` (`LaunchNative` os-plumbing entry). Wired behind
  `PAIR_NATIVE_LAUNCH` in `cmd/pair-go/main.go` (create-only preview; attach/pick/
  in-pane/unsupported-verbs → `ErrFallbackToShell` → shell). Decisions worth noting:
  (a) **fork+wait, not `syscall.Exec`** — `LaunchSession` is `cmd.Run()` with tty
  passthrough so the launcher regains control for M3's quit/restart (change-code M3
  crux, pinned now as the `LaunchSession` contract). (b) **`persistedConfigArgs`
  used as the single resume-strip** at every persist + re-compose site — a superset
  of the shell's per-site resume-only strips (it also drops a stray `--session-id`),
  which is safer for a fresh/re-composed launch and keeps M1's ARCH-DRY
  consolidation. (c) **agent inference** ported for the `resume <tag>` path (agent
  unset by ParseArgs) — `InferAgent` reads `agent-<tag>` then the config-filename
  agent, defaulting to claude. (d) `osfs.FS.Touch` now `MkdirAll`s its parent (the
  create path Touches the draft before any `WriteAtomic` makes the data dir — the
  shell `mkdir -p`s `$DATA_DIR` early; opener's `Touch(log)` is unaffected).
  Verified: `go test ./cmd/internal/launcher` green (pure-helper unit tests +
  fake-`Runtime` loop tests for create / name-prompt / tag-restart picker / codex /
  explicit-resume / fallbacks / agent-inference); full `make test` green;
  `go vet` clean; a **real-OSRuntime end-to-end smoke** (stub zellij+agent, temp
  HOME/XDG) confirmed `resume <freshtag>` mints a real uuid, writes config
  (resume-stripped) + agent record + truncated adapt recorder + seeded draft, ran
  the name-length probe + EXITED-clear, and handed off `--new-session-with-layout
  --session pair-<tag>`. Nothing user-facing changes (shell stays default).
- **M2 boundary review: FIX-THEN-SHIP → SHIP** (via `sdlc judge milestone-review
  --base <merge-base>`; the auto-window bug did not recur this milestone since
  `main == merge-base`, but the manual base was used to be safe). No Critical. The
  one **Important** (fixed): the `OSRuntime` zellij-output parsers (#54/#67 logic)
  + two `RunLaunch` error branches shipped untested. Fix: extracted the row-parse
  into pure `sessionRowState`/`familyRows`/`sessionNameRejected` (`zellijparse.go`,
  also dedups the two IO methods — ARCH-DRY) with a table test, and added
  fake-`Runtime` tests for the probe-too-long + pre-handoff-collision exits.
  Minors taken: `extractExplicitResume` now keeps scanning past a bare
  `--conversation=` (shell-faithful); the `cmd/pair-go` gate uses
  `errors.Is(err, ErrFallbackToShell)` so only the sentinel defers to the shell
  (a future non-fallback error is surfaced, not silently re-run). Minors left
  (degenerate/corruption-only drifts where Go's behavior is equal-or-safer):
  empty `agent-<tag>` falls to the config glob; a malformed config skips the
  picker. go test + smoke re-green after fixes.
- 2026-07-02: closed M1 — go test ./cmd/internal/launcher green — pure per-agent-arg/config/format helpers + named idempotence/collision/strip tests; boundary review verdict FIX-THEN-SHIP (all findings fixed: agy/codex persist-strip completed, strconv dedup); go build ./... + vet clean; zero behavior change (unwired). (The "not-run" suffix below is sdlc's `--no-judge` marker, NOT the review outcome: the boundary review DID run — via `sdlc judge milestone-review --base <branch-base>` — because milestone-close's auto-window picked a wrong far-back base → 6.8 MB diff → `fork/exec claude: argument list too long`; verdict is the FIX-THEN-SHIP above, and the M1 commit carries the real `Review-Verdict:` trailer.); review verdict: not-run
- **change-code:** plan-quality CLEAN, estimate-quality INFO (branch created).
  Fixed the one blocking plan-quality finding first: boundary tags were `Lx` but
  `sdlc`'s milestone-verdict gate only recognizes `M\d+`, so `Lx` would have made
  the final-close review gate a silent no-op — renamed to M1–M5 (splitting the
  dominant work into M2 create-flow + M3 attach/restart/quit). INFOs to fold:
  M1 → named unit tests for the idempotence/collision behaviors (claude
  `--session-id` retry, codex `--no-alt-screen`); M2 → compose `Runtime` from
  sub-interfaces (zellij/ui/markers/config), not a god-interface; M3 → the zellij
  handoff must be fork+wait (Go regains control for `cleanup_quit_marker`), an
  explicit `Runtime` contract, and revisit M3's impl weight (light for its
  complexity) at M3 start with concrete scope rather than back-fitting now.
- Created by extracting #93 M5 (the launcher) into its own ticket — the surface
  (~900 lines new IO orchestration + a new effect seam + the trickiest lifecycle
  logic in the tree, P0) is categorically larger than the M1–M4 leaf ports and
  warrants its own estimate + isolated actuals. Design surveyed + approved in the
  #93 plan; moved to `workshop/plans/000099-port-launcher-to-go-plan.md`.
