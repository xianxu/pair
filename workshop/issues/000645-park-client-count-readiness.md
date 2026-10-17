---
id: 000645
status: open
deps: []
github_issue:
created: 2026-10-17
updated: 2026-10-17
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'b9105e0d4dd605e7ab8586b7984dd7e02dbbda8c' # card fields mirrored from issue-cards; edit via sdlc
---

# Park fixture accepts empty client baseline as attachment readiness

## Problem

Read-only #636 audit found that cmd/internal/couchcore/park_lifecycle_live_test.go zellijClientCount directly invokes list-clients, counts every nonempty stdout line (including the header), and accepts successful empty stdout as count0. startProductionParkHelper snapshots that baseline, starts a helper, and accepts any later count greater than baseline as attachment. If baseline is an empty successful reply and the next response is only CLIENT_ID / ZELLIJ_PANE_ID / RUNNING_COMMAND header, the count moves0 to1 even with no attached client. Source behavior establishes this false-readiness sequence; no native reproduction or causal link to #636/#640 is claimed.

## Spec

Make the park live fixture preserve unknown query results and count actual clients consistently with the production boundary. Do not interpret exit0/empty stdout as zero clients or a header as a client. Preserve lifecycle assertions and setup/deadline separation; no blind retries, sleeps, longer deadlines or production transport change. Use the existing shared query/parser seam where practical (ARCH-DRY/ARCH-ORDER).

## Done when

- A stateful fake regression feeds empty baseline then header-only response and proves helper attachment cannot be established by that sequence.
- A complete header-only response represents known zero clients; a real added client can prove attachment against a known baseline.
- Malformed/empty responses remain uncertain, fixture callers share the rule, and existing lifecycle tests pass.
- Native validation is serialized with #640 and uses a disposable session after the shared native race is classified; do not claim this fixture weakness explains #636 without evidence.

## Plan

## Log

2026-10-17: Filed unclaimed from #636 read-only audit. Relevant source: cmd/internal/couchcore/park_lifecycle_live_test.go zellijClientCount and startProductionParkHelper; canonical production semantics in cmd/internal/launcher/zellij.go and zellijparse.go reject empty stdout before counting rows. No code, test fixture or native environment changed.
