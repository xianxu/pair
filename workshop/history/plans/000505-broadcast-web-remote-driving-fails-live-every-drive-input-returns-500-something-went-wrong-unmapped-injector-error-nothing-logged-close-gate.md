---
gate: boundary-review
issue: 505
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T17:30:11-07:00"
      dispatched: "2026-10-12T17:28:17-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: Recorder errClosed→ErrClosed, cdpTab.Dispatch→ErrInput (ErrDialog pass-through), Chrome.WithTab nil→ErrNotLive have no test
          detail: 'Reverting any of recorder/drive.go:36-38, recorder/drive.go:189-195 or cast_drive.go:27 leaves the suite green; the route test injects a pre-built ErrInput via failingInjector. Add driveTab cases: Input.* failure → ErrInput, dialog → ErrDialog, closed recorder → ErrClosed; and a nil-rec Chrome.WithTab → ErrNotLive.'
          family: behavior-change-without-regression-test
          round: 1
        - id: BR-2
          severity: Minor
          title: live_drive_real_test.go and recorder/drive_test.go are not gofmt-clean
          detail: gofmt -l lists both; the errSpy fields and the new driveTab field comments are misaligned. Base drive_test.go was clean.
          family: gofmt-clean
          round: 1
        - id: BR-3
          severity: Minor
          title: gestures.reset() does not clear lastClick (ARCH-ORDER)
          detail: A click before control ends and one after it is granted again, within DOUBLE_MS and DRAG_PX, is sent as a double click; reset() should also set lastClick = null.
          family: state-reset-completeness
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-12T17:43:03-07:00"
      dispatched: "2026-10-12T17:39:46-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: TestWithTabFailuresAreDriveErrors, TestWithTabGivesUpOnADialog and TestUnrecordedChromeWithTabIsNotLive each fail when the matching fix is reverted (mutation-checked).
          round: 2
        - id: BR-2
          disposition: addressed
          note: gofmt -l on cmd/internal/webcast lists only query/events.go and server.go, neither touched in this window; both files named in BR-2 are clean.
          round: 2
        - id: BR-3
          disposition: addressed
          note: drive.js:607 reset() clears lastClick; the node test for a click across a reset fails without it (mutation-checked).
          round: 2
      recipe: milestone-review
      isolation: fenced
      reviewed: 9ea549903dfc09362cffd0168ba595e27cf9b832
      blocked: false
---

# Gate ledger — pair#505 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T17:30:11-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `behavior-change-without-regression-test` Recorder errClosed→ErrClosed, cdpTab.Dispatch→ErrInput (ErrDialog pass-through), Chrome.WithTab nil→ErrNotLive have no test
  Reverting any of recorder/drive.go:36-38, recorder/drive.go:189-195 or cast_drive.go:27 leaves the suite green; the route test injects a pre-built ErrInput via failingInjector. Add driveTab cases: Input.* failure → ErrInput, dialog → ErrDialog, closed recorder → ErrClosed; and a nil-rec Chrome.WithTab → ErrNotLive.
- **BR-2** [Minor] `gofmt-clean` live_drive_real_test.go and recorder/drive_test.go are not gofmt-clean
  gofmt -l lists both; the errSpy fields and the new driveTab field comments are misaligned. Base drive_test.go was clean.
- **BR-3** [Minor] `state-reset-completeness` gestures.reset() does not clear lastClick (ARCH-ORDER)
  A click before control ends and one after it is granted again, within DOUBLE_MS and DRAG_PX, is sent as a double click; reset() should also set lastClick = null.

## Round 2 — 2026-10-12T17:43:03-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — TestWithTabFailuresAreDriveErrors, TestWithTabGivesUpOnADialog and TestUnrecordedChromeWithTabIsNotLive each fail when the matching fix is reverted (mutation-checked).
- BR-2 — addressed — gofmt -l on cmd/internal/webcast lists only query/events.go and server.go, neither touched in this window; both files named in BR-2 are clean.
- BR-3 — addressed — drive.js:607 reset() clears lastClick; the node test for a click across a reset fails without it (mutation-checked).

## Open findings

(none — every finding has been disposed)
