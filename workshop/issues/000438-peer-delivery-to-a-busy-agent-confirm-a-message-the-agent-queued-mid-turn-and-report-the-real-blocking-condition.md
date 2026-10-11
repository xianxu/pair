---
id: 000438
status: codecomplete
deps: []
github_issue:
created: 2026-10-10
updated: 2026-10-11
estimate_hours:
card_mirror: '6addc0ca7a7502a7e4e551946e94883f92d560ca' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-11T01:01:02-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "ab1c1655", done: "3100e153"}
actual_hours: 0.16
---

# Peer delivery to a busy agent: confirm a message the agent queued mid-turn, and report the real blocking condition

## Problem

On 10-10 (23:44–23:48) the ops TL sent three messages to parley.nvim:1 while it was four minutes into a turn with its
spinner streaming (ops learnings kink 63):
- 034cf323 **expired before its paste** (90 s paste-by) and never arrived.
- de593285 **was delivered** (Claude Code queued it behind the running turn) but its receipt read `indeterminate`:
  no new turn opened within the 30 s confirmation window, so the wrapper had no evidence of delivery.
- The TL's resend, 42e4e03e, was then a **duplicate** the worker saw twice, and it spent another unit of the
  worker's 8-message allowance.

All three receipts said "waiting for rendered output". In `dispatchPeer` (`wrapcmd/peer_delivery.go`) the checks
run in order and the receipt keeps the last one that failed. `outputPending > 0` comes before the composer checks,
and a streaming agent nearly always has output in flight, so the detail hides what actually blocked the paste.

## Spec

- **Confirm a queued message.** When the agent accepts input mid-turn and shows it as queued (Claude Code: the
  queued-message line above the composer, "Press up to edit queued messages"), that is delivery evidence: the
  receipt reads `submitted` (or a distinct `queued-by-agent`), not `indeterminate`. Agent-specific recognition
  lives with the existing composer recognizers; agents without it keep today's behavior.
- **Report the real blocker.** The receipt's detail names the condition that held the delivery for most of the
  window (or every condition seen), not the last check evaluated. A streaming agent's transient output-in-flight
  doesn't mask a composer that isn't ready.
- **Busy-agent policy.** Decide in the plan between pasting mid-turn (today) and holding until the agent's turn
  ends (`LastActivity` quiet for `QuietInterval`), with the deadline counted per pair#435's rule. Whichever is
  chosen, a delivery to an agent that streams for minutes must not expire silently.
- Overlap: pair#435 (pair:3) changes how holds spend the deadline; pair#433 (pair:2) the allowance. Coordinate
  through the TL.

## Done when

- [ ] A message queued by a busy agent reads `submitted` (or `queued-by-agent`), never `indeterminate` (wrapper test with a recorded Claude Code queued-message screen).
- [ ] With continuous output and a not-ready composer, the receipt names the composer condition (test).
- [ ] A delivery to an agent streaming for 3+ minutes either lands or reports a hold, per the chosen policy (test).

## Plan

Measured cause (Log 2026-10-11): the `outputPending > 0` gate starves a streaming agent. Fix the gate, keep
the agent-agnostic confirm, and make the expiry detail name the dominant wait.

- [x] Frame-complete gate: the terminal model's control observer tracks synchronized output (DECSET 2026);
  `FrameComplete()` is true once the stream has closed a synchronized update and is not inside one. In
  `dispatchPeer`, output in flight waits only when the model is not at a complete frame. Streams that never
  frame keep today's rule. (ARCH: one observer owns terminal-control facts; no agent-specific code.)
- [x] Busy-agent policy: **paste mid-turn** (today's rule, now reachable). Claude Code and Codex queue
  mid-turn input natively (Claude Code measured; Codex's captured screens frame, its queueing unmeasured); the recorded queued screen clears the composer, which is the existing confirm
  evidence (occupied after the paste, then empty) — so a queued message reads `submitted` with no new
  recognizer. Holding to turn end would spend minutes of a 90 s paste-by on a long turn.
- [x] Wait accounting: each dispatch charges the time since the previous one to the detail that stood
  during it, per delivery half (reset at enqueue and at the paste). On expiry, the receipt detail lists the
  conditions longest first with their seconds; the `held past` prefix stays. In memory, bounded by the fixed
  detail set, dies with the delivery (ARCH-FUNERAL: nothing durable).
- [x] Fixtures: three screens re-rendered from the recorded parley.nvim:1 stream (2026-10-10 23:45–23:46,
  Claude Code 2.1.29x): mid-turn empty composer, the paste mid-turn, the queued message.
- [x] Tests: queued screen reads Empty; framed continuous output with a chunk always in flight delivers and
  confirms `submitted` from the queued screen; 3+ min of streaming lands; unframed starvation expires naming
  `rendered output`; a not-ready composer under continuous output is named on expiry; observer unit test.

## Log

### 2026-10-10

### 2026-10-11
- 2026-10-11: closed — Review fixes only: RIS clears frame flags (TestTerminalModelFrameComplete gains the ESC c step); ranked waits join with " | " (TestPeerExpiryNamesTheLongestWait updated); doc/atlas scope Codex as unmeasured. go test ./cmd/internal/wrapcmd green in env -i after merging origin/main. (fixed-then-ship); review verdict: SHIP
- 2026-10-11: closed — Root cause measured from the recorded parley.nvim:1 stream (Log 2026-10-11): output-in-flight gate starved a streaming agent; Claude Code frames every repaint with DECSET 2026 and 1111/1153 chunk ends fall between frames with clean composer reads. New tests (peer_busy_test.go) on screens re-rendered from that recording: queued message -> submitted; 3.5 min of framed streaming lands, pasted once; unframed starvation expires naming rendered output; not-ready composer named longest-first on expiry; FrameComplete unit test. Mutation: reverting the gate fails 3 tests, reverting the summary fails 2. Full suite: make -k test fails only test-pair-embedded-runtime (identical on origin/main; session env leak) + test-changelog (passes with scratchpad TMPDIR); go test ./... wrapcmd green; launcher/couchcmd/couchcore failures pass under env -i (session env leak); artifactpath/gcruntime pre-existing, artifactpath wrapcmd entries identical to merge base; -race shows only a pre-existing race in codex_working_test.go vt helper.; review verdict: SHIP

- Replayed `scrollback-1-parley-nvim-25-claude.raw` (pair repo dir 475fc7ba…) through the composer
  recognizer at the recorded chunk boundaries (wrap-events master-chunk offsets). Claude Code brackets every
  frame in `ESC[?2026h … ESC[?2026l`; 1111 of 1153 chunk ends in 23:44–23:47 land outside a synchronized
  update, and at every such boundary the composer read is clean: empty through the turn, occupied from the
  paste (23:45:45) to 23:46:03, empty after with "Press up to edit queued messages" (faint, reads Empty).
- So 034cf323 would have pasted at any framed boundary; de593285's submit was held behind output in flight
  (receipt: "uncertain: delivery deadline elapsed after the submit: waiting for rendered output"). The
  resend 42e4e03e at 23:48:23 cleared in about 1 s: Claude Code queues mid-turn input at once.
- The wrapper's handleChunk ran about 150 ms per chunk with tracing on, so a new chunk is nearly always read
  before the previous one is forwarded.
