---
id: 000745
status: open
deps: [743, ariadne#509]
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: '0ebb95cc66f401dd5b1f1d607c7f726749c74998' # card fields mirrored from issue-cards; edit via sdlc
---

# Promote local VM conformance

## Problem

pair#743 introduces the machine-wide Tart conformance runner and nightly burn-in. The operator requires conformance before landing, so required GitHub PR conformance remains active until local verification has seven clean nights and the landing-scoped SDLC capability exists.

## Spec

- Inspect seven distinct clean nightly runs on the approved image/case set; preserve exact main heads, image/tool versions, execution completeness, failures and cleanup evidence. Never count a blind retry as a clean night; fix flakes at their root. A material image/case change requires renewed qualification.
- After ariadne#509, activate conditional local conformance once per landing train after its suite, including train-of-one. Union each included car's touched paths; unrelated cars skip the VM. Use the train/nightly VM position, never a second queue.
- In the same reviewed cutover, remove the GitHub pull_request trigger; retain main pushes, workflow_dispatch and weekly hosted backstop. Until cutover is verified, preserve the required PR check.
- Review every-agent coverage with pair#744: current legacy targets use agent stand-ins. Do not present installed Claude/Codex CLIs as real-agent test coverage; any unavailable Qoder credits are explicit.
- Obtain the operator's local-gates verification/label. Never self-label or waive a red conformance result.

## Done when

- Seven clean nights are linked and the operator accepts promotion.
- A real eligible train and train-of-one execute conformance once after the suite; an unrelated train skips it. Failing conformance prevents landing without suite bisection or automatic retry.
- Hosted main/weekly backstop remains, PR conformance is replaced by the verified local gate, and atlas describes ownership and recovery.

## Plan

- [ ] Wait for dependencies and burn-in; review evidence and obtain promotion approval before changing gates.

## Log

### 2026-10-20

Filed as the explicit rollout split approved by ops:0 for pair#743. The original #743 activation and GitHub demotion requirements land here, so nightly infrastructure can reach main without weakening pre-landing conformance.
