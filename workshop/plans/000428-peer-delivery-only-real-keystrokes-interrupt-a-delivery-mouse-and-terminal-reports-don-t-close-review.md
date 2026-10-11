# Boundary Review — pair#428 (whole-issue close)

| field | value |
|-------|-------|
| issue | 428 — peer delivery: only real keystrokes interrupt a delivery; mouse and terminal reports don't |
| repo | pair |
| issue file | workshop/issues/000428-peer-delivery-only-real-keystrokes-interrupt-a-delivery-mouse-and-terminal-reports-don-t.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4c147e5d0f099458bdda2c706414bf5cb8212b6b..4619b6a26d7adbe4e8ad52fe7982f7d4ad17fe0b |
| command | sdlc close --issue 428 |
| reviewer | claude |
| timestamp | 2026-10-10T18:15:09-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Reviewing the #428 diff now. Next I'll check the test names and run the wrapcmd tests.

`TestPeerDraftPasteWaitsForDelivery` failed for a reason other than the sandbox. Looking into it.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

The work meets the issue's Spec and every Done-when item. Peer delivery now treats only keystrokes as operator input (`operatorKeystrokes` / `terminalReport`), and orientation keeps its stricter rule. The kink 47 bytes (focus, kitty Alt+Enter, focus) were measured and are now a regression test. A lone ESC resolves at the 30ms flush. Delivery waits while the operator has the agent pane scrolled up, and a draft-pane paste is held behind an in-flight delivery for at most 1s. `atlas/couch.md` documents all of this. Two things stand between this and a clean SHIP:
- **Flaky test:** `TestPeerDraftPasteWaitsForDelivery` failed 2 times in 21 runs. The test races its own fixture; the production code is fine.
- **Atlas mismatch:** the atlas says "Typed keystrokes are never held", but the hold queues every chunk that arrives after the held paste, keystrokes included.

**1. Strengths**
- `orientation_replies.go`: the two readings are now separate functions, `operatorData` for orientation and `operatorKeystrokes` for peer, sharing one `solicited` helper. This replaces the old boolean-flag variant. CPR is deliberately left out of the unsolicited reports because modified F3 has the same shape, and the code says so.
- The kink 47 regression comes from measured wrap-events, not a guess. The log also explains why there were no mouse bytes: that pane's Claude never enables mouse tracking.
- `peer_viewport.go`: when the viewport can't be judged (no zellij, a failed dump, no alignment), the composer counts as in view. The subprocess runs off the dispatch locks. The pure core (`viewportShift`, `cursorRowInView`) is table-tested.
- The held input replays with its original spacing (`heldInput.next`), so the draft's paste → settle → Alt+Enter rhythm survives. `TestHeldInputKeepsGaps` pins this.
- The hold is fenced under `inputAdmission` through `admitStdin`, so held and released chunks go through one admission path (ARCH-DRY).

**2. Critical**
None.

**3. Important**
- **Flaky hold test:** `cmd/internal/wrapcmd/peer_input_test.go:250-256`. The test checks `d.holding` as soon as the output holds two `\r` bytes. The draft's trailing `\x1b[O` can still be queued behind its replay gap at that point, so `holding` is legitimately still true. I reproduced the failure 1 time in 20 runs, plus once in the first run, with "hold outlived its queue". Fix: wait for the condition instead, e.g. `waitFor(time.Second, func() bool { d.mu.Lock(); defer d.mu.Unlock(); return !d.holding })`, or wait for the `\x1b[O` suffix first.
- **Atlas contradicts the hold behaviour:** `atlas/couch.md` (draft-hold bullet) vs `peer_delivery.go` `holdLocked`. Once `holding` is set, `if d.holding { return true }` queues every later chunk, including keystrokes typed in the agent pane. Those keystrokes are also not counted as operator input until they are released. "Typed keystrokes are never held" is only true in the sense that a keystroke never *starts* a hold. Fix the prose: a keystroke never starts a hold, but input that follows a held paste queues behind it to keep order.

