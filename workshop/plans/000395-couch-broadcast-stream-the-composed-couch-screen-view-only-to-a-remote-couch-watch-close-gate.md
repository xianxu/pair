---
gate: boundary-review
issue: 395
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-07T15:36:37-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: Hub.Activate arms grace even when the last accepted frame shows LIVE, so an idle screen ends the broadcast after 1s
          detail: 'hub.go Activate sets missing=true whenever !missing, but before Activate the missing flag is never set from frame content. Reproduced with an overlay test: offer(live), Activate(), fire grace leads to ErrIndicatorHidden. Track a shown bool from every accept (or collapse active/missing/graceC into an explicit inactive|shown|hidden state, ARCH-ORDER) and arm only when not shown; add a regression test and randomize the Activate position in TestHubRandomInterleavings.'
          family: hub-state-flag-constellation
          round: 1
        - id: BR-2
          severity: Minor
          title: Plan promises a final End message to subscribers; hub only closes channels
          detail: M2 server must read Hub.Err() and distinguish a hub end from Subscription.Close. Implement the End message or add a Revisions entry.
          family: plan-code-drift
          round: 1
        - id: BR-3
          severity: Minor
          title: TestTapNotCalledOnFailedPaint omits the refused PresentView transition case the plan lists
          family: plan-test-coverage-gap
          round: 1
        - id: BR-4
          severity: Minor
          title: Hub resync ticker runs for the hub lifetime even with no resyncing subscriber
          family: idle-background-work
          round: 1
      boundary: M1
      recipe: milestone-review
      blocked: true
    - "n": 2
      timestamp: "2026-10-07T15:52:10-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: Tagged off/shown/hidden watch; overlay mutant (Activate ignores shown) turns the new subtest and TestHubRandomInterleavings red.
          round: 2
        - id: BR-2
          disposition: addressed
          note: Plan Revisions 2026-10-07 documents close-queues + Hub.Err() and the M2 server mapping; matches hub.go end().
          round: 2
        - id: BR-3
          disposition: addressed
          note: Plan Revisions entry records the refused-PresentView branch as defence in depth with no public trigger; consistent with presenter.go:380.
          round: 2
        - id: BR-4
          disposition: addressed
          note: run() selects tickC only while resyncing > 0; TestHubRealTickerResyncsQuietScreen exercises the gate and checks resyncing returns to 0.
          round: 2
      boundary: M1
      recipe: milestone-review
      blocked: false
    - "n": 3
      timestamp: "2026-10-07T19:24:50-07:00"
      agent: claude
      findings:
        - id: BR-5
          severity: Important
          title: SSE end reason read via Hub.Err() before hub.done closes; viewers can be told the wrong reason (reproduced flake)
          detail: 'server.go:131-133 reacts to the closed sub channel, which Hub.end closes inside the loop before run() closes h.done, so Hub.Err() can return nil and the reason becomes "the operator stopped broadcasting". go test -count=300 -cpu=1,2,8 failed TestSessionTunnelExitEndsSession once with exactly that. Fix: wait on <-Hub.Done() before Err(), or carry the reason in a final Message. Rule: a consumer reacting to a closed channel reads ending state through an edge ordered after its write.'
          family: cross-channel-state-read
          round: 3
        - id: BR-6
          severity: Important
          title: viewer.js shows nothing when EventSource closes for good, leaving a frozen screen that looks live
          detail: A non-200 response (Cloudflare 502/530, 503 at the viewer cap, 410/404 after the end) closes EventSource without a reconnect. onerror skips the CLOSED state, so a watching viewer keeps a stale screen with no status, and a viewer turned away at the cap sees "Connecting…" forever. In the CLOSED branch, reset or dim the screen and show "Disconnected", optionally retrying with backoff. Add a node test with a fake EventSource.
          family: viewer-liveness-visible
          round: 3
        - id: BR-7
          severity: Minor
          title: Node fit test runs through Go's TestViewerFit, not the planned Makefile target, and no Revision records it
          detail: '2nd finding in this family. Rule: when the implementation departs from a plan step, record it in Revisions in the same commit. Also unrecorded: the hub ends viewers by closing the channel, not with Message End as Task 1.3 said.'
          family: plan-code-drift
          round: 3
        - id: BR-8
          severity: Minor
          title: session.go drops the error from go srv.Serve(l), so a server that dies unexpectedly leaves the session reporting live
          family: silent-error-swallow
          round: 3
      boundary: M2
      recipe: milestone-review
      blocked: true
    - "n": 4
      timestamp: "2026-10-07T19:33:25-07:00"
      agent: claude
      dispose:
        - id: BR-5
          disposition: addressed
          note: Reason stored atomically before queues close (hub.go:365); TestHubReasonVisibleWhenQueueCloses fails at iteration 0 against an Err-gated-on-done overlay mutant; 200x -cpu=1,2,8 session/server stress green.
          round: 4
        - id: BR-6
          disposition: addressed
          note: connect(deps) state machine dims + shows Disconnected/retry on CLOSED (viewer.js:60-78); connect.test.mjs drives CLOSED, cap-refusal, backoff exhaustion via a fake EventSource, run by TestViewerNode.
          round: 4
        - id: BR-7
          disposition: addressed
          note: Plan Revisions 2026-10-07 records the Go-driven node tests; the End-message-to-channel-close departure is recorded at plan line 777.
          round: 4
        - id: BR-8
          disposition: addressed
          note: session.watch selects on served and stops with ErrServerFailed; TestSessionEndsWhenServerFails (BreakListener) passes only through that case.
          round: 4
      findings:
        - id: BR-9
          severity: Important
          title: endReason forwards the raw wrapped Serve error (local addr/socket path) to remote viewers
          detail: session.go:141 wraps the OS error text into ErrServerFailed, and server.go:162 sends err.Error() in the SSE end event, exposing 127.0.0.1:port today and a unix-socket path (username) with cloudflared in M4. Map sentinels to fixed strings via errors.Is with a generic fallback, and test that a wrapped error yields the fixed text (ARCH-SECURE).
          family: viewer-text-closed-set
          round: 4
        - id: BR-10
          severity: Minor
          title: atlas/broadcast.md:69 still cites TestViewerFit after the rename to TestViewerNode
          detail: '3rd finding in plan-code-drift. Rule: a rename or departure greps the old identifier across atlas, plan and lessons in the same commit; extend the lessons.md Revisions bullet to cover atlas references.'
          family: plan-code-drift
          round: 4
      boundary: M2
      recipe: milestone-review
      blocked: true
    - "n": 5
      timestamp: "2026-10-07T19:35:47-07:00"
      agent: claude
      dispose:
        - id: BR-9
          disposition: addressed
          note: server.go:156-181 closed endReasons table via errors.Is plus generic fallback; TestEndReasonIsAClosedVocabulary feeds wrapped secret-bearing errors and would fail under the old TrimPrefix(err.Error()) path; other viewer-facing writes (http.Error at server.go:60,79,112,115) are fixed literals.
          round: 5
        - id: BR-10
          disposition: not-addressed
          note: atlas/broadcast.md:69 still reads "node-tested via TestViewerFit" at c4c000ad; lessons.md Revisions bullet was not extended to cover renames/atlas references (4th plan-code-drift instance, fix the rule not the site).
          round: 5
      boundary: M2
      recipe: milestone-review
      blocked: false
    - "n": 6
      timestamp: "2026-10-07T21:02:31-07:00"
      agent: claude
      findings:
        - id: BR-11
          severity: Critical
          title: startBroadcast reads the captured adopted flag after runTerminalCommand may have returned early; reproduced -race failure
          detail: 'console_broadcast.go:94-102. runTerminalCommand (terminal.go:37-44) returns on ctx/stop while the queued closure may still run on the loop, so the goroutine reads adopted at :100 unsynchronised with the write at :97 (go test -race -count=8 -run TestBroadcast failed TestBroadcastStoppedOnShutdown). Logically the goroutine can stop a session the loop then adopts. 2nd in family. Rule: a caller that can return early must not read state written by work it handed to another goroutine; a single claim (atomic CAS) owns disposal of produced resources. Sweep done: only this call site captures a result.'
          family: cross-channel-state-read
          round: 6
        - id: BR-12
          severity: Important
          title: Shutdown's 6s broadcast wait is defeated by workers.Wait on GoTracked waiters blocked on s.Done()
          detail: The Done watcher (console_broadcast.go:146) and the late-start stop (:100) wait unbounded; teardown's c.workers.Wait (console.go:970) joins them after endBroadcastForShutdown's 6s, so a slow or stuck tunnel Close hangs Couch exit. Select on c.stop as well; add a shutdown test with FakeTunnel CloseBlock asserting a bounded return.
          family: declared-bound-unenforced
          round: 6
        - id: BR-13
          severity: Minor
          title: TestBroadcastOffHasNoClickTarget from Task 3.3 is absent with no Revision
          detail: '2nd in family. Rule: every test named in a plan step exists or a Revision explains its absence.'
          family: plan-test-coverage-gap
          round: 6
        - id: BR-14
          severity: Minor
          title: Start context is never cancelled after adoption or a SetTap failure
          detail: Each broadcast leaves a child context registered on c.lifetime until Couch exits; cancel it once the start completes, or document why it must outlive the session (M4 cloudflared).
          family: context-lifetime-leak
          round: 6
        - id: BR-15
          severity: Minor
          title: README describes a remote link, but M3 broadcasts are local-only, and the M3 local smoke is not recorded in the Log
          detail: '4th in plan-code-drift; rule: docs and the Log state the milestone''s actual delivered state. Add an M4 note to the README, or make sure M4 updates it; record the operator smoke in the Log or in --verified.'
          family: plan-code-drift
          round: 6
      boundary: M3
      recipe: milestone-review
      blocked: true
    - "n": 7
      timestamp: "2026-10-07T21:08:38-07:00"
      agent: claude
      dispose:
        - id: BR-11
          disposition: addressed
          note: startClaim CAS (console_broadcast.go:47-61,124-131,143) gives one owner; go test -race -count=8 -run TestBroadcast|TestStartClaim passes.
          round: 7
        - id: BR-12
          disposition: addressed
          note: awaitDown and the Done watcher select on c.stop, which teardown closes before workers.Wait; TestBroadcastShutdownBoundedByStuckTunnel pins it.
          round: 7
        - id: BR-13
          disposition: addressed
          note: TestBroadcastOffHasNoClickTarget added (console_broadcast_test.go), checking phase, session and the statusControl span.
          round: 7
        - id: BR-14
          disposition: addressed
          note: cancel() runs right after broadcast.Start returns; the Tunnel contract now says a tunnel must outlive ctx.
          round: 7
        - id: BR-15
          disposition: addressed
          note: README says links are local-only until M5; the issue Log records the operator's M3 local smoke at 0dbe0802.
          round: 7
      findings:
        - id: BR-16
          severity: Minor
          title: Tunnel.Open doc says Start cancels its context; the Console caller does
          detail: '5th in family. Rule: a contract comment names the actor that actually performs the action, checked against the code at that point. broadcast.Start (session.go:51) never cancels ctx; only Console.startBroadcast does, so say "callers may cancel ctx once Start returns".'
          family: plan-code-drift
          round: 7
        - id: BR-17
          severity: Minor
          title: No deterministic test drives the late-closure-after-abandon ordering behind BR-11
          detail: '3rd in family. Rule: every handoff that can return early has a test that injects the order (closure after the early return). TestStartClaimDecidesOnce only exercises atomic CAS; the integration ordering is covered only by -race sampling.'
          family: cross-channel-state-read
          round: 7
      boundary: M3
      recipe: milestone-review
      blocked: false
    - "n": 8
      timestamp: "2026-10-07T21:34:18-07:00"
      agent: claude
      findings:
        - id: BR-18
          severity: Minor
          title: Issue Plan M4 row still names COUCH_BROADCAST_FONT_FILE after the bundled-font revision
          detail: 'This is the 6th finding in family plan-code-drift. Rule: a plan Revision that replaces a mechanism also rewrites the issue''s Plan row naming it, in the same commit. The issue file at line 179 still promises the env-var font file.'
          family: plan-code-drift
          round: 8
        - id: BR-19
          severity: Minor
          title: 'parseOSC4Reply writes its own rgb: parser while OSC 10/11 use ansi.XParseColor'
          detail: ARCH-DRY. The strict parser is defensible, but short components round differently from XParseColor's shift, so background and ANSI colours can disagree by one step. broadcast.Hex also repeats toRGBA. Use one strict parser for both, or say why there are two.
          family: duplicate-color-parser
          round: 8
        - id: BR-20
          severity: Minor
          title: The input_test OSC 4 row pins only Reply, not the UnknownOscEvent with the full prefix that capturePalette needs
          detail: 'This is the 3rd finding in family plan-test-coverage-gap. Rule: when a feature consumes an event from an external decoder, at least one test feeds the real decoder''s output to the consumer instead of a hand-built event. Fix: set event uv.UnknownOscEvent(tc.raw) on that row (input_test.go:99).'
          family: plan-test-coverage-gap
          round: 8
      boundary: M4
      recipe: milestone-review
      blocked: false
