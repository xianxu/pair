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
item: larger-go-subsystem design=1.5 impl=4.5
item: greenfield-go-module design=1.0 impl=2.5
item: greenfield-go-module design=0.7 impl=2.5
item: milestone-review design=0.0 impl=1.5
item: atlas-docs design=0.2 impl=0.7
total: 17.7
```

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.* Items map to the plan: L1 pure-logic
completion, **L2 Runtime seam + native orchestration (the dominant item + the
dominant uncertainty)**, L3 in-process cutover + e2e, L4 subcommands + shell
retirement, the four boundary reviews, and the atlas sweep. Design subtotal 3.9 ×
1.15 (thorough-plan-doc buffer) = 4.5; impl subtotal 13.2; total 17.7.

**Honest uncertainty:** this is interaction-heavy lifecycle work (blocking zellij
handoffs, restart re-exec, TTY handling, quit cleanup) — the exact class the M4
estimate-quality judge warned the model under-weights ("6.0h for a 2287-line
launcher is optimistic"). L2/L3 could each run high; the honest band is ~13–22h.
That the launcher alone ≈ #93's original whole-issue 17.4h is the point: the old
6.0h M5 placeholder was the under-scope, now corrected by extracting this ticket.

## Plan

Each `Lx` is a merge-safe review boundary closed on its own (`sdlc
milestone-close`). Independently mergeable; the shell launcher stays the default
until L3 flips it, so pair stays usable throughout.

- [ ] L1 — pure-logic completion: port the remaining pure pieces into
      `cmd/internal/launcher` (full `ParseArgs` incl. `continue`/`rename`/`list`;
      resume-token strip/compose — one helper for the 4 duplicated shell loops;
      config-migration decision rules; per-agent launch-arg composition — claude
      `--session-id` shape, codex `--no-alt-screen` idempotence; `rename`
      plan-build; title/`format_age`/`age_color`). Unit-tested, not yet wired —
      zero behavior change.
- [ ] L2 — Runtime seam + native orchestration: define `launcher.Runtime`
      (zellij exec/query, fzf/prompt, markers, cmux, config read/write, nvim reap,
      spawns, tty, env); build `RunLaunch` driving decision → effects → blocking
      handoff → cleanup/restart. Fake-`Runtime` loop tests for create/attach/
      picker/name-prompt/tag-restart/restart-re-entry/compaction/quit.
- [ ] L3 — in-process cutover: flip `cmd/pair-go` to run the native launcher
      in-process under `PAIR_NATIVE_LAUNCH`; convert `bin/pair-shell` to a thin
      shim → `pair-go launch`; restart re-exec → in-process loop. Full e2e vs the
      shell, then flip the default.
- [ ] L4 — subcommands + retirement: port `list`/`rename`/`continue`; retire the
      shell fallback + `bin/pair-restart.sh` markers → in-process; drop the flag.

## Log

### 2026-07-02
- Created by extracting #93 M5 (the launcher) into its own ticket — the surface
  (~900 lines new IO orchestration + a new effect seam + the trickiest lifecycle
  logic in the tree, P0) is categorically larger than the M1–M4 leaf ports and
  warrants its own estimate + isolated actuals. Design surveyed + approved in the
  #93 plan; moved to `workshop/plans/000099-port-launcher-to-go-plan.md`.
