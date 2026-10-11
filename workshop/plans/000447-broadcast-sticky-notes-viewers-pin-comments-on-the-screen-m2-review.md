# Boundary Review — pair#447 (milestone M2)

| field | value |
|-------|-------|
| issue | 447 — Broadcast sticky notes: viewers pin comments on the screen |
| repo | pair |
| issue file | workshop/issues/000447-broadcast-sticky-notes-viewers-pin-comments-on-the-screen.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | ed708490599e897fea71c69359314cc653155e5c..e3468f3992546d909319ab6c3dc77350cf57da88 |
| command | sdlc milestone-close --issue 447 --milestone M2 |
| reviewer | claude |
| isolation | checkout |
| timestamp | 2026-10-11T15:12:40-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Read the plan; now inspecting the M2 code diff.

Tests pass; checking the issue Log for the M2 smoke evidence and the end-of-broadcast path.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M2 does what the plan says it should. The viewer gets its pure helpers (`noteVisible`/`noteAt`/`noteCell`/`noteBytes`/`parseNotes`). Taps are routed in a fixed order: 💭 first, then pointing off, then comment mode, then a stroke. There is one fixed-size dialog that only ever uses `textContent`, and the page's single `fetch` sits behind a route allow-list. The 56 node tests and `TestViewerNode`/`TestViewerPageLoadsOnlySameOrigin`/`TestServerServesAssetsWithHeaders` all pass on the pinned head, and `go test ./cmd/internal/broadcast` passes. `go test ./cmd/internal/couchtty` couldn't build in this checkout because the generated `runtimebundle` assets are missing. That's an environment problem, and M2 doesn't touch couchtty. Two things stand between this and SHIP, and neither blocks the gate:
- **Stream end:** the viewer keeps its notes list, pointing state and open dialog after `event: end`, so stale notes can still be opened on a blank screen.
- **Late responses:** a save or resolve response isn't tied to the dialog that sent it, so a late 204 can close a newer draft.

The carried plan finding BR-1 is still open.

**1. Strengths**
- `viewer.js` `postJSON` + `POST_ROUTES`: the old single-`fetch` page test was narrowed rather than dropped. It is pinned in `server_test.go:161-184`, and HTML sinks (`innerHTML`, `outerHTML`, `insertAdjacentHTML`, `document.write`) are banned in both scripts.
- The dialog's state is a tagged union (`null | {kind:'draft'} | {kind:'view'}`) rather than a set of separate flags, which is good on ARCH-ORDER.
- The guard width comes over the wire in the notes event instead of being copied into JS (`notes.js:16-24`). That is the only Go/JS mirror, and it's justified.
- `noteDialog` takes its elements and `post` as injected arguments. The fake element in its test throws if `innerHTML` is ever assigned (`notes.test.mjs` `el()`), so the plain-text rule is tested as behaviour, not just as a source grep.
- The tap-routing tests cover every branch the plan lists, including that a tap on the guarded controls opens no draft and that comment mode never posts anything.

**2. Critical findings**
None.

**3. Important findings**
- **`viewer.js:200-206` with `start()` wiring at `viewer.js:380-395`: `event: end` leaves note state live (ARCH-ORDER).** On `end`, `deps.reset()` only resets the terminal. `notes`, `pointer.on`, the mode button and any open dialog all stay.
  - The server's event loop at `server.go:358-374` picks among end, caps and notes in a random order, so the empty notes list from `stop` isn't guaranteed to arrive before `end`.
  - Result: after the broadcast ends, the screen is blank, yet tapping where a 💭 was opens the old note with a Resolve button, and comment mode still opens drafts. Both fail with "Couldn't reach the broadcast."
  - Fix sketch: on the terminal event, reset everything derived from the stream: `notes = {guard:0,notes:[]}`, `pointer.setOn(false)` (which already closes the dialog and resets the mode). Add a connect/start test.

