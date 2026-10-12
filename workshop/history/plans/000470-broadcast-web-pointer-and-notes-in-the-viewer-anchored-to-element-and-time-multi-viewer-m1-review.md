# Boundary Review — pair#470 (milestone M1)

| field | value |
|-------|-------|
| issue | 470 — broadcast-web: pointer and notes in the viewer — anchored to element and time, multi-viewer |
| repo | pair |
| issue file | workshop/issues/000470-broadcast-web-pointer-and-notes-in-the-viewer-anchored-to-element-and-time-multi-viewer.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | e2d4f6d78340cdefe22317faef3a139f5039374e..4758c58740b747e30b9808f8531fd3a9761bb08c |
| command | sdlc milestone-close --issue 470 --milestone M1 |
| reviewer | claude |
| isolation | checkout |
| timestamp | 2026-10-12T03:22:16-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Wiring diff read; now checking the reducer's note handling and the recorder's Send/refusal paths for pending-leak cases.

Build failed on an unrelated embed pattern (generated assets missing in this checkout); running tests on the touched packages directly.

Touched webcast packages pass with -race; the artifactpath failures come from generated runtime assets missing in this checkout, not from this diff. Next: the NoteCrops seam and the tests.

Tests and design read through; running one mutation check on the recorder's document guard to confirm its regression test bites.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M1 delivers what it claims. `notes.Model.NoteCrops` now carries region notes into crops for both the real model and the fake, and the model contract covers the region case. The new `board` package parses requests strictly, counts notes still in flight toward the per-color limit, and passes the checked document to the writer. The writer re-checks that document when it orders the record, in the recorder (`recorder.Emit{Doc}`) and in the reference `board.Writer`, using one rule: `recording.State.OnDoc`. The POST routes use the same admission rules as the terminal pointer routes, and the atlas is updated.

I ran `go test -race` on every `webcast/...` and `broadcast` package and all passed. `artifactpath` fails only because `runtimebundle` assets are missing from this checkout. Those failures are environmental and not in this diff. As a mutation check I removed the recorder's `OnDoc` guard: `TestRecorderEmitChecksTheDocument` failed, and after restoring the line the tree was clean. Nothing blocks shipping. The one Important item is that the plan's Architecture section no longer matches the wiring that was built after #458 landed.

**Strengths**
- `cmd/internal/webcast/board/board.go:161-185`: the board checks a note, reserves its place as `pending` under the lock, emits outside the lock, and the reservation ends on emit error, on seeing the record come back, or on a new document. `TestColorLimitHoldsUnderConcurrency` (40 goroutines, exactly 8 land) and `TestSilentDropEndsPendingAtTheNextDocument` test this sequence.
- `recording.State.OnDoc` (`reduce.go:119`) is the only statement of the document check. The recorder (`recorder.go:506`) and `board.Writer.Emit` both call it (ARCH-DRY pass).
- `wire.go`: strict decoding with UTF-8 validated before the decoder, pointer fields so a missing field is distinguishable from a zero, exact arity on points and boxes, the reserved `operator` id refused, errors that never echo input, and `FuzzParse` seeded as the plan said.
- `server_board.go:166`: a single `boardAnswer` maps every outcome to a status and a fixed sentence, and the atlas lists the same table.
- `Crops` now delegates to `NoteCrops` (`anchorsOnly`), so the old and new crop paths can't drift apart, and `TestNoteCropsARegionByHand` checks that equivalence.

**Critical findings:** none.

**Important findings**
1. **The plan's Architecture section is stale against the built wiring** (ARCH-PURPOSE, plan traceability). The plan still says:
   - `CastConfig.Board *board.Board → NewServer(..., board)` and "couchcmd constructs a board".
   - `board.OnDoc(st, doc)`.

   The code instead:
   - builds the board inside `StartCast` (`cast.go:92`) and attaches it with `Server.WithBoard`.
   - puts the rule in `recording.State.OnDoc`.
   - forwards through `webcast.Emitter`/`Chrome.Emit`.

   The issue Log records the change, but the durable plan does not. Fix: add a `## Revisions` entry (below).

