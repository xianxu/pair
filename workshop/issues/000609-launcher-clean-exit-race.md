---
id: 000609
status: open
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
severity: medium
card_mirror: 'd745515c0d7e5e181ebb2be3af547826773d35ea' # card fields mirrored from issue-cards; edit via sdlc
---

# Launcher clean-exit cancellation test races TERM trap readiness

## Problem

The recorded full suite for #589 at head ff788e65bb02408429d7e974389a2511d8492db2 failed TestCancelledHandoffReportsAChildsCleanExit in cmd/internal/launcher/osruntime_launch_test.go:85: handoff = -1, <nil>; want the child's own 0, nil. All other packages passed, including couchcmd, couchmessage and couchcore. Launcher source and tests have no diff from origin/main in #589.

The fixture sends cancellation after a fixed 100 ms without confirming that its shell installed the TERM trap. A delayed shell startup could therefore receive TERM before the trap is ready. This is a hypothesis, not a controlled reproduction. The neighboring TERM-ignoring test already waits for a trap-ready file before cancellation.

## Spec

Measure the failure ordering with a controllable fixture. Synchronize cancellation with the child's actual trap readiness while preserving the intended contract: a child that exits cleanly after cancellation returns its own exit status. Do not hide signalled exits, add sleeps, or extend deadlines.

## Done when

- A controlled ordering reproduction identifies why the clean-exit test observed a signalled exit.
- Cancellation observes actual child readiness rather than elapsed time.
- Isolated repeated and full-suite verification pass; production cancellation semantics remain covered.

## Plan

- [ ]

## Log

2026-10-16: Filed unassigned from pair:1 at the TL's direction, separate from #589. Failed recorded command: sdlc local-check run --name suite (make PAIR_GO_TRIMPATH=-trimpath test), head ff788e65bb02408429d7e974389a2511d8492db2. Full log: /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/515ad3a2617b9f0f7d1aa19c39691236.log (sha256 prefix 0c6257440f80). This was the only failing test in the completed run.

Qualification on the same unchanged head: go test -trimpath ./cmd/internal/launcher -run '^TestCancelledHandoffReportsAChildsCleanExit$' -count=30 passed in 4.425s. This establishes intermittent behavior; it does not prove the suspected trap-installation ordering. No launcher changes or production slot effects. The TL authorized one unchanged recorded-suite rerun for #589 after this filing.
