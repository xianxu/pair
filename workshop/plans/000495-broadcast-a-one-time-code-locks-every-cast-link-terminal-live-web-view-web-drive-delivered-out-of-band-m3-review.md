# Boundary Review — pair#495 (milestone M3)

| field | value |
|-------|-------|
| issue | 495 — broadcast: a one-time code locks every cast link (terminal LIVE, web view, web drive); delivered out of band |
| repo | pair |
| issue file | workshop/issues/000495-broadcast-a-one-time-code-locks-every-cast-link-terminal-live-web-view-web-drive-delivered-out-of-band.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 8a71b81daf9da9f16f9869eddb7ede06301a46ca..87c863182f914158919258023f1ecdf9b0ee0b57 |
| command | sdlc milestone-close --issue 495 --milestone M3 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T17:25:49-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Reviewed the lock registry refactor; now inspecting the Couch codes UI (the bulk of M3).

Checking the issue's Spec/Done-when and M3 log against what shipped.

Now verifying castlock's `Gen` semantics for a suspected late-rotation race across casts.

Running the targeted test packages (with -race) to confirm the M3 tests are green at head.

Tests green. Quick revert check: does the tap test fail if the operator layer leaks into the tapped frame?

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

M3 delivers what it set out to build. `broadcast.LinkLocks` replaces the two hand-rolled per-cast lock sets and becomes the one registry each cast uses. The new `terminal.OperatorLayer` is applied after the tapped frame is captured, so codes cannot reach a broadcast. The Couch tooltip, the Alt+click Codes menu, the three-flash and the coalesced guess alert all draw only in that layer. Targeted tests pass with `-race` at head:
- `broadcast`, `castlock`, `terminal` (118 s, 8 s and 66 s);
- the couchtty cast tests (24 s, after generating the runtime bundle);
- the webcast `Lock|Gate` tests.

I reverted the `tapped` fix in `presenter.go` and `TestOperatorLayerNeverReachesTheTap` failed with "the operator-only layer reached the tap", so the key security property has a real regression test. Four things keep this short of SHIP:
- **Late rotation from an ended cast.** Couch orders rotations by `(cast kind, link)`, not by which lock sent them. A rotation from an ended cast that arrives late can show a dead code, and it can hide the next cast's first guess-rotation alerts.
- **README is out of date.** It still says viewers "open it in a browser and see the composed Couch screen live". It never mentions the code, the lock page, the hover tooltip or Alt+click.
- **Smoke test not recorded.** The desktop/iPad smoke test is an M3 checklist item and a Done-when item, and it isn't in the issue Log.
- **Stale spec item.** Spec item 6 still describes clicking an inline code to rotate it.

## 1. Strengths
- **The registry consolidates the duplication** (`cmd/internal/broadcast/linklocks.go:28-114`). It replaces the copy-pasted `rotated` closures, `Codes`, `RotateCode` and `endLocks` in `session.go` and `cast.go`. A lock is minted with its token, before the token can be routed, and `Mint` after `End` fails closed. `LockFor` compares every token, so its timing doesn't reveal which one matched.
- **The tap/operator split is structural, not checked by convention** (`cmd/internal/terminal/presenter.go:431-439, 506`). `tapped := f` is fixed before the layer runs, and the revert check confirmed the test bites.
- **Notices never carry a code.** `TestCodesMenuRotatesAndDisconnects` and `TestGuessRotationAlertAndFlash` assert that the broadcast notice row doesn't contain the code. `TestCastCodesShowOnlyOnTheOperatorsScreen` checks the real viewer stream.
- **Clean locking.** The layer draws from the leaf-locked `castCodes` snapshot only, and `NotesMenu.Draw` clones (`notes_menu.go:156`), so nothing mutates the tapped frame in place.
- **Atlas threat model** (`atlas/broadcast.md:104-142`) is accurate to the code, including the honest "operator's own local diagnostic capture" exclusion. The replay follow-up pair#509 is filed and linked.

## 2. Critical findings
None.