---

# Gate ledger — pair#395 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-07T15:36:37-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `hub-state-flag-constellation` Hub.Activate arms grace even when the last accepted frame shows LIVE, so an idle screen ends the broadcast after 1s
  hub.go Activate sets missing=true whenever !missing, but before Activate the missing flag is never set from frame content. Reproduced with an overlay test: offer(live), Activate(), fire grace leads to ErrIndicatorHidden. Track a shown bool from every accept (or collapse active/missing/graceC into an explicit inactive|shown|hidden state, ARCH-ORDER) and arm only when not shown; add a regression test and randomize the Activate position in TestHubRandomInterleavings.
- **BR-2** [Minor] `plan-code-drift` Plan promises a final End message to subscribers; hub only closes channels
  M2 server must read Hub.Err() and distinguish a hub end from Subscription.Close. Implement the End message or add a Revisions entry.
- **BR-3** [Minor] `plan-test-coverage-gap` TestTapNotCalledOnFailedPaint omits the refused PresentView transition case the plan lists
- **BR-4** [Minor] `idle-background-work` Hub resync ticker runs for the hub lifetime even with no resyncing subscriber

## Round 2 — 2026-10-07T15:52:10-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — Tagged off/shown/hidden watch; overlay mutant (Activate ignores shown) turns the new subtest and TestHubRandomInterleavings red.
- BR-2 — addressed — Plan Revisions 2026-10-07 documents close-queues + Hub.Err() and the M2 server mapping; matches hub.go end().
- BR-3 — addressed — Plan Revisions entry records the refused-PresentView branch as defence in depth with no public trigger; consistent with presenter.go:380.
- BR-4 — addressed — run() selects tickC only while resyncing > 0; TestHubRealTickerResyncsQuietScreen exercises the gate and checks resyncing returns to 0.

