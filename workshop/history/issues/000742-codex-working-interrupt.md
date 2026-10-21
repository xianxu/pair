---
id: 000742
status: done
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'd24148715adb22e3cfe6b4aac518cbd7ba4f4691' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T12:53:03-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "114b2d48", done: "b07177f9"}
actual_hours: 1.54
---

# Couch interrupt refuses a working Codex turn

## Problem

Follow-up to #717 (landed in PR #483): the morning live check reported couch --interrupt refusing a Codex turn that was working. The existing admission proof combines reducer-owned working state, recognized composer, current viewport interrupt hint, and automatic-input ownership. Diagnose which proof is missing from the real Codex path; do not weaken idle/panel/picker or second-interrupt protections to make the working case pass.

## Spec

Use the agent's current coherent work evidence, defined once per agent. For Codex, extend the existing RecognizeCodexWorking pure recognizer to the explicit Working and Waiting for status verb prefixes, each requiring esc to interrupt; counters, truncated summaries and command-detail rows after the hint do not affect authority; locate the native status relative to the qualified native composer, not the physical viewport bottom. Both the lifecycle observer and interrupt admission consume this recognizer. Claude keeps reducer-owned Working(), no stop grace, a recognized composer and its current interrupt footer.

An incomplete redraw does not emit a Codex stop transition. At admission and immediately before the one write, Codex's positive current work proof authorizes interrupt without requiring cached notification activity flags. Project a positively recognized working Codex screen as turn-active even if the notification lifecycle missed it. The existing picker, complete-frame, operator/peer/automatic-input ownership and lease fences remain. A later fresh complete *idle composer* must be observed to confirm completion; absence of working evidence on an unknown/picker/panel screen never confirms. There is no retry of a possibly written interrupt. Interrupt observation, start and writer rechecks bind to the last operator-input activity sequence while freshly re-proving current complete work/composer/ownership. Benign native redraws do not revoke that input generation; typing, Esc, image and submit do. Output already read but not yet parsed cannot authorize from an old rendered frame. Other drive shapes retain the all-activity sequence.

Actions: interrupt once; inspect a refusal or unconfirmed result; request a new interrupt only on a newly observed working turn.

| Internal state / failure mode | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Recognized working Codex or Claude, exclusive owned input | One Ctrl+C, awaiting turn end | Inspect final result | Current turn is intentionally interrupted |
| Idle, unknown, partial frame, picker/panel or input ownership conflict | Refused; nothing sent | Inspect current slot | None |
| Turn ends or ownership changes before dispatch | Refused/cancelled before write | Inspect current slot | None |
| Ctrl+C written but no fresh idle composer before deadline | Unconfirmed; never resent | Inspect current slot | Work may already have been interrupted |
| Fresh idle composer after the write | Confirmed | Inspect result; second interrupt refuses | Current turn stopped; conversation remains |

ARCH-DRY: Codex's one pure recognizer serves lifecycle and drive proof, reusing codexComposerBounds. ARCH-PURE: retain native capture/PTY IO in tests and wrapper adapters. ARCH-CONSTRAINTS: one bounded viewport/composer scan per completed observation; no new goroutine, cache, store or growing state. ARCH-STATE: keep the existing drive reducer and one input lease; require positive idle completion rather than infer completion from an absent work signal. Creates nothing durable beyond existing bounded receipts because this repair changes only recognition and confirmation.

## Done when

- Reproduce the reported Codex working-turn refusal with captured evidence and an automated regression that fails before the fix.
- A current working Codex turn admits exactly one Ctrl+C, then confirms only the observed turn end. A second interrupt after completion refuses and sends nothing.
- Idle, Claude stop grace, panels, pickers, stale ownership and competing input continue to refuse without a write. Current coherent Codex work evidence overrides a stale notification stop/grace report, as it does stale completion.
- Targeted tests, the recorded local check, boundary review and a hub-coordinated live Codex check pass before landing.

## Plan

- [x] Capture coherent owned native Codex Working and running-shell frames; record the failing admission flags and composer/status relationship. Keep the reported Waiting wording in explicitly synthetic fixtures; no native Waiting frame was captured.
- [x] Add regressions through handleChunk and the production mailbox/writer with no manually set Codex activity flags: top-anchored and tall composer, both status forms, complete/incomplete redraw and stale lifecycle observations. Assert one Ctrl+C, fresh positive idle confirmation, then second-interrupt refusal.
- [x] Repair the shared recognizer and per-agent admission/confirmation proof. Preserve Claude stop-grace and all negative/ownership cases. Mutate each corrected guard and require its oracle to fail.
- [ ] Run targeted packages/race and reader inventory; document the per-agent work proof in atlas. Push PR and record affected check; stage the tested head for hub-owned live Codex sleep/interrupt/second-refusal check, then close and merge.


