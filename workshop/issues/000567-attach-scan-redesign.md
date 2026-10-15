---
id: 000567
status: open
deps: []
github_issue:
target: workbench-latency
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: '10e24a349383195575ce99859f6bdc98cea4ea61' # card fields mirrored from issue-cards; edit via sdlc
---

# Lower attachment and owner-pass process-table scans to one

## Problem

#204 establishes strict selected Couch attachment <=1 full process-table scan (parent0 + helper1), but the other paths still enumerate repeatedly. Existing #218 owner-pass sharing counts only its initial capture; each live fence also enumerates a full table.

## Spec

Redesign other attachment/owner workflows toward <=1 full process-table enumeration per operation while preserving fresh targeted PID/generation checks, uncertainty, exact terminal authority and effect-time revalidation. Use #204's counted ratchets as the regression oracle; do not rename additional full enumerations as targeted reads or remove safety fences merely to satisfy the count.
Current source-derived baselines to confirm through #204's counters: standalone attach <=2 (ancestry + orphan sweep), continuation fallthrough <=3 (extra ancestry), owner pass <=1+2N for N Probe+Revalidate callers (one shared initial capture plus two native full-table SessionServers reads per caller). SnapshotNamed still enumerates the full table; it only filters argv enrichment.

## Done when

- Standalone/continuation attachment and owner workflows have an explicit counted <=1 full-table budget at their reviewed operation boundary, with successful exact attachment and unchanged safety behavior.
- Ratchets tightened from standalone2, continuation3, owner1+2N; independent counterfactuals prove redundant enumeration fails.
- Permission/read failures and PID/generation changes stay uncertain or revoked, never fabricated absence.

## Plan

- [ ]

## Log

2026-10-15: Follow-up approved by TL receipt dcb3f338-41ac-45ae-a098-39ff395ae578. #204 owns today's count ratchets; this issue owns lowering them, not weakening their definitions.
