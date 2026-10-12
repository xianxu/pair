# Boundary Review — pair#458 (milestone M2)

| field | value |
|-------|-------|
| issue | 458 — broadcast-web: recorder — CDP DOM stream per frame plus isolated-world observer |
| repo | pair |
| issue file | workshop/issues/000458-broadcast-web-recorder-cdp-dom-stream-per-frame-plus-isolated-world-observer.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 1a2cc4dca2a541cb60ff50326c7d7c157a84544f..6496339f0ad999a70ff7014d00a48371b966a57f |
| command | sdlc milestone-close --issue 458 --milestone M2 |
| reviewer | claude |
| isolation | checkout |
| timestamp | 2026-10-12T01:35:11-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Tests pass for webcast/recorder/cdp; couchcmd needs a generated asset bundle. Now reading the recorder code.

Reviewing the Chrome loop and Cast wiring next.

Race-clean. Checking whether the live headless test actually ran, then the atlas/doc diff.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

M2 delivers what the plan asked for at its core. There is a pure mirror that pushes every record through its own reducer, a path-resolved isolated-world observer, a single-writer Recorder shell, CDP message numbering (`Seq`) used to drop events a snapshot already includes, and the Chrome-loop and Cast wiring. The webcast, recorder and cdp tests pass, including under `-race`. Nothing found is a Critical correctness bug, but several Important items should be fixed before the boundary is crossed:

- **Silent failure in production:** recorder errors and losses reach no log, and `Stats` is never read.
- **Lost assets:** an asset fetched while paused is stored but never recorded.
- **Queue overflow drops more than the plan says:** it discards observer reports and execution-context events too, and an unknown context then makes reports resolve against the wrong document.
- **Big untested areas:** frames, CSS, assets and layout have no tests against the fakes.
- **Stale README:** it still tells the operator that viewers see only the title and URL.
- **Hot-path scan:** once the graveyard is full, every removal scans the whole node map.

I couldn't reproduce the live check. `TestLiveRecording` is skipped unless `WEBCAST_LIVE_CHROME=1` is set, and with it set Chrome would not launch on this machine (sandbox/ProcessSingleton error, status 21). The couchcmd and couchtty packages don't build here because the generated `assets/runtime/files` is missing; that predates this range.

1. **Strengths**
   - `mirror.emit` (`recorder/mirror.go`) applies every record to its own `recording.State` first, so the output validates by construction and losses are counted rather than written out.
   - The CDP sequence numbering is small and correct (`cdp/conn.go` dispatch, `CallSeq`). `TestSeqOrdersEventsAndReplies` and `TestRecorderDropsEventsTheSnapshotIncludes` pin it with injected interleavings.
   - `TestMirrorMatchesThePageUnderRandomMutations` is a real property test against a stateful fake DOM: 40 seeds × 150 steps, covering moves, shadow roots and password games, checking against `sameTree`, `Validate` and a planted-secret scan. `FuzzResolve` covers the path resolver.
   - Passwords are protected in depth: the observer only ever sends `secret`, `wiredAttrs` drops `value`, `attrDiff` removes before it adds (so flipping a type drops the value first), and the reducer refuses anything that slips through.
   - The click reported synchronously in its handler, resolved against the mirror, closes #457's lost-click gap (`TestRecorderClickBeforeNavigation`).

2. **Critical:** none.

