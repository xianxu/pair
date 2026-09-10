---
gate: plan-quality
issue: 209
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-09T20:19:11-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Critical
          title: Mode 2's "do not clear when there is nothing to draw" regresses the deliberate redrawTab(nil) clear at run.go:845
          detail: |-
            termcmd/run.go:845 calls m.redrawTab(nil) on tab creation specifically to
            clear before releasing startup output, because replaying the buffer there
            would duplicate it (BR-9). An empty replay because the ring could not
            answer and an empty replay because the caller wants a clean screen are
            different facts; a nil check collapses them. The plan needs a distinct
            signal (or caller-owned decision), not "replay == nil".
          family: absent-data-is-not-intent
          round: 1
        - id: PQ-2
          severity: Critical
          title: Mode 3/4 fix asks Screen for state at the replay START, but Screen only exposes stream-END scalars
          detail: |-
            AltScreen/Mouse/SGRMouse/HoldsCursorSave (screen.go:113-162) reflect the
            whole stream including the tail. Prepending final state before a tail that
            itself transitions can paint primary-screen content onto the alt buffer and
            then have the tail's own ?1049h clear it -- worse than today. Likewise
            replaySafeEnd is a rolling scalar (screen.go:331,349,369,412) and Screen
            never learns the ring window, so a symmetric ReplaySafeStart scalar cannot
            express "earliest boundary still inside the ring"; it also guards
            notification envelopes, a narrower class than mode 3's arbitrary sequences.
          family: state-at-offset-not-at-end
          round: 1
        - id: PQ-3
          severity: Important
          title: Both quoted code blocks and both file:line cites for the existing takeover path are wrong
          detail: |-
            redrawTab is run.go:1658 and is one enqueue call, not the two writes quoted
            at run.go:1021-1024; the clear+write is applyTakeover (run.go:982-983), which
            also feeds hostScan, replays pendingDiag and repaints the strip.
            takeOverScreen is console.go:992-1015 and is not "exactly two writes" -- it
            resets hostScan first and FeedFramings the body back after, so composed
            mode-assertion bytes will reach SafeToPaint's alt-screen carve-out (BR-79/82).
          family: unbacked-existing-behavior-claim
          round: 1
        - id: PQ-4
          severity: Important
          title: The resize nudge has no failure path, no chosen dimension, and no owner for the restore size
          detail: |-
            Child carries no size field and Resize is fire-and-forget (console.go:949,
            run.go:1558), so the restore leg needs a stated owner and a defined outcome
            when it fails -- otherwise the pane is left permanently mis-sized. Say which
            dimension changes and why (cols forces a rewrap down through wrapcmd's
            terminal model; rows usually does not), and what happens when a host resize
            races a switch.
          family: nudge-ordering-and-extent
          round: 1
        - id: PQ-5
          severity: Important
          title: '"zellij repaints its pane on SIGWINCH" carries the whole mode-1 fix with no fake model or conformance check'
          detail: |-
            ARCH-MOCK: name the fake's state model for the nudge and the live
            conformance cadence against real zellij. NewFakeChild only records into
            fake.resizes, so the mode-1 regression row can assert a nudge was issued,
            not that "the repaint path now produces a correct screen" as the test row
            currently promises.
          family: external-behavior-unverified
          round: 1
        - id: PQ-6
          severity: Important
          title: Tests row enumerates cases in prose and names no function under test
          detail: |-
            Replace the enumeration with the functions by name -- the shared repaint
            composer, ReplayThrough's start clamp, the new boundary tracker -- and one
            strategy line each. The boundary tracker is a byte scanner over arbitrary
            child output: fuzz it, seeded with truncated CSI/OSC/DCS and lone ESC.
          family: test-strategy-by-function
          round: 1
        - id: PQ-7
          severity: Minor
          title: No latency budget or coalescing rule for the nudge on the switch path
          detail: |-
            ARCH-CONSTRAINTS: a thread switch is a keystroke-path interaction and the
            nudge cascades a reflow through zellij into every wrapped pane. State a
            budget and what happens on rapid back-to-back switches (coalesce, or
            accept N nudges).
          family: operating-envelope-unstated
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-09T20:23:36-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Intent now carried in the type; run.go:845 + BR-9 verified as a deliberate clear.
          round: 2
        - id: PQ-2
          disposition: not-addressed
          note: Inversion fixes the sampling half but mirrors the destructive-composition half.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: 'All corrected cites verified: console.go:992, run.go:1658, run.go:890/963/982-983.'
          round: 2
        - id: PQ-4
          disposition: addressed
          note: Rows-not-cols, c.size/childSizeLocked as owner, error logged-and-dropped, no coalescing.
          round: 2
        - id: PQ-5
          disposition: addressed
          note: Fake state model named (resize produces a full repaint) plus a live zellij check; cadence is only "after an upgrade".
          round: 2
        - id: PQ-6
          disposition: addressed
          note: Prose enumeration replaced by per-surface strategy with mechanical guards; the fuzz ask is moot now mode 3 is an accepted limitation.
          round: 2
        - id: PQ-7
          disposition: addressed
          note: Rapid-switch rule stated with its reason; no numeric latency budget, non-blocking.
          round: 2
      findings:
        - id: PQ-8
          severity: Critical
          title: Mode-4's replay-then-assert order discards the replay when the incoming child is on the primary screen
          detail: |-
            This is the same family as PQ-2, disposed not-addressed above rather than
            re-raised, because it is the surviving half of that finding rather than a
            new site. The rule the family names: an assertion whose bytes have a frame
            side effect must be ordered relative to the paint BY that side effect, not
            by sampling convenience. Order the buffer switch before HomeAndClear and
            the buffer-independent modes after the replay, and state what an
            unobserved alt-screen belief asserts.
          family: state-at-offset-not-at-end
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-09-09T20:28:21-07:00"
      agent: claude
      dispose:
        - id: PQ-2
          disposition: addressed
          note: Start-state sampling and ReplaySafeStart are both dropped; mode 3 is now an accepted limitation with its reason.
          round: 3
        - id: PQ-8
          disposition: not-addressed
          note: Ordering is correct, but the stated alt-screen belief is falsified by screen.go:41-46 and console_mouse_test.go:456.
          round: 3
      findings:
        - id: PQ-9
          severity: Critical
          title: Two of the four modes the plan asserts have no observed bit, so unwitnessed reads as "off"
          detail: |-
            This is the 2nd finding in family absent-data-is-not-intent, so state the
            rule rather than patching the alt-screen instance: a Screen field may be
            written to the host only when that Screen witnessed the child establish
            it. The enumeration is altScreen, mouse, sgrMouse, cursorSaved
            (screen.go:37-52); only mouseObserved exists (set at screen.go:588,598),
            covering mouse and sgrMouse. 2 of 4 asserted fields are unguarded. A
            reattached zellij (fresh Screen, never re-emits its startup DECSET) gets
            "?1049l" prepended and is pulled off the alt buffer, and the mouse
            assertion fails TestAReattachedChildKeepsItsTrackingMode, which the
            plan's own Done-when requires to pass unmodified.
          family: absent-data-is-not-intent
          round: 3
        - id: PQ-10
          severity: Critical
          title: The named shared home, ptychild, cannot hold the composed repaint - hostty imports ptychild, not the reverse
          detail: |-
            This is the 2nd finding in family unbacked-existing-behavior-claim, so the
            deliverable is the rule: every structural claim the design rests on gets
            checked against the tree before it is committed to, not just the ones a
            prior finding named. Here, ptychild imports only procutil/ansi/notifyosc,
            while hostty imports ptychild (host.go:7, os.go:13, fake.go:8). A function
            composing ?1049h/?1049l + HomeAndClear + replay + EnableMouseClicks needs
            hostty constants, so ptychild is a cycle and copying the constants is
            ARCH-DRY. hostty already serves both consumers and control.go:53 already
            names them as wanting the same thing. Name the owning package explicitly.
          family: unbacked-existing-behavior-claim
          round: 3
        - id: PQ-11
          severity: Important
          title: Step 4 lists cursor-save among the modes to re-assert, but no byte sequence can restore it
          detail: |-
            HoldsCursorSave (screen.go:148-160) reports that the child sits between
            its own DECSC and DECRC. The host's save slot holds a POSITION the console
            does not know, so emitting DECSC to re-assert the belief writes the
            console's own position into the shared slot - the precise hazard that
            docstring exists to prevent. It is a SafeToPaint gate input, not replayable
            state, and the takeover already rebuilds it by feeding the body back
            (console.go:1013, run.go:987). The rule: only state that is both knowable
            by the writer and idempotently expressible as bytes belongs in an
            assertion list; everything else is a gate input. Drop it, or say what
            sequence it emits.
          family: asserted-state-must-be-reconstructible
          round: 3
      blocked: true
    - "n": 4
      timestamp: "2026-09-09T20:31:12-07:00"
      agent: claude
      dispose:
        - id: PQ-8
          disposition: addressed
          note: Composition now orders buffer assert before the clear and buffer-independent modes after the tail; unobserved asserts nothing.
          round: 4
        - id: PQ-9
          disposition: addressed
          note: 'Rule stated plus the enumeration: altScreenObserved/sgrMouseObserved mirror mouseObserved, and item 4 says when absence is admissible.'
          round: 4
        - id: PQ-10
          disposition: addressed
          note: Home moved to hostty; verified hostty imports ptychild (host.go:7, fake.go:8, os.go:13) and not the reverse, so no cycle and no copied constants.
          round: 4
        - id: PQ-11
          disposition: addressed
          note: Cursor-save dropped as a category error, with the reason stated as a rule rather than a one-off exclusion.
          round: 4
      findings:
        - id: PQ-12
          severity: Minor
          title: Tests row still lists cursor-save in the mode-assertion cross-product the design just dropped
          detail: |-
            This is the 2nd finding in family asserted-state-must-be-reconstructible, but
            the rule is already stated correctly in the plan (Mode-4 item 2: no sequence
            injects a previously-saved position, so HoldsCursorSave is a gate input, not
            replayable state). Only the Tests row lags it, still asking for
            {alt-screen, mouse, SGR-mouse, cursor-save} x {set, unset} — half of whose
            cells assert bytes the design will not emit. Drop cursor-save from that
            cross-product so the test surface matches the three fields that carry an
            observed bit.
          family: asserted-state-must-be-reconstructible
          round: 4
      blocked: false
