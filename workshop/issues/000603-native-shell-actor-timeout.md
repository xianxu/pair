---
id: 000603
status: open
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: '77f5fefc133d55634222bcbb89085bb51774162a' # card fields mirrored from issue-cards; edit via sdlc
---

# Native wrapped Zellij conformance times out finding shell actor

## Problem

GitHub macOS conformance intermittently stalls waiting for the real shell actor in TestNativeConsoleWrapperZellij/wrapped-zellij. This blocks unrelated PRs after their local full suite and review pass.

First observed here: 2026-10-16 06:27:17 UTC (2026-10-15 23:27:17 PDT), run 38860695250, attempt 1, head 550c434561059eeda72ed209041b5468916081f1, while landing #571. The subtest failed after 284.48 seconds at terminal_native_test.go:227: timed out waiting for real shell actor.

Run: https://github.com/xianxu/pair/actions/runs/38860695250/job/116842060496

Local diagnostic log: /tmp/pair571-ci-conformance-38860695250.log (ephemeral; the issue does not depend on its retention).

## Spec

Find why the wrapped Zellij fixture does not observe its real shell actor. Keep the assertion meaningful; do not address this by raising its timeout. In the failing run the direct-zellij-baseline passed in 1.73 seconds. Wrapped receipts recorded BOOT, CPR and submitted input bytes; wrapper.log recorded canonical OSC 777 and native-fixture-complete notifications, but shell-actor observation did not complete. These facts do not yet establish the root cause. Runner scheduling, process discovery and terminal lifecycle remain hypotheses.

This is distinct from #601's park helper client-disconnect failure. No terminal fixture, shell actor or Zellij lifecycle code is changed by #571's reduced Codex exact-ID home routing. An intervening conformance run on the same reviewed code passed before a documentation-only main integration created this head. One observed failure is not a calibrated flake rate.

## Done when

- Root cause is reproduced with retained, bounded diagnostics that distinguish actor launch from discovery failure.
- A regression verifies the identified race without a larger timeout.
- Repeated runner/headless checks demonstrate the correction, reporting failure counts and conditions.

## Plan

- [ ]

## Log

- 2026-10-15: Filed unassigned per TL direction. #571 will retry only this failed CI job once on its unchanged head; a repeat failure is escalated rather than retried indefinitely.
