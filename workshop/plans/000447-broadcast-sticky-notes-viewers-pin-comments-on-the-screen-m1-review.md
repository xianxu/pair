# Boundary Review — pair#447 (milestone M1)

| field | value |
|-------|-------|
| issue | 447 — Broadcast sticky notes: viewers pin comments on the screen |
| repo | pair |
| issue file | workshop/issues/000447-broadcast-sticky-notes-viewers-pin-comments-on-the-screen.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 1dea34113f9e040f2fb548192906acfe19dc0f74..a75133cda8eb5134ebf176c963a6520fd96725d8 |
| command | sdlc milestone-close --issue 447 --milestone M1 |
| reviewer | claude |
| isolation | checkout |
| timestamp | 2026-10-11T14:10:32-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Reviewing the M1 diff now. Next I'm reading the core notes, pointer, server and session code.

The broadcast tests pass under the race detector. The couchtty package won't build in this checkout because its generated runtime assets are missing. I'm looking for how those assets get made.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M1 delivers what it promises: the note model, strict parsing, drawing, the board inside `PointerState` under its one lock, the `note`/`resolve` routes, the `notes` event to every stream, the session checks, and the 💭 overlay plus the `💭N` count in Couch. The code matches the Spec and the plan's Revisions, including the generation-guarded add and the monotonic version. `go test -race ./cmd/internal/broadcast` passes. The notes, pointer, status-row and broadcast tests in couchtty pass under `-race` three times over. The full couchtty package fails here, but only on PTY tests that can't start a process in this environment (`ptychild: start … operation not permitted`) and a soak-test race inside the `vt` emulator's cleanup. None of those failures touch this diff. One thing stands between this and SHIP: the README isn't updated for the new operator-visible `💭N` count, even though README §"Remote pointer (#412)" describes that same status row. That fix is cheap. The rest is Minor.

Note: couchtty would not build until I ran `make runtimebundle-generate`, which writes `cmd/internal/runtimebundle/assets/runtime/`. That happened only in this disposable checkout.

### 1. Strengths
- **Ordering design (ARCH-ORDER):** clearing the notes happens in the same critical section as the flip to off (`pointer.go:103-105`, `:135`). `addNote(n, gen)` refuses when the generation is stale (`pointer_notes.go:30`), and the frame check runs without the lock (`session.go` `landsNow`). `TestPointerNotesAddFlipInterleavings` covers each interleaving of one add and one flip through the generation, which is a real ordering seam, not a single sample.
- **Pure core (ARCH-PURE):** `NoteBoard`, `ParseNote`, `NoteVisible` and `DrawNotes` are pure. Their tests need no IO, and they include `FuzzParseNote` and a property test showing `DrawNotes` gives the same result in any note order. `DrawNotes` reads the output frame, so neighbouring notes come out the same in either order, and the property test checks this.
- **One shared admission path (ARCH-DRY):** `readPointerBody` (`server.go`) keeps the original order: pointing on, content type, slot, rate check, deadline, body. The slot is held through handling and released on every early exit. `landsNow` is factored out of `acceptPoint`.
- **Untrusted input (ARCH-SECURE):** the body is checked as valid UTF-8 before decoding, so invalid bytes can't become U+FFFD. Control and Cf characters are refused. The refusal sentences are a closed set that never echo input. The SSE payload goes through `json.Marshal`, so a newline in a note can't inject an event. The view link still answers 405 on POST for `note` and `resolve` (`pointer_test.go` route table, `server_test.go` routes).
- **Real end-to-end test:** `TestNoteDrawnOnOperatorAndViewers` runs a real session and console. It POSTs a note over HTTP, checks the 💭 in both the operator's emulator and a viewer's, checks `💭1` in the tab bar, and checks that resolve removes both.

### 2. Critical findings
None.

### 3. Important findings
- **README update appears missing for the `💭N` tab-bar count.** `README.md:876-891` documents the live status row as `LIVE ⏸ 👆 👽`. M1 adds an operator-visible `💭N` after `👽` (`reserve.go:227-230`), and turning pointing off now deletes every note. Fix: add a sentence or two to the "Remote pointer" paragraph covering the count, that it isn't clickable, and that turning pointing off deletes all notes. The viewer-side how-to can wait for M2.