content_hash: 1899e6ebb6fa798a46e92de4cfb1bfbba42a7e79544880504e4f33ad1a72a43e
---

# Gate ledger — pair#209 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-09T20:19:11-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Critical] `absent-data-is-not-intent` Mode 2's "do not clear when there is nothing to draw" regresses the deliberate redrawTab(nil) clear at run.go:845
  termcmd/run.go:845 calls m.redrawTab(nil) on tab creation specifically to
  clear before releasing startup output, because replaying the buffer there
  would duplicate it (BR-9). An empty replay because the ring could not
  answer and an empty replay because the caller wants a clean screen are
  different facts; a nil check collapses them. The plan needs a distinct
  signal (or caller-owned decision), not "replay == nil".
- **PQ-2** [Critical] `state-at-offset-not-at-end` Mode 3/4 fix asks Screen for state at the replay START, but Screen only exposes stream-END scalars
  AltScreen/Mouse/SGRMouse/HoldsCursorSave (screen.go:113-162) reflect the
  whole stream including the tail. Prepending final state before a tail that
  itself transitions can paint primary-screen content onto the alt buffer and
  then have the tail's own ?1049h clear it -- worse than today. Likewise
  replaySafeEnd is a rolling scalar (screen.go:331,349,369,412) and Screen
  never learns the ring window, so a symmetric ReplaySafeStart scalar cannot
  express "earliest boundary still inside the ring"; it also guards
  notification envelopes, a narrower class than mode 3's arbitrary sequences.