**4. Minor findings**
- **`notes.js:74-81`: a late response isn't checked against the current dialog.** If you Cancel while a save is in flight and open a new draft, the old request's 204 runs `close()` and throws away the new draft's text. A late 409 shows its refusal on the new draft. Fix: capture a draft token when the request starts and ignore the response if `state` has changed.
- **Escape doesn't close a viewed note.** The listener is on the dialog root (`notes.js:89`), and only a draft moves focus inside it. The dialog also says `aria-modal="true"` but doesn't trap focus.
- **README:** the Sticky notes paragraph (README.md:894) never says how a viewer places a note (the 👆/💭 mode button below the swatches).
  - **This is the 3rd finding in family `docs-gate-user-surface`.** The rule: the README's feature paragraph names every control a person uses for the feature, and every milestone that adds or removes a control updates it.
  - M3 removes the hint bar and M4 adds the menu, so fold this into M3's README task.
- **Plan drift:** Task 8 gives `noteAt(notes, col, row, cols, rows, guard)`, but the code has `noteAt(list, col, row, cols, rows)` with the guard carried inside `list`.

**5. Test coverage notes**
- The tap-routing, dialog and connect tests cover the Task 9 and Task 10 lists.
- Missing:
  - nothing tests the end-of-broadcast reset (the Important finding above);
  - nothing tests a late or out-of-order save response.
- Rejection of an unknown route by `postJSON` is only pinned by grepping the source text. That's acceptable given how small the function is.

**6. Architecture**

| Principle | Result | Why |
|---|---|---|
| ARCH-DRY | pass | `postPoint` now goes through `postJSON`. Hint text lives only in `drawMode`. The visibility mirror is justified and shares only the guard, which arrives over the wire. |
| ARCH-PURE | pass | The helpers are pure. The dialog and `pointerMode` take their IO as injected dependencies. |
| ARCH-PURPOSE | pass | Every M2 deliverable is present. The live walk was replaced by headless Chrome plus the operator's smoke, which is recorded in Revisions. |
| ARCH-MOCK | N/A | No external binary. The DOM and `fetch` are faked at the injected `post` and element seams. |
| ARCH-CONSTRAINTS | pass | A tap is a linear scan of at most about 48 notes. |
| ARCH-SECURE | pass | The notes event is parsed into typed entries. Server refusal text and note text go only through `textContent`. |
| ARCH-ORDER | flag | The end event doesn't reset derived state, and late completions aren't tied to the draft that sent them. |
| ARCH-FUNERAL | pass | Nothing durable is created. The viewer's state lives in memory and dies with the page. |

For M3 and M4: when the hint bar and `band()` are removed, keep the reset on `end` in the same place as `setOn(false)`. M4's terminal menu has the same "close when the snapshot empties" lifecycle, so reuse that idea there.

**7. Plan revision recommendations**
- In Task 3 Step 3, replace "using `f.Clone()` and `uv.EmptyCell` for blanks" with "using `f.Clone()`; blanks per Core concepts (DrawNotes)". This is BR-1.
- Task 8: change the signature to `noteAt(list, col, row, cols, rows)`, with the guard taken from `list.guard`.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Plan line 147 (Task 3 Step 3) still reads "using f.Clone() and uv.EmptyCell for blanks"; code blank() at notes.go:208 keeps style, so only the plan text remains wrong.
findings:
  - id: new
    severity: Important
    family: stream-end-resets-derived-state
    title: |
      Viewer keeps notes, pointing-on, mode button and open dialog after event end (viewer.js:200-206, 380-395)
    detail: |
      deps.reset only resets the terminal; server.go selects end/caps/notes in random order, so stale notes stay tappable (with Resolve) on the blank ended screen and comment mode still drafts. Reset all stream-derived state (notes list, setOn(false)) on the terminal event and add a test. ARCH-ORDER.
  - id: new
    severity: Minor
    family: late-completion-identity
    title: |
      noteDialog applies a save/resolve response to whatever dialog is open now, not the one that sent it (notes.js:74-81)
    detail: |
      Cancel during an in-flight save and open a new draft: the old 204 closes the new draft (text lost), or an old 409 shows on it. Capture a token per request and drop mismatched completions. ARCH-ORDER.
  - id: new
    severity: Minor
    family: docs-gate-user-surface
    title: |
      README Sticky notes paragraph never says how a viewer places a note (the 👆/💭 mode button)
    detail: |
      3rd finding in this family. Rule: the README feature paragraph names every control a person uses for the feature, and each milestone that adds or removes a control updates it; fold into M3's README task (hint bar removal, tooltips).
  - id: new
    severity: Minor
    family: dialog-keyboard-reach
    title: |
      Escape does not close a viewed note; aria-modal dialog has no focus move or trap in view mode
    detail: |
      The keydown listener is on #note, and only a draft moves focus into it (notes.js:89, draft focuses input).
  - id: new
    severity: Minor
    family: plan-self-consistency
    title: |
      Plan Task 8 gives noteAt(notes, col, row, cols, rows, guard); code takes noteAt(list, col, row, cols, rows) with list.guard
    detail: |
      2nd in family. Rule: task steps cite Core concepts for signatures rather than restating them, which also covers BR-1; update Task 8 to match.