### 4. Minor findings
- **ARCH-DRY:** `ParsePointBatch` (`point.go:52-58`) still has its own copy of the strict decode logic that `decodeStrict` (`notes.go:140`) now provides. Have `ParsePointBatch` call `decodeStrict`.
- **Untested path:** `forgetNotes`, called from `resetPointer` (`console_pointer.go:238`), has no test. `TestNotesIgnoredAfterBroadcastEndsOrFromAnotherSession` stops a broadcast that holds no notes, so plan Task 6's "after the broadcast stops … the count is 0" is never exercised with notes present. The overlay is removed at stop (`console_broadcast.go:291`), so this guards only the held snapshot.
- **Per-color limit can be dodged:** the color comes from the client (`ParseNote`) and isn't tied to the stream's claimed color, so one viewer can spread notes across all six colors. The 48-note total cap and the rate limit still apply, and the setting is collaborative by design.
- **Signature drift:** the plan says `resolveNote(id int) error`; the code has `(bool, error)` (`pointer_notes.go:41`). Harmless.
- **Two clones per frame:** `screenOverlay` can clone each public frame twice (notes, then marks) while both are present. Not on a hot path at 48 notes; noted for ARCH-CONSTRAINTS.
- **Smoke-test substitution:** Task 7's curl smoke was replaced by an automated end-to-end test, and the `## Log` says so. Acceptable.
- **Stale comment:** `drawnNotes`' comment says "empty unless pointing is on", but the function doesn't check that. It holds only because every phase change resyncs. Reword the comment.

### 5. Test coverage notes
- Plan Task 4 (a)–(j) and Task 5 are covered by `notes_session_test.go` and `pointer_notes_test.go`. That includes the hub-loop deadlock test (`TestNotesDoNotDeadlockWithPointerWatch`), the stale watch report, and a `-race` stress run.
- Plan Task 6 is covered: version ordering, another session, after stop, private frames, pointing off, and a pre-off snapshot dropped after re-enable. The one gap is the snapshot-cleared-at-stop case above.
- The fake `OnNotes` in `startPointerSessionWith` records calls on the real session, so the limits and versions under test are the production ones, not mocks.

### 6. Architectural notes for upcoming work (M2)
- **ARCH-DRY:** the guard width already goes over the wire as `guard`. Keep `noteVisible` in JS a strict mirror of `NoteVisible` (`col+1 < cols`, plus the guard), and consider shared test vectors so the two can't drift.
- **ARCH-SECURE:** the dialog must render text only through `textContent`. Plan Task 10 already tests that `innerHTML` is never assigned; keep that test.
- Checks for the other principles:
  - ARCH-PURPOSE: pass. M1 is everything on the Couch side; the viewer page is legitimately M2.
  - ARCH-MOCK: pass. Tests use `FakeTunnel` with real hub and server.
  - ARCH-CONSTRAINTS: pass. Two requests/s with a burst of 5, an 8 KB body cap, and at most 48 notes of 1 KB each.
  - ARCH-FUNERAL: pass. Notes live in memory only and are cleared on every turn-off and at the end of the broadcast. The `onNotes` goroutines are bounded by `c.lifetime` and the rate limit.

### 7. Plan revision recommendations
- Optional: "2026-10-11 — M1 review: `PointerState.resolveNote` returns `(changed bool, err error)` so the session reports only real changes; the status-row tests live in `reserve_broadcast_test.go`."

```findings
findings:
  - id: new
    severity: Important
    family: docs-gate-user-surface
    title: |
      README update appears missing for the 💭N tab-bar count and notes cleared on pointing off
    detail: |
      README.md:876-891 documents the live status row as LIVE ⏸ 👆 👽; M1 adds an operator-visible 💭N after 👽 (reserve.go:227-230) and deletes all notes when pointing turns off. Add a sentence or two to the Remote pointer paragraph.
  - id: new
    severity: Minor
    family: shared-helper-not-reused
    title: |
      ParsePointBatch duplicates the strict-decode logic now in decodeStrict (ARCH-DRY)
    detail: |
      point.go:52-58 repeats DisallowUnknownFields plus the trailing-token check that notes.go:140 decodeStrict provides; have ParsePointBatch call decodeStrict.
  - id: new
    severity: Minor
    family: untested-cleanup-path
    title: |
      forgetNotes at broadcast end is not exercised with notes held
    detail: |
      TestNotesIgnoredAfterBroadcastEndsOrFromAnotherSession stops a broadcast that holds no notes, so plan Task 6's count-is-0-after-stop case with notes present is not tested.
  - id: new
    severity: Minor
    family: client-asserted-identity
    title: |
      Per-color note limit uses a client-supplied color, so one viewer can use all six colors
    detail: |
      ParseNote accepts any color 0-5 that is not tied to the stream's claimed color; the 48-note total cap and the rate limit still hold, and the setting is collaborative by design.
  - id: new
    severity: Minor
    family: comment-claims-unenforced-invariant
    title: |
      drawnNotes comment says empty unless pointing is on, but the function does not check
    detail: |
      This holds only because every pointer-phase change resyncs the snapshot (console_notes.go); reword the comment to say that.
```

