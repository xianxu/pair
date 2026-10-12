---
id: 000484
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: 'fd4cd192446d1fd7b97e4c9945800450066d0d90' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T10:28:32-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
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

- Trace (temporary `fmt.Fprintf` in `paintNow`, widened window): `computed notice=false` → `computed notice=true` → `published notice=true` → `published notice=false` (+18 ms). The stale snapshot published last.
- Tests: `TestStaleChromeSnapshotCannotOverwriteNotice` and `TestNoticeDuringResizeUsesOneGeometry` (`terminal_migration_test.go`) park the first paint at the hook and let the notice try to overtake. With `publishMu` disabled: the first fails with row `"[brain]"`, the second with `terminal: incompatible chrome geometry`. With the fix, both pass.
- Soak: the race test binary ran `TestEndpointChrome*|TestStaleChrome*|TestNoticeDuringResize*` with `-count=200` under 12 self-terminating CPU burners (`perl -e 'alarm 150; 1 while 1'`; this machine has no `timeout` binary): PASS, 0 failures. The same soak of the old tests on HEAD's code also passed. At that load the poll rarely misses the 18 ms window, so the load soak is a weak detector (original rate ~1 in 128 under #479's heavier soak); the deterministic tests are the evidence.

