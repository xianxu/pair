# Boundary Review — pair#464 (whole-issue close)

| field | value |
|-------|-------|
| issue | 464 — couch: messages to an idle slot expire 'waiting for picker to close' when no picker is open |
| repo | pair |
| issue file | workshop/issues/000464-couch-messages-to-an-idle-slot-expire-waiting-for-picker-to-close-when-no-picker-is-open.md |
| boundary | whole-issue close |
| milestone | — |
| window | 5fad484267be4e2938c9a55d9cc93127cc2bdb0f..11f5b1fc8986c19f710379bb1213aa5ba99fee4b |
| command | sdlc close --issue 464 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T00:39:10-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Checking how the stdout pump feeds the terminal model relative to overlay detection.

Tests pass; now running a quick mutation check on the feed hook.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The fix targets the right cause and is solid. Claude Code auto-denies an unattended permission prompt after two minutes, and Esc or a digit key can also answer it. None of those is a plain Enter through the wrapper, and only that Enter cleared the `pickerActive` latch, so the latch stayed armed and held every later peer message. The new `observePickerScreen` rule clears the latch when a picker has hidden the composer and the composer then comes back. It runs on every terminal feed and once at arming. It shares `consumeOverlayLocked` with the Enter path and is limited to the `claude` profile. I checked the ordering: `stdoutPump.appendOutput` feeds the terminal model synchronously, before `checkOverlayOpen` runs, so the read at arming sees the picker's frame. The full `wrapcmd` suite passes. The new tests pass under `-race`. Deleting the feed hook at `wrap.go:531` makes three of the new tests fail. Nothing is Critical. Two cheap fixes are needed before close: the atlas still states two things this change made false, and one Spec clause (a sender-actionable "can't deliver" reason) was neither delivered nor descoped.

1. **Strengths**
   - `wrap.go:2181` `consumeOverlayLocked`: one function now ends the overlay for both the Enter and the screen, and it resets the carries and the new hidden flag in one place (ARCH-DRY).
   - `wrap.go:2198`: the "composer hidden first" rule correctly handles the case where the open OSC arrives before the picker is painted. `TestPickerLatchHoldsUntilThePickerHidTheComposer` pins it.
   - `wrap.go:2017`: the read at arming fixes a real ordering bug (a picker that closes in the very next frame). Mutation testing confirms the tests cover it.
   - The fixtures were re-rendered from the actual pair:3 recording and documented in the testdata README. The end-to-end test `TestPeerDeliveryAfterAutoDeniedPrompt` drives the reported failure through to `Submitted`.
   - `TestPickerLatchScreenCloseIsPerProfile` pins both the per-profile flag and the Enter-only fallback.

2. **Critical:** none.

3. **Important**
   - **Atlas makes two claims this diff falsified.**
     - `atlas/architecture.md:743` ends with "Known edge inherited from the one-shot overlay design: dismissing an overlay without Enter leaves the flag set until the next plain Enter." For Claude this is no longer true.
     - `atlas/architecture.md:739` says "Two independent layers defend this." For Claude, the overlay layer's close now depends on the composer recognizer, so the layers are no longer independent.
     - These are the only two instances in the window. Fix: scope the "known edge" sentence to profiles without `overlayHidesComposer`, and qualify "independent" for Claude.
   - **A Spec clause was neither delivered nor descoped.** The Spec says: "A message that can't be delivered says why in terms the sender can act on." `peer_delivery.go:456` still sets `"waiting for picker to close"`, and the plan has no Revisions entry. A stale latch can still happen on every non-Claude profile, and Claude can still have a picker genuinely open. In both cases the sender gets the same text and no next step. Fix: say what the sender can do (for example, that the slot shows a picker and someone must answer it in the pane), or add a `## Revisions` entry that descopes the clause.

4. **Minor**
   - Done-when's "live check" clause is unverified at close. The plan explicitly defers it to the TL. Make sure it is tracked somewhere so it doesn't disappear when the issue closes.
   - Untested false-close risk. The latch now clears whenever `claudeComposerActive` reads true after a hidden frame. No fixture covers a Claude picker that contains a text input, such as "No, and tell Claude what to do differently" or AskUserQuestion's "Type something". If the recognizer read one of those as a composer, both defensive layers would fail together, and a peer paste could land inside the picker.
   - ARCH-PURPOSE note: the same stuck-latch class remains for codex, grok, qoder, muse and agy, which keep the Enter-only rule. The plan scopes this deliberately ("measured for claude only"), which is acceptable. Only the error-message item above makes it visible to senders.

