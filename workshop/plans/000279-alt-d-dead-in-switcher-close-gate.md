---
gate: boundary-review
issue: 279
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-18T17:05:38-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Minor
          title: Done-when clause names a SendKey mechanism the delivered test does not use
          detail: |-
            workshop/issues/000279-alt-d-dead-in-switcher.md:96 claims the Couch test "encodes the key
            with the host emulator's own SendKey", but hostty.FakeHost has no SendKey; the test
            hand-encodes via keyboardHost.press (keyboard_host_test.go:61). Family enumeration in this
            window: (a) this clause; (b) terminal_input.go:140's "(Alt+d and Alt+n)" enumeration, which
            omits Ctrl+Alt+n, Alt+Shift+N, Alt+c, Ctrl+Alt+c and Alt+Shift+Enter, all CSI-u-only in
            workbenchshortcut/shortcut.go:415-433. Fix both in one pass.
          family: doc-claim-accuracy
          round: 1
        - id: BR-2
          severity: Minor
          title: terminal_input.go names two enhanced-only chords where the table has seven
          detail: |-
            cmd/internal/couchtty/terminal_input.go:140 says the product chords that exist only in
            enhanced encoding are "Alt+d and Alt+n". shortcut.go:415-433 registers no legacy row for
            ChordCtrlAltN, ChordAltShiftN, ChordAltC, ChordCtrlAltC or ChordAltShiftEnter either. Same
            family as the Done-when clause above.
          family: doc-claim-accuracy
          round: 1
        - id: BR-3
          severity: Minor
          title: hostty.EnableKeyboardDisambiguation is the mechanism f32bb4cf removed and now has no consumer
          detail: |-
            cmd/internal/hostty/control.go:78 exports "\x1b[=1;2u" with a doc comment describing the #251
            per-batch re-assert that #255 replaced; grep over cmd/ finds no reference outside its own
            declaration, not even a test. The package states the rule it violates at enterAltScreen
            ("Exported surface needs a consumer outside its own package's tests"). Same family in this
            window: LeaveAltScreen (control.go:63), referenced only by hostty_test.go and couchtty's
            core_concepts_contract_test.go. Delete or re-home both in one sweep.
          family: dead-exported-surface
          round: 1
        - id: BR-4
          severity: Minor
          title: writeFramePacket spells the same write-and-record step three different ways
          detail: |-
            presenter.go:859-885 records ownership as "if wholeWrite(...) { set true }", then
            "= wholeWrite(...)", then "= !wholeWrite(...)". One helper -- writeOwned(ctx, seq) (bool,
            error) -- collapses the three to identical call sites and removes the inversion as a place to
            get it wrong. ARCH-DRY.
          family: repeated-write-then-record
          round: 1
        - id: BR-5
          severity: Minor
          title: keyboardHost re-implements the kitty stack and encoder that third_party/vt already provides
          detail: |-
            cmd/internal/couchtty/keyboard_host_test.go carries its own CSI parser, per-screen kitty stack
            and key encoder (press), while third_party/vt models both (pair_keyboard.go, pair_limits.go:50)
            and exposes SendKey (key.go:26) -- and the presenter test in this same window uses vt for
            exactly this purpose. Pre-existing, extended here from one key to three. Consolidating also
            resolves the Done-when wording finding. ARCH-DRY.
          family: repeated-write-then-record
          round: 1
        - id: BR-6
          severity: Minor
          title: The operator smoke evidence exists only in the uncommitted working tree
          detail: |-
            At the pinned head the last Done-when box is "[ ]"; the working tree has it ticked plus a
            09-18 Log entry ("works (global detach)"). Make sure the close commits the issue body. The
            tree also carries unrelated uncommitted work (merge-check.yml, Makefile, bootstrap.sh,
            scripts/issue-sync.sh deleted, scripts/merge-checks.d/40-duplicate-issue-id.sh) that must not
            be swept into the close commit.
          family: durable-record-uncommitted
          round: 1
      recipe: small-diff-review
      blocked: false
---

# Gate ledger — pair#279 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-18T17:05:38-07:00 (claude) — passed

### Raised