## Round 3 — 2026-10-07T19:24:50-07:00 (claude) — BLOCKED

### Raised

- **BR-5** [Important] `cross-channel-state-read` SSE end reason read via Hub.Err() before hub.done closes; viewers can be told the wrong reason (reproduced flake)
  server.go:131-133 reacts to the closed sub channel, which Hub.end closes inside the loop before run() closes h.done, so Hub.Err() can return nil and the reason becomes "the operator stopped broadcasting". go test -count=300 -cpu=1,2,8 failed TestSessionTunnelExitEndsSession once with exactly that. Fix: wait on <-Hub.Done() before Err(), or carry the reason in a final Message. Rule: a consumer reacting to a closed channel reads ending state through an edge ordered after its write.
- **BR-6** [Important] `viewer-liveness-visible` viewer.js shows nothing when EventSource closes for good, leaving a frozen screen that looks live
  A non-200 response (Cloudflare 502/530, 503 at the viewer cap, 410/404 after the end) closes EventSource without a reconnect. onerror skips the CLOSED state, so a watching viewer keeps a stale screen with no status, and a viewer turned away at the cap sees "Connecting…" forever. In the CLOSED branch, reset or dim the screen and show "Disconnected", optionally retrying with backoff. Add a node test with a fake EventSource.
- **BR-7** [Minor] `plan-code-drift` Node fit test runs through Go's TestViewerFit, not the planned Makefile target, and no Revision records it
  2nd finding in this family. Rule: when the implementation departs from a plan step, record it in Revisions in the same commit. Also unrecorded: the hub ends viewers by closing the channel, not with Message End as Task 1.3 said.
