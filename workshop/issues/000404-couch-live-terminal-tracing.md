---
id: 000404
status: open
deps: []
github_issue:
created: 2026-10-07
updated: 2026-10-07
estimate_hours:
card_mirror: 'd90abbd78e873fd46786b87f72c9e1e9fb4ccd41' # card fields mirrored from issue-cards; edit via sdlc
---

# Opt-in Couch live terminal tracing

## Problem

The intermittent Claude turn-end display corruption tracked by #379 cannot yet
be reproduced. Existing logs miss the Zellij-to-Couch and Couch-to-host streams.
Tracing must ship independently so the operator can use regular Couch while
waiting for an occurrence; shipping instrumentation does not resolve #379.

## Spec

Own the opt-in two-boundary capture implementation originally developed under
#379, its tests, runbook, and implementation plan. COUCH_CAPTURE_DIR explicitly
enables capture for one regular Couch process; isolation is optional. Preserve
ordered endpoint bytes/geometry, host write receipts, thread identity, private
files, bounded resources, default-off behavior, and terminal operation on failure.

The first real regular-session capture stopped after 6.88 seconds at 5.35 MiB
with `terminal capture queue limit reached`. Resolve realistic startup/burst
handling and expose failure while Couch is running, rather than only at exit.
Settle and document retention for a long-running wait (including the 256 MiB
limit and whether bounded rolling capture is needed), with replay completeness
explicit. The imported implementation is not yet ready to ship.

## Done when

- Explicit opt-in works in regular and isolated Couch; disabled mode writes nothing.
- Both terminal boundaries retain exact bytes, ordering, geometry and accepted-write receipts with tested completeness semantics.
- A regression test represents the observed startup burst; a real regular Couch smoke capture survives that workload.
- Queue, disk and write failures are visible during the session without corrupting terminal output; resource limits and long-running retention are documented and tested.
- Tests, review and runbook support shipping tracing to main independently, without closing #379 or claiming its display bug fixed.

## Plan

- [x] Import the existing implementation, tests and runbook from #379 onto a branch based on main.
- [ ] Revise the transferred implementation plan for the real overflow finding and long-running capture behavior.
- [ ] Fix and verify remaining capture reliability and failure visibility requirements.
- [ ] Review and ship instrumentation independently; keep #379 open for reproduction and diagnosis.

## Log

### 2026-10-07 — Split tracing delivery from diagnosis

- Operator requested a separate implementation ticket and branch; #379 retains the long debugging session and all evidence.
- Original implementation commits: 9ca17478 (capture) and 0b384b45 (regular-session activation). Imported work will preserve their provenance; historical test results do not establish live-capture reliability.
- Real evidence: `/Users/xianxu/.local/share/pair/captures/session-2417847440/events.jsonl`, 5,609,068 bytes, 3,280 records, 6.88 seconds, final incomplete marker `terminal capture queue limit reached`. The capture had already stopped, so it cannot record later incidents. Current queue bounds are 128 records / 8 MiB; disk cap is 256 MiB.
- The underlying display fault remains unproven. No root-cause fix is included in this ticket.
