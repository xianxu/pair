---
id: 000479
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '39549c7ddb39442f78050a2d1aeb2bc77d05f3f7' # card fields mirrored from issue-cards; edit via sdlc
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

## Done when

- No test in the list fails because an awaited event arrived late; each site either waits on its event or injects the clock it asserts.
- The touched packages pass 200 runs under `-race` and parallel load.

## Plan

- [ ] Per site, prove the window: a temporary product sleep past the bound fails the test.
- [ ] couchtty `waitUpTo`: wait on the condition with no window (binary timeout catches a hang); convert its callers.
- [ ] Remaining listed select guards: wait on the event, or inject the clock where the bound is the subject.
- [ ] 200 `-race` runs of each touched package under parallel load; full `make -k test`.

## Log

### 2026-10-12
