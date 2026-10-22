---
id: 000818
status: open
deps: [pair#791, pair#796]
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
collaboration-mode: collaboration
card_mirror: '63fbf958b57ce18bf51de2fcdc43d6df4bc0977b' # card fields mirrored from issue-cards; edit via sdlc
---

# Couch: reassess resize recovery after every wrapper carries #735

## Problem

#810 proposed a recovery mechanism for resize-then-silence stalls, but the first native capability experiment failed: Claude 2.1.308 did not repaint after an unchanged-size group SIGWINCH. The issue is parked. A new live ops:1 capture admits on current code but reproduces peek-visible/admission-hidden behavior when the pre-#735 always-invalidating resize behavior is restored. TL reports many fleet wrappers predate #735's landing, providing a strong stale-binary explanation for chronic stalls; the exact live wrapper revision remains unproved.

## Spec

After the operator refreshes all wrappers onto code carrying #735, and #791's capture plus #796's default collapse are live, determine whether #810 needs any further recovery implementation. Establish actual wrapper rollout evidence, observe real delivery behavior and inspect any new retained refusal capture. Separate duplicate same-size resize handling already fixed by #735 from an actual geometry-change/no-repaint case. Do not treat an observation interval with no recurrence as proof that all possible resize stalls are fixed.

## Done when

- Record which wrappers carry #735 and when the refresh completed, with unknowns explicitly retained rather than inferred from supervisor revision.
- Record a TL-agreed observation window, any recurring refusal conditions and the relevant private capture provenance.
- Bring an evidence-based recommendation to TL: retire #810 if no further mechanism is warranted, or re-plan only a demonstrated remaining case. Any geometry bounce or other recovery effect requires explicit new approval before code.

## Plan

- [ ]

## Log

2026-10-22: Filed unclaimed at authenticated ops:0 request 6e54f49d-cdcf-44db-b1f4-53714a10c64b. The operator was asked by TL to Alt+n ops:0 now; tonight's full shutdown is expected to refresh the rest. These are planned actions, not verified completed rollout. #810 remains parked; #791 landing and then #807 retain priority.