**4. Minor**
- **ARCH-ORDER (a race the tests can't see):** the held paste is not admitted until it is released. The delivery's "no operator keystroke in the last second" gate therefore can't see it, and nothing in `dispatchPeer` checks `d.holding` before a *next* message pastes. In practice the 1s settle after A's submit output, plus the loop draining `held.next` at the top of each iteration, likely prevents interleaving. Still, a `!d.holding` paste gate would make it structural.
- **ARCH-ORDER (split state):** the hold lives in two places, `peerDelivery.holding`/`holdUntil` and the writer-owned `heldInput.base`/`chunks`. Only the writer calling `endHold` keeps them consistent. That is acceptable for a single owner goroutine, but worth noting.
- **ARCH-CONSTRAINTS:** each reading renders the full scrollback (`ViewRows`) under the model lock, then scans every shift: O(scrollback × height). The design caps this at one reading per 500ms, and only while the delivery is ready and waiting. That is fine, but there is no recorded measurement for a long scrollback.
- **ARCH-MOCK:** `composerInView` runs `zellij action dump-screen` directly. `viewProbe` is an injected seam and the pure part is tested, but there is no fake or conformance check for the zellij 0.45 dump-screen semantics the log relies on (scrolled viewport, cursor not moved by scroll).
- `flushPartial` counts any unfinished frame at the 30ms flush as a keystroke. That includes a terminal OSC reply split across a slow read. This is rare, and the error is in the conservative direction.

**5. Test coverage**
- Table tests cover each byte class through `admitInput` and through dispatch, plus the kink 47 regression, the lone-ESC flush, the hold with its bound, the gap replay, the viewport shift and the cursor row.
- The integration tests run the real input writer over a pipe, which is good, but each sees only one interleaving. The flaky failure above is exactly that showing up.
- Other failures in my sandboxed run were sandbox `/tmp` and socket denials, unrelated to this change.

**6. Architecture**
- ARCH-DRY: pass.
- ARCH-PURE: pass. The classifier, `viewportShift` and `cursorRowInView` are pure; the IO shell is thin.
- ARCH-PURPOSE: pass. All four operator-folded Done-when items are delivered.
- ARCH-MOCK: minor flag (zellij dump-screen, above).
- ARCH-CONSTRAINTS: minor flag (scrollback scan, above).
- ARCH-SECURE: pass. The input is the pane's own stdin and the zellij dump; parse failures fail open, visibly, through a `PEER-view-fail` debug line.
- ARCH-ORDER: minor flags (above).
- ARCH-FUNERAL: pass. Nothing durable is created; the hold queue is in memory, bounded, and drained on EOF.

**7. Plan revisions**
None needed beyond the atlas wording fix.

```findings
findings:
  - id: new
    severity: Important
    family: test-observes-single-interleaving
    title: |
      TestPeerDraftPasteWaitsForDelivery is flaky (about 1 in 20): it asserts !holding while the trailing focus-out chunk is still queued
    detail: |
      peer_input_test.go:250-256 reads d.holding right after the second CR shows up; the draft's trailing focus-out can still be waiting on its replay gap. Wait on !holding with waitFor instead (or on the focus-out suffix). Reproduced with go test -count=20.
  - id: new
    severity: Important
    family: docs-contradict-behavior
    title: |
      atlas/couch.md says typed keystrokes are never held, but holdLocked queues every chunk after a held paste
    detail: |
      peer_delivery.go holdLocked returns true for all input while holding. Reword: a keystroke never starts a hold, and later input queues behind a held paste to keep order.
  - id: new
    severity: Minor
    family: hold-state-not-gating-dispatch
    title: |
      dispatchPeer does not check d.holding before pasting the next message
    detail: |
      A held draft paste is not admitted until release, so it never touches lastInput. Today only the settle window and the loop order keep the next delivery from pasting ahead of the released draft. An explicit !d.holding paste gate would make this structural (ARCH-ORDER).
  - id: new
    severity: Minor
    family: external-binary-without-fake
    title: |
      composerInView runs zellij dump-screen with no fake or conformance check of its scrolled-viewport semantics
    detail: |
      The viewProbe seam exists and the pure part is tested, but the zellij 0.45 behaviour the design depends on is only recorded in the Log (ARCH-MOCK).
  - id: new
    severity: Minor
    family: unmeasured-hot-path-cost
    title: |
      Each viewport reading renders the full scrollback under the model lock and scans every shift
    detail: |
      Capped at one reading per 500ms while a delivery waits; no measurement recorded for a long scrollback (ARCH-CONSTRAINTS).
```