3. **Important**
   - **Recorder failures are silent in production** (`cast.go` `RecordOptions{ID, Out}`; `chrome.go:553`). `Logf` is never set, so a failed open, a write error, overflow resyncs and drop examples all go to a no-op. `Close`'s error is dropped when `Logf` is nil, and nothing reads `Stats`. A full disk or an mkdir failure leaves a cast that looks recorded and isn't. Fix: wire `Logf` to Couch's log, and report `Stats` and the `Close` error when the cast ends.
   - **Assets that finish while paused are lost** (`recorder.go` `case assetDone: if r.live`; `side.go` `queueAsset`). The store keeps the body, the `asset` record is dropped, and `goAssets` on resume skips the URL because `Lookup` says it's stored. Fix: emit `asset` records whatever the phase (the reducer keeps `Assets` across segments), or deduplicate against `r.m.st.Assets` rather than the store.
   - **Overflow drops every CDP event, not just DOM events** (`Recorder.Send`). The plan says only the live sessions' DOM events are dropped. Losing `Runtime.executionContextCreated` leaves `contexts` incomplete. Later `bindingCalled` reports then get frame `""`, and `frameDoc("")` resolves them against the session's root, so input, scroll, hover or focus from an in-process iframe can land on the wrong node. Reports themselves (clicks) are also dropped. Fix: apply the queue bound only to `DOM.*` events, and treat an unknown frame as stale instead of falling back to the root.
   - **Unit tests against fakes are missing for whole areas** (measured with `go test -cover`). These functions are at 0%: `FrameDoc`, `frameOwner`, `emitSessionDoc`, `frameOffset`, `Drop`, `FullText`, `readText`, `InlineStyle`, `ConstructedSheet`, `OwnedSheet`, `adopt`, `queueAsset`, `shotDone`, `captureLayout`. `attachFrames` is at 6.7%. The plan's PQ-3 sequence tests are also absent:
     - the active page detaching, then `seg tab`;
     - a frame session detaching;
     - a desync at recorder level triggering a resync;
     - `documentUpdated` arriving while a resync is pending.

     Only the env-gated live test covers these. Fix: extend `fakeChrome` with `Target`/`Page`/`CSS`/`Network`/`DOMSnapshot` replies, and add sequence tests for each.
   - **README is stale for the new user-facing surface** (`README.md:914-930`).
     - Every cast is now written to `<couch store>/webcast/recordings/<id>/` and kept until deleted.
     - Viewers now receive the DOM record stream (`rec` events), including typed non-password values.
     - The README still says "Viewers see only whether the cast is live … title and URL" and "once the recorder lands". This touches privacy.
   - **Hot-path O(nodes) scan per removed node once the graveyard is full** (`mirror.go` `bury` → `liveBackend`). The same pattern recurs in `FullText`, `observe.go` `frameDoc` (every report from an in-process iframe) and `frameOwner`. On the stress page this eats the 250 ms p95 budget. Fix: one `backend → *cnode` index per `msession`, kept in `index` and `forget`, and used by all four.
   - **This is the 2nd finding in family `test-ignores-setup-errors`.** The rule that covers both: a test must not discard an error whose result feeds an assertion. Enforce it with `errcheck -blank` (or an equivalent vet analyzer) over `cmd/internal/webcast/...` test files, rather than fixing sites one by one.
     - The instance that matters: `TestRecorderSegments` calls `h.r.Close()` and then `h.validate()`. Inside, `st, _ := h.r.Stats(...)` returns `errClosed` with a zero `st`, so the "dropped" check is vacuous.
     - Other sites: `recorder_test.go:262` and `:267`, and `live_record_test.go:69`, `:192` and `:241` (`ReadDir`, where a nil result panics).

4. **Minor**
   - `frameOffset` assumes an in-process iframe's owner sits in the main frame. Coordinates are wrong for nested same-process iframes, and for an out-of-process frame owned inside a same-process subframe.
   - `childNodeInserted` with an unknown `previousNodeId` silently inserts at index 0 instead of reporting a desync.
   - If `snapshotTop` fails during a tab switch, the recorder sets `live=false` while the reducer's segment stays open, with no `stop`. The separate `live`/`opened`/`resync` flags have unwritten legal combinations (ARCH-ORDER).
   - The `errStale && it.K != "click" { continue }` branch in `binding` is dead code.
   - The `Seg` comment says every session's tree is dropped; the code doesn't. `attachFrames` emits stale docs for out-of-process frames, then `FrameDoc` emits them again on each segment.
   - The id-map bound (PQ-6) doesn't hold: `setChildNodes` and `Resnapshot` forget nodes without burying them, so those ids are never pruned.
   - `Detached` leaves the `frameIDs`, `loader`, `requests` and `headers` entries behind.
   - `TestRecorderOverflowResyncs` depends on timing (`Sleep`) and never asserts that an overflow actually happened.
   - The enqueue-and-signal block is copied three times (`Send`, `ask`, `post`); it should be one `enqueue` (ARCH-DRY).
   - The observer runs a full-document `querySelectorAll('*')` every 250 ms; its page cost is unmeasured (ARCH-CONSTRAINTS).
   - The observer and out-of-process frame auto-attach are installed even when nothing is recorded.
   - Production uses `recording.NewFakeHub`.
   - With no page left the recording ends `stop pause`, not the planned `stop end`.
   - Recordings are cleaned up only by the operator, per cast (ARCH-FUNERAL; the plan accepted this).