## 3. Important findings
- **I1 — ARCH-ORDER, late cross-cast rotation (`console_castcodes.go:283-336`, `console_broadcast.go:161`, `console_webcast.go:62`).**
  - **Cause:** `castlock` calls `OnRotated` after it releases its mutex (`castlock.go:140-145, 221-224`), and every lock's `Gen` starts again at 1. Couch keys `gens` by `linkKey(kind, link)`, and `forgetCastCodes` clears that key when a cast ends.
  - **Trigger:** cast A's guess rotation is delivered after `broadcastEnded`/`webcastEnded` has run `forgetCastCodes`, for example from an unlock that was in flight while `teardown` ran.
  - **Effects:**
    - It writes A's `Gen` back into the map, so cast B's first rotations with `Gen ≤ A's` are dropped as "older": no alert and no flash.
    - It flashes A's dead code.
    - If the operator is hovering B's cell, it overwrites B's `hoverCodes` with A's dead code, and the operator would read a wrong code aloud.
  - **Rule (this is the 2nd `unordered-async-notifications` finding):** a consumer orders async notifications by *(producer instance, Gen)*, never by a key that a later producer reuses. It drops a notification from a producer that is no longer current.
  - **Fix:** capture the cast's `attempt` in the `OnCodeRotated` closure (the attempt counter is already in `broadcastState`/`webcastState`). Make `onCodeRotated` drop the notification unless that attempt is still the current one. Key `gens` by `(kind, attempt, link)`.
  - **Sweep:** apply the same check to the other cast callbacks into Console in this range: the screen and web `OnCodeRotated`.
  - **Test:** end cast A, deliver A's rotation, start B, deliver B's `Gen 1`. Assert B's alert shows and A's code never shows.
- **I2 — README is missing the code lock and the new gestures (`README.md:865-872, 950-955, 982-995`).**
  - The README still promises viewers see the screen live on opening a link.
  - It never mentions the code, the lock page and name prompt, the hover tooltip on LIVE/Chrome, Alt+click for the Codes menu, rotation disconnecting viewers, or that the drive link takes a 6-digit code.
  - **Rule (this is the 2nd `docs-gate-surface` finding):** the README gate covers *changed behaviour of an already-documented surface*, not only new commands. Enumerate every README paragraph that describes what a cast link's recipient or the operator does, and update each one in the same range: screen view, pointer, web view, drive, notes.
- **I3 — The operator smoke test (desktop + iPad) is unrecorded.**
  - It is an M3 checklist item and a Done-when item, and the issue Log has no entry for it.
  - On iPad specifically, hover and Alt+click need a trackpad and hardware keyboard. The smoke test must confirm the codes are reachable at all there, or record a fallback.
  - Record the result before closing M3.

## 4. Minor findings
- **M1 — Issue Spec item 6 is stale.** It still says "clicking the code in Couch's tab bar rotates it", which contradicts item 7 ("no codes inline"). This is the 5th `spec-moves-with-decisions` finding. Rule: a decision that replaces another greps every artifact that restates it (issue Spec, Done-when, plan, atlas) for the old terms ("click", "inline", "tab bar") in the same commit.
- **M2 — "rotate all" is never exercised.** `TestWebCastCodesOnTheChromeCell` only checks that the item is present. This is the 3rd `behavior-without-default-test` finding. Rule: every Spec'd user action, meaning each menu item, has a test that invokes it and asserts its effect.
- **M3 — The link-copy notices don't say a code is needed** (`console_broadcast.go:228`, `console_pointer.go:130,148`, `console_webcast.go:268`). An operator who copies a link isn't told to hover for its code.
- **M4 — `castAnchorAt` has two identical `case` arms** (`console_castcodes.go:110-113`), and `forgetCastCodes` rebuilds the `linkKey` prefix by hand (`:351`).
- **M5 — `installCodesLayer` uses `context.Background()`,** where the sibling `SetOverlay` uses `c.lifetime` (`console_pointer.go:72`).
- **M6 — The guess-coalescing and Gen policy in `onCodeRotated` sits inside Console under its locks,** and the tests reach into `f.con.codes` internals. It could be a small pure `codeNotices.Apply(rotation, now) → (flash, notice)` (ARCH-PURE).

## 5. Test coverage notes
- **Well covered:**
  - the layer-vs-tap invariant (revert-verified);
  - notices never naming the code;
  - per-link rotate disconnecting a viewer;
  - stale `Gen` within a single cast;
  - the 30 s coalescing;
  - the Chrome cell tooltip with the 6-digit drive code;
  - `LinkLocks` routing, cross-link session refusal, and `End`.
- **Gaps:**
  - a cross-cast late rotation (I1);
  - "rotate all" (M2);
  - flash and menu frames against a real viewer (covered indirectly by the layer test).
- **Setup note:** couchtty tests need `go run ./cmd/internal/runtimebundle/generatecmd` first. I generated untracked assets in this disposable checkout to run them.

