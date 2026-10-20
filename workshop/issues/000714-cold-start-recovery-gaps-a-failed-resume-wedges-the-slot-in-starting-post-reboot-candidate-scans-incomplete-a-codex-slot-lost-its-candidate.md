---
id: 000714
status: codecomplete
deps: []
github_issue:
created: 2026-10-19
updated: 2026-10-19
estimate_hours: 1.24
collaboration-mode: collaboration
card_mirror: 'ae8bc0010e77e9e159f3be99adc86f8bf3eb7a6c' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-19T21:04:51-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
actual_hours: 0.60
---

# Cold-start recovery gaps: a failed resume wedges the slot in starting; post-reboot candidate scans incomplete; a codex slot lost its candidate

## Problem

First real cold start (10-19 ~15:55 hard crash, full machine reboot, 11 working slots down). Recovering the fleet by
`couch --resume-candidates` + `couch --resume` / `--reboot` per slot worked for 9 of 11 on the first try, but surfaced
four gaps:

1. **A failed resume wedges the slot in "starting".** `couch --resume ariadne:2` failed with "Pair exited before
   registering (exit 1)" and recorded no diagnostics (`failures.json` has no entry for nonce start-f65945f19dabb48e).
   Every retry afterwards refused for several minutes with "resume-starting: source is still starting; wait for its
   bounded launch … diagnostics could not be saved: invalid launch failure identity". Only `--reboot` (fresh, losing
   the conversation) got out.
2. **Every candidate scan was incomplete after the reboot:** each slot printed "last activity unknown", "current
   recovery witness names another file generation", "Conversation evidence unavailable", "Candidate scan incomplete".
   Resume still worked, so the warnings were noise and gave the TL nothing to judge by.
