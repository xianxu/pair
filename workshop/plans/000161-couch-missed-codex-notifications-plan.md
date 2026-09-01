# Couch Notification Lifecycle Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reliably surface one Couch notification when Claude or Codex stops working, including short Codex turns with no stable visual completion marker.

**Architecture:** Keep `emitOuter` as the single delivery boundary and add a pure per-turn reducer ahead of it. Extend Pair's existing session watcher/inventory authority to follow the validated Codex root transcript and publish launch-scoped observations to `pair-wrap`; native OSC, Claude progress/marker state, rendered Codex activity, and an activity-gated watchdog remain deduplicated secondary signals.

**Tech Stack:** Go, PTY/ANSI parsing, incremental JSONL, launch-scoped sidecars, fake clock/runtime tests, Couch terminal integration tests.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `TurnObservation` | `cmd/internal/wrapcmd/notification_lifecycle.go` | new |
| `NotificationLifecycle` | `cmd/internal/wrapcmd/notification_lifecycle.go` | new |
| `LifecycleDecision` | `cmd/internal/wrapcmd/notification_lifecycle.go` | new |
| `CodexWorkingRecognizer` | `cmd/internal/wrapcmd/codex_working.go` | new |
| `LifecycleRecord` | `cmd/internal/sessionwatch/lifecycle_event.go` | new |
| Claude marker grammar | `cmd/internal/wrapcmd/wrap.go` | modified |
| Codex native-event grammar | `cmd/internal/sessioninventory/event.go` | modified |

- **`TurnObservation`** normalizes native OSC, markers, progress OSC,
  transcript events, rendered state, and watchdog expiry. N observations reduce
  into one turn generation; transcript observations may carry one native
  `turn_id`.
- **`NotificationLifecycle`** is immutable reducer state: observed activity,
  generation/turn identity, notification state, and richer-message grace.
  This is semantic dedup; `emitOuter`'s 500 ms guard remains transport throttling
  (`ARCH-PURE`, `ARCH-DRY`).
- **`LifecycleDecision`** contains notify/message and timer actions, with no IO
  or clock reads.
- **`CodexWorkingRecognizer`** recognizes only the live rendered
  `• Working (… esc to interrupt)` status. `Worked for…`, prose, and quoted
  copies have no authority.
- **`LifecycleRecord`** is versioned launch-scoped JSONL containing agent,
  launch ordinal, source/outcome, optional turn ID/message, transcript path, and
  absolute record position and native event timestamp. Its stable identity is
  `(launch ordinal, authorized artifact generation, transcript record offset)`;
  journal byte position is transport state, never semantic identity.
- **Claude marker grammar** accepts one or more Unicode General Category `L`
  code points in the verb, with no normalization.
- **Codex native-event grammar** distinguishes successful completion from abort
  and fatal error. It also exposes captured `task_started` plus its `turn_id` so
  the reducer keys a generation before completion. Add `turn_complete` only
  after capturing its real envelope.

### Lifecycle transition contract

| Current state | Observation | Result |
|---|---|---|
| idle | user submission, rendered/progress working, or transcript `task_started(turnID)` | open generation; attach `turnID` when known |
| active, unkeyed | `task_started(turnID)` | attach key to the same generation |
| active | native/marker/visual completion | notify or enter richer-message grace; retain generation as completed tombstone |
| active | transcript terminal for its attached `turnID` | notify once; close as keyed tombstone |
| completed tombstone | same keyed terminal or unkeyed source before next generation | suppress as duplicate; a richer message may replace a pending generic during grace |
| any | next user submission or a different `task_started(turnID)` | close prior correlation window and open a new generation |
| active | abort/error terminal | notify attention with distinct outcome, once; do not label successful completion |
| idle | stop/watchdog without prior activity/submission | ignore |

User submission is the hard boundary that makes two rapid keyless turns
distinct. Transcript `task_started` is the hard keyed boundary and is published
even if the TUI Working bar never persists. A delayed keyed terminal is matched
to its keyed generation/tombstone, not to whichever generation is current;
unknown old keys are ignored after their native timestamp precedes the current
generation boundary. Reducer tests enumerate native->transcript,
transcript->native, visual->transcript, marker->progress-stop, two keyless rapid
turns separated by submission, and a delayed duplicate after new activity.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `emitOuter` | `cmd/internal/wrapcmd/wrap.go` | modified | outer TTY OSC 777 + slug refresh |
| `NotificationRewriter` | `cmd/internal/wrapcmd/notification_rewriter.go` | modified | OSC 9/99/777 and 9;4 stream |
| `AuthorizedTranscriptFollower` | `cmd/internal/sessionwatch/run.go` | modified | validated Codex rollout + child lifetime |
| `LifecycleJournal` | `cmd/internal/sessionwatch/lifecycle_event.go` | new | append-only launch event sidecar |
| `LifecycleJournalTailer` | `cmd/internal/wrapcmd/lifecycle_journal.go` | new | incremental event delivery to proxy loop |
| `terminalModel` | `cmd/internal/wrapcmd/terminal_model.go` | modified | rendered Codex screen |
| Couch projection | `cmd/internal/couchtty/console.go` | verified | canonical OSC to unread/status/switcher |

