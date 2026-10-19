---
id: 000683
status: open
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
severity: high
collaboration-mode: collaboration
card_mirror: '0e42822b9cf2ac778e301e825e516323be5c02b1' # card fields mirrored from issue-cards; edit via sdlc
---

# Successful park cleanup remains unresolved and lifecycle failures lose diagnostics

## Problem

Operator UI relaunches of ariadne:0 and pair:0 failed on 2026-10-18 around 17:49–17:51 PDT. The reported Couch build included #651, #676 and #670. Each cleanup helper published a matching success completion and preserved scrollback, but Couch left the park awaiting_completion and did not publish VerifiedPark. Subsequent resume refused resume-parking. The original UI failure message was not retained with an inspectable operation, and the hub reported no new launch-diagnostics/failures.json entry; the initiating cause could not be established from the surviving evidence.

The operator later recovered through the menu; both outcomes were reported as recovered (Fresh). Read-only inspection found new live tags, while the old records had been archived with their parks still unresolved. Whether the operator selected Fresh or Couch selected it is not established; do not infer an automatic Fresh transition from the label alone.

## Spec

Reconcile a matching successful cleanup into a verified park after exact owner-death proof; keep unknown/replaced ownership refusal and never manufacture success. Make interrupted post-cleanup finalization recoverable on the exact retained transaction. Persist every lifecycle failure's message with its operation, including UI-triggered relaunch/park/recovery failures before a new launch, so later diagnosis does not depend on a transient banner or a launch-only diagnostic entry. Preserve bounded diagnostics, identity and effect-phase evidence without recording transcript bodies or secrets.

Distinguish verified park recovery, retained unresolved evidence and explicit Fresh in the UI and durable operation outcome. Investigate why the menu path archived these unresolved records; do not silently equate archival or a new Fresh slot with successful park finalization. Preserve prior conversations and capture authority.

## Done when

- [ ] Stateful coverage reproduces cleanup success followed by interrupted/failed Couch finalization; exact recovery finalizes the retained transaction without repasting, restarting or silently choosing Fresh.
- [ ] Lifecycle errors from both menu and CLI remain durably inspectable with message, exact target/transaction, phase and outcome, including failures with no new launch attempt.
- [ ] Recovery and Fresh outcomes explicitly distinguish finalized park from archived unresolved evidence; unknown/replaced owner guards remain intact.
- [ ] The retained incident is diagnosed as far as evidence permits; missing original error and uncertain menu choice remain explicit, not reconstructed as facts.
- [ ] Required verification and review pass before normal landing.

## Plan

Inspect retained lifecycle/archived records and the operation-diagnostic seam first. Design failure retention and recovery ordering with explicit partial outcomes before implementation. #681 separately owns the confirmed two-second CLI admission timeout; do not duplicate that fix here.

## Log

2026-10-18 — Read-only incident evidence gathered before any worker repair or relaunch. Hub requested this unassigned filing in verified delivery 89941932-b309-48df-b397-e5f5cbadfb58. No production metadata was repaired by the worker.

Before menu recovery, global threadstore records were pair revision 685 and ariadne revision 546, both awaiting_completion, attempt 1 open, no VerifiedPark. The recorded helper PIDs were absent; Zellij sessions 1-282 and 1-284 were absent. Matching success completions and all three parked capture bundle files existed. No persistent lifecycle/store lock holder was found during inspection.

After menu recovery, read at approximately 18:17–18:18 PDT:

- pair:0 old scope e108517d46ab4575, tag 1-pair-2, archived revision **686**, park nonce park-678704160c5bde5f, attempt 1 still awaiting_completion, no VerifiedPark. Original owner PID 56488, kernel identity 1792370871.658511. Success completion at 17:51:15.714059 PDT; capture token 20261018T175115; raw 377597 bytes, events 868 bytes, capture metadata 785 bytes. Prior Codex native ID **01a12720-0b3f-7ee0-a5da-1474622794f5**, retained transcript 3187553 bytes. New active tag 1-pair-69, session 1-310, revision 4, one live incarnation and no open park/start/continuation.
- ariadne:0 old scope b325dbb782174007, tag couch-539ef32aa4744e03, archived revision **547**, park nonce park-98bb4395906efc9d, attempt 1 still awaiting_completion, no VerifiedPark. Original owner PID 56753, kernel identity 1792370872.494384. Success completion at 17:49:13.701393 PDT; capture token 20261018T174913; raw 154319 bytes, events 4625 bytes, capture metadata 828 bytes. Prior Claude native ID **bee2be25-757b-4027-8573-2bdcaf857203**, retained transcript 1337921 bytes. New active tag 1-ariadne-67, session 1-308, revision 4, one live incarnation and no open park/start/continuation.

The old records reside under the global threadstore archive, not records; their request/completion JSON and parked bundles remain in their repository scope directories. No pending threadstore journal was found. New live slots are settled; archived old park bookkeeping is explicitly unresolved. Native transcript existence was checked by path and size only; contents were not printed or attached.

Code audit: runActiveAttempt returns immediately if the terminal-delete trigger reports an error after an effect; awaitCompletionAndChildDeath also returns immediately on completion-observation errors; finalization/store errors can leave the same awaiting record. Without the original UI message those paths cannot be distinguished. Target client exit does not itself cancel the UI operation context, which derives from console lifetime. This audit does not establish an inherent deadlock or a particular initiating failure.

Confirmed separate #681: recover-park prepared full fleet inventory before receipt allocation inside a 2-second CLI admission exchange. Hub verified both timed-out requests had unknown receipts. Adjacent normal resume preparations took 3.055s and 5.233s. This explains the failed CLI recovery attempts, not the original UI relaunch failure.

2026-10-18 — Additional evidence from hub delivery a4fd4c1b-e50a-437b-96d9-28addcead9a2, verified read-only: launch-diagnostics/failures.json contains a later pair:0 launch failure at 18:15:07.913087 PDT, scope e108517d46ab4575, intermediate tag 1-pair-68, nonce start-21db4bb41655e148, stage registration, helper started, before/after exit 1. launch_error is "context canceled / recovery attempt registration is not confirmed". Decoded retained output says the exact Couch session name 1-309 was not accepted because the Zellij name probe was killed and its context deadline was exceeded. This is a later attempted launch, plausibly the operator's Resume from disk before Fresh; that association is the hub's inference, not a proved reconstruction. It does not supply the original relaunch/park failure text. The earlier report of no new launch diagnostic referred to the original approximately17:49–17:51 relaunch failures, not this later launch.

The hub also reports parley.nvim:0 subsequently returned "recovered (Fresh)" with no new failure record, giving approximate time18:40. The time and chosen menu action are not independently verified here; this report is retained as supplied rather than used to infer an automatic Fresh choice or precise ordering. Production state was read only; no additional recovery, launch or diagnostic write was performed by this worker.