## 6. Architectural notes
- **ARCH-DRY: pass.** LinkLocks is a real consolidation; only the M4 nits remain.
- **ARCH-PURE: pass, with M6.** `codeTip` is pure; the rotation policy could be.
- **ARCH-PURPOSE: pass on code, with I3 and I2 open.** Every consumer (screen `Session`, web `Cast`, both `LockFor`s) derives from `LinkLocks`, with no leftover hand-maintained lock fields. The purpose isn't met until the operator has verified it and the README tells users.
- **ARCH-MOCK: pass.** No new external dependency.
- **ARCH-CONSTRAINTS: pass.**
  - Flash goroutines are bounded by the per-lock rate limit (about one forced rotation per 8 s) and by `c.lifetime`.
  - Hover repaints only on a change.
  - The layer runs only leaf locks on the Presenter goroutine.
- **ARCH-SECURE: pass.** Codes appear only in the post-tap layer; notices are code-free; the threat model is documented.
- **ARCH-ORDER: flag (I1).**
- **ARCH-FUNERAL: pass, except I1's re-insert.** All state is in memory and cleared on cast end, and the locks end with the cast via `LinkLocks.End`.
- **When pair:5 adds the web pointer link,** it is one `Mint`. The tooltip, menu and rotate-all derive from `Codes()` automatically.

## 7. Plan revision recommendations
- Add a Revisions entry: "M3 as built: `LinkLocks.Mint` (not `Add`); `terminal.OperatorLayer` applied after the tap's frame; the guess-rotation coalesce and newest-Gen ordering are scoped per cast instance (after I1)." Fix line 11 and line 185, which say `Add`, to `Mint`.
- Update the Status line at close, and record the smoke-test outcome, including whether the gestures work on iPad.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Plan now leads with a Current design section (Couch layer/tooltip/Alt+click menu/flash, SameSite=Strict at line 16) and fences the draft under an explicit superseded banner (line 32); current M3 deliverables live in the issue Plan checklist.
findings:
  - id: new
    severity: Important
    family: unordered-async-notifications
    title: |
      Code-rotation ordering is keyed by (cast kind, link), so a late rotation from an ended cast poisons the next cast
    detail: |
      castlock fires OnRotated outside its mutex and Gen restarts at 1 per lock; onCodeRotated (console_castcodes.go:283) keys gens by linkKey(kind, link), which forgetCastCodes clears at cast end. A late callback re-inserts the old Gen, so the next cast's first guess rotations are dropped (no alert or flash), a dead code flashes, and a hovered tooltip shows the dead code. 2nd in family. Rule: order async notifications by (producer instance, Gen) and drop ones from a non-current producer. Fix: capture the cast attempt in the OnCodeRotated closures (console_broadcast.go:161, console_webcast.go:62) and check it on delivery; add a test that ends cast A, delivers A's rotation, starts B and delivers B's Gen 1.
  - id: new
    severity: Important
    family: docs-gate-surface
    title: |
      README never mentions the cast-link code, the lock page, the hover tooltip or Alt+click Codes menu
    detail: |
      README.md:865-872 still says viewers open the link and see the screen live; the pointer (950-955), web view and drive (982-995) paragraphs omit the code, and the drive link needs a 6-digit code. 2nd in family. Rule: the README gate covers changed behaviour of already-documented surfaces, not only new commands. Sweep every README paragraph describing a cast link recipient's or the operator's actions in the same range.
  - id: new
    severity: Important
    family: done-when-item-unverified
    title: |
      M3's operator smoke test on desktop and iPad is not recorded
    detail: |
      It is in the M3 checklist and Done-when, and absent from the issue Log. The iPad pass must confirm that hover and Alt+click (which need a trackpad and keyboard) can reach the codes, or record a fallback.
  - id: new
    severity: Minor
    family: spec-moves-with-decisions
    title: |
      Issue Spec item 6 still says clicking the code in the tab bar rotates it
    detail: |
      This contradicts Spec item 7 (no inline codes; the Alt+click Codes menu). 5th in family. Rule: a decision that replaces another greps every artifact restating it (issue Spec, Done-when, plan, atlas) for the old terms in the same commit.
  - id: new
    severity: Minor
    family: behavior-without-default-test
    title: |
      The Codes menu's "rotate all" item is never invoked by a test
    detail: |
      TestWebCastCodesOnTheChromeCell only asserts the item is present. 3rd in family. Rule: every Spec'd user action (each menu item) has a test that invokes it and asserts its effect, here that every link's code changes and its viewers disconnect.
  - id: new
    severity: Minor
    family: operator-affordance-discoverability
    title: |
      Link-copy notices do not tell the operator that viewers need a code or where to see it
    detail: |
      console_broadcast.go:228, console_pointer.go:130 and :148, and console_webcast.go:268 copy a now-locked link without hinting to hover for its code.
```