- **`NotificationRewriter`** must preserve progress bytes while emitting typed
  observations. Existing every-split stream tests are the stateful fake.
- **`AuthorizedTranscriptFollower`** reuses the existing `sessionwatch.Runtime`
  fake and scanner-authorized root selection. No second `lsof`, newest-file, or
  cwd resolver is permitted (`ARCH-DRY`, `ARCH-MOCK`).
- **Journal/tailer** bridge the detached watcher to the wrapper. The current
  launch ordinal owns records; malformed, oversized, partial, truncated, or
  stale input fails closed. The tailer feeds a proxy channel so notification
  state remains single-goroutine-owned.
- **Journal commit protocol** mirrors `sessionledger`: one watcher owns the
  advisory lock; a newline is the record commit marker; writes loop to full
  length and sync before reporting committed. Open/write/sync/close/unlock
  failures distinguish non-authoritative, indeterminate, and committed outcomes.
  On indeterminate append, rescan for the stable composite identity before retry.
  The tailer opens before child spawn at the prior EOF, accepts only the current
  launch, and never replays earlier launches. Partial final lines wait for commit;
  replacement/truncation fails closed and stops the tailer with a diagnostic.
- **Couch projection** receives no new policy. A production-path test proves the
  emitted OSC updates both existing surfaces.

### Operating envelope (`ARCH-CONSTRAINTS`)

- Online terminal path: native/marker delivery stays immediate. After binding,
  transcript polling is 100 ms and journal tailing is 50 ms, budgeting <=150 ms
  scheduling latency and <250 ms end-to-end local detection. Basis: existing
  watcher fast poll and wrapper 50 ms capture tick; enforce with fake-clock
  boundary tests and record one live-fixture measurement.
- One incremental root follower per Pair session; never reread the whole
  transcript. Expected scale is <=100 concurrent local Pair sessions and <=10
  completion records/minute/session (domain-informed upper bound). Reuse existing
  JSONL pending-size limits; journal records are bounded to 64 KiB.
- Multiple Codex sessions may share cwd/storage. Only current launch ordinal plus
  validated root binding publishes.
- Screen recognition runs only after a terminal chunk over the bounded screen.
- Invalid, ambiguous, replaced, or stale input produces diagnostics and no
  notification. Observation must never block PTY forwarding.
- One proxy-loop event channel has capacity 32. Journal delivery runs in its own
  goroutine and may pause at the durable journal cursor under backpressure; PTY
  forwarding never waits on it. Activity observations may coalesce to latest
  state, but terminal records are never dropped. Sustained overflow logs once per
  generation and recovers from the journal.
- Network/GPU: N/A.

## Chunk 1: Claude marker correctness (M1)

### Task 1: Pin and correct the Unicode marker grammar

**Files:**
- Modify: `cmd/internal/wrapcmd/wrap.go`
- Create: `cmd/internal/wrapcmd/notification_marker_test.go`
- Modify: `cmd/internal/wrapcmd/update_agent_output_test.go`

- [ ] Write table tests using `endOfTurnByAgent["claude"]`: accept `Baked` and
  precomposed `Sautéed`; reject empty, combining-mark, digit, punctuation, and
  whitespace-containing verbs.
- [ ] Run `go test ./cmd/internal/wrapcmd -run TestClaudeEndOfTurnGrammar -count=1`;
  expect the `Sautéed` case to fail.
- [ ] Change only `[A-Za-z]+` to `\p{L}+`; retain the anchored star, whitespace,
  duration, and prefix-only trailing text.
- [ ] Feed a finalized colored SGR span containing
  `✻ Sautéed for 34s · done 1:39 PM` through `updateAgentOutput` /
  `finalizeSpan` and a temp outer-TTY seam. Assert one canonical notification;
  duplicates and quoted/uncolored variants add none.
- [ ] Run `go test ./cmd/internal/wrapcmd -run 'TestClaudeEndOfTurnGrammar|Test.*Saut' -count=1`; expect PASS.
- [ ] Commit `couch: #161 M1 accept Unicode Claude marker verbs`, tick M1,
  append evidence, update atlas only if its grammar is stale, and run
  `sdlc milestone-close --issue 161 --milestone M1`.

## Chunk 2: Unified lifecycle policy (M2)

### Task 2: Build the pure reducer and semantic dedup

**Files:**
- Create: `cmd/internal/wrapcmd/notification_lifecycle.go`
- Create: `cmd/internal/wrapcmd/notification_lifecycle_test.go`
- Modify: `cmd/internal/wrapcmd/wrap.go`

