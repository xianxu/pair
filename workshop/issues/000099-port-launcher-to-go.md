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
- [ ] M2 — Runtime seam + create-flow orchestration: define `launcher.Runtime`
      (zellij exec/query, fzf/prompt, markers, cmux, config read/write, nvim reap,
      spawns, tty, env); build `RunLaunch` for the **create** path (native create
      behind `PAIR_NATIVE_LAUNCH`; shell stays default). `RunLaunch` stays a thin
      orchestrator over pure deciders — no business logic inline. Fake-`Runtime`
      tests for create / name-prompt / tag-restart config picker.
- [ ] M3 — attach / restart / quit / compaction orchestration: native attach, the
      restart-marker re-entry (in-process loop, not `exec $0`), in-session
      compaction, quit cleanup. Fake-`Runtime` loop tests for each.
- [ ] M4 — in-process cutover: flip `cmd/pair-go` to run the native launcher
      in-process under `PAIR_NATIVE_LAUNCH`; convert `bin/pair-shell` to a thin
      shim → `pair-go launch`. Full e2e vs the shell, then flip the default.
- [ ] M5 — subcommands + retirement: port `list`/`rename`/`continue`; retire the
      shell fallback + `bin/pair-restart.sh` markers → in-process; drop the flag;
      resolve `bin/pair-shell` shim-vs-remove via an explicit `git ls-files bin/` +
      caller check.

## Log

### 2026-07-02
- 2026-07-02: closed M1 — go test ./cmd/internal/launcher green — pure per-agent-arg/config/format helpers + named idempotence/collision/strip tests; boundary review ran via sdlc judge milestone-review --base (branch base) → FIX-THEN-SHIP, all findings fixed (agy/codex persist-strip completed, strconv dedup); go build ./... + vet clean; zero behavior change (unwired); review verdict: not-run
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
