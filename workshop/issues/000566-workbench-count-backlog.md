---
id: 000566
status: open
deps: []
github_issue:
target: workbench-latency
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: 'f64ef613f8fb19331650c017ff78f8c356601eea' # card fields mirrored from issue-cards; edit via sdlc
---

# Count submit, span reads and build fan-out; preserve nonblocking timing backlog

## Problem

#204's original backlog also covered #201/#202/#203 interaction costs and nonblocking timing capture. The TL approved closing #204 on the later three Couch invariants first; preserve the earlier work here rather than claiming it complete.

## Spec

Add deterministic counted invariants for one Zellij round trip per alt+Return (#201), one agent-span file read per change rather than keystroke (#202), and build fan-out bounded by available cores rather than session count (#203). Determine current behavior first; do not introduce deliberately failing normal-suite tests. Any unfixed defect gets its own fix with a failing regression.
Tier2 is optional nonblocking timing capture: every reading records concurrently working agent count and relevant conditions; timings cannot fail a build. Reuse #204's suite structure and extension guide. No timing threshold or machine changes implied.

## Done when

- The three earlier interaction count contracts have meaningful positive/negative controls through their production seams and run in the normal suite.
- Any tier2 timing capture records agent count for every reading and cannot fail the build.
- Documentation names reproducible commands and what each count covers.

## Plan

- [ ]

## Log

2026-10-15: Split from #204 by TL approval dcb3f338-41ac-45ae-a098-39ff395ae578. This is the preserved earlier backlog, not a dependency of #204's counted Couch delivery.