5. **Test coverage**
   - The mirror's DOM translation, password rules, path resolution, segments, the `Seq` filter and click-before-navigate are well covered.
   - Frames, CSS, assets, screenshots/layout and detach handling rest entirely on the env-gated live test.
   - The live check is manual, with no conformance cadence (ARCH-MOCK).

6. **Architecture**
   - ARCH-DRY: flagged (the enqueue copies). The `cdpconv` duplication was accepted in the plan.
   - ARCH-PURE: pass. The mirror is pure and its tests need no IO.
   - ARCH-PURPOSE: pass for M2; the asset store isn't wired yet because #459 hasn't landed.
   - ARCH-MOCK: flagged. The fake CDP is stateful only for the DOM, and there is no scheduled live conformance check.
   - ARCH-CONSTRAINTS: flagged (the hot-path scan and the observer's scan).
   - ARCH-SECURE: pass. The binding exists only in the isolated world, password values never leave the observer, and the recordings directory is created 0700.
   - ARCH-ORDER: flagged (flag combinations, the overflow test, and the missing interleaving tests).
   - ARCH-FUNERAL: Minor (in-memory per-session maps; operator-only cleanup of recordings).

7. **Plan revisions**
   - Record that `CastConfig.Launch` is `func(ctx, *RecordOptions)` and that `Assets` became `NewAssets func(recording.AssetWriter)`.
   - Record that overflow handling must spare Runtime and observer events.
   - Record that the PQ-3 sequence tests and the frame/CSS/asset fake coverage are still owed (M2), or move them to M3 explicitly.

```findings
findings:
  - id: new
    severity: Important
    family: silent-error-swallowing
    title: |
      Recorder errors, drops and overflow are invisible in production: Logf never wired, Stats unread, Close error dropped
    detail: |
      cast.go builds RecordOptions without Logf; chrome.go:553 drops the Close error when Logf is nil. A failed Output open or a sink write error leaves an unrecorded cast that looks recorded. Wire Logf to Couch's log and report Stats and the Close error at cast end.
  - id: new
    severity: Important
    family: asset-record-dedupe-source
    title: |
      Asset fetched while paused is stored but its asset record is dropped, and goAssets later skips it via store Lookup
    detail: |
      recorder.go handle assetDone emits only if r.live; side.go queueAsset skips URLs already in the store. Emit asset records whatever the phase, or deduplicate against r.m.st.Assets instead of the store.
  - id: new
    severity: Important
    family: overflow-drops-non-dom-events
    title: |
      Queue overflow drops every CDP event (contexts, bindingCalled), not only DOM events; unknown frame then resolves against the root
    detail: |
      Send bounds every Event. A lost executionContextCreated makes binding() pass frame "", and frameDoc("") returns the session root, so a report from an in-process iframe can be applied to the wrong node. Bound only DOM.* events, and treat an unknown frame as stale.
  - id: new
    severity: Important
    family: fake-coverage-gap
    title: |
      Frames, CSS sheets, truncated text, inline style, assets, screenshots/layout and detach have 0% unit coverage against fakes; PQ-3 sequence tests missing
    detail: |
      Measured with go test -cover: FrameDoc, frameOwner, frameOffset, Drop, FullText, InlineStyle, ConstructedSheet, OwnedSheet, queueAsset, shotDone, captureLayout are at 0%. Missing tests: active page detach then seg tab, frame detach, desync then resync at recorder level, documentUpdated during a pending resync. Only the env-gated TestLiveRecording covers them.
  - id: new
    severity: Important
    family: readme-user-surface
    title: |
      README is stale: casts are now recorded to <store>/webcast/recordings/<id>/ and viewers receive the DOM stream
    detail: |
      README.md:914-930 still says viewers see only live state, title and URL, and that the DOM is cast once the recorder lands. Document the recordings directory, its retention, and what viewers now receive (typed non-password values included).
  - id: new
    severity: Important
    family: per-event-cost-proportional-to-change
    title: |
      bury/liveBackend scans every node per removed node once the 50k graveyard is full; same O(nodes) scan in FullText, frameDoc, frameOwner
    detail: |
      Rule: per-event work in the mirror must be O(change), not O(document). Keep one backend-to-node index per msession, maintained in index and forget, and use it at all four sites.
  - id: new
    severity: Important
    family: test-ignores-setup-errors
    title: |
      Tests discard errors that feed assertions; TestRecorderSegments' dropped-records check is vacuous (Stats after Close)
    detail: |
      2nd finding in this family. Rule: no discarded error whose result feeds an assertion; enforce with errcheck -blank over cmd/internal/webcast test files. Sites: recorder_test.go:132 (validate after Close returns errClosed with a zero Stats), :262, :267; live_record_test.go:69, :192, :241 (ReadDir, nil result panics).
  - id: new
    severity: Minor
    family: frame-coordinate-mapping
    title: |
      frameOffset assumes an in-process iframe's owner is in the main frame; wrong for nested same-process iframes
  - id: new
    severity: Minor
    family: desync-not-guessed
    title: |
      childNodeInserted with an unknown previousNodeId silently inserts at index 0 instead of reporting a desync
  - id: new
    severity: Minor
    family: state-flag-constellation
    title: |
      snapshotTop failure sets recorder live=false while the reducer's segment stays open, with no stop (ARCH-ORDER)
  - id: new
    severity: Minor
    family: id-map-bound
    title: |
      PQ-6 bound not held: setChildNodes and Resnapshot forget nodes without burying them, so their ids are never pruned
  - id: new
    severity: Minor
    family: duplicated-enqueue
    title: |
      Enqueue-and-signal block copied three times in Send, ask and post (ARCH-DRY)
  - id: new
    severity: Minor
    family: order-test-oracle
    title: |
      TestRecorderOverflowResyncs is timing-based and never asserts that an overflow actually happened
  - id: new
    severity: Minor
    family: comment-code-drift
    title: |
      Seg comment says trees are dropped; attachFrames re-emits stale out-of-process frame docs that FrameDoc then emits again; dead errStale branch in binding
  - id: new
    severity: Minor
    family: plan-drift
    title: |
      Plan says Launch(ctx, recorder.Output) and CastConfig.Assets; code has *RecordOptions and NewAssets; no page left ends with stop pause, not stop end
```

---

## Re-review — 2026-10-12T01:50:13-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 458 — broadcast-web: recorder — CDP DOM stream per frame plus isolated-world observer |
| repo | pair |
| issue file | workshop/issues/000458-broadcast-web-recorder-cdp-dom-stream-per-frame-plus-isolated-world-observer.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 1a2cc4dca2a541cb60ff50326c7d7c157a84544f..83de5fcf406a74e5bc89b0cb7cbbe9bbab60a9ce |
| command | sdlc milestone-close --issue 458 --milestone M2 |
| reviewer | claude |
| isolation | checkout |
| timestamp | 2026-10-12T01:50:13-07:00 |
| verdict | REWORK |

## Review

Now the fix commit diff in detail.

Reviewing the fix round; writing a quick repro test for a suspected id-loss bug in `install`'s same-document bury.

Confirmed. Checking that the BR-14 change in `install` caused it.

Confirmed: the BR-14 change causes it, and removing it leaves the whole suite green (so BR-14 has no regression test). Next I'm running the full suite and checking the other fixes.

Now reverting each claimed behavior fix individually to see whether a test goes red.

Checking whether BR-4's Close-error and Logf wiring have regression tests.

Two last revert checks (BR-17's Seg change, BR-9 indexes) before writing the verdict.

```verdict
verdict: REWORK
confidence: high
```

**Summary.** The fix round does a lot well. The Stats/Close account, the per-session `byBackend`/`frames` indexes, one snapshot per input through a `dirty` set, the single `enqueue`, and the new path tests (OOP frame, styles, assets, screenshots/layout, active-page detach, one-snapshot-per-reason) are all solid. Webcast and recorder tests pass under `-race`. But the BR-14 fix brought in a **Critical regression**: `install` now buries the old tree *between* `forget` and `index`. When the graveyard overflows, the prune sees every node of the old tree as dead and deletes ids that still belong to live nodes. I reproduced it in this checkout. A same-document resnapshot of a 30k-node page, run twice, gives the `div` a new id (7 → 30014). The `Kids` record is refused (`dropped map[kids:1]`) because the root's id changed, so resync breaks on large or often-resynced pages. Removing the change makes the repro pass and leaves the whole suite green, so BR-14 had no regression test. Separately, BR-4, BR-6, BR-12, BR-13 and BR-17 change behavior, but reverting each one leaves every test green. They still lack their regression evidence. BR-7 is still missing `frameOffset` coverage (0%).

### 1. Strengths
- **One snapshot per input.** The `dirty` set (`recorder.go` afterInput) gives one snapshot per session per input whatever the reasons, page first. The page's snapshot uses `markAll` when it turns out to be a new document. `TestRecorderOneSnapshotServesEveryReason` pins this with an injected interleaving (`gatedCaller`). That is a good ARCH-ORDER test.
- **Stats after Close.** `Stats` now returns the final account after Close (`r.final`), which fixes the vacuous check from BR-10. The overflow test now asserts `Lost > 0 && Overflow == 0` and checks `Losses()` (BR-16).
- **`enqueue` (BR-15).** Send, ask and post now share one path, and only `DOM.*` events are bounded.
- **New tests found real bugs.** `TestRecorderAssets` and the screenshot/layout test caught the `getResourceContent` `content` field and top-document layout bugs. That is evidence the fake now models the dependency rather than echoing the implementation.
- **Asset records (BR-5).** Assets are emitted whenever `r.opened`. Reverting that change makes `TestRecorderAssets` fail, so the fix is real.

### 2. Critical findings
- **`mirror.go:521-529` (install) and `mirror.go:711-713` (setChildNodes): the prune deletes ids of live nodes (ARCH-ORDER, id-map-bound).** `bury` prunes the oldest graveyard entries using `liveBackend`. Both sites call it right after `forget` has emptied `byBackend` and before `index` has added the new tree back, so every pruned entry looks dead.
  - **Repro:** 30k text nodes plus one div, then `Resnapshot` ×2. The div gets a new id, `Kids` is refused, and the recording falls out of sync.
  - **Fix:** index the new tree first, then bury only the old nodes whose backend id is absent from the new `byBackend`. Apply this at both sites.
  - **Test:** add one at more than `graveSize/2` nodes, or make `graveSize` a variable the test can set.

### 3. Important findings
These are prior findings disposed `not-addressed`:
- **BR-4:** the code is wired, but there is no test. With the loss-error join in `run()` and `Logf: cfg.Logf` in `cast.go` both removed, the suite stays green. Needed:
  - a recorder test that `Close()` returns the loss error;
  - a webcast test that `CastConfig.Logf` reaches the recorder.
- **BR-6:** reverting either the `DOM.` bound or `frameDoc("")` → nil leaves the suite green. Needed:
  - a test that an overflowing burst does not drop `executionContextCreated` or `bindingCalled`;
  - a test that an unknown frame's report is counted stale, not applied to the root.
- **BR-7:** `frameOffset` is still at 0% coverage. That is the in-process and out-of-process frame-to-page coordinate mapping. It needs a fake `DOM.getFrameOwner`/`DOM.getBoxModel` test, including a nested in-process frame (BR-11's case).

### 4. Minor findings
- **Unbounded queue for non-DOM inputs (ARCH-CONSTRAINTS).** After BR-6, `Network.*`, `CSS.*` and `Runtime.bindingCalled` queue without a bound while the recorder is stalled in a call. The plan should state the bound or argue why the input rate is bounded.
- **BR-12, BR-13, BR-17:** each fix is present, but reverting it leaves the suite green.
- **Notice spam.** `Logf` posts a console notice on every resync (`recorder.go:483`). Under sustained overflow this could crowd the notice line. Consider rate-limiting it, or reporting only at the end.

### 5. Test coverage notes
- **Passing:** `go test -race` on `webcast/...` and `cdp`. Recorder coverage is 78.9%.
- **Not built here:** `couchtty` and `couchcmd` fail setup in this checkout because `runtimebundle/embed.go` has no generated assets. That is environmental, but it means the console `Logf` wiring was not compiled or tested in this review.
- **Timing-based:** `TestRecorderAssets` still relies on `time.Sleep(50ms)`. Its "finishing as the cast pauses" step races `Pause` against `assetDone`, so it does not reliably hit the paused path. The test passes either way, but revert-to-`r.live` did go red.

### 6. Architecture
| Principle | Result | Note |
|---|---|---|
| ARCH-DRY | pass | `enqueue` consolidated |
| ARCH-PURE | pass | mirror stays pure; shell is thin |
| ARCH-PURPOSE | pass | |
| ARCH-MOCK | pass | fake CDP is stateful; a live check exists (env-gated) |
| ARCH-CONSTRAINTS | flag (Minor) | unbounded non-DOM queue |
| ARCH-SECURE | pass | password rule held; random-mutation test checks the planted secret |
| ARCH-ORDER | flag (Critical) | the forget → bury → index window is exactly an unwritten ordering invariant |
| ARCH-FUNERAL | pass | recordings retained until the operator deletes them, declared in plan line 115 and the README |

For upcoming work (M3), write the mirror's subtree-replace invariant once as a helper used by every site that replaces a subtree, e.g. `replace(old, new)`: index new, then bury old minus new.

### 7. Plan revision recommendations
- Add a Revisions entry: "Subtree replacement (install same-document, setChildNodes) indexes the new tree before burying the old; ids are pruned only for backends absent after the update (id-map bound, BR-14 follow-up)."
- State the queue bound, or the reason none is needed, for non-DOM inputs.

```findings
dispose:
  - id: BR-4
    disposition: not-addressed
    note: |
      Wiring is present but untested: reverting the loss join in run() and Logf in cast.go leaves all tests green.
  - id: BR-5
    disposition: addressed
    note: |
      Reverting the assetDone r.opened check to r.live fails TestRecorderAssets.
  - id: BR-6
    disposition: not-addressed
    note: |
      Reverting the DOM-only bound or frameDoc("") returning nil leaves the suite green; no regression test.
  - id: BR-7
    disposition: not-addressed
    note: |
      Most paths are now covered, but frameOffset is still at 0% (no fake getFrameOwner/getBoxModel test).
  - id: BR-8
    disposition: addressed
    note: |
      README.md:925-938 documents the recordings dir, retention, viewer stream and the password rule; matches cast.go/recordingOutput.
  - id: BR-9
    disposition: addressed
    note: |
      byBackend/frames indexes are used at all four sites; a semantics-preserving refactor that existing tests exercise.
  - id: BR-10
    disposition: addressed
    note: |
      Named sites fixed and Stats outlives Close; the errcheck enforcement suggested was not added.
  - id: BR-11
    disposition: addressed
    note: |
      Comment at recorder.go:1028-1033 now matches the code (box in the session's local-root viewport plus the main-frame offset).
  - id: BR-12
    disposition: not-addressed
    note: |
      Desync on unknown previousNodeId is present, but reverting it leaves tests green.
  - id: BR-13
    disposition: not-addressed
    note: |
      Stop pause on snapshotTop failure is present, but reverting it leaves tests green.
  - id: BR-14
    disposition: not-addressed
    note: |
      The fix introduced a Critical regression (raised new) and has no test: removing it leaves the suite green.
  - id: BR-15
    disposition: addressed
    note: |
      Send, ask and post all go through enqueue.
  - id: BR-16
    disposition: addressed
    note: |
      Test now asserts Lost>0, Overflow==0 and the Losses() text, using waitEntered instead of a sleep.
  - id: BR-17
    disposition: not-addressed
    note: |
      Comment fixed and errStale branch removed, but the Seg forget of OOP roots is untested (reverting it leaves tests green).
  - id: BR-18
    disposition: addressed
    note: |
      Plan Revisions entry (plan line 195) matches CastConfig.Launch/NewAssets/NewLive/Logf and tellRecorder's no-page behavior.
findings:
  - id: new
    severity: Critical
    family: id-map-bound
    title: |
      Graveyard prune between forget and index deletes ids of live nodes; same-document resnapshot of a large page churns ids and its Kids is refused
    detail: |
      This is the 2nd finding in family id-map-bound. Rule: an id may be pruned only when its backend is absent from the session's tree after the update is applied, so bury runs after index at every subtree-replace site (install same-document at mirror.go:521, setChildNodes at mirror.go:711). Repro: 30k-node page, Resnapshot twice: div id 7 -> 30014, dropped map[kids:1]. Removing the install bury makes it pass and leaves the suite green. Add a test above graveSize/2.
  - id: new
    severity: Minor
    family: queue-bound-per-input-class
    title: |
      Non-DOM CDP events and observer reports now queue unbounded while the recorder is stalled in a call
    detail: |
      After BR-6 only DOM.* is bounded; Network.*, CSS.* and Runtime.bindingCalled grow the queue without limit for as long as a call stalls (ARCH-CONSTRAINTS). The plan should state a bound or argue the input rate is bounded.
```