- **PQ-3** [Important] `unbacked-existing-behavior-claim` Both quoted code blocks and both file:line cites for the existing takeover path are wrong
  redrawTab is run.go:1658 and is one enqueue call, not the two writes quoted
  at run.go:1021-1024; the clear+write is applyTakeover (run.go:982-983), which
  also feeds hostScan, replays pendingDiag and repaints the strip.
  takeOverScreen is console.go:992-1015 and is not "exactly two writes" -- it
  resets hostScan first and FeedFramings the body back after, so composed
  mode-assertion bytes will reach SafeToPaint's alt-screen carve-out (BR-79/82).
- **PQ-4** [Important] `nudge-ordering-and-extent` The resize nudge has no failure path, no chosen dimension, and no owner for the restore size
  Child carries no size field and Resize is fire-and-forget (console.go:949,
  run.go:1558), so the restore leg needs a stated owner and a defined outcome
  when it fails -- otherwise the pane is left permanently mis-sized. Say which
  dimension changes and why (cols forces a rewrap down through wrapcmd's
  terminal model; rows usually does not), and what happens when a host resize
  races a switch.
- **PQ-5** [Important] `external-behavior-unverified` "zellij repaints its pane on SIGWINCH" carries the whole mode-1 fix with no fake model or conformance check
  ARCH-MOCK: name the fake's state model for the nudge and the live
  conformance cadence against real zellij. NewFakeChild only records into
  fake.resizes, so the mode-1 regression row can assert a nudge was issued,
  not that "the repaint path now produces a correct screen" as the test row
  currently promises.
