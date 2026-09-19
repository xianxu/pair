# Boundary Review — pair#279 (whole-issue close)

| field | value |
|-------|-------|
| issue | 279 — alt+d no longer detaches from the switcher |
| repo | pair |
| issue file | workshop/issues/000279-alt-d-dead-in-switcher.md |
| boundary | whole-issue close |
| milestone | — |
| window | 955cdd95b31ed3263073b0d31c0495023df0ccd7..189b6801a0709653799b79a4b9cc410f6a71591b |
| command | sdlc close --issue 279 |
| reviewer | claude |
| timestamp | 2026-09-18T17:05:38-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The fix is correct and the evidence is real. I traced the state machine by hand and then mutation-tested a scratch copy of `189b6801`: removing the push after `?1049h` turns `TestPresenterKeyboardPushFollowsTheScreen` red (`alternate: flags=9, want 3`) **and** reproduces the operator's report verbatim in the end-to-end test (`physical Alt+d encoded "\x1bd" (flags 0) and dispatched nothing`, 2 of 4 subtests — exactly the two that end on the alternate screen); removing the pop before `?1049l` and removing `releaseAlt`'s pop each go red on the restoration assertions (the latter at `cut 21` of the write-cut sweep). The accounting is correct at every write cut I could construct, the fix lives in the one owner of parent modes so Pair's `termcmd` inherits it, and the doc rewording in `park.go` actually matches `couchkeys.go:87`, README:338/470 and the three tests it cites. Nothing blocks the boundary. What remains is a small family of claims written slightly ahead of the code — one in the issue's own `## Done when`, one in a comment the diff touched — plus leftovers from the #251→#255 migration this issue diagnosed.

**1. Strengths**

- `cmd/internal/terminal/presenter.go:857` — the screen accounting moved out of the generic `write()` into `writeFramePacket`, so one function owns "which screen is the parent on, and does it carry our push". `wholeWrite` records ownership only for a write that fully landed, which is what makes the cut-write sweep's restoration assertion hold at every byte.
- The oracle is a real per-screen protocol model, not a mock of the write: `parentKeyboard`/`assertKeyboardRestored` (`presenter_test.go:441-455`) replay the parent stream into `third_party/vt`, whose keyboard stack is genuinely per screen (`pair_limits.go:50`, `keyboard[e.screenIndex()]`), with distinct ambient flags 5/9 so a push or pop landing on the wrong screen cannot pass.
- `console_keyboard_test.go:51` (`awaitScreen`) repairs a test that was green for no reason: under mutation 1, `TestKeyboardPhysicalNotificationJump` also fails now, confirming the race fix made an existing case discriminating rather than just tidier.
- Constants `altEnter`/`altLeave`/`keyboardPush`/`keyboardPop` replace four inline literals that were previously spelled out in three files — ARCH-DRY improvement inside the window.
- `park.go:107-118` — the reworded contract is verifiable: `menuLiveActions` (`menu.go:1208`) really does carry per-row `detach`, and all three cited tests exist and pass.

**2. Critical findings** — none.

**3. Important findings** — none.

**4. Minor findings**

- `workshop/issues/000279-alt-d-dead-in-switcher.md:96` — the `## Done when` clause says the Couch test "encodes the key with the host emulator's own `SendKey`"; it does not. `hostty.FakeHost` has no `SendKey`, and the test hand-encodes via `keyboardHost.press` (`keyboard_host_test.go:61`). The guarantee (encoded from live per-screen flags) is delivered; the named mechanism is not.
- `cmd/internal/couchtty/terminal_input.go:140` — "Some product chords intentionally exist only in enhanced encoding (Alt+d and Alt+n)". The chord table (`workbenchshortcut/shortcut.go:415-433`) has no legacy row for Ctrl+Alt+n, Alt+Shift+N, Alt+c, Ctrl+Alt+c or Alt+Shift+Enter either.
- `cmd/internal/hostty/control.go:78` — `EnableKeyboardDisambiguation` (`\x1b[=1;2u`), the #251 mechanism `f32bb4cf` replaced, now has zero consumers anywhere, including tests. The package states its own rule at `enterAltScreen` ("Exported surface needs a consumer outside its own package's tests"). `LeaveAltScreen` is in the same state (only two tests reference it).
- `cmd/internal/terminal/presenter.go:859-885` — the "write, then record ownership" step appears three times in three shapes (`if wholeWrite(...) {set true}`, `= wholeWrite(...)`, `= !wholeWrite(...)`); one `writeOwned(ctx, seq) (bool, error)` helper would make them identical.
- `cmd/internal/couchtty/keyboard_host_test.go` — `keyboardModel` + `press` re-implement a CSI parser, a per-screen kitty stack and a kitty key encoder that `third_party/vt` already provides and the presenter test already uses. Pre-existing, extended here from one key to three.
- The operator smoke is recorded only in the **uncommitted** working tree (issue `## Log`, 09-18, "works (global detach)"); at the pinned head the clause is still `[ ]`. It must land in the close commit. Note the tree also carries unrelated uncommitted work (`merge-check.yml`, `Makefile`, `bootstrap.sh`, `scripts/issue-sync.sh` deleted, `scripts/merge-checks.d/40-duplicate-issue-id.sh`) — don't sweep it into the close.

**5. Test coverage notes**