- [ ] Write reducer tests for native completion; activity then completion;
  duplicate sources more than 500 ms apart; distinct transcript turn IDs inside
  500 ms; stop without activity; activity-gated watchdog; user input clearing a
  stale fallback; richer-message grace; abort/error attention outcomes. Include
  every ordering in the lifecycle transition contract, including two keyless
  turns separated by submission and a delayed old keyed event after new work.
- [ ] Run `go test ./cmd/internal/wrapcmd -run TestNotificationLifecycle -count=1`;
  expect compile failure for missing lifecycle types.
- [ ] Implement pure
  `Reduce(state, observation, now) (state, decision)`. Prefer transcript turn ID;
  otherwise use activity generation. Decisions request notify or timer actions;
  the reducer performs no IO and reads no clock.
- [ ] Add one proxy-owned observation path. Route existing native OSC and marker
  sources through it; only reducer decisions call `emitOuter`. Use `p.now` and
  one owned resettable timer in the proxy `select`, with all lifecycle state on
  that goroutine. Every arm returns a monotonically increasing timer token plus
  generation; a firing is ignored unless both still match. Stop, drain, reset,
  child-exit, and stale-fire/rearm behavior get deterministic loop tests.
- [ ] Detect the existing agent-facing submit action in the stdin translation
  path and enqueue `submitted`; ordinary typing does not open generations.
- [ ] Run focused lifecycle/rewriter/marker tests; expect one same-turn
  notification across source permutations and two distinct rapid turns.

### Task 3: Turn Claude progress OSC into activity observations

**Files:**
- Modify: `cmd/internal/wrapcmd/notification_rewriter.go`
- Modify: `cmd/internal/wrapcmd/notification_rewriter_test.go`
- Modify: `cmd/internal/wrapcmd/osc_test.go`
- Modify: `cmd/internal/wrapcmd/wrap.go`

- [ ] Add every-split tests proving OSC `9;4;3` and `9;4;0` pass through
  byte-for-byte while producing typed working/stopped observations. Pin startup
  `9;4;0` without prior working as silent; retain malformed/overlong behavior.
- [ ] Extend the rewriter result with progress observations and map Claude
  `3 -> working`, `0 -> stopped`; other states remain non-authoritative.
- [ ] On `3 -> 0`, briefly await a richer marker/native message before generic
  notify. Never infer completion from ten seconds without progress. Arm 60-second
  recovery only after activity and emit once per generation.
- [ ] Run `go test ./cmd/internal/wrapcmd -count=1`; expect PASS.
- [ ] Commit `couch: #161 M2 unify notification lifecycle signals`, document
  precedence/grace/timer ownership in the atlas, tick/log M2, and run
  `sdlc milestone-close --issue 161 --milestone M2`.

## Chunk 3: Codex transcript authority (M3)

### Task 4: Add the launch-scoped lifecycle journal

**Files:**
- Create: `cmd/internal/sessionwatch/lifecycle_event.go`
- Create: `cmd/internal/sessionwatch/lifecycle_event_test.go`
- Modify: `cmd/internal/artifactpath/paths.go`
- Modify: `cmd/internal/artifactpath/paths_test.go`
- Modify: `cmd/internal/launcher/scoped_paths.go`
- Modify: `cmd/internal/launcher/scoped_paths_test.go`
- Create: `cmd/internal/wrapcmd/lifecycle_journal.go`
- Create: `cmd/internal/wrapcmd/lifecycle_journal_test.go`

- [ ] Write codec/tailer tests for the stable composite identity, partial trailing
  line, malformed/oversized record, replayed transcript identity, truncation,
  replacement, stale launch, and tailer restart at prior EOF. Use portable temp
  folders.
- [ ] Add one canonical path through `artifactpath`; make launcher/watcher/wrapper
  derive from it. Update manifest, cleanup, and path-family tests.
- [ ] Mirror `sessionledger` append outcomes and lock protocol. Test open, lock,
  partial/full write, sync, close, unlock, indeterminate rescan/retry, and exact
  committed replay reconciliation. Newline is commit authority.
- [ ] Implement bounded incremental tailing from the pre-spawn EOF. Validate the
  composite identity/current launch, stop on replacement/truncation, and deliver
  terminal records through a capacity-32 channel without blocking PTY IO.
- [ ] Run `go test ./cmd/internal/sessionwatch ./cmd/internal/wrapcmd ./cmd/internal/launcher -run 'Lifecycle|ScopedPaths' -count=1`; expect PASS.

### Task 5: Continue following the authorized Codex root