```

---

## Re-review — 2026-10-11T15:15:54-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 447 — Broadcast sticky notes: viewers pin comments on the screen |
| repo | pair |
| issue file | workshop/issues/000447-broadcast-sticky-notes-viewers-pin-comments-on-the-screen.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | ed708490599e897fea71c69359314cc653155e5c..706a32676d10c63478c5fdae9f61984e62812423 |
| command | sdlc milestone-close --issue 447 --milestone M2 |
| reviewer | claude |
| isolation | checkout |
| timestamp | 2026-10-11T15:15:54-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Checking whether view links ever receive `caps`, since the new end handler now synthesizes one.

Fix for BR-8 synthesizes a `caps` event even on view links (which never receive caps). Now running the JS tests and doing revert checks on the claimed fixes.

All three claimed fixes go red when reverted. Now checking the plan's Task 3/Task 8 wording and M2 checklist.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

All four executable fixes in 706a3267 hold. When I put the pre-fix `notes.js` and `viewer.js` back, three regression tests fail: BR-8 end reset, BR-9 stale answer, and BR-11 view focus. After restoring the files, the node viewer suite passes (59/59) and `go test ./cmd/internal/broadcast` is green. The prose fixes for BR-10 (README) and BR-12 (Task 8) are in place. Nothing blocks the boundary. One plan nit is still open: BR-1, Task 3 Step 3 still says "`uv.EmptyCell` for blanks". The BR-8 fix also adds one small new problem. It sends a made-up `caps` event on *every* page, but the server only sends `caps` to pointer pages (`server.go:325-340`). So when a broadcast ends, a view-only page now shows the "Pointing is off" hint band it never showed before.

1. **Strengths**
   - `noteDialog` binds each answer to its dialog with a generation counter (`notes.js:62,79-91`). It is simple, every open and close bumps it, and a test drives the real cancel → new draft → late 204 sequence (`notes.test.mjs:157`).
   - The end handler no longer depends on the order in which `end`, `caps` and `notes` arrive (`viewer.js:200-212`). Because `events.close()` runs first, no late event can bring state back afterwards.
   - Every request goes through one `postJSON` with a route allowlist (`viewer.js:409-426`), and note text is only ever set as `textContent`. The page test enforces both, so the security surface is narrow.
   - The operator smoke on a real browser (headless Chrome over CDP, plus the 390px viewport check) is recorded in the issue Log, so Task 11 is evidenced.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - `viewer.js:208`: on a view link, the made-up `caps({pointer:false})` runs `setOn(false)`. That un-hides `#hint` with the text "Pointing is off" and calls `band()`, which lifts the stage. Both go against "a view page has none and keeps the whole window" (`viewer.js:329-330`). This is the **2nd finding in family `stream-end-resets-derived-state`**.
     - **Rule:** an end must put each piece of stream-derived state back to the state that page variant had *on join*. It must not replay events the variant never receives.
     - **Fix:** `connect` remembers whether a `caps` event was ever seen and only sends `caps(off)` if so. Alternatively, give the viewer a single `ended()` dep that owns the whole reset. Add a test where a view-link harness gets `end` and the caps callback is not called.
     - M3 Task 14 removes the hint bar, which will hide most of the symptom, but the stray `band()` call stays.
   - BR-1 is still open (`plan.md:147`).