Three mutations, all caught, all verified by me rather than taken from the `## Log`. Both `## Done when` behavioral clauses are exercised in both states: main **and** alternate screen (four subtests including `alternate-and-back` and `back-again`), and restoration after release **and** after a cut at every byte (both sweep variants). Gaps, all small: the presenter test's first paint is always a main-screen actor, so "first paint is an alt-screen actor" (setup push then enter-push in one paint) isn't directly covered; restoration is asserted via flags only, not stack depth — an imbalance is caught only because 3 ≠ ambient 5/9 (vt exposes no depth accessor, so this is a note, not a demand); and Alt+n / Ctrl+Alt+n have no physical-encoding test, which I'd leave alone since the presenter-level test covers the mechanism for every enhanced-only chord. Environment: `go test ./cmd/internal/terminal/` is green; `./cmd/internal/couchtty/` fails only `TestNotificationPTYConformance` and `TestCouchProductionSoak` with "operation not permitted" on `/tmp` and pty spawn — sandbox residue, unrelated to this diff.

**6. Architectural notes**

ARCH-DRY — **pass** with the minors above (the diff net-reduces literal duplication; the remaining duplication is the test-side kitty model, pre-existing). ARCH-PURE — **pass**: the accounting sits at the write seam where it belongs, `wholeWrite` is pure, and the tests drive a protocol model rather than mocking the writer. ARCH-PURPOSE — **pass**: the fix is at the single owner, so it sweeps the whole class of enhanced-only chords on both screens rather than the one chord the report named, and Pair's `termcmd` inherits it. Forward-looking: the setup delta still writes `?1004h`/`?2004h`/mouse modes exactly once, and the claim that those are screen-global (unlike the kitty stack) is currently unstated anywhere. `parentKeyboard`'s ambient-replay harness is the cheap place to pin it if that assumption is ever doubted.

**7. Plan revision recommendations**

- Append to `## Revisions`: the `## Done when` clause 1 names `SendKey`; as delivered the test encodes with `keyboardHost.press` over the per-screen `keyboardModel`. Either reword the clause or switch `keyboardHost` to a `vt.Emulator` and use its `SendKey` (which also disposes of the DRY note).

```findings
findings:
  - id: new
    severity: Minor
    family: doc-claim-accuracy
    title: |
      Done-when clause names a SendKey mechanism the delivered test does not use
    detail: |
      workshop/issues/000279-alt-d-dead-in-switcher.md:96 claims the Couch test "encodes the key
      with the host emulator's own SendKey", but hostty.FakeHost has no SendKey; the test
      hand-encodes via keyboardHost.press (keyboard_host_test.go:61). Family enumeration in this
      window: (a) this clause; (b) terminal_input.go:140's "(Alt+d and Alt+n)" enumeration, which
      omits Ctrl+Alt+n, Alt+Shift+N, Alt+c, Ctrl+Alt+c and Alt+Shift+Enter, all CSI-u-only in
      workbenchshortcut/shortcut.go:415-433. Fix both in one pass.
  - id: new
    severity: Minor
    family: doc-claim-accuracy
    title: |
      terminal_input.go names two enhanced-only chords where the table has seven
    detail: |
      cmd/internal/couchtty/terminal_input.go:140 says the product chords that exist only in
      enhanced encoding are "Alt+d and Alt+n". shortcut.go:415-433 registers no legacy row for
      ChordCtrlAltN, ChordAltShiftN, ChordAltC, ChordCtrlAltC or ChordAltShiftEnter either. Same
      family as the Done-when clause above.
  - id: new
    severity: Minor
    family: dead-exported-surface
    title: |
      hostty.EnableKeyboardDisambiguation is the mechanism f32bb4cf removed and now has no consumer
    detail: |
      cmd/internal/hostty/control.go:78 exports "\x1b[=1;2u" with a doc comment describing the #251
      per-batch re-assert that #255 replaced; grep over cmd/ finds no reference outside its own
      declaration, not even a test. The package states the rule it violates at enterAltScreen
      ("Exported surface needs a consumer outside its own package's tests"). Same family in this
      window: LeaveAltScreen (control.go:63), referenced only by hostty_test.go and couchtty's
      core_concepts_contract_test.go. Delete or re-home both in one sweep.
  - id: new
    severity: Minor
    family: repeated-write-then-record
    title: |
      writeFramePacket spells the same write-and-record step three different ways
    detail: |
      presenter.go:859-885 records ownership as "if wholeWrite(...) { set true }", then
      "= wholeWrite(...)", then "= !wholeWrite(...)". One helper -- writeOwned(ctx, seq) (bool,
      error) -- collapses the three to identical call sites and removes the inversion as a place to
      get it wrong. ARCH-DRY.
  - id: new
    severity: Minor
    family: repeated-write-then-record
    title: |
      keyboardHost re-implements the kitty stack and encoder that third_party/vt already provides
    detail: |
      cmd/internal/couchtty/keyboard_host_test.go carries its own CSI parser, per-screen kitty stack
      and key encoder (press), while third_party/vt models both (pair_keyboard.go, pair_limits.go:50)
      and exposes SendKey (key.go:26) -- and the presenter test in this same window uses vt for
      exactly this purpose. Pre-existing, extended here from one key to three. Consolidating also
      resolves the Done-when wording finding. ARCH-DRY.
  - id: new
    severity: Minor
    family: durable-record-uncommitted
    title: |
      The operator smoke evidence exists only in the uncommitted working tree
    detail: |
      At the pinned head the last Done-when box is "[ ]"; the working tree has it ticked plus a
      09-18 Log entry ("works (global detach)"). Make sure the close commits the issue body. The
      tree also carries unrelated uncommitted work (merge-check.yml, Makefile, bootstrap.sh,
      scripts/issue-sync.sh deleted, scripts/merge-checks.d/40-duplicate-issue-id.sh) that must not
      be swept into the close commit.
```
