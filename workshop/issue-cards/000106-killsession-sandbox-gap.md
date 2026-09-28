---
id: '000106'
status: wontfix
created: 2026-07-07
updated: 2026-07-07
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
