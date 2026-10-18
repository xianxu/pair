---
id: 000667
status: open
deps: [pair#665]
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: '5dff1663220743852d440d19d0f2b69e6bdf8db1' # card fields mirrored from issue-cards; edit via sdlc
---

# Replace four unresolved readiness waits

## Problem

#665 M2 replaces readiness guesses with observed completion, but four existing sites have no qualified completion witness. Hub approval42233925 explicitly defers them into this one followup. They remain visible in the exact shrink-only debt file with per-entry reasons; they are not fixed or relabeled as duration subjects.

## Spec

Find and qualify a trustworthy completion signal for each site, then remove its sleep and retire that exact debt. Preserve every original behavioral assertion. Do not substitute quiet-frame intervals, navigation retries, immediate zero snapshots, a generic CDP flush claim, or a new fixed wait. Ask the hub before any native/visible/operator-agent run; use disposable state and leave operator sessions/configuration untouched.

| Baseline site at M1 | Missing witness |
| --- | --- |
| cmd/internal/wrapcmd/agent_drive_live_test.go, TestAgentDriveLiveCapture, time.Sleep(1500 * time.Millisecond), ordinal1 | The model/effort picker and styled selection are drawn, but native post-render input-handler mounting is not acknowledged. |
| cmd/internal/wrapcmd/peer_live_test.go, TestPeerLiveConformance, time.Sleep(time.Second), ordinal1 | The trust picker paints its No/Yes selection before later mount effects can reset immediate input; there is no mounted-handler generation witness. |
| cmd/internal/webcast/live_drive_test.go, TestLiveDrive, time.Sleep(2 * time.Second), ordinal1 | A driver's denied download has no qualified terminal deny outcome. Network ERR_ABORTED alone also happens for downloads handed off to another path. |
| cmd/internal/webcast/live_drive_test.go, TestLiveDrive, time.Sleep(time.Second), baseline ordinal2 | After interception is disabled, no qualified terminal native-chooser decision proves the absence of a future Page.fileChooserOpened event. Runtime.evaluate and SyncEvents do not establish that claim. Removing the earlier identical sleep in M2 may renumber this surviving call; use the final debt entry as current identity. |

## Done when

- Each of the four sites has a documented completion contract and a regression that delays the relevant work beyond an unrelated paint/CDP reply, proving the contract does not release early.
- Tests retain model/effort/config, trust choice, no-new-download and restored-chooser assertions as applicable.
- Statefully faked tests and permitted real conformance qualify each external boundary; uncertainty refuses or remains explicit.
- The four sleeps and their exact current debt entries are removed without admitting replacement debt; targeted normal/race checks and recorded suite pass.

## Plan

Deferred and unassigned; design the external completion contracts before implementation. Evidence and implementation context live in #665 R7 and workshop/plans/000665-m2-implementation-notes.md.

## Log

Filed under authenticated hub42233925. No investigation or native/browser run was performed for these four in #665 M2; the retained waits are known unresolved debt.