- **BR-8** [Minor] `silent-error-swallow` session.go drops the error from go srv.Serve(l), so a server that dies unexpectedly leaves the session reporting live

## Round 4 — 2026-10-07T19:33:25-07:00 (claude) — BLOCKED

### Disposed

- BR-5 — addressed — Reason stored atomically before queues close (hub.go:365); TestHubReasonVisibleWhenQueueCloses fails at iteration 0 against an Err-gated-on-done overlay mutant; 200x -cpu=1,2,8 session/server stress green.
- BR-6 — addressed — connect(deps) state machine dims + shows Disconnected/retry on CLOSED (viewer.js:60-78); connect.test.mjs drives CLOSED, cap-refusal, backoff exhaustion via a fake EventSource, run by TestViewerNode.
- BR-7 — addressed — Plan Revisions 2026-10-07 records the Go-driven node tests; the End-message-to-channel-close departure is recorded at plan line 777.
- BR-8 — addressed — session.watch selects on served and stops with ErrServerFailed; TestSessionEndsWhenServerFails (BreakListener) passes only through that case.

### Raised

- **BR-9** [Important] `viewer-text-closed-set` endReason forwards the raw wrapped Serve error (local addr/socket path) to remote viewers
  session.go:141 wraps the OS error text into ErrServerFailed, and server.go:162 sends err.Error() in the SSE end event, exposing 127.0.0.1:port today and a unix-socket path (username) with cloudflared in M4. Map sentinels to fixed strings via errors.Is with a generic fallback, and test that a wrapped error yields the fixed text (ARCH-SECURE).
