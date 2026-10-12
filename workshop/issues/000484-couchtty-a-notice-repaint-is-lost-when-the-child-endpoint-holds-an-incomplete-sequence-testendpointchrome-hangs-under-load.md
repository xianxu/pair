---
id: 000484
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: 'b86d60dca5105566bc48a5bcb85cd3ea25fe0b6e' # card fields mirrored from issue-cards; edit via sdlc
---

# couchtty: a notice repaint is lost when the child endpoint holds an incomplete sequence (TestEndpointChrome* hangs under load)

## Problem

Found by the #479 soak (200 `-race` runs per touched package under parallel load). Two tests in `cmd/internal/couchtty/terminal_migration_test.go` waited ~560 s (the test's whole deadline, via `testwait`) for a status-row repaint that never came:

- `TestEndpointChromeDuringIncompleteChildSequence`: child feeds `\x1b[38;2;`, then `con.setNotice("CHROME-DURING-PARTIAL")`; row 8 never shows it ("timed out waiting for independent chrome").
- `TestEndpointChromeDoesNotUseChildCursorSave/\x1b7`: child feeds `\x1b7`, then `setNotice("INDEPENDENT-STATUS")`; never painted.

Each happened once in ~128 couchtty runs. This is a lost event, not a late one: before #479 the 3 s `waitFor` turned it into an ordinary-looking flake. A notice set while the endpoint holds an unfinished sequence (or right after a cursor save) can be dropped: a repaint request lost or coalesced away, or a paint that ran before the notice was visible to it. This is a product bug: a status-row update the operator never sees.

## Spec

## Done when

- The root cause is named and measured (which repaint request is lost, and why).
- A deterministic test reproduces the loss without load (e.g. by injecting the interleaving) and passes after the fix.
- Both tests pass 200 `-race` runs under parallel load.

## Plan

- [ ] Reproduce: soak the two tests under load with a trace of repaint requests and paints; find the dropped request.
- [ ] Fix the request/paint ordering at its root; pin the interleaving in a test.
- [ ] Soak.

## Log

### 2026-10-12