5. **Test coverage**
   - Both states of Done-when are exercised: the idle-suggestion and shell-footer screens read deliverable, and the picker still on screen keeps the latch armed.
   - Mutating the feed hook fails three tests. I did not independently mutate the arming read, the hidden-first rule or the profile flag. The Log claims those mutations were checked, and the test structure supports it.

6. **Architecture**
   - ARCH-DRY: pass. The consume logic was consolidated rather than duplicated.
   - ARCH-PURE: pass. `observePickerScreen` is thin glue: it reads a snapshot and calls the pure recognizer under the lock. The decision is simple enough that extracting it would add little.
   - ARCH-PURPOSE: pass, with the per-harness note above.
   - For later work: if other harnesses get `overlayHidesComposer`, each needs a recorded picker-closes-without-Enter fixture first.

7. **Plan revisions**
   - Add a `## Revisions` entry that either records the delivered sender-actionable message or explicitly descopes the Spec clause "says why in terms the sender can act on".

```findings
findings:
  - id: new
    severity: Important
    family: doc-claim-drift
    title: |
      Atlas still says dismissing an overlay without Enter leaves the latch set, and calls the two layers independent
    detail: |
      atlas/architecture.md:743 "Known edge inherited from the one-shot overlay design: dismissing an overlay without Enter leaves the flag set until the next plain Enter" is now false for claude; atlas/architecture.md:739 "Two independent layers" no longer holds for claude because the overlay close now reads the composer recognizer. These are the only two instances in the window. Scope both to profiles without overlayHidesComposer.
  - id: new
    severity: Important
    family: spec-clause-undelivered
    title: |
      The Spec's sender-actionable undeliverable reason was neither delivered nor descoped
    detail: |
      peer_delivery.go:456 still reports "waiting for picker to close" with no next step for the sender, and every non-claude profile can still produce a stale latch. Either word it so the sender knows what to do (answer the picker in the pane) or add a plan Revisions entry descoping the clause.
  - id: new
    severity: Minor
    family: done-when-unverified-at-close
    title: |
      Done-when live check deferred to the TL and unverified at close
    detail: |
      The plan item is unchecked by design; make sure the TL's post-relaunch check is tracked somewhere so it is not lost when the issue closes.
  - id: new
    severity: Minor
    family: layer-coupling-unfixtured
    title: |
      Screen close couples the overlay latch to the composer recognizer with no picker-with-text-input fixture
    detail: |
      If claudeComposerActive reads a picker's free-text field (permission "No, and tell Claude…", AskUserQuestion "Type something") as a composer, the latch clears while the picker is open and both defensive layers fail together. Add such a fixture asserting the latch holds.
```

---

## Re-review — 2026-10-12T00:44:01-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 464 — couch: messages to an idle slot expire 'waiting for picker to close' when no picker is open |
| repo | pair |
| issue file | workshop/issues/000464-couch-messages-to-an-idle-slot-expire-waiting-for-picker-to-close-when-no-picker-is-open.md |
| boundary | whole-issue close |
| milestone | — |
| window | 5fad484267be4e2938c9a55d9cc93127cc2bdb0f..ae4e297f1eabb62c974344517709a4b0f27419b3 |
| command | sdlc close --issue 464 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T00:44:01-07:00 |
| verdict | SHIP |

## Review

Probing whether the new screen-close can clear the latch early when the picker's paint arrives split across chunks. The test build needs generated runtime assets first.

The split probe was clean: none of the 5,174 split points cleared the latch early. Now running the new tests and the full package.

```verdict
verdict: SHIP
confidence: high
```

**Verdict: SHIP.** The root cause is real and the fix targets it. `pickerActive` was a latch that only a plain Enter could clear. When Claude Code auto-denied a permission prompt, or the user pressed Esc or a digit, the latch stayed set and every peer message was held "waiting for picker to close". The fix adds a per-profile `overlayHidesComposer` flag, set only for Claude. With it, the latch clears when the composer comes back after the picker hid it, and this goes through the same `consumeOverlayLocked` that the Enter path uses. All four prior findings are disposed below. The new tests and the full `wrapcmd` package pass. I also probed the new path myself (results under test coverage). Nothing blocks SHIP.