**Minor findings**
- `board.go:41/104`: once the board has refused a record (`broken`), it silently answers every later request with `ErrStale` ("The page changed. Try again."), forever, and logs nothing. Its `Put` also returns `ErrClosed` after `Close`, which contradicts the Log's "never fails the recording's Tee". Give it a distinct error or a log hook (ARCH-SECURE: degrade visibly).
- `cast.go:148-153`: the Tee feeds the Hub before the board. A viewer could act on a new `seg` it has already received before the board has applied it, and get a spurious 409. The window is microseconds; putting `bd` ahead of `live` in the Tee closes it (ARCH-ORDER).
- `wire.go` `boxOf`: the viewer's `box` is only bounded to ±1e6 px. Because crops widen the page to hold every note's own box, one bad box can produce a page-sized crop in wave 3. Consider clamping it to the element's layout box (ARCH-SECURE).
- The board keeps a second full `recording.State` (DOM plus every stroke and note for the cast's life) alongside the recorder's mirror. Strokes are bounded only by the rate limit (60/s × 64 pts). The board doesn't need old strokes; note it for a long-cast memory budget (ARCH-CONSTRAINTS/ARCH-FUNERAL).
- `board.go:142`: a stroke's record id is `viewer.id + "." + stroke id`. The atlas `stroke` row doesn't document this, and M2's JS store joins chunks by id.
- The refusal sentences differ from the plan ("That isn't valid.", "The cast isn't showing a page yet."). The atlas matches the code, so this is fine; just note it in the revision.
- `boardAnswer` maps `ErrNotRecording` (a cast that isn't an Emitter) to 503 with "Too many requests", which is misleading. It can't happen with production Chrome.

**Test coverage notes**
- Each named error has a test, at both board level (`TestRefusals`, `TestDocumentRules`) and route level (`TestBoardAnswers`). The check-then-emit race is tested with an injected navigation (`TestNavigationBetweenCheckAndWriteDropsTheNote`). A fixture replay covers pause, resume and tab switch. A cast-level test runs from the POST through to `notes.jsonl` and checks that the operator's resolve lands.
- The asynchronous production writer (the recorder queue, which returns nil and drops later) is covered only by a stub emit plus the recorder unit test. Nothing checks that `board.Writer` and `recorder.Emit` agree (ARCH-MOCK, minor).
- The rate-limit test depends on wall-clock time (`TestBoardColorLimitAndRate`, `time.Sleep`). It passed under `-race`, but `NewRateLimit` isn't fed an injected clock.

**Architecture**
- ARCH-DRY: pass. One palette, shared note limits, `broadcast.PlainText` exported and reused, one `OnDoc`.
- ARCH-PURE: pass. The parsers are pure, and the Board is pure apart from its lock and the injected `Emit`. HTTP admission lives in a thin `server_board.go`.
- ARCH-PURPOSE: pass for M1 scope. Region crops reach `NoteCrops`, and the board is wired to the real recorder. Plan drift is Important #1.
- ARCH-MOCK: pass with a note. `board.Writer` is the stateful reference writer behind the same `Emit` seam, but there's no conformance check against the recorder (above).
- ARCH-CONSTRAINTS: pass. Budgets are enforced (4 in flight, 60/s strokes, 2/s with burst 5 for notes, 2 s read deadline, `MaxBytesReader`). The duplicated State is a minor note.
- ARCH-SECURE: pass with minors. Bodies are parsed into typed values at the boundary, identity is documented as attribution only, `operator` is reserved, and text is plain text with no format characters. The viewer box bound and the silent `broken` state are noted above.
- ARCH-ORDER: pass. Pending entries are tagged by document, and their end conditions are listed and tested. `closed` and `broken` are terminal flags, not a combination of independent states. The Tee order is a minor note.
- ARCH-FUNERAL: pass. Pending entries end on emit error, on being seen, or on a new document. Subscriptions close at `Close`. Records go with the recording directory, and the atlas states the bound.

**Notes for upcoming work**
- M2's JS stroke store must join chunks on the composed `viewer.stroke` id.
- M3's resolve-as-operator path goes through `Cast.ResolveNote`, which validates against `model.Live`. A note the operator sees through `Subscribe` is always resolvable, but one still pending is not.

**Plan revision recommendations**
- `## Revisions` (M1, after #458 landed):
  - Wiring is `StartCast` builds `board.New(notes.Structural{}, emit→webcast.Emitter)`, tees it into the recording beside the DirWriter and the Hub, and attaches it with `Server.WithBoard`. There is no `CastConfig.Board`, and couchcmd constructs nothing.
  - The document rule is `recording.State.OnDoc`, called by `recorder.Emit{Record, Doc}` and `board.Writer`.
  - Couch's surface is `Cast.Notes`/`Cast.ResolveNote`.
  - Stroke record ids are `<viewer id>.<stroke id>`.
  - The refusal sentences are those listed in atlas/webcast-format.md.

```findings
findings:
  - id: new
    severity: Important
    family: plan-drift-after-rewire
    title: |
      Plan Architecture still describes CastConfig.Board, couchcmd construction and board.OnDoc; the code wires in StartCast/WithBoard and uses recording.State.OnDoc
    detail: |
      The issue Log records the change but the durable plan does not; add a Revisions entry covering the wiring, the OnDoc location, Cast.Notes/ResolveNote, the composed stroke id and the final refusal sentences.
  - id: new
    severity: Minor
    family: silent-degradation
    title: |
      A broken board answers ErrStale forever with no log, and Put returns ErrClosed after Close despite the never-fails-the-Tee claim
    detail: |
      board.go Put/pageLocked. Give it a distinct error or a log hook so the operator can see the board has stopped (ARCH-SECURE).
  - id: new
    severity: Minor
    family: sink-order-before-consumer-check
    title: |
      The Tee feeds the Hub before the board, so a viewer can act on a seg the board has not applied yet and get a spurious 409
    detail: |
      cast.go recordingOutput: append bd before live (ARCH-ORDER).
  - id: new
    severity: Minor
    family: untrusted-geometry-bound
    title: |
      A viewer-supplied note box is bounded only to 1e6 px and widens the page box in crops
    detail: |
      Clamp it to the element's layout box or the page before it reaches NoteCrops in wave 3 (ARCH-SECURE).
  - id: new
    severity: Minor
    family: duplicate-reducer-state-growth
    title: |
      The board keeps a second full recording.State, including every stroke for the cast's life, beside the recorder's mirror
    detail: |
      Bounded only by the stroke rate; the board needs neither old strokes nor older documents (ARCH-CONSTRAINTS/ARCH-FUNERAL).
  - id: new
    severity: Minor
    family: atlas-wire-detail
    title: |
      The atlas stroke row does not say the record id is the viewer id plus "." plus the stroke id, which M2's JS store joins on
  - id: new
    severity: Minor
    family: fake-conformance
    title: |
      Nothing checks that board.Writer (synchronous, returns ErrStale) and recorder.Emit (asynchronous, drops silently) behave the same
    detail: |
      ARCH-MOCK: a shared contract run over both writers would catch drift.
```

---

## Re-review — 2026-10-12T03:26:36-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 470 — broadcast-web: pointer and notes in the viewer — anchored to element and time, multi-viewer |
| repo | pair |
| issue file | workshop/issues/000470-broadcast-web-pointer-and-notes-in-the-viewer-anchored-to-element-and-time-multi-viewer.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | e2d4f6d78340cdefe22317faef3a139f5039374e..b9e409c6697e10ab4f30010b3ea0be541955f2d6 |
| command | sdlc milestone-close --issue 470 --milestone M1 |
| reviewer | claude |
| isolation | checkout |
| timestamp | 2026-10-12T03:26:36-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Prior-round fixes check out in the code so far. Next I'm checking that the tests exist and running them.

Tests pass. Now reverting the board-before-Hub ordering to see whether any test catches it.

Checking a few edge cases now: stroke-id collisions, and whether the recording can reopen after the board has closed.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

**Summary.** M1 does what the plan, as revised, says it does. The `board` package is in place: strict parsing, the named refusals, pending entries that count toward a color's limit, and the document re-check at write time (`recording.State.OnDoc`). It is wired through `StartCast`, `Chrome.Emit` and `recorder.Emit{Doc}`, and `Server.WithBoard` serves the three POST routes. `go test -race ./cmd/internal/webcast/...` passes. Of the seven prior findings, four are fixed with evidence (BR-1, BR-2, BR-5, BR-6). Three stay open, all Minor:
- **BR-3:** the sink order is fixed, but no test fails without it.
- **BR-4:** deferred to wave 3, with a reason.
- **BR-7:** declined, with a reason.

Nothing blocks. The one cheap fix worth making first is a test that pins the board-before-Hub order.

### 1. Strengths
- **One ordering rule.** `recording.State.OnDoc` (`recording/reduce.go:123`) is used by both the recorder at dequeue (`recorder.go:506`) and `board.Writer.Emit` (`writer.go:40`). `TestRecorderEmitChecksTheDocument` also checks that a dropped record isn't counted as refused.
- **Locking.** The board's lock is a leaf lock: emit and subscriber sends happen outside it, and `refreshLocked` empties the one-slot buffer before sending, so the deadlock the plan predicted through Tee → `Put` can't happen.
- **A broken board fails visibly.** It answers `ErrBroken` (503, logged once through `Logf`, which is wired from `cfg.Logf` at `cast.go`). `Put` never fails the recording's Tee, including after `Close`. `TestBrokenBoardStopsAndSaysSo` pins all of this.
- **One mapping from outcome to answer.** `boardAnswer` (`server_board.go:167`) holds fixed sentences that never echo input. The id regex excludes `.`, so the composed stroke id `<viewer>.<stroke>` can't collide.
- **Region notes have a contract.** `NoteCrops` is covered by the shared `RunModelContract` (`notes/contract.go:108`), so the fake and `Structural` are held to the same region behavior.

### 2. Critical
None.

### 3. Important
None.

### 4. Minor
- **BR-3 (still open):** I swapped `bd` and `live` in the sink list in `recordingOutput` (`cast.go`), and `go test -race ./cmd/internal/webcast/` still passed. Nothing pins the order. Fix sketch: build the sinks list in a small helper, or expose the Tee's order, and assert `[dir, board, hub]`. Or write a test where a Hub subscriber posts a note as soon as it receives a `doc` record and expects 204.
- **BR-4 (deferred, accepted as deferred):** `NoteCrops` has no production caller outside `notes/` yet, so the clamp belongs with wave 3's submission. It still needs doing there.
- **BR-7 (declined):** the reasoning is sound for the rule they share, but the gap is real. Server and cast tests run against `board.Writer`, which answers a navigation race with 409. Production answers the same race with 204 and drops the note silently. Tests at this seam see a different answer than viewers do.
- **Misleading answer (not filed as a finding):** an emit failure (`ErrNotRecording`, or a writer error) maps to 503 with the busy sentence "Too many requests. Wait a moment." (`server_board.go:186`), which tells the viewer the wrong cause.

### 5. Test coverage notes
I reverted fixes to check that tests catch them:
- **BR-3 order swap:** stays green, so there is no regression evidence.
- **BR-5:** pinned by the `len(r.bd.st.Strokes) == 0` check (`board_test.go:431`).
- **BR-2:** pinned by the same test's `ErrBroken`, log-once and Put-after-Close assertions.

The issue Log also reports two mutation checks I didn't re-run: the pending-count test and the silent-drop test.

### 6. Architecture (ARCH-*)
- **ARCH-DRY: pass.** One palette, the shared note limits (`broadcast.PlainText` is now exported for reuse), one OnDoc rule, and `Crops` defined as `NoteCrops` without regions.
- **ARCH-PURE: pass.** `Parse*` functions are pure; the Board is pure apart from its lock; the routes are a thin admission layer.
- **ARCH-PURPOSE: pass for M1's scope.** Viewer UI and Couch's `💭N` are M2 and M3 as planned.
- **ARCH-MOCK: minor gap (BR-7).** The reference Writer's synchronous `ErrStale` differs from the recorder's silent drop.
- **ARCH-CONSTRAINTS: pass.** Budgets are enforced per cast (strokes 60/s, notes and states 2/s with a burst of 5, 4 in flight, a body read deadline).
- **ARCH-SECURE: pass.** Parsing is strict, `operator` is reserved, refusals never echo input; the box clamp is deferred (BR-4).
- **ARCH-ORDER: pass on the race the plan named.** Pending entries end on seen-back, on emit error, or at a new document. The sink-order rule lacks a test (BR-3).
- **ARCH-FUNERAL: pass.** The board keeps no strokes, pending entries die as listed, the board is closed with the recording or the cast, and `Out` is opened only once (`recorder.go:621`), so a closed board is never teed into a later recording.

### 7. Plan revisions
None needed. The 2026-10-12 Revisions entry matches the code: the wiring, `State.OnDoc`, `Cast.Notes`/`ResolveNote`, the stroke id, the answers and the budgets.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Plan Revisions 2026-10-12 covers the StartCast/WithBoard wiring, State.OnDoc, Cast.Notes/ResolveNote, the stroke id, the answers and budgets; all checked against cast.go, chrome.go, recorder.go and server_board.go.
  - id: BR-2
    disposition: addressed
    note: |
      ErrBroken (503, logged once through Logf, wired from cfg.Logf); Put returns nil after Close; pinned by TestBrokenBoardStopsAndSaysSo.
  - id: BR-3
    disposition: not-addressed
    note: |
      The order is fixed in recordingOutput, but swapping bd and live back leaves go test ./cmd/internal/webcast green; add a test that pins dir, board, hub.
  - id: BR-4
    disposition: not-addressed
    note: |
      Deferred to wave 3 with a reason; NoteCrops has no production caller yet, so it is still open but not due at M1.
  - id: BR-5
    disposition: addressed
    note: |
      Board.Put truncates st.Strokes after each Apply (board.go:126); pinned by board_test.go:431.
  - id: BR-6
    disposition: addressed
    note: |
      The atlas stroke row now says the id is the viewer id, a dot, then the shell's stroke id.
  - id: BR-7
    disposition: not-addressed
    note: |
      Declined with a reason; the gap remains: tests on board.Writer see 409 for a navigation race where production answers 204 and drops silently.
```
