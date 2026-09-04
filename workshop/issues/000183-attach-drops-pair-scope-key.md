---
id: 000183
status: open
deps: []
github_issue:
created: 2026-09-04
updated: 2026-09-04
estimate_hours:
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

## Plan

Single-pass: one plumbing fix plus the declaration that prevents its class. No
`Mx` tags — one review boundary (AGENTS.md §3).

- [ ] Failing test first: assert the attach path's exported env satisfies the
      title poller's declared requirement. Red on `PAIR_SCOPE_KEY`.
- [ ] Extract the poller's environment requirement to one declaration; have
      `createflow` and `AttachExistingSession` both consume it. Resolve the
      scope key on the attach path the way `createflow` does rather than
      deriving it from `env.DataDir`'s basename — a path-shape coupling is how
      this becomes a second silent failure.
- [ ] Make an absent scope key non-silent inside `contextcmd`, distinct from a
      binding that is genuinely not established.
- [ ] Delete the stale comment at `lifecycle.go:42-43` — the "pair-shell
      exports these globally" premise is no longer true and is what made the
      subset look complete.
- [ ] Real-stack verification: detach and reattach a live thread; confirm the
      count returns to the frame title.
- [ ] Atlas: the poller's env contract beside the create/attach split, if the
      surface warrants a line.

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
