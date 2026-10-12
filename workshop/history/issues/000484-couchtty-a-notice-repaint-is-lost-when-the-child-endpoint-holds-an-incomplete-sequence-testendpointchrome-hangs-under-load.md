---
id: 000484
status: done
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: 'ff5bed0e972a5d9cf89b3dab378d339bfc6a1cf2' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T10:28:32-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "b00bd64d", done: "75e14700"}
actual_hours: 1.03
---

# couchtty: a notice repaint is lost when the child endpoint holds an incomplete sequence (TestEndpointChrome* hangs under load)

## Problem

Found by the #479 soak (200 `-race` runs per touched package under parallel load). Two tests in `cmd/internal/couchtty/terminal_migration_test.go` waited ~560 s (the test's whole deadline, via `testwait`) for a status-row repaint that never came:

- `TestEndpointChromeDuringIncompleteChildSequence`: child feeds `\x1b[38;2;`, then `con.setNotice("CHROME-DURING-PARTIAL")`; row 8 never shows it ("timed out waiting for independent chrome").
- `TestEndpointChromeDoesNotUseChildCursorSave/\x1b7`: child feeds `\x1b7`, then `setNotice("INDEPENDENT-STATUS")`; never painted.

Each happened once in ~128 couchtty runs. This is a lost event, not a late one: before #479 the 3 s `waitFor` turned it into an ordinary-looking flake. A notice set while the endpoint holds an unfinished sequence (or right after a cursor save) can be dropped: a repaint request lost or coalesced away, or a paint that ran before the notice was visible to it. This is a product bug: a status-row update the operator never sees.

## Spec

Root cause (measured): presentation is published unordered. Every paint snapshots console state (status model, size, focus) and then publishes it to the presenter outside any ordering, from several goroutines: Run, and notice publishers (`GoTracked(setNotice)` in `console_webcast.go`; the tests' own `setNotice`). The fixture's initial `selectActorContext` ends with a `paintNow` on Run; the test's `setNotice` paints concurrently. A trace showed the Run paint snapshotted `notice=false`, the notice paint snapshotted and published `notice=true`, then the stale one published last (18 ms later). The screen kept the stale row and nothing repainted it. The test passed only when its poll caught the notice in that window; under load the poll missed it and waited out the deadline. The child's incomplete sequence was incidental timing, not the cause.

The same race between `onResize` (sets `c.size`, then `presenter.Resize`) and a concurrent notice paint renders a row of the new width against the old geometry. The presenter refuses with `terminal: incompatible chrome geometry`, and `terminalError` ends couch (measured with the lock removed).

Fix: `Console.publishMu` is held from snapshot through publish and commit in every console-driven publication: `paintNow`, `showMenu`, `publishSelection` (the select path), and `onResize`'s size update plus `presenter.Resize`. The last publication is then always the latest snapshot (ARCH: one ordering owner for presentation). `publishHook` is a test seam inside the lock.

## Done when

- The root cause is named and measured (which repaint request is lost, and why).
- A deterministic test reproduces the loss without load (e.g. by injecting the interleaving) and passes after the fix.
- Both tests pass 200 `-race` runs under parallel load.

## Plan

- [x] Reproduce: trace snapshot/publish order in `paintNow`; find the stale publish.
- [x] Fix the request/paint ordering at its root (`publishMu`); pin the interleaving in a test.
- [x] Soak.

## Log

### 2026-10-12
- 2026-10-12: closed — Post-close delta is comment-only (console.go routing-answer comment names the Attach path, per the close review's advisory). go build ok; full make -k test exit 0 on the merged tree before it. (fixed-then-ship); review verdict: SHIP
- 2026-10-12: closed — Root cause measured: unordered presentation; stale status-row snapshot published after the notice paint (trace computed false, computed true, published true, published false +18ms); also notice during resize -> incompatible chrome geometry -> couch exits. Fix: publishMu spans snapshot-to-commit in paintNow, showMenu, selection (through the focus commit, BR-1), onResize. Four hook tests (chrome, resize, select, panel) each fail on revert and pass fixed. Soak 200x -race under 12 CPU burners: 0 failures. couchtty+terminal -race green; make -k test exit 0 under the suite lock on the tree merged with origin/main. no-atlas: internal ordering fix, no new surface.; review verdict: SHIP

- Trace (temporary `fmt.Fprintf` in `paintNow`, widened window): `computed notice=false` → `computed notice=true` → `published notice=true` → `published notice=false` (+18 ms). The stale snapshot published last.
- Tests: `TestStaleChromeSnapshotCannotOverwriteNotice` and `TestNoticeDuringResizeUsesOneGeometry` (`terminal_migration_test.go`) park the first paint at the hook and let the notice try to overtake. With `publishMu` disabled: the first fails with row `"[brain]"`, the second with `terminal: incompatible chrome geometry`. With the fix, both pass.
- Soak: the race test binary ran `TestEndpointChrome*|TestStaleChrome*|TestNoticeDuringResize*` with `-count=200` under 12 self-terminating CPU burners (`perl -e 'alarm 150; 1 while 1'`; this machine has no `timeout` binary): PASS, 0 failures. The same soak of the old tests on HEAD's code also passed. At that load the poll rarely misses the 18 ms window, so the load soak is a weak detector (original rate ~1 in 128 under #479's heavier soak); the deterministic tests are the evidence.
- Close review round 1 (FIX-THEN-SHIP). BR-1: the select path committed `c.focus`/`c.active` after releasing `publishMu`, so a paint in that gap read the menu focus and re-panelled over the new selection. `selectActorContext` now holds `publishMu` from the focus read through the focus commit. BR-2: the select and panel sites were unpinned. Added `TestPaintDuringSelectionCannotRestoreTheMenu` (hook `select` at the focus-commit gap; fails on the BR-1 shape with the menu on row 1) and `TestNoticeDuringMenuPublicationIsShown` (hook `panel`; fails with `showMenu` unlocked: row `"[brain]"`). The Minor stale comments (`publishHook` doc, the routing-answer guard in `paintNow`) are updated.

