---
gate: boundary-review
issue: 458
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T01:05:32-07:00"
      dispatched: "2026-10-12T01:04:28-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Minor
          title: withAttr copies the attribute map on every attr record, even for elements that aren't inputs (reduce.go:306)
          detail: Put an n.N == "input" guard before building the copy; attr is the most frequent mutation kind when replaying.
          family: hot-path-allocation
          round: 1
        - id: BR-2
          severity: Minor
          title: query.UserAction leaves out the new Hover kind with no recorded decision
          detail: 'Leaving Ptr out is correct. Decide whether hover belongs in a note''s action context, and write the decision down when #461 consumes it.'
          family: consumer-kind-classification
          round: 1
        - id: BR-3
          severity: Minor
          title: TestValidateDirReportsStrayFrameFiles discards the errors from Put, PutFrame, Close and WriteFile
          family: test-ignores-setup-errors
          round: 1
      boundary: M1
      recipe: milestone-review
      isolation: checkout
      reviewed: a005266576968cee74282ce83f5d768f166bd78d
      blocked: false
    - "n": 2
      timestamp: "2026-10-12T01:35:11-07:00"
      dispatched: "2026-10-12T01:28:51-07:00"
      agent: claude
      findings:
        - id: BR-4
          severity: Important
          title: 'Recorder errors, drops and overflow are invisible in production: Logf never wired, Stats unread, Close error dropped'
          detail: cast.go builds RecordOptions without Logf; chrome.go:553 drops the Close error when Logf is nil. A failed Output open or a sink write error leaves an unrecorded cast that looks recorded. Wire Logf to Couch's log and report Stats and the Close error at cast end.
          family: silent-error-swallowing
          round: 2
        - id: BR-5
          severity: Important
          title: Asset fetched while paused is stored but its asset record is dropped, and goAssets later skips it via store Lookup
          detail: recorder.go handle assetDone emits only if r.live; side.go queueAsset skips URLs already in the store. Emit asset records whatever the phase, or deduplicate against r.m.st.Assets instead of the store.
          family: asset-record-dedupe-source
          round: 2
        - id: BR-6
          severity: Important
          title: Queue overflow drops every CDP event (contexts, bindingCalled), not only DOM events; unknown frame then resolves against the root
          detail: Send bounds every Event. A lost executionContextCreated makes binding() pass frame "", and frameDoc("") returns the session root, so a report from an in-process iframe can be applied to the wrong node. Bound only DOM.* events, and treat an unknown frame as stale.
          family: overflow-drops-non-dom-events
          round: 2
        - id: BR-7
          severity: Important
          title: Frames, CSS sheets, truncated text, inline style, assets, screenshots/layout and detach have 0% unit coverage against fakes; PQ-3 sequence tests missing
          detail: 'Measured with go test -cover: FrameDoc, frameOwner, frameOffset, Drop, FullText, InlineStyle, ConstructedSheet, OwnedSheet, queueAsset, shotDone, captureLayout are at 0%. Missing tests: active page detach then seg tab, frame detach, desync then resync at recorder level, documentUpdated during a pending resync. Only the env-gated TestLiveRecording covers them.'
          family: fake-coverage-gap
          round: 2
        - id: BR-8
          severity: Important
          title: 'README is stale: casts are now recorded to <store>/webcast/recordings/<id>/ and viewers receive the DOM stream'
          detail: README.md:914-930 still says viewers see only live state, title and URL, and that the DOM is cast once the recorder lands. Document the recordings directory, its retention, and what viewers now receive (typed non-password values included).
          family: readme-user-surface
          round: 2
        - id: BR-9
          severity: Important
          title: bury/liveBackend scans every node per removed node once the 50k graveyard is full; same O(nodes) scan in FullText, frameDoc, frameOwner
          detail: 'Rule: per-event work in the mirror must be O(change), not O(document). Keep one backend-to-node index per msession, maintained in index and forget, and use it at all four sites.'
          family: per-event-cost-proportional-to-change
          round: 2
        - id: BR-10
          severity: Important
          title: Tests discard errors that feed assertions; TestRecorderSegments' dropped-records check is vacuous (Stats after Close)
          detail: '2nd finding in this family. Rule: no discarded error whose result feeds an assertion; enforce with errcheck -blank over cmd/internal/webcast test files. Sites: recorder_test.go:132 (validate after Close returns errClosed with a zero Stats), :262, :267; live_record_test.go:69, :192, :241 (ReadDir, nil result panics).'
          family: test-ignores-setup-errors
          round: 2
        - id: BR-11
          severity: Minor
          title: frameOffset assumes an in-process iframe's owner is in the main frame; wrong for nested same-process iframes
          family: frame-coordinate-mapping
          round: 2
        - id: BR-12
          severity: Minor
          title: childNodeInserted with an unknown previousNodeId silently inserts at index 0 instead of reporting a desync
          family: desync-not-guessed
          round: 2
        - id: BR-13
          severity: Minor
          title: snapshotTop failure sets recorder live=false while the reducer's segment stays open, with no stop (ARCH-ORDER)
          family: state-flag-constellation
          round: 2
        - id: BR-14
          severity: Minor
          title: 'PQ-6 bound not held: setChildNodes and Resnapshot forget nodes without burying them, so their ids are never pruned'
          family: id-map-bound
          round: 2
        - id: BR-15
          severity: Minor
          title: Enqueue-and-signal block copied three times in Send, ask and post (ARCH-DRY)
          family: duplicated-enqueue
          round: 2
        - id: BR-16
          severity: Minor
          title: TestRecorderOverflowResyncs is timing-based and never asserts that an overflow actually happened
          family: order-test-oracle
          round: 2
        - id: BR-17
          severity: Minor
          title: Seg comment says trees are dropped; attachFrames re-emits stale out-of-process frame docs that FrameDoc then emits again; dead errStale branch in binding
          family: comment-code-drift
          round: 2
        - id: BR-18
          severity: Minor
          title: Plan says Launch(ctx, recorder.Output) and CastConfig.Assets; code has *RecordOptions and NewAssets; no page left ends with stop pause, not stop end
          family: plan-drift
          round: 2
      boundary: M2
      recipe: milestone-review
      isolation: checkout
      blocked: true
    - "n": 3
      timestamp: "2026-10-12T01:50:14-07:00"
      dispatched: "2026-10-12T01:43:05-07:00"
      agent: claude
      dispose:
        - id: BR-4
          disposition: not-addressed
          note: 'Wiring is present but untested: reverting the loss join in run() and Logf in cast.go leaves all tests green.'
          round: 3
        - id: BR-5
          disposition: addressed
          note: Reverting the assetDone r.opened check to r.live fails TestRecorderAssets.
          round: 3
        - id: BR-6
          disposition: not-addressed
          note: Reverting the DOM-only bound or frameDoc("") returning nil leaves the suite green; no regression test.
          round: 3
        - id: BR-7
          disposition: not-addressed
          note: Most paths are now covered, but frameOffset is still at 0% (no fake getFrameOwner/getBoxModel test).
          round: 3
        - id: BR-8
          disposition: addressed
          note: README.md:925-938 documents the recordings dir, retention, viewer stream and the password rule; matches cast.go/recordingOutput.
          round: 3
        - id: BR-9
          disposition: addressed
          note: byBackend/frames indexes are used at all four sites; a semantics-preserving refactor that existing tests exercise.
          round: 3
        - id: BR-10
          disposition: addressed
          note: Named sites fixed and Stats outlives Close; the errcheck enforcement suggested was not added.
          round: 3
        - id: BR-11
          disposition: addressed
          note: Comment at recorder.go:1028-1033 now matches the code (box in the session's local-root viewport plus the main-frame offset).
          round: 3
        - id: BR-12
          disposition: not-addressed
          note: Desync on unknown previousNodeId is present, but reverting it leaves tests green.
          round: 3
        - id: BR-13
          disposition: not-addressed
          note: Stop pause on snapshotTop failure is present, but reverting it leaves tests green.
          round: 3
        - id: BR-14
          disposition: not-addressed
          note: 'The fix introduced a Critical regression (raised new) and has no test: removing it leaves the suite green.'
          round: 3
        - id: BR-15
          disposition: addressed
          note: Send, ask and post all go through enqueue.
          round: 3
        - id: BR-16
          disposition: addressed
          note: Test now asserts Lost>0, Overflow==0 and the Losses() text, using waitEntered instead of a sleep.
          round: 3
        - id: BR-17
          disposition: not-addressed
          note: Comment fixed and errStale branch removed, but the Seg forget of OOP roots is untested (reverting it leaves tests green).
          round: 3
        - id: BR-18
          disposition: addressed
          note: Plan Revisions entry (plan line 195) matches CastConfig.Launch/NewAssets/NewLive/Logf and tellRecorder's no-page behavior.
          round: 3
      findings:
        - id: BR-19
          severity: Critical
          title: Graveyard prune between forget and index deletes ids of live nodes; same-document resnapshot of a large page churns ids and its Kids is refused
          detail: 'This is the 2nd finding in family id-map-bound. Rule: an id may be pruned only when its backend is absent from the session''s tree after the update is applied, so bury runs after index at every subtree-replace site (install same-document at mirror.go:521, setChildNodes at mirror.go:711). Repro: 30k-node page, Resnapshot twice: div id 7 -> 30014, dropped map[kids:1]. Removing the install bury makes it pass and leaves the suite green. Add a test above graveSize/2.'
          family: id-map-bound
          round: 3
        - id: BR-20
          severity: Minor
          title: Non-DOM CDP events and observer reports now queue unbounded while the recorder is stalled in a call
          detail: After BR-6 only DOM.* is bounded; Network.*, CSS.* and Runtime.bindingCalled grow the queue without limit for as long as a call stalls (ARCH-CONSTRAINTS). The plan should state a bound or argue the input rate is bounded.
          family: queue-bound-per-input-class
          round: 3
      boundary: M2
      recipe: milestone-review
      isolation: checkout
      blocked: true
---

# Gate ledger — pair#458 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T01:05:32-07:00 (claude) — passed

### Raised

- **BR-1** [Minor] `hot-path-allocation` withAttr copies the attribute map on every attr record, even for elements that aren't inputs (reduce.go:306)
  Put an n.N == "input" guard before building the copy; attr is the most frequent mutation kind when replaying.
- **BR-2** [Minor] `consumer-kind-classification` query.UserAction leaves out the new Hover kind with no recorded decision
  Leaving Ptr out is correct. Decide whether hover belongs in a note's action context, and write the decision down when #461 consumes it.
- **BR-3** [Minor] `test-ignores-setup-errors` TestValidateDirReportsStrayFrameFiles discards the errors from Put, PutFrame, Close and WriteFile

## Round 2 — 2026-10-12T01:35:11-07:00 (claude) — BLOCKED

### Raised

- **BR-4** [Important] `silent-error-swallowing` Recorder errors, drops and overflow are invisible in production: Logf never wired, Stats unread, Close error dropped
  cast.go builds RecordOptions without Logf; chrome.go:553 drops the Close error when Logf is nil. A failed Output open or a sink write error leaves an unrecorded cast that looks recorded. Wire Logf to Couch's log and report Stats and the Close error at cast end.
- **BR-5** [Important] `asset-record-dedupe-source` Asset fetched while paused is stored but its asset record is dropped, and goAssets later skips it via store Lookup
  recorder.go handle assetDone emits only if r.live; side.go queueAsset skips URLs already in the store. Emit asset records whatever the phase, or deduplicate against r.m.st.Assets instead of the store.
- **BR-6** [Important] `overflow-drops-non-dom-events` Queue overflow drops every CDP event (contexts, bindingCalled), not only DOM events; unknown frame then resolves against the root
  Send bounds every Event. A lost executionContextCreated makes binding() pass frame "", and frameDoc("") returns the session root, so a report from an in-process iframe can be applied to the wrong node. Bound only DOM.* events, and treat an unknown frame as stale.
- **BR-7** [Important] `fake-coverage-gap` Frames, CSS sheets, truncated text, inline style, assets, screenshots/layout and detach have 0% unit coverage against fakes; PQ-3 sequence tests missing
  Measured with go test -cover: FrameDoc, frameOwner, frameOffset, Drop, FullText, InlineStyle, ConstructedSheet, OwnedSheet, queueAsset, shotDone, captureLayout are at 0%. Missing tests: active page detach then seg tab, frame detach, desync then resync at recorder level, documentUpdated during a pending resync. Only the env-gated TestLiveRecording covers them.
- **BR-8** [Important] `readme-user-surface` README is stale: casts are now recorded to <store>/webcast/recordings/<id>/ and viewers receive the DOM stream
  README.md:914-930 still says viewers see only live state, title and URL, and that the DOM is cast once the recorder lands. Document the recordings directory, its retention, and what viewers now receive (typed non-password values included).
- **BR-9** [Important] `per-event-cost-proportional-to-change` bury/liveBackend scans every node per removed node once the 50k graveyard is full; same O(nodes) scan in FullText, frameDoc, frameOwner
  Rule: per-event work in the mirror must be O(change), not O(document). Keep one backend-to-node index per msession, maintained in index and forget, and use it at all four sites.
- **BR-10** [Important] `test-ignores-setup-errors` Tests discard errors that feed assertions; TestRecorderSegments' dropped-records check is vacuous (Stats after Close)
  2nd finding in this family. Rule: no discarded error whose result feeds an assertion; enforce with errcheck -blank over cmd/internal/webcast test files. Sites: recorder_test.go:132 (validate after Close returns errClosed with a zero Stats), :262, :267; live_record_test.go:69, :192, :241 (ReadDir, nil result panics).
- **BR-11** [Minor] `frame-coordinate-mapping` frameOffset assumes an in-process iframe's owner is in the main frame; wrong for nested same-process iframes
- **BR-12** [Minor] `desync-not-guessed` childNodeInserted with an unknown previousNodeId silently inserts at index 0 instead of reporting a desync
- **BR-13** [Minor] `state-flag-constellation` snapshotTop failure sets recorder live=false while the reducer's segment stays open, with no stop (ARCH-ORDER)
- **BR-14** [Minor] `id-map-bound` PQ-6 bound not held: setChildNodes and Resnapshot forget nodes without burying them, so their ids are never pruned
- **BR-15** [Minor] `duplicated-enqueue` Enqueue-and-signal block copied three times in Send, ask and post (ARCH-DRY)
- **BR-16** [Minor] `order-test-oracle` TestRecorderOverflowResyncs is timing-based and never asserts that an overflow actually happened
- **BR-17** [Minor] `comment-code-drift` Seg comment says trees are dropped; attachFrames re-emits stale out-of-process frame docs that FrameDoc then emits again; dead errStale branch in binding
- **BR-18** [Minor] `plan-drift` Plan says Launch(ctx, recorder.Output) and CastConfig.Assets; code has *RecordOptions and NewAssets; no page left ends with stop pause, not stop end

## Round 3 — 2026-10-12T01:50:14-07:00 (claude) — BLOCKED

### Disposed

- BR-4 — not-addressed — Wiring is present but untested: reverting the loss join in run() and Logf in cast.go leaves all tests green.
- BR-5 — addressed — Reverting the assetDone r.opened check to r.live fails TestRecorderAssets.
- BR-6 — not-addressed — Reverting the DOM-only bound or frameDoc("") returning nil leaves the suite green; no regression test.
- BR-7 — not-addressed — Most paths are now covered, but frameOffset is still at 0% (no fake getFrameOwner/getBoxModel test).
- BR-8 — addressed — README.md:925-938 documents the recordings dir, retention, viewer stream and the password rule; matches cast.go/recordingOutput.
- BR-9 — addressed — byBackend/frames indexes are used at all four sites; a semantics-preserving refactor that existing tests exercise.
- BR-10 — addressed — Named sites fixed and Stats outlives Close; the errcheck enforcement suggested was not added.
- BR-11 — addressed — Comment at recorder.go:1028-1033 now matches the code (box in the session's local-root viewport plus the main-frame offset).
- BR-12 — not-addressed — Desync on unknown previousNodeId is present, but reverting it leaves tests green.
- BR-13 — not-addressed — Stop pause on snapshotTop failure is present, but reverting it leaves tests green.
- BR-14 — not-addressed — The fix introduced a Critical regression (raised new) and has no test: removing it leaves the suite green.
- BR-15 — addressed — Send, ask and post all go through enqueue.
- BR-16 — addressed — Test now asserts Lost>0, Overflow==0 and the Losses() text, using waitEntered instead of a sleep.
- BR-17 — not-addressed — Comment fixed and errStale branch removed, but the Seg forget of OOP roots is untested (reverting it leaves tests green).
- BR-18 — addressed — Plan Revisions entry (plan line 195) matches CastConfig.Launch/NewAssets/NewLive/Logf and tellRecorder's no-page behavior.

### Raised

- **BR-19** [Critical] `id-map-bound` Graveyard prune between forget and index deletes ids of live nodes; same-document resnapshot of a large page churns ids and its Kids is refused
  This is the 2nd finding in family id-map-bound. Rule: an id may be pruned only when its backend is absent from the session's tree after the update is applied, so bury runs after index at every subtree-replace site (install same-document at mirror.go:521, setChildNodes at mirror.go:711). Repro: 30k-node page, Resnapshot twice: div id 7 -> 30014, dropped map[kids:1]. Removing the install bury makes it pass and leaves the suite green. Add a test above graveSize/2.
- **BR-20** [Minor] `queue-bound-per-input-class` Non-DOM CDP events and observer reports now queue unbounded while the recorder is stalled in a call
  After BR-6 only DOM.* is bounded; Network.*, CSS.* and Runtime.bindingCalled grow the queue without limit for as long as a call stalls (ARCH-CONSTRAINTS). The plan should state a bound or argue the input rate is bounded.

## Open findings

- **BR-1** [Minor] `hot-path-allocation` withAttr copies the attribute map on every attr record, even for elements that aren't inputs (reduce.go:306)
- **BR-2** [Minor] `consumer-kind-classification` query.UserAction leaves out the new Hover kind with no recorded decision
- **BR-3** [Minor] `test-ignores-setup-errors` TestValidateDirReportsStrayFrameFiles discards the errors from Put, PutFrame, Close and WriteFile
- **BR-4** [Important] `silent-error-swallowing` Recorder errors, drops and overflow are invisible in production: Logf never wired, Stats unread, Close error dropped
- **BR-6** [Important] `overflow-drops-non-dom-events` Queue overflow drops every CDP event (contexts, bindingCalled), not only DOM events; unknown frame then resolves against the root
- **BR-7** [Important] `fake-coverage-gap` Frames, CSS sheets, truncated text, inline style, assets, screenshots/layout and detach have 0% unit coverage against fakes; PQ-3 sequence tests missing
- **BR-12** [Minor] `desync-not-guessed` childNodeInserted with an unknown previousNodeId silently inserts at index 0 instead of reporting a desync
- **BR-13** [Minor] `state-flag-constellation` snapshotTop failure sets recorder live=false while the reducer's segment stays open, with no stop (ARCH-ORDER)
- **BR-14** [Minor] `id-map-bound` PQ-6 bound not held: setChildNodes and Resnapshot forget nodes without burying them, so their ids are never pruned
- **BR-17** [Minor] `comment-code-drift` Seg comment says trees are dropped; attachFrames re-emits stale out-of-process frame docs that FrameDoc then emits again; dead errStale branch in binding
- **BR-19** [Critical] `id-map-bound` Graveyard prune between forget and index deletes ids of live nodes; same-document resnapshot of a large page churns ids and its Kids is refused
- **BR-20** [Minor] `queue-bound-per-input-class` Non-DOM CDP events and observer reports now queue unbounded while the recorder is stalled in a call
