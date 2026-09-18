---
id: 000277
status: open
deps: []
github_issue:
created: 2026-09-17
updated: 2026-09-17
estimate_hours:
---

# Context meter shows the dead conversation's count after Shift+Alt+N

## Problem

Shift+Alt+N restarts the agent with a **fresh conversation**, but the agent
pane's frame meter (`<agent> (<count>)`, #71) keeps rendering the **previous**
conversation's context-window size. Observed sequence (operator, 2026-09-17):

1. Shift+Alt+N → the frame still shows the old, now-dead count (e.g. `claude (562k)`).
2. First `Return` in the new conversation → the count blanks to bare `claude`.
3. Once the new session's transcript carries usage → the real (small) count appears.

Steps 2 and 3 are correct. Step 1 is the defect: for the whole window between the
restart and the first submit — unbounded, since it's however long the operator
takes to type the first prompt — the meter is actively lying about which
conversation it measures. The failure mode is the same *class* as #98 (a frozen
number sourced from a dead transcript), but a different mechanism: #98 rendered a
stale *twin agent*; this renders a stale *paint*.

## Spec

### Why it happens

`titlepoller.updateFrameTitles` (`cmd/internal/titlepoller/run.go:196`) is the
**only** writer of the agent pane's frame title — grep confirms no other
`RenamePane`/`rename-pane` call site touches it. Two properties combine:

- **The restart path never repaints.** `planRestart` (`launcher/markers.go:183`)
  handles `NewSession` by dropping the saved config so the create path mints a
  fresh session id (`createflow.go:196`). It changes the agent's identity but
  leaves the pane title exactly as the last poll painted it.
- **The poller only repaints on recent activity.** `Run` calls
  `updateFrameTitles` only when `age < 2*PollInterval` over `activityMTime` =
  max(nvim draft mtime, `SessionActivity(tag, agent)`) — and the loop ticks at
  `PollInterval` (default 60s). A fresh conversation generates no activity until
  the operator submits, so no repaint fires. That first submit is exactly the
  step-2 trigger the operator observed.

The count itself is correct at every step: `contextcmd.Run` resolves the ledger's
`CurrentLaunch` binding and prints nothing unless `TokenUsageForRoot` finds a
usage record, so a just-minted session legitimately renders bare `<agent>`
(`frameTitle(agent, "")`). Nothing is wrong with the resolver — the frame is
simply never asked to re-render at the moment the conversation dies.

### Direction

Treat the restart as an **event that invalidates the meter**, rather than waiting
for the pull loop to notice:

- **(a) Push, at the event.** The `NewSession` restart path repaints the agent
  pane to `frameTitle(agent, "")` at the same point it drops the config. Cheap,
  deterministic, and owned by the layer that knows the conversation ended.
- **(b) Pull, close the gate.** Make a restart count as activity (e.g. include
  the config/ledger mtime in `activityMTime`), so the next tick repaints within
  one `PollInterval` instead of never.

(a) alone has a race worth settling during design: between the config drop and
the new launch record landing in the per-tag ledger, `CurrentLaunch` can still
name the **old** root, so a poller tick in that window could repaint the dead
count right back over the cleared title. Either (a)+(b) together, or a
`contextcmd`-side check that the resolved root matches the *current* config
session id, closes it. Verify the window empirically before choosing — the
operator's step 2 suggests the ledger flips at launch, not at first usage.

The invariant worth stating (and testing): **the meter never displays a count
sourced from a conversation the agent is no longer in.** Blank is always a
correct answer; a stale number never is.

## Done when

- [ ] Shift+Alt+N blanks the agent pane's count immediately — no dead count
      survives into the new conversation, even if the operator never submits.
- [ ] A poller tick during the restart window cannot repaint the dead count
      (race in Direction (a) settled, not assumed).
- [ ] The real count reappears once the new session's first usage record lands
      (no regression on step 3).
- [ ] Regression coverage at the pure seam: a fake-runtime test that restarts
      mid-loop and asserts the rendered title sequence.

## Plan

- [ ] Reproduce with an instrumented poller: log `activityMTime`, `age`, and the
      rendered title across a real Shift+Alt+N; confirm no repaint fires until
      first submit, and measure how long the ledger keeps naming the old root.
- [ ] Choose the fix per the race finding: (a), (a)+(b), or (a)+resolver check.
- [ ] Implement + regression test.
- [ ] Atlas: note the meter-invalidation rule wherever the frame meter is
      described.

## Log

### 2026-09-17

- Filed from a brain advisor session on the operator's report. Root cause traced
  by code read (no live repro yet): the restart path never repaints and the
  poller's activity gate never fires for a conversation that hasn't started.
  Observed sequence (stale → blank on first Return → correct) confirmed live by
  the operator, including step 3.
- Distinct from #98 (wontfix, resolved by #97): same visible symptom class
  (a count from a dead transcript), different mechanism — that was a stale pane
  twin winning the glob, this is a stale paint nobody invalidates.