5. **Test coverage notes**
   - The new tests check what the user sees (the dialog stays open, the input text survives, Escape closes the dialog). They don't just repeat the implementation.
   - The BR-8 test checks the last `caps`/`notes` value in only one interleaving (caps, then notes, then end). That is enough here, because `end` closes the stream.
   - No test covers a view link receiving `end`; that is the gap behind the new Minor finding.
   - BR-11: a viewed note now moves focus into the dialog, so Escape works. There is still no focus trap under `aria-modal`, in either mode. That's acceptable for now.

6. **Architectural notes (ARCH-\*)**
   - **ARCH-DRY: pass.** `noteVisible` mirrors Go's `NoteVisible` on purpose, and the guard width comes from the server rather than a JS constant.
   - **ARCH-PURE: pass.** The `notes.js` helpers are pure, and the dialog and `pointerMode` take fake elements and deps.
   - **ARCH-PURPOSE: pass.** The full viewer-side scope of M2 is delivered.
   - **ARCH-MOCK: pass (N/A).** There are no new external dependencies; the EventSource fake sits at the `connect` seam.
   - **ARCH-CONSTRAINTS: pass.** Taps are O(n) over at most 8 notes per color × 6 colors.
   - **ARCH-SECURE: pass.** `parseNotes` drops malformed entries, text only goes into `textContent`, and routes are allowlisted.
   - **ARCH-ORDER: pass for the dialog.** The generation counter makes stale answers unrepresentable. The end-reset ordering is fixed. The view-variant gap is the Minor finding above.
   - **ARCH-FUNERAL: pass.** The page creates nothing durable: the draft lives only in the textarea, and the notes list dies with the page.
   - For M3/M4: `pointerMode` now has two booleans and a mode string (`on`, `mode`, plus the open dialog). If M4 adds terminal-menu states, turn them into one tagged state rather than adding more flags.

7. **Plan revision recommendations**
   - Task 3 Step 3 (`plan.md:147`): replace "using `f.Clone()` and `uv.EmptyCell` for blanks" with "implement per Core concepts → DrawNotes" (BR-1).

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      plan.md:147 Task 3 Step 3 still reads "using f.Clone() and uv.EmptyCell for blanks", contradicting Core concepts line 37.
  - id: BR-8
    disposition: addressed
    note: |
      viewer.js:208-209 delivers caps-off and an empty notes list on end; connect.test.mjs BR-8 test fails with the pre-fix viewer.js (verified by revert). New view-link side effect raised separately.
  - id: BR-9
    disposition: addressed
    note: |
      notes.js:62-91 generation token; notes.test.mjs "an answer applies only to the dialog that sent it" fails on revert (verified).
  - id: BR-10
    disposition: addressed
    note: |
      README.md:895-896 now names the 👆/💭 mode button under the swatches.
  - id: BR-11
    disposition: addressed
    note: |
      notes.js:141 focuses Close on view; test fails on revert. No focus trap yet, acceptable at Minor.
  - id: BR-12
    disposition: addressed
    note: |
      plan.md:232 Task 8 now cites Core concepts for the noteAt signature instead of restating it.
findings:
  - id: new
    severity: Minor
    family: stream-end-resets-derived-state
    title: |
      End sends a made-up caps-off to view links, which never get caps, so they show the "Pointing is off" hint band
    detail: |
      viewer.js:208 calls deps.caps({pointer:false}) on every page. server.go:325-340 sends caps only to pointer pages, so on a view link setOn(false) un-hides #hint and band() lifts the stage after the end. 2nd in family. Rule: an end puts stream-derived state back to the page variant's join state and never replays events that variant does not receive. Fix: only send caps-off when a caps event was seen (or use one ended() dep), and add a view-link end test.
```
