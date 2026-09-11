---
id: 000183
status: working
deps: []
github_issue:
created: 2026-09-04
updated: 2026-09-10
estimate_hours: 1.62
started: 2026-09-10T16:57:25-07:00
---

# Attach drops PAIR_SCOPE_KEY, so the context meter vanishes after reattach

## Problem

A pane's zellij frame title is `<agent> (<count>)` — `claude (184k)` — where the
count is the agent's context-window size (#71). **After a thread is reattached,
the count is gone and the frame reads bare `claude`.** Observed 2026-09-04 on
all three reattached threads in one couch session, and not on a thread that had
been launched normally and never detached.

The failure is silent in both directions: nothing errors, and the title still
renders — it just quietly loses the half the operator was reading. The context
meter is the one surface that says "this session is nearly full", so losing it
without a signal is worse than losing it loudly.

## Spec

### Root cause

`AttachExistingSession` exports four environment variables before spawning the
title poller (`cmd/internal/launcher/lifecycle.go:42-48`):

```go
// Export what the spawned poller inherits (pair-shell exports these globally
// before the branch; the attach branch itself only re-exports PAIR_TAG).
rt.SetEnv("PAIR_HOME", opts.PairHome)
rt.SetEnv("PAIR_DATA_DIR", env.DataDir)
rt.SetEnv("PAIR_TAG", tag)
rt.SetEnv("PAIR_SESSION_NAME", session)
...
rt.SpawnTitlePoller(tag, agent, session)
```

`PAIR_SCOPE_KEY` is not among them. The comment records the assumption that
broke: in the shell, `bin/pair` exported the whole block **before** branching
create-vs-attach, so the attach branch only needed `PAIR_TAG`. In the Go port
the full export block moved *inside* the create branch
(`createflow.go:550-557`, `PAIR_SCOPE_KEY` at :554), and attach kept a
hand-copied four-line subset of what used to be global.

The chain to the symptom:

1. The poller's `ContextCount` calls `contextcmd.Run(..., contextcmd.EnvFromOS(), ...)`
   (`titlepoller/runtime.go:98-102`).
2. `EnvFromOS` reads `PAIR_SCOPE_KEY` from the poller's own environment —
   empty (`contextcmd/contextcmd.go:26`).
3. `sessioninventory.QuerySession(runtime, "", tag, agent)` matches nothing, so
   the status is not `BindingEstablished`; `contextcmd.Run` returns 0 having
   printed nothing (`contextcmd.go:43-46`).
4. `ContextCount` is `""`, so `frameTitle(agent, "")` returns bare `agent`
   (`titlepoller/titlepoller.go:65-70`).

An empty scope key is not an error at any step — it just matches nothing. That
is why this presents as a missing feature rather than a failure.

### Why it needs a reattach to show

A created session's poller is spawned from `createflow` with the full env, so
the meter works. Detach kills the poller along with the actor's process group;
couch's reattach respawns it through `AttachExistingSession`, without the scope
key. So the meter works until the first detach/reattach cycle and never again
for that incarnation.

### The fix, and the thing behind it

Exporting `PAIR_SCOPE_KEY` in `AttachExistingSession` restores the meter. The
value is available: `createflow` resolves it via
`ResolveRepoScope(envScopeRoot(env))`, and `env.DataDir` is `repos/<scopeKey>`
by construction. Note couch already passes the same value into the process as
`COUCH_THREAD_SCOPE` — the data is present under another name, which is what
makes the omission a plumbing gap rather than a missing capability.

**The defect behind the defect is two hand-maintained export lists that must
agree** (ARCH-DRY). Attach is also missing `PAIR_AGENT`,
`PAIR_LAUNCH_ORDINAL`, `PAIR_WORKBENCH_LAYOUT` and every artifact binding.
Those do not affect the meter today only because the zellij panes still carry
their env from the original create — attach spawns no panes. So the next thing
the poller learns to read breaks exactly the same way, silently.

The target is **one declaration of what a title poller needs in its
environment**, consumed by both branches — not making the two lists identical.
They legitimately differ: `PAIR_LAUNCH_ORDINAL`, `PAIR_SESSION_ID` and
`PAIR_AGENT_ARGS` are create-specific and have no meaning on attach.

## Done when

- A thread that has been detached and reattached shows `<agent> (<count>)` in
  its zellij frame title, verified on the real stack.
- The poller's environment requirement is declared in ONE place that both the
  create and the attach path consume; neither re-lists it by hand.
- A regression proves the attach path exports everything that declaration
  names — failing today on `PAIR_SCOPE_KEY`. It must assert against the
  declaration, not against a second hand-written list, or it reproduces the
  bug it exists to catch.