- **BR-10** [Minor] `plan-code-drift` atlas/broadcast.md:69 still cites TestViewerFit after the rename to TestViewerNode
  3rd finding in plan-code-drift. Rule: a rename or departure greps the old identifier across atlas, plan and lessons in the same commit; extend the lessons.md Revisions bullet to cover atlas references.

## Round 5 — 2026-10-07T19:35:47-07:00 (claude) — passed

### Disposed

- BR-9 — addressed — server.go:156-181 closed endReasons table via errors.Is plus generic fallback; TestEndReasonIsAClosedVocabulary feeds wrapped secret-bearing errors and would fail under the old TrimPrefix(err.Error()) path; other viewer-facing writes (http.Error at server.go:60,79,112,115) are fixed literals.
- BR-10 — not-addressed — atlas/broadcast.md:69 still reads "node-tested via TestViewerFit" at c4c000ad; lessons.md Revisions bullet was not extended to cover renames/atlas references (4th plan-code-drift instance, fix the rule not the site).

## Round 6 — 2026-10-07T21:02:31-07:00 (claude) — BLOCKED

### Raised

- **BR-11** [Critical] `cross-channel-state-read` startBroadcast reads the captured adopted flag after runTerminalCommand may have returned early; reproduced -race failure
  console_broadcast.go:94-102. runTerminalCommand (terminal.go:37-44) returns on ctx/stop while the queued closure may still run on the loop, so the goroutine reads adopted at :100 unsynchronised with the write at :97 (go test -race -count=8 -run TestBroadcast failed TestBroadcastStoppedOnShutdown). Logically the goroutine can stop a session the loop then adopts. 2nd in family. Rule: a caller that can return early must not read state written by work it handed to another goroutine; a single claim (atomic CAS) owns disposal of produced resources. Sweep done: only this call site captures a result.
- **BR-12** [Important] `declared-bound-unenforced` Shutdown's 6s broadcast wait is defeated by workers.Wait on GoTracked waiters blocked on s.Done()
  The Done watcher (console_broadcast.go:146) and the late-start stop (:100) wait unbounded; teardown's c.workers.Wait (console.go:970) joins them after endBroadcastForShutdown's 6s, so a slow or stuck tunnel Close hangs Couch exit. Select on c.stop as well; add a shutdown test with FakeTunnel CloseBlock asserting a bounded return.
- **BR-13** [Minor] `plan-test-coverage-gap` TestBroadcastOffHasNoClickTarget from Task 3.3 is absent with no Revision
  2nd in family. Rule: every test named in a plan step exists or a Revision explains its absence.
