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

## Open findings

- **BR-1** [Minor] `hot-path-allocation` withAttr copies the attribute map on every attr record, even for elements that aren't inputs (reduce.go:306)
- **BR-2** [Minor] `consumer-kind-classification` query.UserAction leaves out the new Hover kind with no recorded decision
- **BR-3** [Minor] `test-ignores-setup-errors` TestValidateDirReportsStrayFrameFiles discards the errors from Put, PutFrame, Close and WriteFile
- **BR-4** [Important] `silent-error-swallowing` Recorder errors, drops and overflow are invisible in production: Logf never wired, Stats unread, Close error dropped
- **BR-5** [Important] `asset-record-dedupe-source` Asset fetched while paused is stored but its asset record is dropped, and goAssets later skips it via store Lookup
- **BR-6** [Important] `overflow-drops-non-dom-events` Queue overflow drops every CDP event (contexts, bindingCalled), not only DOM events; unknown frame then resolves against the root
- **BR-7** [Important] `fake-coverage-gap` Frames, CSS sheets, truncated text, inline style, assets, screenshots/layout and detach have 0% unit coverage against fakes; PQ-3 sequence tests missing
- **BR-8** [Important] `readme-user-surface` README is stale: casts are now recorded to <store>/webcast/recordings/<id>/ and viewers receive the DOM stream
- **BR-9** [Important] `per-event-cost-proportional-to-change` bury/liveBackend scans every node per removed node once the 50k graveyard is full; same O(nodes) scan in FullText, frameDoc, frameOwner
- **BR-10** [Important] `test-ignores-setup-errors` Tests discard errors that feed assertions; TestRecorderSegments' dropped-records check is vacuous (Stats after Close)
- **BR-11** [Minor] `frame-coordinate-mapping` frameOffset assumes an in-process iframe's owner is in the main frame; wrong for nested same-process iframes
- **BR-12** [Minor] `desync-not-guessed` childNodeInserted with an unknown previousNodeId silently inserts at index 0 instead of reporting a desync
- **BR-13** [Minor] `state-flag-constellation` snapshotTop failure sets recorder live=false while the reducer's segment stays open, with no stop (ARCH-ORDER)
- **BR-14** [Minor] `id-map-bound` PQ-6 bound not held: setChildNodes and Resnapshot forget nodes without burying them, so their ids are never pruned
- **BR-15** [Minor] `duplicated-enqueue` Enqueue-and-signal block copied three times in Send, ask and post (ARCH-DRY)
- **BR-16** [Minor] `order-test-oracle` TestRecorderOverflowResyncs is timing-based and never asserts that an overflow actually happened
- **BR-17** [Minor] `comment-code-drift` Seg comment says trees are dropped; attachFrames re-emits stale out-of-process frame docs that FrameDoc then emits again; dead errStale branch in binding
- **BR-18** [Minor] `plan-drift` Plan says Launch(ctx, recorder.Output) and CastConfig.Assets; code has *RecordOptions and NewAssets; no page left ends with stop pause, not stop end