3. **pair:3 had no resume candidate at all** (codex, issue pair#712, its suite running at the crash): only "Start
   fresh" was offered, though its transcript should be on disk like the other codex slots'.
4. **No fleet view:** the TL ran 11 `--resume-candidates` calls and joined them with sdlc claims by hand (the "resume
   the fleet" verb is ariadne#456's scope; this issue only needs the per-slot outputs to be trustworthy).

## Spec

Approved scope: failed-launch disposition and diagnostics only. Native recovery discovery/context work is split to #721. See [durable plan](../plans/000714-cold-start-recovery-plan.md).

Under the existing per-thread operation hold, an owned helper that has completed plus an absent exact proposed terminal permits a nonce/revision-fenced terminal failure transition. Retain the selection/context, park and thread while removing only the failed creating incarnation. Present/unknown terminal or unknown helper evidence preserves the occupied claim. All uncertain dispositions are visibly Launch unconfirmed, with Inspect as recovery. Optional diagnostics must never decide that disposition.

Compose diagnostic identities atomically: partial outer evidence cannot replace a complete captured launch address or combine a tag/scope from different targets. Keep strict codecs and one terminal publisher. Unresolved refusals remain persistable by path/reference. No device/inode relaxation or scan-budget change is part of #714.

## Done when

- A recovery helper that exits before registration with its exact terminal absent records full launch diagnostics and leaves the same retained conversation immediately retryable; a second resume proceeds without waiting for stale-start age or choosing Fresh.
- Present/unknown terminal, unknown helper, canceled/failed probe, or failed/stale disposition commit preserves the exact attempt and reports Launch unconfirmed; no new agent, signal or terminal deletion follows uncertainty.
- Repo-scope-only outer resume dispatch preserves complete child launch identity and writes one readable failure record; unresolved refusals also persist, and diagnostic publication failure does not change lifecycle disposition.
- Ordering/race, file-backed restart and bounded lock-contention tests pass; the pushed final head has a recorded full suite pass before normal merge.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only; calibration currently marked stale by sdlc, so provisional.

Two smaller-go-module extensions cover failed-start transition/finalization with portable store/order tests and diagnostic composition with dispatcher regression. Both reuse existing stores, process fakes and transition patterns: no greenfield library dependency. Upper-end v2 implementation 0.5h becomes 0.20h each at v3.1 40%; thorough accepted spec discounts design 0.20h to 0.04h. Local lint is a third small extension (0.15 design *0.2=0.03; 0.30 impl *0.4=0.12). Issue/spec uses 1h design *0.2 and 0.20 impl *0.4. Docs uses 0.10 design *0.2 and 0.10 impl *0.4; one boundary review uses 0.10 design *0.2 and 0.50 impl *0.4. Familiarity 1.0 for recent #712/#715 owners; +15% design buffer for accepted durable plan. Sum design 0.35, implementation 0.84: 0.35*1.15+0.84=1.2425h, recorded 1.24h.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.20 impl=0.08
item: smaller-go-module design=0.04 impl=0.20
item: smaller-go-module design=0.04 impl=0.20
item: smaller-go-module design=0.03 impl=0.12
item: atlas-docs design=0.02 impl=0.04
item: milestone-review design=0.02 impl=0.20
design-buffer: 0.15
total: 1.24
```

## Plan

- [x] Implement confirmed failure disposition and immediate retry with state/order regressions.
- [x] Repair complete-address diagnostic composition and enforce the mechanical class.
- [x] Verify focused/race behavior, bounded failure paths and lint/source audits. Recorded suite and landing remain normal publication gates, tracked below.

## Log

### 2026-10-19
- 2026-10-19: closed — Production scope-only resume RED reproduced retained start and missing diagnostics; GREEN proves immediate same-native retry and reopened durable history. Focused race family 13.609s, CLI 1.049s, deadline race 1.343s, repolint/artifactpath 6.735s/6.008s; counted 8/16 releases each one observation and commit; diagnostic mutation control RED/restored GREEN and 35069 fuzz inputs. TL explicitly directed no stage/live check and owns next natural-failure smoke. Recorded suite f362417d7963a354d4add24c0f3358f2 queued on pushed implementation f767ed8491dd08bf6256eb9dedb7559501d56267; review overlaps queue per tl-worker, no landing before suite passes.; review verdict: SHIP

- 10-19 17:30 TL root-cause read (read-only, from records + source):
  - **pair:3 (gap 3): never bound, not lost.** Binding rows live in `~/.local/share/pair/repos/<scope>/ledger-<tag>.jsonl`
    (`kind:"binding"`, `confirmation_reason:"correlation"`). pair:6's ledger-1-pair-59 has them; pair:3's
    ledger-1-pair-58 has only the 10-16 15:02 launch row. sessionwatch (`cmd/internal/sessionwatch/run.go` ~232,
    send gate #420) correlates turns from `log-<tag>.md` after the launch offset; `readPairLog` treats a missing log
    as no turns, so it never binds. pair:3 had no log-1-pair-58.md. The codex rollout exists
    (`~/.codex/sessions/2026/10/16/rollout-2026-10-16T15-02-33-01a146bc-….jsonl`, cwd pair-slot3).
    `ResumeCandidates` (couchcore/recovery_preview.go:229) offers resume only via `ResolveEstablished`. Open: why
    no Pair-log turn in 3 days. Fix direction: bind by rollout cwd + launch time as a weaker candidate rather than
    offering only fresh.
  - **ariadne:2 (gap 1): failure path leaves the start claim.** `launch_existing.go:235-239` (unconfirmed) calls
    `fail(h, cause, nil)` with no rollback; the archived record kept `state:"creating"`, nonce
    start-f65945f19dabb48e, owner_pid = the live supervisor. `resume_route.go:171` refuses while `DecideStaleStart`
    (actionableinventory.go:700) is false; with a live owner only the 5-min age releases it.
  - **Diagnostics lost:** `couch --resume` sets repo-scope but no tag; `lifecycleFailureTargetForCall`
    (lifecyclefailure_record.go:54) builds an empty-Tag address that overwrites the launch's full address;
    `normalizeLaunchFailure` (launchfailure.go:95) rejects it ("invalid launch failure identity"). Also hit the first
    exit-1, whose message still claimed diagnostics went to failures.json (last written 15:49). Exit-1 cause unknown.
  - **Possible 4th:** supervision.jsonl has had no write since 15:56 though the post-reboot supervisor is its writer.

### 2026-10-19 — slot 3 planning

- Claimed and started #714 on its own main-based branch after the #715 landing. Read current launch/failure/recovery owners; no implementation changes. #699 workflow files are clean and remain outside this item.
- Confirmed current source paths from the TL read: recovery's unconfirmed tail retains every post-ack claim; outer diagnostic enrichment treats a repo-scope-only address as complete. Existing helper-exited uncertainty tests deliberately retain claims when observation is unavailable, so an unconditional rollback would violate the current safety contract (ARCH-ORDER).
- Read-only metadata for old pair:3 proves a device-ID mismatch: witness device 16777232, current stat device 16777230; raw inode 731744298 and events inode 731744297 match their witness values. This does not prove durable volume identity and is not a reason to ignore generation checks. Old raw capture size is 381260612 bytes, also beyond the 256 MiB recovery-read budget. Native discovery must not depend on accepting that screen (ARCH-SECURE, ARCH-CONSTRAINTS).
- A bounded read-only subagent traced existing candidate discovery: lack of established binding is not by itself why all inferred choices disappear; usable screen/witness and a complete scan currently gate them. The old missing Pair log explains absent causal binding, but its underlying capture/logging cause remains unproved. A native-only explicit candidate variant is proposed rather than changing watcher binding rules (ARCH-DRY, ARCH-PURPOSE).
- Durable plan written with state/action/loss table, terminal event enumeration, counted growth tests, v1/v2 compatibility boundary and three review milestones. Waiting for hub approval before change-code or implementation.

## Revisions

- 2026-10-19: Initial design separates definite failed launches from uncertain outcomes and proposes independent native-only candidate evidence. Reason: current on-disk device mismatch plus an oversized historical screen prevents safe reuse of the capture-backed inference contract; old acceptance criteria remain in scope.

- 2026-10-19, TL approval: close #714 after former M1; moved original M2/M3 into #721 (claimed for next, dependency #714). Merged disposition persistence failures into visible Launch unconfirmed. Native-only v2 and five-minute policy are approved for #721; no device/inode relaxation or budget increase. Current Done when and Plan now match the approved narrow contract; original broader requirements remain in the historical plan commit and #721 Spec.

- 2026-10-19, plan gate refinement: PQ-1 names a missing function-level testing contract; plan now gives one adversarial strategy/mechanical guard per risky owner. PQ-2 verified against threadrecord/recovery_context_test.go TestRecoverySchemaRejectsUnknownNestedFields: current ThreadStore rejects unsupported nested fields, so #714 retains that unchanged schema/refusal behavior rather than claiming existing preservation. No implementation started before gate acceptance.

- 2026-10-19: Implementation RED reproduced the actual outer scope-only resume, exit 23, retained creating claim, empty reopened failure store and immediate resume-starting refusal. Reused pure ReconcileStart rather than duplicate its policy; StartFailedConfirmed retains exact recovery choice. Added completed-child/live-group, terminal uncertainty, CAS race, wrong-helper and held-store tests. A separate RED proved outer recorder save refusal was hidden by OperatorError; corrected its concise notice without affecting disposition. Diagnostic subagent independently reproduced partial-identity loss and mutation-tested the repair.

- 2026-10-19: Focused race verification passed (couchcore 13.609s); CLI recovery/lifecycle tests passed (couchcmd 1.049s). Bounded terminal-observation test passed under race (1.343s); complete repolint/artifactpath audits passed (6.735s / 6.008s). Counted 8/16 failures use exactly one terminal observation and one revision increment each. Held real store lock refuses before release while preserving the exact claim. Diagnostics fuzz executed 35069 inputs; controlled direct-address replacement failed persisted-history regression and restored source passed. Added RED/GREEN proof that both save refusal and Launch unconfirmed survive concise operator rendering.

- 2026-10-19: TL explicitly directed no stage to :0 and no contrived disposable live check; land on the production scope-only resume RED/GREEN evidence, recorded suite, close and merge. TL owns verification at the next natural failed resume in smoke calibration. PR #471 published; recorded suite attempt f362417d7963a354d4add24c0f3358f2 is queued on implementation head f767ed8491dd08bf6256eb9dedb7559501d56267. Boundary review may overlap the queued suite under tl-worker rules; a recorded pass must precede landing.

- 2026-10-19: Recorded full branch suite f362417d7963a354d4add24c0f3358f2 PASS on f767ed8491dd08bf6256eb9dedb7559501d56267, exit 0 in 9m59s, log sha256 dae383f38fff (after 1486s machine queue). couchcmd 42.258s, couchcore 194.836s, wrapcmd 63.592s. Boundary round 1 SHIP, no Critical/Important/Minor findings; independent reviewer ran focused/race and complete source/codec audits. Entering normal merge; no staged or induced live check by explicit TL decision.

- 2026-10-19: Landing train held this car on a conflict with newly landed #695. Integrated origin/main and resolved the sole conflict in StartClaimed by retaining main's transcript-binding validation/store before the separate StartFailedConfirmed case. Extended the transition regression across fresh/proven/inferred recovery to assert both claim-time binding and failure-time preservation. Integrated focused race checks passed: couchcore 39.218s, couchcmd 2.995s; the added transcript-preservation regression passed separately under race (1.326s). Re-closing and recording a fresh integrated suite before re-entering normal landing; no gate bypass.