- **BR-14** [Minor] `context-lifetime-leak` Start context is never cancelled after adoption or a SetTap failure
  Each broadcast leaves a child context registered on c.lifetime until Couch exits; cancel it once the start completes, or document why it must outlive the session (M4 cloudflared).
- **BR-15** [Minor] `plan-code-drift` README describes a remote link, but M3 broadcasts are local-only, and the M3 local smoke is not recorded in the Log
  4th in plan-code-drift; rule: docs and the Log state the milestone's actual delivered state. Add an M4 note to the README, or make sure M4 updates it; record the operator smoke in the Log or in --verified.

## Round 7 — 2026-10-07T21:08:38-07:00 (claude) — passed

### Disposed

- BR-11 — addressed — startClaim CAS (console_broadcast.go:47-61,124-131,143) gives one owner; go test -race -count=8 -run TestBroadcast|TestStartClaim passes.
- BR-12 — addressed — awaitDown and the Done watcher select on c.stop, which teardown closes before workers.Wait; TestBroadcastShutdownBoundedByStuckTunnel pins it.
- BR-13 — addressed — TestBroadcastOffHasNoClickTarget added (console_broadcast_test.go), checking phase, session and the statusControl span.
- BR-14 — addressed — cancel() runs right after broadcast.Start returns; the Tunnel contract now says a tunnel must outlive ctx.
- BR-15 — addressed — README says links are local-only until M5; the issue Log records the operator's M3 local smoke at 0dbe0802.

### Raised

- **BR-16** [Minor] `plan-code-drift` Tunnel.Open doc says Start cancels its context; the Console caller does
  5th in family. Rule: a contract comment names the actor that actually performs the action, checked against the code at that point. broadcast.Start (session.go:51) never cancels ctx; only Console.startBroadcast does, so say "callers may cancel ctx once Start returns".
- **BR-17** [Minor] `cross-channel-state-read` No deterministic test drives the late-closure-after-abandon ordering behind BR-11
  3rd in family. Rule: every handoff that can return early has a test that injects the order (closure after the early return). TestStartClaimDecidesOnce only exercises atomic CAS; the integration ordering is covered only by -race sampling.

## Round 8 — 2026-10-07T21:34:18-07:00 (claude) — passed

### Raised

- **BR-18** [Minor] `plan-code-drift` Issue Plan M4 row still names COUCH_BROADCAST_FONT_FILE after the bundled-font revision
  This is the 6th finding in family plan-code-drift. Rule: a plan Revision that replaces a mechanism also rewrites the issue's Plan row naming it, in the same commit. The issue file at line 179 still promises the env-var font file.
- **BR-19** [Minor] `duplicate-color-parser` parseOSC4Reply writes its own rgb: parser while OSC 10/11 use ansi.XParseColor
  ARCH-DRY. The strict parser is defensible, but short components round differently from XParseColor's shift, so background and ANSI colours can disagree by one step. broadcast.Hex also repeats toRGBA. Use one strict parser for both, or say why there are two.
- **BR-20** [Minor] `plan-test-coverage-gap` The input_test OSC 4 row pins only Reply, not the UnknownOscEvent with the full prefix that capturePalette needs
  This is the 3rd finding in family plan-test-coverage-gap. Rule: when a feature consumes an event from an external decoder, at least one test feeds the real decoder's output to the consumer instead of a hand-built event. Fix: set event uv.UnknownOscEvent(tc.raw) on that row (input_test.go:99).

## Open findings

- **BR-10** [Minor] `plan-code-drift` atlas/broadcast.md:69 still cites TestViewerFit after the rename to TestViewerNode
- **BR-16** [Minor] `plan-code-drift` Tunnel.Open doc says Start cancels its context; the Console caller does
- **BR-17** [Minor] `cross-channel-state-read` No deterministic test drives the late-closure-after-abandon ordering behind BR-11
- **BR-18** [Minor] `plan-code-drift` Issue Plan M4 row still names COUCH_BROADCAST_FONT_FILE after the bundled-font revision
- **BR-19** [Minor] `duplicate-color-parser` parseOSC4Reply writes its own rgb: parser while OSC 10/11 use ansi.XParseColor
- **BR-20** [Minor] `plan-test-coverage-gap` The input_test OSC 4 row pins only Reply, not the UnknownOscEvent with the full prefix that capturePalette needs