## Log




- 2026-10-20: closed — Full review of approved interrupt input-generation correction: native observeOutput->handleChunk6s/7s/9s/Waiting regression RED on old activity token and GREEN with input token across observation/start/write. Typing/Esc/image/submit revoke; ordinary drives retain activity fence; existing pending-output count preserves parsed-current-frame proof. Full wrapcmd PASS146.685s; focused race wrapcmd34.101s/couchcmd2.850s/couchmessage14.035s/agentdrive1.459s; vet and reader8 PASS66.451s. Output-token/operator-stamp/unparsed-output mutation controls RED; restored sources. Corrective code e767dc64d4e1 pushed PR493, new recorded check and stage build started. no-plan-check retains pending suite/CI/live/landing; no live-success claim. Per TL747 merge re-enters once742 review is recorded;742 stage held. (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP
- 2026-10-20: closed — Full review of TL-approved stable-marker contract after BR-3: native truncated Waiting/detail regression RED before fix and GREEN after; captured Working6s passes replay, initial live refusal cause remains unassigned. Full wrapcmd PASS110.787s; focused race PASS wrapcmd7.522s/agentdrive1.176s; vet and reader inventory8 PASS; hint and whitelist mutations RED/restored. Corrective production3027d14a2cb8 staged/build verified; recorded suite queues on that pushed code and hub live check pending. no-plan-check acknowledges final row includes those external checks and landing; neither suite nor live acceptance is waived.; review verdict: SHIP
- 2026-10-20: closed — BR1 acceptance now explicitly retains Claude grace refusal while current coherent Codex work overrides cached notification stop/grace. Existing Claude negative and new retained Codex stop-grace positive pass under race (1.921s); cached-grace mutation fails the new case. BR2 completed Plan row distinguishes captured Working/running-shell frames from reported Waiting wording in synthetic fixtures. No production code changed after a16907dd. Prior focused race/vet/native-capture/mutation evidence stands; PR493 CI on a16907dd passed. Final recorded check requeued on 69bd42d1 because retained test changed; old run cancelled through local-check cancel. a16907dd remains staged and build-verified for hub live acceptance, blocked only on operator switcher relaunch of a Codex slot. --no-plan-check acknowledges the final row includes pending recorded suite, live acceptance and landing; neither suite nor live acceptance is waived, both must pass before merge.; review verdict: SHIP
- 2026-10-20 — Dispatched by ops:0 after #726 landed, before #733. #717 is already done and its old branch must not be reused. Current archived #717 Log describes BR-4/BR-6 fixes but lacks the reported failed live-check detail; asked ops:0 for target/time/refusal while tracing the production path.

- 2026-10-20 12:56 PDT — ops:0 supplied the morning evidence: target parley.nvim:4, Codex freshly switch-agent launched around 09:00 on f3c597f8 wrapper; requested sleep 60. UI showed Working (3s, esc to interrupt), then Waiting for background terminal (15s, esc to interrupt), 1 background terminal running. couch --interrupt refused with the nothing-sent/no-working-turn message. Source audit: Codex already has a visual observer; its recognizer only accepts Working in fixed physical bottom rows, so Waiting is a confirmed unsupported status. A top-anchored composer plus work status is also missed by fixed rows. No evidence yet assigns the first Working refusal to one guard.
- Existing Codex working.raw is a 154-byte status delta without a composer or complete synchronized frame; it cannot establish a composer-relative recognition contract. Capturing a fresh owned native PTY run before defining geometry. ARCH-DRY: reuse one Codex work recognizer for lifecycle and interrupt evidence, and preserve Claude working/stop-grace guards.

- 2026-10-20 12:59 PDT — Owned native Codex 0.163.0 capture reproduced initial Working refusal. In a 120x38 complete frame the status is at zero-based row10, the qualified bold composer at row13 (two blank rows between), footer row15. Diagnostics: composer known, picker false, lifecycle Generation0/Active=false/ActivitySeen=false, Interruptible=false. Coherent frames alternate • Working and ◦ Working; exact bullet matching is a second false-negative. Starting MCP servers appears at the same geometry and must stay excluded by wording. The owned fresh agent executed sleep20 and replied done; no background-wait frame appeared within the bounded70s capture. Child/reader teardown joined, temporary tests and copied login removed. Evidence under /Volumes/TestScratch/tmp/pair-slot2/codex-interrupt-vxjpuco9: capture.raw, working-prefix.raw, working.screen, coherent-statuses.txt, diagnostic.txt, version.txt. Existing targeted interrupt/Codex-recognizer tests pass and miss the combined production path.

## Revisions

- 2026-10-20 12:59 PDT — Fresh capture qualifies the proposal: accept both captured •/◦ spinner markers in the explicit Working/background-wait status grammar; keep MCP startup and arbitrary text excluded. This is evidence for the same per-agent recognition repair, not a broadened interrupt action. Design approval requested from ops:0 before implementation; no production/test edits made.

- 2026-10-20 — ops:0 approved the per-agent proof design (no new state): explicit Working/Waiting whitelist, both observed spinner glyphs, composer-relative location; MCP remains excluded. Waiting fixture is synthetic from the morning live-check text because the owned capture did not show it. change-code inferred quick flow. Native/synthetic production-output/writer tests went red before the Codex fix. Pure reducer test caught unknown-screen false confirmation and now requires fresh positive idle. Captured model-picker and recovery-key regressions caught over-broad screen projection during implementation; unknown is not invented work and recovery authority now derives after current work. Four mutations (composer region, Waiting wording, alternate spinner, cached flags) each fail the intended regression; restored afterward.

- Full captured stream replay additionally qualified native running-shell status: ◦/• Working (3s • esc to interrupt) · 1 background terminal running · /ps to view · /stop to close at row12 above composer15. Added sanitized captured background-prefix fixture; its positive and synthetic Waiting+summary cases failed before recognizing the optional count/singular-plural summary. Long status continuation rows are rejoined within the same bounded composer-relative region. Malformed/zero-count/unknown suffixes refuse. Full wrapcmd passed before this final status grammar refinement (137.581s); final focused race and recorded check follow. Vet and eight reader-inventory checks passed earlier.

- 2026-10-20 — BR-1 accepted: blanket stop-grace refusal in Done when contradicted the approved per-agent Spec. Qualified it as Claude grace refusal and explicit Codex current-work override, and added retained Codex grace coverage alongside the existing Claude refusal test. BR-2 accepted: completed capture row now separates captured Working/running-shell evidence from reported Waiting wording in synthetic fixtures. No production code changes.

### 2026-10-20 — review acceptance clarification (BR-1/BR-2)

The original Done-when phrase “Idle, stop grace, panels…” was too broad: Claude stop grace refuses, while Codex uses its current coherent work proof despite stale notification grace/completion. This clarifies the already-approved Spec; no action or state changes. The completed capture task no longer implies a native Waiting capture: Working/running-shell frames were captured; Waiting is reported text exercised synthetically.

### 2026-10-20 — failed staged live check and stable-marker repair

ops:0 live check failed on parley.nvim:3 after the operator relaunched it at13:48:07. Wrapper92530 was verified from a16907dd; supervisor was older. Supervisor work-proof consumption and drive wire files have identical git blobs in f3c597f8 and a16907dd, while Interruptible is computed by the wrapper: a Couch restart would not repair this recognition miss. The CLI's relaunch-Couch suffix is generic no-receipt wording. Current live peek showed Waiting for background terminal (1m02s • esc to interrupt) · 1 background terminal running… plus a separate └ sleep60 row before composer. Native capture witness confirms94x39, raw offset0 from13:48:09; preserved raw is being replayed read-only.

ops:0 approved the corrective simplification: anchor the current status group above the qualified composer on an explicit Working/Waiting verb and esc-to-interrupt marker, ignoring following counters, truncation and command-detail rows. Retain MCP exclusion, complete-frame/input ownership fences, one-write and positive-idle confirmation. This replaces fragile incidental metadata parsing within the existing recognizer; no new state or input action. Pin all three native captures. The queued check on69bd42d1 is cancelled through the managed verb because production will change; requeue on the corrected head.

- 2026-10-20 — Corrective evidence: sanitized styled viewport reconstructions from the authorized94x39 operator capture now cover Working at6s (row33/composer36) and Waiting at15s (row32, detailrow33, composer36). The Working frame passes the staged recognizer and writer replay, so it does not establish the cause of the initial Working refusal. Waiting reproduces a definite false refusal: the status ends with a truncated background summary and has a separate command-detail row. The production-output/writer regression fails before the repair. TL approved replacing incidental grammar with the explicit Working / Waiting for verb prefixes plus esc-to-interrupt hint in the same composer-relative status group, ignoring suffix content. Removed duration/count/singular-plural/summary validation; both spinner markers, startup exclusion, current complete proof, input ownership, one write and fresh idle confirmation remain. Original120x38 and operator94x39 fixtures pass focused package tests after the repair.

- Corrective verification: focused Codex/interrupt race tests pass (wrapcmd7.522s, agentdrive1.176s), vet passes, and all8 reader-inventory checks pass. Removing the interrupt-hint requirement makes missing-hint refusal fail; removing the verb whitelist makes MCP refusal fail. Both mutations restored and focused tests pass again (wrapcmd1.643s, agentdrive0.073s). Full wrapcmd and final recorded check remain pending; live acceptance still blocks landing.

- 2026-10-20 — BR-3 accepted: the approved stable-marker correction changes the executable recognition contract beyond BR-1/BR-2 documentation fixes, so the narrow re-close cannot admit it. Reconciled Spec to explicit Working / Waiting for prefixes plus interrupt hint and rerun the full close boundary. No production changes after the staged corrective head; its recorded suite remains valid.

- 2026-10-20 14:54 PDT — Recorded full suite passed on corrective production head3027d14a2cb85212e8a07e65202f416fb8457baa, exit0 in10m36s, sdlc-local/suite, logsha25677baa2e446e1. PR493 CI on current pushed close head is green (conformance2m16s, merge-check1m35s, local-gates10s). Full close SHIP remains current; no production changes since the tested/staged corrective head. ops:0 confirms live acceptance still waits on the operator relaunching parley.nvim:3. Keep the stage and do not merge before that check. #733 reassigned to pair:4; this slot awaits a different next item after742.

### 2026-10-20 — revised plan after partial live acceptance: work proof across redraws

The corrected3027d14a wrapper accepted admission on the21:47-relaunched parley.nvim:3, but requests d6903541/fdb77c3b refused at the supervisor pre-write guard. Confirmed source cause: DriveObservation.Sequence is the mailbox d.sequence; observeOutput increments it for every native output chunk. The supervisor guard, wrapper start admission and writer pre-write admission all require exact equality. No timer text is hashed or compared. Fixing only the supervisor would leave two later failures.

The existing complete qualified composer + whitelisted work verb/interrupt hint is the work authority. Its stable token must reflect operator input, not output. Proposed bounded change: retain the most recent operator-input value of the existing mailbox sequence (one uint64 per wrapper, initialized with the mailbox; updated alongside operatorInputLocked, admitImage and humanSubmit). For a positively interruptible observation, expose that value through the existing opaque Sequence field. Start and writer gates select the same token for InterruptOnly. Other drive requests keep the existing all-activity sequence. Instance, complete frame, current work/composer, ownership, picker, buffered input and one-write/positive-idle fences stay in place. No new wire field, store, retry, timer parser or deadline.

Simplest alternative: remove sequence equality for interrupts and trust current flags. Rejected because operator input can arrive and be forwarded between observations while current work remains eligible; a current predicate cannot prove the same input generation. Reusing submissions misses typing/Esc/image; lastInput is a timestamp and does not cover every path. The input-generation snapshot is held by the existing mailbox owner for its lifetime and makes that recheck one equality.

Actions and visible outcomes remain the existing table: explicit interrupt, inspect refusal/unconfirmed result, and a later new request only for current work. Benign working redraws now remain eligible; any operator input invalidates the request even if work still renders. No additional possible loss: one Ctrl+C still intentionally stops the current turn; uncertain writes are never retried.

- [ ] Add regression from an observed Working fixture at6s to7s/9s and Waiting detail redraw: call observeOutput before handleChunk exactly as production does, rather than fixture output alone. Admit with the FIRST observed token; tick once before wrapper start and once before writer dispatch. Require exactly one Ctrl+C, fresh idle completion and second refusal.
- [ ] Exercise supervisor recheck and both wrapper gates. At each gap, operator typing/Esc/image/submit must invalidate the request; picker/unknown/partial/changed wrapper/work-end/ownership still refuse. Ordinary model/recovery drives retain all-output sequence invalidation.
- [ ] Implement one shared selector for interrupt input token versus ordinary activity token and update all input-generation event sites in the same round. Mutation controls remove output tolerance or input invalidation and make the corresponding tests fail.
- [ ] Full close review, targeted/race/vet/inventory, new recorded check and CI on corrected code. Restage/build, then ask the hub to relaunch onto the corrected wrapper and repeat live acceptance. Keep old suite evidence as history only once code changes.

ARCH-DRY: one sequence-selection helper consumed by observation, start and writer admission. ARCH-PURE: token selection and current proof remain deterministic; no clock hash. ARCH-STATE: retain same drive/input lease, one bounded derived input token, no new user-action state. ARCH-CONSTRAINTS/ARCH-FUNERAL: constant work per existing input/output event, one uint64 per wrapper collected with that wrapper; no new durable artifact.

747 was ready but not admitted (733 held landing); its own queued retry was paused with SIGINT and suite-pass/ready-state checkpoint committed on its branch.742 stage is still3027d14a. This revised plan is sent to ops:0 before production/test changes because it corrects a third mechanism gap and adds an input-generation snapshot.

- TL approved the revised input-generation plan, with747 re-entering merge once742 reaches review. change-code admitted. Native tick regression RED: work stayed Interruptible but activity Sequence changed1->2 between6s and7s. Updated one lastOperatorInputSequence snapshot on typing/Esc/image/submit; observation/start/writer select that token only for interrupts. No supervisor production change or wire field. Service guard regressions retain fresh proof/instance/operator generation checks. A second regression pinned reader-seen-but-unparsed output: all3 gates wrongly trusted the old frame without the activity fence; central interrupt proof now also requires existing outputPending==0. This preserves the approved current complete-frame requirement, adds no new state, and all focused tick/input/unparsed/ordinary/service cases PASS. Full race/vet/wrapcmd and mutation controls in progress.

- Recheck mutation controls: restoring all-output token selection makes native timer regression RED; dropping operator-input stamps makes typing/Esc/image/submit revocation RED (including a stale Ctrl+C on submit); dropping outputPending proof makes unparsed-output observation/start/write RED. All overlays leave working sources untouched. Vet PASS; reader inventory positive preflight PASS so far, final8-test completion pending.

- Final focused verification on the recheck candidate: race PASS wrapcmd34.101s, couchcmd2.850s, couchmessage14.035s, agentdrive1.459s. Vet PASS. Reader inventory8 PASS66.451s. Native tick variants are explicitly labeled synthetic status-row repaints on the captured94x39 composer/footer; Waiting remains the captured operator frame. Full wrapper run and new recorded suite/CI/live acceptance follow; no claim of live interrupt success yet.

### 2026-10-20 22:06 — hub-owned live acceptance PASS

ops:0 verified parley.nvim:3's wrapper revision e767dc64d4e161b2539ae87251d276872b9a7bee after its22:06:30 relaunch. With sleep60 running, the native screen showed • Working (8s • esc to interrupt) ·1 background terminal running ·… above the composer. Public couch --interrupt parley.nvim:3 --confirm succeeded: Ctrl+C written and the turn ended (request06a82060). Codex displayed Conversation interrupted and returned its composer. A second interrupt refused with no working turn/nothing sent. This is the required native live acceptance for the corrected recognition and redraw-tolerant pre-write proof. Waiting/detail variants remain covered by captured/synthetic automated cases; no separate native Waiting interrupt is claimed.

ops:0 directed landing742 after747's turn, then clearing742's stage on pair:0 so the coordinated restart can rebuild main.747's merge runner remains active in the slot's primary checkout; this acceptance-only commit is made on742 in a temporary checkout without changing747's branch. Final recorded suite/CI still gate742 landing; live success does not waive them.

- 2026-10-20 23:10 PDT — Final corrected-code recorded full suite PASS on e767dc64d4e161b2539ae87251d276872b9a7bee: exit0 in9m45s, sdlc-local/suite, logsha2563e442eb1ea75. Full close SHIP and hub-owned live acceptance above are complete. #747 landed as PR498, so resume742 landing from the owning pair:2 checkout; preserve the accepted stage until742 lands, then clear it for the coordinated main rebuild.
