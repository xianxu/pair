---
id: 000672
status: open
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
severity: critical
collaboration-mode: collaboration
card_mirror: '25022670b6fc6f57f565e42822ec0c4543497706' # card fields mirrored from issue-cards; edit via sdlc
---

# Accept native Zellij client replies without blocking launches

## Problem

Operator launches/relaunches fail on the new build containing #647. Observed by the hub: exact session name 📁1-292 is rejected because ProbeSessionName's successful list-clients reply fails strict decoding with invalid Zellij clients header. Installed Zellij is0.45.1. This blocks new slots and takes priority over #665 M2.

## Spec

Capture the real list-clients bytes including header and the exact probe context. Fix the confirmed strict-decoder compatibility defect while preserving fail-closed handling of malformed/wrong-kind/duplicate replies. No fallback to loose parsing. Use the captured native output as a permanent fixture. No operator session mutations, local Zellij build or unrelated #665 mechanism changes.

## Done when

- Exact native reply/header causing refusal is retained and the cause reported to the hub.
- Captured format is accepted by the strict contract; malformed/wrong-kind/duplicate identities still refuse, with deterministic regression red before green.
- Relevant consumers normal/race and repository checks pass; required recorded suite and quick close review pass before landing.
- Land through the line and report the PR/merge so the hub can rebuild pair:0.

## Plan

- Capture and classify the native successful reply and parser mismatch.
- Add captured fixture plus adversarial controls, reproduce red, implement the smallest strict compatibility repair.
- Qualify, review, recorded suite, land; return to checkpointed #665 M2 afterward.

## Log

2026-10-18: urgent dispatch hubda698b0e. #665 M2 clean checkpoint da22f58a48e89393717f617a30d51f867997a884; new work begins from fresh main, not stacked. Disposable nonexistent-session control on installed0.45.1 emits exit1, empty stdout, stderr There is no active session!; this is already an accepted diagnostic and does not reproduce the reported successful malformed-header path. Keep diagnosis open until real bytes are captured.
