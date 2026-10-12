---
id: 000479
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: 'ed05ba5ffb9bcc6c7d16641fcaa19992ac979d7d' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T07:00:54-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
---

# tests: replace short real-time hang guards with event waits (couchtty waitUpTo 250/500ms, ~47 sites, and siblings)

## Problem

Found by the #476 sweep (fixed real-time windows asserted under load, the #473/#476 flake family). Ranks are a reading of the code, not measured failures. Paths are under `cmd/internal/`.

These tests wait for an event with a short fixed bound (100 ms to 1.5 s), and fail if the event is late. The event is right; the bound is a performance assertion that a loaded `-race` suite can miss:
- couchtty `waitUpTo(t, 250ms/500ms, ...)`, about 47 sites: console_run_menu (~30), console_menu_operation, console_newest_page, console_menu, console_preview, console_completion, console_slotgit:116. Highest volume.
- couchtty select guards at 250/500 ms: console_run_menu_test.go:274,279,304,481,489,550,555,583; console_menu_test.go:54,65,71,119,131,136; console_preview_test.go:78,84,89,96,130,162,167; console_menu_operation_test.go:227,239; console_attach_transaction_test.go:76,81; console_mouse_test.go:184 (`timeoutAfter` 400 ms).
- couchcore/supervisorlease_subprocess_test.go:97 (750 ms poll on a re-exec'd subprocess; high), actionableinventory_test.go:352,360 (250 ms), launchhelper_test.go:114 (1 s).
- couchcmd/run_test.go:1299 (250 ms).
- wrapcmd: harness_tty_live_test.go:362 (1 s, kill and reap of a re-exec'd binary), overlay_test.go:72, terminal_model_test.go:910,918,926, composer_recognizers_test.go:608, stdout_batch_test.go:96,102,142, translate_stdin_test.go:101,121,157,188 and picker_overlay_test.go:218,228 (`waitFor(200ms)`).
- launcher/osruntime_launch_test.go:27,44 (1 s / 1.5 s on a real `sh` child).
- terminalqualify/candidate_test.go:182, couchsingleton/evidence_test.go:107, notifytransport/transport_test.go:342,402, couchtty/terminal_pressure_test.go:183 (redundant 1 s total next to its select).

## Spec

As #476 did for the retention lock test: where the event is the assertion, wait on it with no real-time window, and let the test binary's timeout catch a hang (its goroutine dump names the blocked call). Where a bound is part of the subject, keep it and inject the clock. Don't just raise the constants. couchtty's `waitUpTo` helper is one change for most sites. Before changing a site, prove it is a window: widen it with a temporary sleep in the product and watch the test fail.

Design (agreed with the TL, 2026-10-12): one shared helper, `cmd/internal/testwait`, bounds every wait by the test's own deadline minus a margin (10 s, or half the time left), not by a guess at event latency. A late event passes; a lost one fails with a message naming what the test waited for, before the binary's timeout. `testwait.After(t)` is the guard case of a select, `Until`/`Wait` poll. Sites that asserted a wall-clock ceiling (`elapsed > 1s`) assert the outcome that proves the bound instead: the product's own timeout error (`ErrThreadStoreBusy`, "lock timed out", "timed out") or how a child ended (SIGTERM vs SIGKILL). The binary timeout catches a lost bound. Absence windows (`case <-time.After(...):` with an empty body, "nothing happens in N ms") stay: under load they get less sensitive but never fail falsely.

## Done when

- No test in the list fails because an awaited event arrived late; each site either waits on its event or injects the clock it asserts.
- The touched packages pass 200 runs under `-race` and parallel load.

## Plan

- [x] Prove the window with a temporary product sleep: couchtty `paintNow` +300 ms fails 23 tests; wrapcmd stdin read +300 ms fails 3 `waitFor(200ms)` tests. The ceiling sites are windows by construction.
- [x] `cmd/internal/testwait` (`After`, `Wait`, `Until`) plus its unit tests.
- [x] couchtty: `waitFor`/`waitLong`/`waitUpTo` become one deadline-bounded `waitFor` (107 calls); 81 positive select guards use `testwait.After(t)`; the slot-git `settle` reads the schedule's idle state on the loop instead of waiting for a 20 ms quiet spell (the probe exposed it).
- [x] Other listed select guards: couchcore actionableinventory, couchcmd run, wrapcmd overlay/terminal_model/composer, terminalqualify candidate.
- [x] wrapcmd `waitFor`/`waitForStdoutBatch`: deadline-bounded; couchcore supervisorlease: `testwait.Wait`.
- [x] Ceilings: launchhelper, harness_tty_live, notifytransport ×2, couchsingleton evidence, terminal_pressure, launcher ×2, each asserting the outcome instead.
- [ ] 200 `-race` runs of the touched tests under parallel load; full `make -k test`.

## Log

### 2026-10-12