- **BR-1** [Minor] `doc-claim-accuracy` Done-when clause names a SendKey mechanism the delivered test does not use
  workshop/issues/000279-alt-d-dead-in-switcher.md:96 claims the Couch test "encodes the key
  with the host emulator's own SendKey", but hostty.FakeHost has no SendKey; the test
  hand-encodes via keyboardHost.press (keyboard_host_test.go:61). Family enumeration in this
  window: (a) this clause; (b) terminal_input.go:140's "(Alt+d and Alt+n)" enumeration, which
  omits Ctrl+Alt+n, Alt+Shift+N, Alt+c, Ctrl+Alt+c and Alt+Shift+Enter, all CSI-u-only in
  workbenchshortcut/shortcut.go:415-433. Fix both in one pass.
- **BR-2** [Minor] `doc-claim-accuracy` terminal_input.go names two enhanced-only chords where the table has seven
  cmd/internal/couchtty/terminal_input.go:140 says the product chords that exist only in
  enhanced encoding are "Alt+d and Alt+n". shortcut.go:415-433 registers no legacy row for
  ChordCtrlAltN, ChordAltShiftN, ChordAltC, ChordCtrlAltC or ChordAltShiftEnter either. Same
  family as the Done-when clause above.
- **BR-3** [Minor] `dead-exported-surface` hostty.EnableKeyboardDisambiguation is the mechanism f32bb4cf removed and now has no consumer
  cmd/internal/hostty/control.go:78 exports "\x1b[=1;2u" with a doc comment describing the #251
  per-batch re-assert that #255 replaced; grep over cmd/ finds no reference outside its own
  declaration, not even a test. The package states the rule it violates at enterAltScreen
  ("Exported surface needs a consumer outside its own package's tests"). Same family in this
  window: LeaveAltScreen (control.go:63), referenced only by hostty_test.go and couchtty's
  core_concepts_contract_test.go. Delete or re-home both in one sweep.
- **BR-4** [Minor] `repeated-write-then-record` writeFramePacket spells the same write-and-record step three different ways
  presenter.go:859-885 records ownership as "if wholeWrite(...) { set true }", then
  "= wholeWrite(...)", then "= !wholeWrite(...)". One helper -- writeOwned(ctx, seq) (bool,
  error) -- collapses the three to identical call sites and removes the inversion as a place to
  get it wrong. ARCH-DRY.
- **BR-5** [Minor] `repeated-write-then-record` keyboardHost re-implements the kitty stack and encoder that third_party/vt already provides
  cmd/internal/couchtty/keyboard_host_test.go carries its own CSI parser, per-screen kitty stack
  and key encoder (press), while third_party/vt models both (pair_keyboard.go, pair_limits.go:50)
  and exposes SendKey (key.go:26) -- and the presenter test in this same window uses vt for
  exactly this purpose. Pre-existing, extended here from one key to three. Consolidating also
  resolves the Done-when wording finding. ARCH-DRY.
- **BR-6** [Minor] `durable-record-uncommitted` The operator smoke evidence exists only in the uncommitted working tree
  At the pinned head the last Done-when box is "[ ]"; the working tree has it ticked plus a
  09-18 Log entry ("works (global detach)"). Make sure the close commits the issue body. The
  tree also carries unrelated uncommitted work (merge-check.yml, Makefile, bootstrap.sh,
  scripts/issue-sync.sh deleted, scripts/merge-checks.d/40-duplicate-issue-id.sh) that must not
  be swept into the close commit.

## Open findings

- **BR-1** [Minor] `doc-claim-accuracy` Done-when clause names a SendKey mechanism the delivered test does not use
- **BR-2** [Minor] `doc-claim-accuracy` terminal_input.go names two enhanced-only chords where the table has seven
- **BR-3** [Minor] `dead-exported-surface` hostty.EnableKeyboardDisambiguation is the mechanism f32bb4cf removed and now has no consumer
- **BR-4** [Minor] `repeated-write-then-record` writeFramePacket spells the same write-and-record step three different ways
- **BR-5** [Minor] `repeated-write-then-record` keyboardHost re-implements the kitty stack and encoder that third_party/vt already provides
- **BR-6** [Minor] `durable-record-uncommitted` The operator smoke evidence exists only in the uncommitted working tree
