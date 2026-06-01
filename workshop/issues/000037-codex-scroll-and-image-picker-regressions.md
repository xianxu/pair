---
id: 000037
status: done
deps: []
created: 2026-06-01
updated: 2026-06-01
estimate_hours: 1.0
actual_hours: 0.6
---

# Codex scroll and image picker regressions

## Done when

- Codex image attach picker can be confirmed with plain Return after Alt+i.
- Codex stdout filtering covers the newly observed terminal mode that may affect zellij mouse/focus handling without removing forensic raw capture.
- Regression tests cover the image-capture overlay arming and stdout filtering.

## Spec

After the synchronized-output filter from #30, mouse scroll stopped working
again in a live Codex session. The same session also showed a Codex image attach
picker where plain Return moved to the next choice and Alt+Return was required
to select.

Live raw scrollback confirms the previous `ESC[?2026h/l` synchronized-output
markers are balanced and the running `pair-wrap` binary contains the #30 filter.
The stream also contains Codex startup terminal mode `ESC[?1004h` focus-event
tracking, with no mouse-reporting modes (`1000/1002/1003/1006`) observed.

For image attach, pair already has a stronger signal than visible picker text:
nvim sends SIGUSR1 to pair-wrap to arm image capture immediately before sending
Ctrl+V to the agent pane. In Codex sessions that should also arm a one-shot
overlay Enter bypass.

## Plan

- [x] Arm Codex overlay Enter bypass from image-capture start.
- [x] Filter Codex focus-event terminal mode from stdout to zellij while keeping raw scrollback unchanged.
- [x] Add focused tests for both behaviors.
- [x] Rebuild and verify.

## Log

### 2026-06-01

- Filed from live report: mouse scroll stopped again; image attach picker needed
  Alt+Return to select.
- Live raw `scrollback-pair-codex.raw` has balanced `2026h/l`, no mouse-reporting
  mode counts, and one `?1004h` focus-event enable at startup.
- Implemented Codex image-capture overlay arming in `pair-wrap`: SIGUSR1 capture
  start now sets the one-shot picker flag so the next plain Return confirms the
  picker.
- Extended Codex stdout filtering to remove `ESC[?1004h/l` focus-event mode from
  zellij-facing output while preserving raw scrollback.
- Verification: `go test ./cmd/pair-wrap`, `make test`, and `make pair-wrap`
  pass.
- Closed: `go test ./cmd/pair-wrap` and `make test` pass; `make pair-wrap`
  rebuilt `bin/pair-wrap` with Codex image-capture Enter bypass and focus-event
  stdout filtering.