---

## Re-review — 2026-10-11T14:22:15-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 447 — Broadcast sticky notes: viewers pin comments on the screen |
| repo | pair |
| issue file | workshop/issues/000447-broadcast-sticky-notes-viewers-pin-comments-on-the-screen.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 1dea34113f9e040f2fb548192906acfe19dc0f74..1c3a0b160976359ea6565092c6ff5dd8017f3feb |
| command | sdlc milestone-close --issue 447 --milestone M1 |
| reviewer | claude |
| isolation | checkout |
| timestamp | 2026-10-11T14:22:15-07:00 |
| verdict | SHIP |

## Review

Let me look at the last fix commit and the prior-finding sites.

Disposing the six prior findings and checking the console wiring; next I'll run the tests.

Generating the embedded runtime assets so the couchtty tests can build, then running them.

PTY failures are environmental (no pty in this checkout) and untouched by the diff. Running the notes-specific tests repeatedly under `-race`:

Checking the BR-4 fix by reverting `forgetNotes` and confirming the new test fails:

```verdict
verdict: SHIP
confidence: high
```

M1 delivers the Couch side of sticky notes as the plan describes. There is a pure note board, strict parsing and drawing in `notes.go`. The board sits under the pointer lock, and an add only lands if pointing has not been turned off or on since the caller read the generation. Turning pointing off empties the board in the same critical section. The `note` and `resolve` routes reuse the `point` route's admission checks through `readPointerBody`, and every stream gets an `event: notes`. The console keeps a versioned snapshot and resyncs it from the session whenever pointing turns on or off. The last commit fixes three of the prior findings with real evidence (BR-2, BR-3, BR-6). For BR-4 I removed the `forgetNotes` call and the new test failed, so its regression test is real. BR-1 is still open: it is a stale sentence in the plan, Minor, and not blocking. BR-5 is withdrawn as consistent with the trust model in the spec. I'm raising one new Minor finding: the README describes viewer-page note features that only arrive in M2. Nothing blocks shipping.