**Files:**
- Modify: `cmd/internal/sessionwatch/run.go`
- Modify: `cmd/internal/sessionwatch/run_test.go`
- Modify: `cmd/internal/sessionwatch/lifecycle.go`
- Modify: `cmd/internal/sessioninventory/event.go`
- Modify: `cmd/internal/sessioninventory/event_test.go`
- Modify: `cmd/internal/sessioninventory/jsonl_incremental.go` only if required to expose existing absolute offsets
- Modify: `cmd/internal/wrapcmd/wrap.go`

- [ ] Add stateful fake-runtime cases: fresh binding whose causal round ends in
  `task_complete`; later appended turns; short response; concurrent sibling/root
  transcripts; rejected subagent; stale launch; partial record; truncate/replace;
  child exit and cleanup. Include pre-launch terminal records, the unique binding
  round's `task_started` plus terminal, and a later turn; only the validated
  binding-round pair and later turn may publish. The short-turn regression binds
  after both causal records already exist, with no native OSC/rendered activity,
  and must notify exactly once.
- [ ] Preserve the captured `event_msg.payload.type=task_complete` schema. Add
  `turn_complete` only from a real sanitized fixture. Keep success, aborted, and
  fatal error outcomes distinct. Capture/publish `task_started(turn_id)` as the
  keyed generation opener.
- [ ] Factor existing authorized target advancement so the watcher persists the
  binding plus an authorization watermark derived from the unique causal round:
  authorized artifact identity/generation and the parser-complete opener/terminal
  offsets beyond the launch artifact boundary. Persist the binding first; then
  publish the validated causal `task_started(turn_id)` followed by its matching
  terminal, even though both are now behind the follow watermark. Follow only
  records beyond the terminal watermark thereafter. Never publish unrelated
  earlier openers/terminals. Reuse process incarnation,
  descendant-FD corroboration, inventory bounds, and ledger guard.
- [ ] Start/stop the wrapper tailer with the child and validate
  `PAIR_LAUNCH_ORDINAL`. Prove native OSC plus transcript deduplicates, while two
  transcript turn IDs notify twice.
- [ ] Use 100 ms post-binding transcript polls and the wrapper's 50 ms journal
  poll. Fake-clock tests enforce <=150 ms scheduling latency; record one live
  fixture measurement under the <250 ms end-to-end budget.
- [ ] Run `go test ./cmd/internal/sessioninventory ./cmd/internal/sessionwatch ./cmd/internal/wrapcmd ./cmd/internal/launcher -count=1`; expect PASS without leaks/stale publication.
- [ ] Commit `couch: #161 M3 follow authorized Codex turn completion`, update
  `atlas/session-identity.md` and architecture, tick/log M3, and run
  `sdlc milestone-close --issue 161 --milestone M3`.

## Chunk 4: Visual recovery and Couch conformance (M4)

### Task 6: Recognize rendered Codex activity

**Files:**
- Create: `cmd/internal/wrapcmd/codex_working.go`
- Create: `cmd/internal/wrapcmd/codex_working_test.go`
- Modify: `cmd/internal/wrapcmd/wrap.go`
- Create: `cmd/internal/wrapcmd/testdata/tty/codex/<captured-version>/working.raw`
- Modify: `cmd/internal/wrapcmd/harness_tty_fixture_test.go`

- [ ] Capture/sanitize a real animated Working-to-final-output PTY fixture and
  record Codex version/terminal size; do not synthesize guessed ANSI.
- [ ] Feed it through `terminalModel` and assert absent -> present -> absent.
  Reject quoted `> • Working…`, prose, and `─ Worked for…`.
- [ ] After each terminal feed, compare pure recognized presence and enqueue only
  transitions. Transcript/native authority deduplicates visual stop.
- [ ] With fake time prove prompt-only idleness is silent, observed activity can
  recover once by disappearance/watchdog, and new activity/input cancels stale
  timers.

### Task 7: Prove Couch status and switcher behavior

**Files:**
- Modify: `cmd/internal/couchtty/notification_test.go`
- Modify: `cmd/internal/wrapcmd/harness_tty_integration_test.go` or add a focused cross-package test at the existing accessible seam
- Modify: `atlas/architecture.md`

- [ ] Pass wrapper-emitted OSC 777 through the real Couch console/screen path.
  Assert the inactive thread becomes unread and the same completion appears in
  status and switcher projections.
- [ ] Run focused package tests, then `go test ./... -count=1` and
  `git diff --check`; expect PASS.
- [ ] Commit `couch: #161 M4 recover and display stopped agent work`, tick/log
  M4 with fixture versions, update atlas, and run
  `sdlc milestone-close --issue 161 --milestone M4`.
- [ ] Re-run `go test ./... -count=1 && git diff --check` from a clean worktree.
- [ ] Close with `sdlc close --issue 161 --verified '<focused lifecycle, watcher concurrency, real TTY fixture, Couch projection, and full Go test evidence>'`; omit guessed actuals so the gate measures them.
