---
id: 000106
status: wontfix
deps: []
github_issue:
created: 2026-07-07
updated: 2026-07-07
estimate_hours:
---

# zellij kill-session blocked under agent command sandbox — compaction restart doesn't complete

## Problem

Split out from **#105** (deterministic writer-triggered restart). #105 fixed the
*detection* half of the compaction restart under a sandboxed agent shell
(`PAIR_FAKE_IN_ZELLIJ` bypasses the sandbox-blocked `InZellijPane` proc-ancestry
walk). Its re-smoke confirmed detection now fires — compaction begins
("compacting pair-ariadne — parking scrollback…") — **but the restart still does
not complete: `zellij kill-session` cannot reach zellij's server socket from the
agent's command sandbox.** The kill runs from the agent's shell, so it inherits
the sandbox's unix-socket restrictions.

The restart *does* complete when the `pair continuation` writer is run
**unsandboxed** (`dangerouslyDisableSandbox`), which isolates the sandbox as the
sole remaining blocker — the whole flow (proc-walk detection + kill-session +
outer relaunch) is correct end-to-end. So this is a **sandbox policy / packaging
gap, not a flow bug**: in the operator's normal setup (agent runs with a command
sandbox), `alt+shift+c` still won't fully complete the restart until the kill can
reach the socket.

See #105 `## Revisions` (2026-07-07 entry) for the full re-smoke evidence.

## Spec

Make the compaction restart complete end-to-end **while the agent shell is
sandboxed** (the operator's default). Two candidate approaches (decide in the
plan):

1. **Allowlist zellij's server socket** in the agent's command-sandbox
   `allowUnixSockets` so the sandboxed writer's `zellij kill-session` reaches the
   server. Smallest change; but it lives in the *agent repo's* sandbox config
   (ariadne `.claude/settings*.json`), not in pair — so pair can't self-contain
   the fix, and every downstream consumer's sandbox needs the same entry.
   Resolve the socket path (mac: under `$TMPDIR`/`~/Library/Caches/...`; linux:
   `/tmp/zellij-<uid>/`) and whether a glob is required.
2. **Make pair's kill sandbox-robust** — perform the session teardown from a
   process *outside* the agent sandbox (e.g. defer the `kill-session` to the
   outer reincarnation loop in `createflow.go`, which runs un-sandboxed, rather
   than from the writer/`pair continue` child). Self-contained in pair; larger
   scope — needs a design pass on who owns the kill and how the marker/kill
   handshake works.

## Done when

- With the agent shell sandboxed (operator default), `alt+shift+c` completes the
  restart: the tag relaunches fresh (pair/pair-dev preserved) seeded from the
  continuation's NEXT ACTION — no unsandboxed escape hatch required.
- Verified by a live `alt+shift+c` smoke in a real sandboxed pair session (the
  same manual path #105 used), plus whatever unit coverage the chosen approach
  admits.

## Plan

- [x] Mark stale as wontfix after confirming the normal `Alt+Shift+C`
      continuation/restart workflow now succeeds.

## Log

### 2026-07-07
- 2026-07-07: wontfix — stale as written. The reported operator workflow now
  succeeds in normal use: `Alt+Shift+C` writes the continuation and restarts pair
  after capture completes. Pair has not landed the larger self-contained
  outside-sandbox teardown design; the observed failure no longer reproduces in
  the current environment, so this issue is no longer actionable.
- Filed from #105. Root cause isolated there: detection fixed (`PAIR_FAKE_IN_ZELLIJ`), but `zellij kill-session` can't reach the server socket from the sandboxed agent shell; restart completes only when the writer runs unsandboxed. This issue tracks closing that last gap.
