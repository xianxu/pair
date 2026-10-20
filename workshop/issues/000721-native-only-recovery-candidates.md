---
id: 000721
status: working
deps: [pair#714]
github_issue:
created: 2026-10-19
updated: 2026-10-19
estimate_hours: 3.47
collaboration-mode: collaboration
card_mirror: '5789e1ac026318794f05d1e85aaa9e244254545a' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-19T21:10:41-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# Offer unconfirmed native recovery candidates without a usable screen

## Problem

Native recovery candidates currently require a usable capture even when the native transcript survives. Post-reboot file device identities differ; pair:3 raw capture also exceeds the recovery scan budget. A missing Pair log prevented causal binding. Native cwd/time candidates must remain weaker explicit alternatives.

## Spec

Transferred from #714 at the TL direction (2026-10-19). The following original M2/M3 plan sections define the approved work; references to #714 milestone numbers are historical and must be renumbered in this issue's durable plan. Do not execute until #714 lands and this issue starts on its own main-based branch.

Operator/TL approved native-only v2 context and five-minute selection lifetime: list unconfirmed candidates with cwd, last activity and native id; never auto-select; revalidate before any effect. No device/inode relaxation and no budget raise. Keep v1 capture-backed meaning unchanged and reject incompatible native-only authority in old readers. Preserve known partial observations and distinguish screen availability from native-scan completeness. All uncertain lifecycle outcomes share visible Launch unconfirmed and the Inspect recovery action.

### Original M2 — Independent screen/native evidence and explicit unbound choices

Files: `threadrecord/recovery.go` and context codec tests; `sessioninventory/recovery_collect.go` and tests; `couchcore/{recovery_evidence_os.go,recovery_preview.go,recovery_prepared.go}` plus new focused `recovery_native_only_test.go`; existing CLI/menu presentation and tests where source labels appear; atlas session identity/couch pages.

- [ ] Add failing fixtures: old device IDs with same inodes; absent/invalid witness; oversized screen; no Pair log/binding with an eligible Codex rollout; multiple same-cwd roots; proven binding plus failed screen; incomplete native scan. Assert last activity is explicitly sourced, native incompleteness is separate, and no listing provisions/binds/writes lifecycle state.
- [ ] Model typed screen availability separately from native scan completeness. Retain partial descriptive evidence when a read fails; report exact capture issue rather than generic conversation unavailable. A file mtime is labelled capture-file activity, never causal conversation activity.
- [ ] Add v2 native-only context source and validation/round-trip tests. Carry a fixed observation cutoff, source revision/digest, existing candidate fingerprints, optional attributable launch hint, five-minute expiry, and no capture. Reject mixed native-only/capture fields and invalid future times. V1 remains byte/meaning compatible.
- [ ] Extend collector request with explicit evidence-source kind. Native-only collection uses observation cutoff and optional launch hint, same root/cwd/interactive checks, no tail score and no assertion of a producing capture. Missing launch hint is honest weaker evidence, not an incomplete native scan. Reuse one metadata/read pass and existing stateful fake.
- [ ] Project no more than five explicit choices labelled unconfirmed native candidates with native ID/activity; `Automatic=false`, `Proven=nil` without established binding, no fabricated tail percentage. Plain resume still requires proven binding; candidate-key resume is the explicit choice. Candidate keys bind the source kind, cutoff, revision and exact ordered evidence; expiry is enforced at selection.
- [ ] Exercise changed native content/identity/cwd, duplicate ID, child/noninteractive session, malicious paths, unknown fields and stale keys. Compare listing/selection/menu JSON and text. Add N/2N visits/read counts and exact per-request byte-budget tests.
- [ ] Run `go test ./cmd/internal/threadrecord ./cmd/internal/sessioninventory ./cmd/internal/couchcore ./cmd/internal/couchcmd ./cmd/internal/couchtty -run 'Recovery|ResumeCandidates' -count=1`; log baseline mutation controls, update atlas and commit, then `sdlc milestone-close --issue 714 --milestone M2`.

### Original M3 — Final authority, restart conformance and landing

Files: `launcher/{recovery_authority_os.go,recovery_authority.go}` and tests; `couchcore/recovery_prepared.go` and end-to-end fixtures; artifactpath source inventory if new production files; atlas/index.md only for new atlas pages.

- [ ] Add failing native-only selection-to-exec fixture, interrupted launch/retry fixture, and independent launcher validation cases. Read/write/reopen records across each boundary using portable fake state. A copied selection or changed root must fail before native exec.
- [ ] Branch inferred preservation and final validation by typed source: capture-backed keeps existing archive/identity requirements; native-only persists its bounded native snapshot before retirement, revalidates chosen artifact/cwd/interactive provenance and expiry before claim/exec, and supplies unavailable-screen orientation only. Preserve existing exact stop-set confirmation and conversation-in-use checks for both.
- [ ] Confirm no ledger binding is manufactured, native argv has the selected resume ID, no prompt is injected, confirmed-binding resume remains available with no screen, Fresh remains available after genuine evidence failure, and late helper registration survives the M1 disposition race.
- [ ] Run `go test -race ./cmd/internal/couchcore ./cmd/internal/launcher ./cmd/internal/sessioninventory -run 'Recovery|Resume' -count=1` and `go test ./internal/repolint ./cmd/internal/artifactpath -count=1`. Run existing detached/warm/cold resume suites and all source/codec audits for new files.
- [ ] Ask hub for staged disposable live qualification using `sdlc stage --build`; verify emitted revisions, early-launch failure diagnostics and immediate retry, and candidate listing/explicit selection on the new binary. Hub owns real-slot actions; clear stage afterward unless landing clears it. Never reproduce by rebooting an active working slot.
- [ ] Update Log with measured failure-finalization bounds, scan counts, evidence classifications and remaining historical gaps. Commit/push final branch, run `sdlc local-check run --name suite`, and close M3/issue through normal gates. On SHIP run `sdlc merge --yes`, report verified PR/landing and stop.

## Done when

- Post-reboot output reports readable activity and the actual missing/invalid screen evidence without describing a complete native scan as incomplete.
- Unbound interactive native transcripts matching physical cwd appear as explicit unconfirmed alternatives with cwd, activity and native id, with no automatic choice or manufactured binding.
- Native-only context v2 has a five-minute choice lifetime and exact source revision/artifact fingerprints; changes refuse before effects and at native exec.
- Existing capture-backed, proven, Fresh, stop-set confirmation, live-conversation protection and scan budgets remain enforced.

## Plan

- [x] Start on main after #714 lands; write durable plan from the transferred Spec and run normal change-code gates.
- [x] Complete native-only codec/collector/selection/authority implementation and targeted verification from the durable plan.
- [ ] Obtain hub qualification decision on the tested candidate; run recorded suite, close review and landing.

Durable design: [native-only recovery candidates plan](../plans/000721-native-only-recovery-candidates-plan.md). Hub approval and full-flow change-code passed. The durable plan uses one atomic issue-close boundary because preview and final launch authority must ship together.

## Log

- 2026-10-19: Split from #714 by TL approval; native-only variant/lifetime approved. #714 lands failed-launch retry and diagnostics first.

- 2026-10-19: Parent #714 landed as PR #471, merge e63e7fd421fd34ad3308f566545527ef3a19666b, after integrating #695 and receiving renewed SHIP plus an integrated recorded suite PASS (8m45s). Hub report receipt 5a4adbd1-af20-4460-a389-fdab7b7ca0c6 submitted. Started this issue on its own main-based branch 000721-native-only-recovery-candidates. No implementation or change-code yet; durable plan remains next.
- 2026-10-19: start-plan refreshed the ariadne dependency and generated pending #699 changes in local-gates-guard.yml and merge-check.yml. Preserved them in stash named 721-start-plan-generated-workflows-preserved-for-699; older 715-start-plan-generated-workflows-preserved remains untouched. Do not include workflow migration in this issue.
- 2026-10-19: Carry forward the approved native-only v2/five-minute design and explicit unconfirmed labels without auto-selection, binding manufacture, device/inode relaxation, or budget increase. Re-read current owners before writing the plan: #695 has now landed transcript binding on the thread record and type-owned unknown-field preservation, so the earlier #714 codec assumptions must be verified against the current tree rather than copied. The Spec's original M2/M3 commands are historical: renumber actual boundaries for #721. Plan-quality and estimate follow normal change-code gates; no new implementation has started.
- 2026-10-19: Resumed saved continuation; inspected current preview/acquisition/collector/selection and launcher/codec owners, including bounded read-only subagent exploration. Confirmed recovery nested codecs remain strict despite record/TranscriptBinding preservation from #695. Drafted durable plan with source-aware v2 compatibility, fixed cutoff propagation through keys and reobservation, independent screen/native outcomes, shared retry byte accounting, state table and named-function adversarial tests (ARCH-DRY, ARCH-ORDER, ARCH-EVOLVE). No code edits or estimate; hub approval precedes change-code.

- 2026-10-19: Hub approved the durable plan and single atomic boundary in verified submitted receipt 6657d462-20bb-4bbe-b22b-39507571b925 (ops:0 to current pair:3 incarnation). Live qualification remains a decision on the tested candidate; hub default is production-path regressions plus the next natural recovery, not permission to touch pair:6. Baseline `go test ./cmd/internal/threadrecord ./cmd/internal/sessioninventory -run 'Recovery' -count=1` PASS; issue validation conforms. Entering change-code plan-quality gate, estimate afterward.

- 2026-10-20: Implementation checkpoint after full-flow admission: source codec/selector, bounded collector, independent evidence acquisition, native-only preview/selection, source-aware CLI/menu and final launcher validation are implemented. Core source/acquisition/selection tests observed RED before fixes; codec/collector/launcher workers likewise reported RED→GREEN. Combined targeted Recovery/ResumeCandidate tests pass in threadrecord, sessioninventory, couchcore, couchcmd, couchtty and launcher. The combined race run for Recovery/Resume passed in core (37.644s), launcher (9.694s) and inventory (2.927s), before final partial-fact/Fresh composition refinements; renewed final race and recorded suite remain pending. Artifactpath/testpair/repolint guard run passed; rerun after final test additions underway.
- 2026-10-20: Portable production constructor test exercises prepareSingleton→NewCouchWith using unchanged run.go evidence wiring, real disposable native home and injected clock one hour from host time; listing→explicit selection confirms exact native ID/saved argv/no prompt and no binding/capture manufacture. Removing the production evidence assignment made that test fail; source restored and targeted acceptance race PASS (1.948s). Cutoff-propagation and pre-retirement-persistence mutation controls also failed as intended, restored afterward. Native counted N=8/16 evidence: metadata 16/32, one listing, stat 16/32, reads 8/16, cwd resolutions 9/17; exact byte/metadata/root bounds covered.
- 2026-10-20: Integration regressions cover same inode with changed device, oversized/missing screen, absent/invalid witness, stale/copied/expired selection, interrupted retirement with durable reopen, and fresh listing after retained native expiry. No device/inode relaxation or larger scan budget. A failed retry now retains earlier descriptive facts; Fresh separately preserves a genuinely readable screen while native-only resume persists no capture. Known partial evidence remains incomplete and cannot issue unbound native keys.

- 2026-10-20: Final targeted six-package Recovery/ResumeCandidate run PASS, and renewed final race Recovery/Resume PASS: core 44.085s, launcher 7.289s, inventory 2.788s. Final artifactpath/testpair/repolint guard run PASS (10.667s/12.632s/11.294s). Restored mutation-control source and diff checks clean. Remaining: concrete hub qualification decision, pushed-head recorded suite, close review and landing.

- 2026-10-20: Verified submitted hub receipt c1e6ca82-f486-49b4-a933-08d3ab6ae897 approves production-path regressions plus next natural recovery on candidate c0c70e0e3f3da050e1d1b7f13478f8a7b21f3632; no stage and no action on pair:6. Hub owns natural verification/smoke calibration. PR #479 opened: https://github.com/xianxu/pair/pull/479. Proceeding to recorded suite, close and merge as directed.

## Revisions


- 2026-10-19: Added the concrete durable plan after main-based start and current-owner inspection. Original M2/M3 scope is retained as one atomic #721 review boundary, with unchanged approved behavior and added explicit codec/cutoff/accounting test contracts.
- 2026-10-19: Plan-quality PQ-1 requires compressed function-level test strategy rather than duplicated case inventories. Revised Tasks 1–4 accordingly; no architecture/scope change. First change-code inferred quick due to design line count; reran with --flow=full to honor TL-approved full flow. No implementation or estimate yet.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only. The named calibration is stale against its newer ledger, so these units remain provisional.

Decomposition follows the settled plan: source codec, collector extension, independent acquisition, selection/persistence, CLI/menu projection, launcher validation, portable conformance, atlas, one close review, and issue shaping. Existing strictjson/native parsers/lifecycle fixtures short-circuit greenfield design: 2.0h × 0.5 library availability × 0.2 spec discount = 0.2h for each greenfield concern. Smaller module design uses 0.2–0.3h × 0.2. All implementation entries already apply the v3.1 40% scale to table values (greenfield 0.8h → 0.32h; smaller 0.4–0.5h → 0.16–0.20h). Familiar codebase means familiarity 1.0; thorough approved plan means 15% design buffer. No external API discovery or cross-repo propagation. Design 1.10h × 1.15 + implementation 2.20h = 3.465h, rounded to 3.47h.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module design=0.20 impl=0.32
item: smaller-go-module design=0.04 impl=0.20
item: greenfield-go-module design=0.20 impl=0.32
item: greenfield-go-module design=0.20 impl=0.32
item: smaller-go-module design=0.04 impl=0.16
item: smaller-go-module design=0.06 impl=0.20
item: greenfield-go-module design=0.20 impl=0.32
item: atlas-docs design=0.02 impl=0.08
item: milestone-review design=0.04 impl=0.20
item: issue-spec design=0.10 impl=0.08
design-buffer: 0.15
total: 3.47
```