- `contextcmd` distinguishes "no established binding" from "no scope key
  supplied" rather than returning the same silent zero for both — a caller that
  forgot to pass scope should be diagnosable. (`pair doctor` is the natural
  home if a surface is wanted; a non-silent internal return is the minimum.)

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec           design=0.35 impl=0.05
item: milestone-review     design=0.00 impl=0.20
item: milestone-review     design=0.00 impl=0.10
item: smaller-go-module    design=0.05 impl=0.12
item: smaller-go-module    design=0.05 impl=0.12
item: smaller-go-module    design=0.05 impl=0.16
item: atlas-docs           design=0.05 impl=0.04
item: milestone-review     design=0.00 impl=0.20
design-buffer: 0.15
total: 1.62
```

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. The rows, in order:

1. The in-window design already spent: claim, then the plan doc. Anchored on
   the measured 0.79h at change-code, because claiming early puts design inside
   the window.
2. Plan-review round 1.
3. Plan-review round 2.
4. `contextcmd`'s names plus the no-scope status and the dispatcher's stderr.
5. `titlepoller.SessionEnv` and the observed-reads test.
6. The launcher wiring on both paths, the regression test and the spawn pin.
   Module design is thin because the plan carries literal code.
7. The atlas.
8. One close review.

The design buffer is +15% because a thorough plan doc exists. Frequency, per
`#201`: this runs once per launch or reattach, and cost is not a concern.
(`sdlc estimate-source` reports the calibration doc `[stale]`, #127.)

## Plan

Durable plan: `workshop/plans/000183-attach-drops-pair-scope-key-plan.md`. It
holds the code, commands, and mutation table, and has had two fresh-context
review rounds. It is single-pass, with one review boundary at `sdlc close`, so
there are no `Mx` tags.

- [ ] Task 0: `sdlc change-code` (branch, plan gate, estimate).
- [ ] Task 1: `contextcmd` names its variables (`EnvDataDir`, `EnvScopeKey`,
      `EnvFrom`), and a missing scope key gets its own status
      (`ExitNoScopeKey`, with a stderr reason), distinct from an unbound owner.
- [ ] Task 2: `titlepoller.SessionEnv` / positional `NewSessionEnv` /
      `Environ`, beside the `optionsFromCLI` parse it inverts, tied to the
      observed reads.
- [ ] Task 3: the regression test, red first, fails on attach's
      `PAIR_SCOPE_KEY` and is measured against the contract's own names.
- [ ] Task 4: both launch paths hand the poller its contract; attach resolves
      scope like create; the stale `lifecycle.go:42-43` comment goes;
      `titlePollerArgv` becomes `titlePollerSpawn`.
- [ ] Task 5: atlas (the poller's launch contract, and a missing scope is not
      an unbound owner).
- [ ] Task 6: mutation sweep (8 rows, each matched to its named failure), an
      unsandboxed `make test`, and real-stack detach/reattach by the operator.

## Log

### 2026-09-04

Diagnosed from an operator report — "the context window display stopped working
after reattachment; all three pair sessions are reattached and none has the
`claude (42k)` display".

Measured, in order, on the live host:

- **The pollers are alive** for all three reattached threads (pids 14291,
  14892, 38232), so it is not a missing sidecar.
- **The data is fine.** `PAIR_SCOPE_KEY=2e51fcf9799b1d8f pair context
  couch-65a74c2a094a07f4 claude` → `193k`. The same command with the variable
  unset prints nothing and exits 0.
- **The environment differs by launch path.** Poller 38232, spawned by `pair
  resume`, carries only `PAIR_HOME`, `PAIR_DATA_DIR`, `PAIR_TAG`,
  `PAIR_SESSION_NAME` (plus couch's own `COUCH_*`). Poller 26468 for
  `arc-agi-3`, a normal launch, carries `PAIR_SCOPE_KEY=f8b24636cbc27cd2` and
  the full artifact-binding block — and that pane still shows its meter.
- `PAIR_SCOPE_KEY` is the first 16 hex chars of SHA-256 over the cleaned
  absolute repo root (`launcher/scope.go:23-34`); confirmed by hashing
  `/Users/xianxu/workspace/brain` → `2e51fcf9799b1d8f`, matching the data dir.

Unrelated observation, low confidence, recorded so it is not lost: the
`📁brain-couch-19` zellij server had two live `pair wrap` panes at diagnosis
time — pid 26570 on `--resume 765a05dd…` (19h) and 54360 on `--session-id
8af77697…` (11h). May be intentional; not investigated.

### 2026-09-10

Claimed. The code was re-read against the issue's diagnosis, and it still
holds: `lifecycle.go:44-47` exports four variables and no `PAIR_SCOPE_KEY`.

**Design.** The poller no longer inherits its session variables. The spawn
hands it `titlepoller.SessionEnv`, the way `SpawnSessionWatcher` already
receives `scopeKey` as an argument. Facts that shaped it:

- **An empty scope key is never legitimate.** `sessionledger.validateRecord`
  rejects scope-less records (`record.go:357`), so `contextcmd` can treat an
  empty key as a caller error.
- **Attach cannot refuse on a missing scope the way create does**
  (`createflow.go:386-389`). The meter is optional; the handoff is not.
- **A scope-less poller already running outlives the upgrade.** The
  single-instance guard keeps the first one (`titlepoller/run.go:95-100`), so
  the real-stack check is detach, then reattach.

**Plan review, two rounds**, each applying Tasks 1–4 to a scratch clone:

- **Round 1** found three problems:
  - A new production file would have failed `artifactpath`'s exhaustive source
    inventory.
  - The regression test was itself a second hand-kept list. It also exposed a
    worse hazard: an unfilled future field would render an empty `PAIR_X=`
    that overrides a good inherited value, because exec keeps the last
    duplicate key.
  - One mutation expectation was false.
- **Round 2** confirmed the design, ran every mutation, and found two needles
  that did not do what their rows claimed. One would have "passed" on an
  unused-variable compile error.

Both rounds are recorded in the plan's `## Revisions`.

## Revisions

### 2026-09-10 — estimate re-derived per the estimate-quality judge (1.10 → 1.62)

Reason: the first derivation had no row for the design and the two plan-review
rounds already spent inside the measured window (0.79h at change-code). Claiming
early puts that time in the window, and the judge showed recent pair v3.1 rows
running around 0.6× for this reason. The judge suggested 1.6–1.8h.
Delta: added an `issue-spec` row and two `milestone-review` rows; module design
cut to 0.05 each. The implementation rows are unchanged.