**Inspection:** I ran the stat and name-status recipes, then the targeted and full diffs. I ran `go test -race` on `broadcast` (ok), and on the notes, reserve and pointer tests in `couchtty` with `-count=5` (ok). Building `couchtty` and running `artifactpath` first needed the embedded runtime assets, which I generated with `make runtimebundle-generate` (they aren't checked in); after that `artifactpath` passed. With the full `couchtty -race` run, the PTY tests failed here (`ptychild: start …: operation not permitted`), and the race detector flagged a data race in `terminal_soak_test.go`'s cleanup after that test failed. All of that is in test code the diff doesn't touch.

1. **Strengths**
   - `pointer.go:100-105,135`: turning pointing off and clearing the board share one critical section, and `addNote` checks the generation (`pointer_notes.go:30`). A note can't land after the clear, and its frame check doesn't run under `p.mu`.
   - `TestPointerNotesAddFlipInterleavings` uses the generation to inject orderings, so it covers each ordering of one add against one flip, not just whichever ordering the scheduler picks.
   - `console_notes.go:27`: `onNotes` hands off to the Run loop without waiting, which avoids a deadlock when a 👆 click on that loop reports the emptied board. The versioned `applyNotes`, together with the resync, drops stale snapshots. `TestNotesGoWhenPointingTurnsOff` checks this both while off and after pointing is turned back on.
   - `server.go:160-198`: the three routes share one admission path, refusals are a closed set of sentences, and the view link still never reads a body.
   - `notes.go:200-207`: `DrawNotes` blanks cut cells while keeping their style, and widens `UsedColumns`. A property test covers note order and leaves no orphaned half of a wide character.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - Plan Task 3 Step 3 (plan:147) still says `uv.EmptyCell`; this is BR-1, still open.
   - `README.md:894-901` presents viewer-page actions as available now ("People on the pointer link can also pin…", "anyone on either link can tap a 💭 to read it"). M1 ships no viewer UI for notes; they can only be posted with `curl` until M2.

5. **Test coverage notes:** Every case the plan names for Tasks 1–6 is present. That includes the fuzz test, the `DrawNotes` property test, the stress test with injected ordering, the hub-deadlock test and the new broadcast-end test. The console's watch-driven turn-off is only covered through the session test, but it goes through the same `pointerOff` → `resyncNotes` code as the click path.

6. **Architecture**
   - **ARCH-DRY: pass.** `decodeStrict`, `landsNow`, `readPointerBody` and `guarded` are each shared.
   - **ARCH-PURE: pass.** The board, parsing, visibility and drawing are pure and tested without IO.
   - **ARCH-PURPOSE: pass.** M1 covers the whole Couch side, as split in the plan.
   - **ARCH-MOCK: pass.** There are no new external dependencies, and the tests use `httptest` and the existing fake tunnel.
   - **ARCH-CONSTRAINTS: pass.** Notes are capped at 48 and 1 KB each, rate-limited to 2/s with a burst of 5, and request bodies to 8 KB. `onNotes` starts one goroutine per change, which the rate limit bounds and `c.lifetime` ends.
   - **ARCH-SECURE: pass.** Parsing is strict and refuses invalid UTF-8 and format characters (bidi overrides, zero-width marks). Errors are fixed values that never echo input. The color is client-asserted, as BR-5 notes and the spec accepts.
   - **ARCH-ORDER: pass.** The generation-conditioned add and the snapshot version only ever increase. The tests reproduce the orderings that matter by injecting them.
   - **ARCH-FUNERAL: pass.** Notes live only in memory and are deleted when pointing turns off or the broadcast stops; the console calls `forgetNotes` at the end.

   For M2, `notes.js` should take the guard width from the event, as planned, and not copy `StatusGuardCols`.

7. **Plan revisions:** Replace "using `f.Clone()` and `uv.EmptyCell` for blanks" in Task 3 Step 3 with a pointer to Core concepts ("blank as Core concepts' DrawNotes entry says"), and add a `## Revisions` line recording BR-1.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Plan line 147 still says "f.Clone() and uv.EmptyCell for blanks"; the code is correct (notes.go:205 blank keeps style), only the plan step is stale.
  - id: BR-2
    disposition: addressed
    note: |
      README.md:894-901 adds the Sticky notes paragraph covering the 💭N count after the controls and that pointing off deletes notes; matches reserve.go:227-230 and pointer.go:103.
  - id: BR-3
    disposition: addressed
    note: |
      point.go:50 now calls decodeStrict; the broadcast point tests pass unchanged.
  - id: BR-4
    disposition: addressed
    note: |
      TestNotesForgottenWhenBroadcastEnds (console_notes_test.go:113) fails when the forgetNotes call in resetPointer is removed ("console still holds 1 notes (v1)"), so the fix is reachable.
  - id: BR-5
    disposition: withdrawn
    note: |
      Consistent with the spec's trust model: the pointer link is a shared capability, point colors (#446) are client-asserted too, and the 48-note total cap plus rate limit are the real bound.
  - id: BR-6
    disposition: addressed
    note: |
      console_notes.go:78-80 comment now says emptiness relies on the resync at each pointer-phase change; applyNotes also gates on pointer==pointerOn.
findings:
  - id: new
    severity: Minor
    family: docs-gate-user-surface
    title: |
      README describes viewer-page note actions that only land in M2
    detail: |
      README.md:894-901 says pointer-link viewers can pin notes and anyone can tap a 💭 to read it, but M1 ships no viewer UI (notes are curl-only until M2 Tasks 8-10). This is the 2nd finding in this family; the rule: README changes describe only the surface present at the boundary's head, so tag M2-only behavior as upcoming or add it with M2.
```