- **PQ-6** [Important] `test-strategy-by-function` Tests row enumerates cases in prose and names no function under test
  Replace the enumeration with the functions by name -- the shared repaint
  composer, ReplayThrough's start clamp, the new boundary tracker -- and one
  strategy line each. The boundary tracker is a byte scanner over arbitrary
  child output: fuzz it, seeded with truncated CSI/OSC/DCS and lone ESC.
- **PQ-7** [Minor] `operating-envelope-unstated` No latency budget or coalescing rule for the nudge on the switch path
  ARCH-CONSTRAINTS: a thread switch is a keystroke-path interaction and the
  nudge cascades a reflow through zellij into every wrapped pane. State a
  budget and what happens on rapid back-to-back switches (coalesce, or
  accept N nudges).

## Round 2 — 2026-09-09T20:23:36-07:00 (claude) — BLOCKED

### Disposed

- PQ-1 — addressed — Intent now carried in the type; run.go:845 + BR-9 verified as a deliberate clear.
- PQ-2 — not-addressed — Inversion fixes the sampling half but mirrors the destructive-composition half.
- PQ-3 — addressed — All corrected cites verified: console.go:992, run.go:1658, run.go:890/963/982-983.
- PQ-4 — addressed — Rows-not-cols, c.size/childSizeLocked as owner, error logged-and-dropped, no coalescing.
- PQ-5 — addressed — Fake state model named (resize produces a full repaint) plus a live zellij check; cadence is only "after an upgrade".
- PQ-6 — addressed — Prose enumeration replaced by per-surface strategy with mechanical guards; the fuzz ask is moot now mode 3 is an accepted limitation.
- PQ-7 — addressed — Rapid-switch rule stated with its reason; no numeric latency budget, non-blocking.

### Raised

- **PQ-8** [Critical] `state-at-offset-not-at-end` Mode-4's replay-then-assert order discards the replay when the incoming child is on the primary screen
  This is the same family as PQ-2, disposed not-addressed above rather than
  re-raised, because it is the surviving half of that finding rather than a
  new site. The rule the family names: an assertion whose bytes have a frame
  side effect must be ordered relative to the paint BY that side effect, not
  by sampling convenience. Order the buffer switch before HomeAndClear and
  the buffer-independent modes after the replay, and state what an
  unobserved alt-screen belief asserts.

## Round 3 — 2026-09-09T20:28:21-07:00 (claude) — BLOCKED

### Disposed

- PQ-2 — addressed — Start-state sampling and ReplaySafeStart are both dropped; mode 3 is now an accepted limitation with its reason.
- PQ-8 — not-addressed — Ordering is correct, but the stated alt-screen belief is falsified by screen.go:41-46 and console_mouse_test.go:456.