**Strengths**
1. `wrap.go:2178-2189`: `consumeOverlayLocked` is now the single way an overlay ends. Enter and the screen-close share it, so the carry-reset logic exists in one place (ARCH-DRY pass).
2. `wrap.go:2198-2216`: the hidden-first rule keeps the latch armed when the open signal arrives before the picker's paint. `TestPickerLatchHoldsUntilThePickerHidTheComposer` pins this.
3. `checkOverlayOpen` (`wrap.go:2015-2017`) reads the screen once when the latch arms. The picker's frame lands before its signal, so a close in the very next frame would otherwise be missed. The call happens after `detectOverlayOpen` has released `overlayMu`, so there is no deadlock.
4. `peer_picker_test.go:144-159`: a table check across every profile that only Claude sets the flag, plus a check that turning the flag off restores the Enter-only behaviour.
5. The fixtures are re-rendered from the actual pair:3 incident, and the end-to-end test (`TestPeerDeliveryAfterAutoDeniedPrompt`) reaches `couchmessage.Submitted`.

**Critical findings:** none.

**Important findings:** none.

**Minor findings:** none new.

**Test coverage notes**
- `go test ./cmd/internal/wrapcmd/` passes, including all six new tests.
- **Split probe:** I wrote a temporary test (since removed) for the riskiest new path. It sends the composer screen plus the permission signal, then the permission-picker frame split at every byte offset, then the frame again. None of the 5,174 split points cleared the latch early. So a partial paint does not read as "composer hidden, then composer back".
- The question-picker fixture does catch the coupling bug: it feeds the same frame repeatedly after arming. If the recognizer accepted it, the first frame would never mark the composer hidden, and with no reading of it hidden the latch could not clear. On the next frame it would see "composer back", so the latch would clear and the test would fail. The test also checks directly that the composer state reads Unknown.

**Done-when clauses**
- "Idle prompt with dim suggestion / background-task footer reads deliverable": covered by `TestRecordedIdleScreensReadDeliverable`.
- "Real pickers still block": covered by the latch holding while the countdown repaints, and by the question-picker test.
- "Live check": deferred to the TL, recorded in the Plan.

Done-when: current — the Spec's only change is moving the sender-actionable wording to #465. The Done-when never mentioned that wording, and both remaining clauses either have tests or are tracked.

**Architecture**
- **ARCH-DRY: pass.** The duplicated overlay consume is consolidated, and nothing in the diff repeats existing logic.
- **ARCH-PURE: pass.** The decision reads `profile.recognize`, a pure function over terminal snapshots, and the hook in `wrap.go` is a few lines of glue.
- **ARCH-PURPOSE: pass.** The latch outliving its picker, which is the point of the issue, is fixed. The one deferred clause (sender wording) has a home: #465's receipts, `blocked by a picker: <title>`, which is a separable extension, not the core of this issue.
- **For later work:** other harnesses still clear the latch only on Enter. When someone measures whether their pickers hide the composer, the change is the one-word `overlayHidesComposer` flag, and the per-profile test enforces that it was measured.

**Plan revision recommendations:** none. The plan's Revisions entry already matches the code.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      atlas/architecture.md:739 now says the layers are independent except for Claude's overlay close and names the test; :743 limits the stale-flag edge to profiles without overlayHidesComposer. Both instances fixed.
  - id: BR-2
    disposition: addressed
    note: |
      Descoped in the plan's Revisions (2026-10-12, at close) to #465, whose Spec owns "blocked by a picker: <title>" receipts and the hub notice. Checked in workshop/issues/000465 lines 21-25.
  - id: BR-3
    disposition: addressed
    note: |
      The Plan item records the owner (TL, ops:0), the trigger (next pair:0 build, Couch restart, slot relaunch) and the repro. It stays unchecked by design.
  - id: BR-4
    disposition: addressed
    note: |
      TestPickerLatchHoldsOverQuestionPickerTextRow re-feeds the recorded question-picker frame after arming and asserts the latch holds and the composer reads Unknown; if the recognizer accepted that frame the test would fail. My split probe also found no premature clear.
```