### Raised

- **PQ-9** [Critical] `absent-data-is-not-intent` Two of the four modes the plan asserts have no observed bit, so unwitnessed reads as "off"
  This is the 2nd finding in family absent-data-is-not-intent, so state the
  rule rather than patching the alt-screen instance: a Screen field may be
  written to the host only when that Screen witnessed the child establish
  it. The enumeration is altScreen, mouse, sgrMouse, cursorSaved
  (screen.go:37-52); only mouseObserved exists (set at screen.go:588,598),
  covering mouse and sgrMouse. 2 of 4 asserted fields are unguarded. A
  reattached zellij (fresh Screen, never re-emits its startup DECSET) gets
  "?1049l" prepended and is pulled off the alt buffer, and the mouse
  assertion fails TestAReattachedChildKeepsItsTrackingMode, which the
  plan's own Done-when requires to pass unmodified.
- **PQ-10** [Critical] `unbacked-existing-behavior-claim` The named shared home, ptychild, cannot hold the composed repaint - hostty imports ptychild, not the reverse
  This is the 2nd finding in family unbacked-existing-behavior-claim, so the
  deliverable is the rule: every structural claim the design rests on gets
  checked against the tree before it is committed to, not just the ones a
  prior finding named. Here, ptychild imports only procutil/ansi/notifyosc,
  while hostty imports ptychild (host.go:7, os.go:13, fake.go:8). A function
  composing ?1049h/?1049l + HomeAndClear + replay + EnableMouseClicks needs
  hostty constants, so ptychild is a cycle and copying the constants is
  ARCH-DRY. hostty already serves both consumers and control.go:53 already
  names them as wanting the same thing. Name the owning package explicitly.
- **PQ-11** [Important] `asserted-state-must-be-reconstructible` Step 4 lists cursor-save among the modes to re-assert, but no byte sequence can restore it
  HoldsCursorSave (screen.go:148-160) reports that the child sits between
  its own DECSC and DECRC. The host's save slot holds a POSITION the console
  does not know, so emitting DECSC to re-assert the belief writes the
  console's own position into the shared slot - the precise hazard that
  docstring exists to prevent. It is a SafeToPaint gate input, not replayable
  state, and the takeover already rebuilds it by feeding the body back
  (console.go:1013, run.go:987). The rule: only state that is both knowable
  by the writer and idempotently expressible as bytes belongs in an
  assertion list; everything else is a gate input. Drop it, or say what
  sequence it emits.

## Round 4 — 2026-09-09T20:31:12-07:00 (claude) — passed

### Disposed

- PQ-8 — addressed — Composition now orders buffer assert before the clear and buffer-independent modes after the tail; unobserved asserts nothing.
- PQ-9 — addressed — Rule stated plus the enumeration: altScreenObserved/sgrMouseObserved mirror mouseObserved, and item 4 says when absence is admissible.
- PQ-10 — addressed — Home moved to hostty; verified hostty imports ptychild (host.go:7, fake.go:8, os.go:13) and not the reverse, so no cycle and no copied constants.
- PQ-11 — addressed — Cursor-save dropped as a category error, with the reason stated as a rule rather than a one-off exclusion.

### Raised

- **PQ-12** [Minor] `asserted-state-must-be-reconstructible` Tests row still lists cursor-save in the mode-assertion cross-product the design just dropped
  This is the 2nd finding in family asserted-state-must-be-reconstructible, but
  the rule is already stated correctly in the plan (Mode-4 item 2: no sequence
  injects a previously-saved position, so HoldsCursorSave is a gate input, not
  replayable state). Only the Tests row lags it, still asking for
  {alt-screen, mouse, SGR-mouse, cursor-save} x {set, unset} — half of whose
  cells assert bytes the design will not emit. Drop cursor-save from that
  cross-product so the test surface matches the three fields that carry an
  observed bit.

## Open findings

- **PQ-12** [Minor] `asserted-state-must-be-reconstructible` Tests row still lists cursor-save in the mode-assertion cross-product the design just dropped
