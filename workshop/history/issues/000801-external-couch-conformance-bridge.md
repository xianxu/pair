---
id: 000801
status: done
deps: [pair#758]
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours: 3.75
collaboration-mode: collaboration
card_mirror: '4b57f70bf46d9dbd2d607d47a5bb7aac7a735900' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-22T07:20:32-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
actual_hours: 6.51
---

# External hold-open Couch conformance bridge

## Problem

parley.nvim#405 needs a real hold-open #758 editor endpoint producer for external Lua conformance. parley.nvim#408 exercised only public adopt-store/direct-store registry commands and did not start a daemon. Pair's cmd/internal/testcouch owns genuine disposable fleets but is a Go testing.TB API, with no external driver entrypoint. Reimplementing its lifetime and teardown in Parley would duplicate authority and risk live-state leakage.

## Spec

Provide a bounded Go testcouch owner/driver and thin external wrapper, reusing Build/New/Prepare/Start/Close/AssertCleaned. Build a candidate or accept an explicit candidate Couch executable, allocate a fresh private isolated root and private PATH from an empty environment, and start only the owned disposable Couch. No live roots, operator sessions, installed agents, network or default singleton authority.

Coordinate final argv/readiness/stop contract with parley.nvim:1 before code. Proposed external shape: explicit candidate binary plus owned scratch selection; one readiness JSON line with Version, canonical Namespace, IPC Socket and complete Environment map for clients (including private store/identity/Couch selection), diagnostics on stderr. Readiness must follow actual service admission, never mere socket creation. Bound readiness/output/startup and total lifetime. Stdin EOF or TERM requests the same idempotent joined stop, returning cleanly only after Close and AssertCleaned prove cleanup. On uncertain cleanup retain exact owned evidence/root and exit failure; no guessed PID cleanup or recursive removal before descendant absence. Parent owns launch timeout/process group; document crash recovery limits.

parley.nvim#405's real-producer acceptance depends on this bridge and #758. Per TL5e43c052, #405 may land its M1 on fakes and native unit evidence; only real-producer acceptance waits. This issue is independent work from main after its producer dependency lands, not added to #758's current review.

## Done when

- External driver can start a given/built producer, read a single confirmed readiness envelope, run Parley's isolated Lua driver in the emitted environment, and stop by EOF or TERM.
- Tests cover startup failure, early EOF/TERM, repeated stop, client/child cancellation and uncertain teardown without live-state access; genuine native conformance proves exact cleanup.
- Interface confirmed with parley.nvim:1, docs/entrypoint and inventories updated, recorded checks and normal whole-issue gates pass.

## Plan

Durable implementation plan: [external Couch bridge](../plans/000801-external-couch-bridge-plan.md). Full flow, one atomic close; hub and consumer approval recorded in Revisions.

- [x] Confirm executable/envelope/bounds with parley.nvim:1 and obtain hub design approval.
- [x] Extract shared error-returning candidate/fleet owner seams; existing testing.TB wrappers retain behavior and cleanup.
- [x] Implement pure protocol/lifecycle and bounded cancellable IO through the same stateful fake/native seam.
- [x] Add the thin executable/atomic build target, genuine native bridge and Parley consumer conformance, inventories and documentation.
Recorded verification, close review and landing remain the binary-owned workflow below; they are not implementation-complete checkbox claims.

## Log


- 2026-10-22: closed — Approved consolidated compatibility re-plan cleared plan-quality round4 CLEAN and change-code. Real public Build RED1.011s then GREEN1.324s; focused wrapper/strict-owner race3.971s; full testcouch25.861s/testpair6.786s PASS. Relative/non-clean placement RED0.208s/0.247s then cohort1.232s and race3.703s PASS. internal/repolint8.146s and final tracked-reader inventory PASS. Earlier strict response, legacy default and concept-map BR1/2/3 independently disposed round2; BR4 now class-wide wrapper candidate/lifetime/normalization controls, standalone isolation unchanged; no fourth hole found. Retained native Claude/EOF and Codex/TERM PASS36.648s and actual Parley driver nvim0.12.5/0.11.7 PASS are explicitly pinned to eee3547f, not relabelled as new code. Public DTO positive and causal decoder/default/wrapper controls qualify later corrections. Fresh pushed510d18f8 recorded affected check requested; review overlaps queue per tl-worker, final-code record required before merge. No extra native run or timeout bump. Hub owns project updates.; review verdict: SHIP
2026-10-22: filed unclaimed by pair:6 per TLe632079d/5e43c052 after inspecting testcouch and #408's actual direct-store probe. parley.nvim:1 provided proposed envelope and stop interface in receipt59bd1da0. TL will dispatch to a free Pair slot; #758 fixes its own review findings first.


## Revisions

### 2026-10-22 — dispatched bridge contract and shared ownership

#758 and #773 are landed; start-plan now owns a fresh main-based #801 branch. Consumer receipt b1bdc65f pins Version/Namespace/Socket/complete Environment, private couch+nvim PATH, explicit candidate/scratch selection, real service roundtrip, stderr-only diagnostics and joined EOF/TERM cleanup; receipt1fb645cd confirms its existing driver will consume this bridge rather than add a Couch launcher. Proposed executable is bin/couch-conformance-bridge with --couch ABS and --scratch-parent ABS; omit --couch to build the candidate. Proposed extra Documents array exposes exactly two actual registered private fixture IDs/paths. Proposed absolute startup300s, active lifetime600s, joined-stop15s and64KiB readiness/diagnostic caps need consumer confirmation and hub approval before code. The durable plan names the existing testing.TB coupling, shared error-returning extraction, stateful fake, late-output/stop ordering, cleanup uncertainty and real conformance. No standalone testing.TB shim, duplicate launcher or installed Pair fallback.

### 2026-10-22 — planning catch-up seed recovery

First start-plan catch-up regenerated .github/workflows/merge-check.yml and correctly refused a dirty resting branch (ariadne#557). Followed authenticated hub26eccf46: saved the920-byte diff to /Volumes/TestScratch/tmp/pair-slot7/pair801-ariadne557-seeded-94ukwvc8.patch, restored only that seeded file, then re-ran bare start-plan successfully. No workflow change is part of #801; local-gates verification is never self-applied.


### 2026-10-22 — consumer contract confirmed and hub approval complete

Hub0cdb7c57 approves subject to consumer confirmation; canonical consumerd0f4eb15 confirms all proposed bounds/argv/envelope with the parent-plus-side-doc layout now specified in the durable plan Revisions. Ordered Documents are artifacts/first.md and artifacts/first.side/second.md (type: side-doc, parent: ../first.md), with distinct registry IDs in Namespace. Keep invocation-PATH-selected native nvim for separate0.11.7/0.12.5 qualification; parent allows630s, closes editors before EOF and retains uncertain evidence. No additional envelope fields or CLI verbs. Proceed to plan-quality/change-code; no implementation before it clears. #739 is separately pushed ready at96062db5b1a73fe327f171b1e93e7a28966d1068 and remains unstaged per hub; it is never stacked into this branch.


### 2026-10-22 — plan-quality round1 refinements, no implementation yet

Automatic compact-design inference selected quick flow; re-ran with full flow to match the approved refactor's known code size. Plan-quality returned PQ-1 (candidate build isolation/provenance) and PQ-2 (function-level adversarial test strategies). Appended the explicit shared testpair.PlaceCandidate extraction, strict offline/allowlisted build config, checkout provenance and deliberate shared Go cache boundary; no installed Pair or ambient PAIR_TEST_BINARY in the strict path. Added the canonical named-function strategy table superseding illustrative test inventories. These refine the approved same-owner/isolation contract without adding a resolver, cache or executable protocol. Restart prompt arrived during review, but the review completed normally; no lost runner or duplicate dispatch. Re-run the gate before estimating/coding.


## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md, using the v2/v2.1 primitive procedure. Method A only, derived after plan-quality CLEAN round2. estimate-source marks calibration stale, so these hour values are provisional. Familiarity1.0: existing Go startup/fixture/ownership stack, not a novel runtime. Thorough plan discounts implementation-component design by0.2 and uses15% design buffer; issue/spec design is counted once without that discount. Implementation values below are already40% of v2/v2.1 rates, not scaled a second time.

Breakdown in block order: issue/spec and consumer contract (raw0.8 design/0.2 implementation); two bounded shared refactors (candidate/testpair and Fleet ownership/reporting, each raw0.4/0.5); pure bridge model/protocol (raw1.0/0.8, design halved because existing standard-library JSON/context and shared owner cover the stack); runner/service/native IO integration (raw1.5/1.5); executable/atomic target (raw0.2/0.5); one native endpoint/conformance discovery surface (raw0/0.6, both nvim versions qualify the same API); atlas/docs (raw0.1/0.2); one close review (raw0/0.5). Tests are included in each implementation primitive, not charged again as a separate test phase. No external library search/dependency or cross-repo code change is needed; consumer coordination uses its existing driver and is included in issue/API work.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.80 impl=0.08
item: cross-cutting-refactor design=0.08 impl=0.20
item: cross-cutting-refactor design=0.08 impl=0.20
item: greenfield-go-module design=0.10 impl=0.32
item: api-integration design=0.30 impl=0.60
item: smaller-go-module design=0.04 impl=0.20
item: real-api-discovery design=0.00 impl=0.24
item: atlas-docs design=0.02 impl=0.08
item: milestone-review design=0.00 impl=0.20
design-buffer: 0.15
total: 3.75
```

Design1.42h×1.15 + implementation2.12h =3.753h, rounded3.75h. Queue waits and operator-controlled #739 shutdown are not implementation work in this estimate.


### 2026-10-22 — implementation checkpoint; native acceptance pending

Full change-code cleared after CLEAN plan-quality round2 and estimate-quality INFO (optimistic component estimates, no blocking finding). Implemented shared error-returning CandidateOwner/NewOwned, keeping the testing wrappers; pure protocol/reducer; actual native service adapter; cancellable owned pipes; thin atomic executable and bounded diagnostics. All implementation began after accepted design48cfe351. No full-suite or native acceptance is claimed yet.

Meaningful RED controls included expired-owner admission, missing public APIs, missing selected build utilities, leaked Go metadata config, live scratch overlap reaching tool lookup, duplicate JSON admission, absent per-call deadline, and blocking stderr defeating cleanup. Targeted portable tests and race cohorts pass. The full unread-stderr regression failed with old blocking os.File.Write, then passed using one nonblocking syscall. Stdin/stdout/stderr are pipe protocol endpoints; unread stderr produces a prompt failed diagnostic sink rather than blocking teardown. No retry, goroutine or queue was added to output. Reader guard now recognizes the new native helper APIs; actual helper consumers use non-Go full fallback, while literal-only controls remain local fixtures. Artifact manifest lists the new qualification-only sources explicitly.

First actual executable diagnostic run built candidates offline and started the genuine Claude fixture, then correctly refused duplicate PATH in the shared Fleet.Environment; exact owner cleanup passed. Captured log: /Volumes/TestScratch/tmp/pair-slot7/pair801-entry-smoke.log. This is diagnostic evidence, not acceptance or a source-complete pass. Root cause: testpair's defaults plus explicit overrides were represented as duplicate entries (exec used last-wins), and EnvironmentAt selected its shell helper from consumer cwd. Added failing regressions and fixed the shared fleet boundary to use the selected source helper with unique last-wins keys; no adapter workaround or ambient fallback. Final native rerun and external consumer handoff remain pending. #739 stays ready and unstaged under the hub's hold.

Shared helper verification after the runtime environment correction: full cmd/internal/testcouch PASS42.052s; bridge/CLI/testpair portable checks and targeted race cohorts passed. The fast reader-inventory check is refreshed only for audited new/changed readers. Native qualification is next.

Native qualification1 reached genuine registration and both public editor-endpoint lookups, then its own editor assertion rejected Neovim's normal cwd-relative argv projection (40.629s). Fixed only the assertion to request native absolute paths; removed literal-ID echo checks because actual ID authority is independently tested by the public lookups. Exact fleet/candidate cleanup succeeded. A second bounded native invocation is running. Artifactpath's complete package also caught the missing root binary .gitignore entry for the new main package; added that entry, and its exact failing test now passes. internal/repolint passed. No cap, timeout or refusal was relaxed.

Native qualification2 failed after303.101s: fixed300s startup expired, no wrapper-startup log or startup sources, then exact cleanup proved all generations absent/direct child and IO joined. Observation receipt (not a full raw capture): workshop/evidence/000801/native-attempt-2-observation.json. No new retry or deadline change. Hub97f22b2d authorizes exactly one further bounded native run after the two ad-hoc audit fixes, with the cached console frame on failure; another startup deadline must be classified and reported before further runs. Ad-hoc audit found detached constructor cancellation and wrong remaining-root recovery evidence; red-test corrections are in progress. PR548 is open; its current four server checks passed before these corrections. Recorded verification and close remain pending.

Both lifetime audit findings are disposed by read-only recheck. Four native-owner causal regressions passed after RED; bridge suite PASS0.261s, native-owner race PASS1.506s. Shared admission/install controls passed after RED4.787s: focused5.273s, race11.572s and full testcouch30.881s. Cancellation is checked between32KiB copy chunks and before later installation effects; completed admission does not cancel the admitted lifetime. Full artifactpath passed10.359s, internal/repolint passed23.365s, complete reader-guard Python suite passed21tests116.850s. Fast inventory refreshed for precisely owner/native reader changes and the literal cancellation fixture. Next action: commit/push these corrections, then one hub-authorized native invocation captured to a file; another300s failure must be classified and reported before further runs.

Hub-authorized native qualification3 PASSED36.648s on eee3547fd06b624351dee871a4f0630a03d2f612: Claude/default candidates/EOF16.09s and Codex/explicit fresh candidate/TERM20.39s, both actual public endpoint lookups, native nvim0.11.7 access to the advertised parent/side documents and exact joined cleanup with fleet/candidate roots absent. Full capture and provenance: workshop/evidence/000801/native-authorized3-eee3547fd.log/json; SHA2562280b09adb67db75226bbf254cda23900f187554557d14ef188f360aff960f7a. No timeout extension, retries or extra native invocation. Hub report d43cf375 and consumer handoff8790868a both submitted. The earlier startup stall remains unclassified, with cached-screen diagnostics now available; this pass does not invent its cause. Actual Parley consumer/nvim0.12.5 confirmation remains outstanding. Executable rebuilt; PR548 updated. Record the pushed-head suite next, then close only with the outstanding consumer disposition explicit.


### 2026-10-22 — checkable implementation and gate-owned completion

The earlier combined checkbox “Record checks, close review and land through sdlc” mixed implementation with the future gate that itself refuses unchecked implementation rows. Preserve those three requirements as workflow instructions, not a pre-landing completion claim. All four implementation rows are complete; recorded suite, review and landing remain explicitly pending until their receipts exist. The durable plan's final workflow row has the same disposition. No Done-when or required evidence is waived.

Consumer738380b8 reports its actual existing #405 driver PASSED against the clean eee3547f executable on invocation-PATH nvim0.12.5 and0.11.7; Git verified consumer source d660ec6f1a3fef56d6fb8a1f583dd1bbd08cc1ff. Both real checks cover native handshake, exact supersession notice, same-attach retry, desktop takeover, stale register/unregister, detach/lookup retirement, side doc and same-editor distinct listeners; corrupted-identity fake rejects. Both bridge sessions exit0 after joined EOF cleanup before consumer runtime removal. Retained the two consumer summary logs plus provenance under workshop/evidence/000801/parley405-*; both summary hashes2d86297f0e5b910895840312755a4bbae5629fe353301a5362c45d86bc1dd078 (the summaries intentionally match; they are not raw producer transcripts). Consumer waits for the eventual landed pin; no native gap remains.

Recorded affected selection expanded to suite on pushed48162a9b506a5e8454988ab9923417ba4e3280e3. Existing run lease-869401233/attempt45de03aef78b2a5e9d3e5127d4d87024 survived the Couch restart and remains queued; restarted nothing. Runtime code still equals eee3547f; later commits carry documentation/evidence only. Per tl-worker's explicit overlap rule, request the close review while this recorded run waits, but require its passing final-code record before landing. No source/runtime changes, suite cancellation, FIFO change or fabricated status.


### 2026-10-22 — close round1 REWORK and supported supersession

Close returned three Critical findings, all accepted: BR-1 exact response member decoding, BR-2 legacy Build default temp aliases, BR-3 concept-map traceability. Cancelled the still-queued old-code lease-869401233/attempt45de03aef78b2a5e9d3e5127d4d87024 with local-check cancel; it exited before acquiring capacity or running tests. This is superseded verification, not a test failure. No FIFO waiver, duplicate runner or hand-posted status.

BR-1 RED reproduced26 invalid acceptances across54 malformed member controls. Shared exact-member projection rejects case aliases, absent/null consumed fields and wrong types at both response envelopes and nested registration; opaque additive metadata including1e10000 survives validation. Actual public EndpointReply marshaling is a positive control. Bridge package PASS0.512s and native-owner race PASS1.378s. BR-2 RED0.239s reproduced the absent-RAM-disk default alias regression; Build and New now share canonical default selection while explicit BuildOwned aliases remain refused before effects. BuildOwned/Owned cohort PASS10.145s; focused race PASS1.239s. BR-3 corrects the durable plan tables to real names/files and includes native, pipe and diagnostic adapters. The prior real native/consumer captures qualify eee3547f, not these later decoding/default fixes; no further native run has been made. Fresh final-code recorded verification and re-close follow.

Final correction cohort: make test-affected-inventory PASS; whole testcouch36.033s, bridge0.514s, CLI0.170s, testpair10.257s, couchmessage26.110s. Bridge/CLI race PASS1.718s/1.219s. Reader certificates refreshed only for audited native_test and binaries_test content. Hub update647f8de5 submitted. Commit/push the corrections before recording a fresh check; re-close may overlap the queue, and landing still requires a final-code passing record.


### 2026-10-22 — close round2: prior fixes verified; wrapper policy regression

Round2 independently reproduced RED for BR-1 and BR-2 by removing their fixes, and disposed BR-1/2/3 as addressed. New BR-4: Build no longer honors suite-selected PAIR_TEST_BINARY or the existing process-owned fallback; it rebuilds Pair by calling strict BuildOwned with no selected candidate. Restore selection through the existing resolver only in the legacy wrapper, keep standalone inputs isolated, and test selection plus source/copy cleanup lifetimes. Plan Revision states the class rule and enumerates three direct consumers. Hub report1eff9b59 sent. Cancelled still-queued lease-1942691207/attempt e106bd3540289b5c44fc1feb633c7402 through local-check cancel; terminal confirmed interrupted before tests, not a failure. No additional native invocation.


### 2026-10-22 — stop and consolidate compatibility boundary

Independent read-only Build/New/testpair sweep found legacy relative placement also tightened by the new strict PlaceCandidate API. No existing in-tree relative consumer is claimed, but preserving the API contract requires an explicit boundary adapter. This is the third compatibility delta; stopped incremental implementation and prepared a consolidated Revision in the durable plan for hub approval, naming the unchanged-wrapper duplication alternative and complete behavioral matrix. No suite or native run is active; both obsolete suite attempts were supported cancellations before tests. BR-4 current atomic work will be retained; no new relative-path shim until approval.

Fleet-owner stopped before any BR-4 edits or tests. Its sweep confirms three Build callers across two packages, both using testpair.Run. Preserve the existing Python3 placement dependency in the wrapper-selected branch. Re-plan send99b4a038 returned uncertain because the supervisor socket was absent; query its receipt before any further send. Await hub approval; no active suite/native run.


### 2026-10-22 — authenticated re-plan approval, before implementation

Restart prompt3f80ea8f authenticated; old uncertain receipt99b4a038 was absent after restart. Explicitly resent the still-relevant approval request as62681647, confirmed submitted. Authenticated hub025f85c5 approves the consolidated thin-adapter boundary and requires plan-quality before code, then a stop/report if a fourth compatibility hole appears. Read-only sweep verifies New defaults/lifetime/cleanup and testpair.Run source ownership, with existing tests; all12 current At consumers use absolute paths. No BR-4 implementation yet. Record and run change-code’s plan-quality gate now; no bypass or extra native invocation.

Plan-quality round4 CLEAN, prior PQ-1/2 remain disposed, no fourth compatibility hole found. Full change-code completed exit0 at c2261ce8d; estimate-quality INFO again notes optimistic original component estimates, with no fabricated derivation or blocker. Do not backfit the original estimate to current actuals. Approved wrapper and placement work now starts RED-first in disjoint existing files; standalone bridge inputs, protocol and native bounds remain unchanged.

Legacy destination-normalization controls reproduced both wrappers refusing relative outputs (RED0.208s), then the already-identified non-clean restriction on absolute outputs (RED0.247s). The thin adapter translates relative destinations against the selected checkout and cleans the destination once before strict placement; it preserves the caller-visible return string. Direct strict PlaceCandidate continues rejecting relative/non-clean explicit paths before any runner. These are the same approved normalization class, not an additional resolver or scope expansion.

BR-4 meaningful real-wrapper RED1.011s reproduced replacement of both supplied and process-fallback Pair executables. Thin BinaryContext/private-selection correction GREEN1.324s; focused wrapper/strict-owner race PASS3.971s. Supplied mode proves zero fallback compilations and source survival after private cleanup/process exit; unset mode proves one fallback across two per-test Build cleanups, then Run removes only its fallback. Python3 is explicitly selected only when the private legacy candidate is present; strict standalone isolation stays green. Full testcouch PASS25.861s and testpair6.786s; normalized-placement/lifetime cohort PASS1.232s and race3.703s. internal/repolint PASS8.146s. Fast inventory PASS after staging the new reader and recording its non-Go fallback; no exemption or new production reader. No fourth compatibility hole found; cross-volume mismatch remains excluded.

Next: commit/push final-code corrections, start a fresh recorded affected selection (may expand to full suite), and re-close with prior real native/consumer evidence honestly pinned to eee3547f plus these causal wrapper/decoder regressions. No further native run, deadline extension or replay of a cancelled check. Hub owns project updates.


### 2026-10-22 — SHIP; recorded verification and landing retained

Close round3 SHIP, all BR-1..4 addressed, no new finding/fourth compatibility hole. Reviewer independently disabled candidate forwarding and legacy normalization with temporary overlays and confirmed meaningful RED, then passed portable packages, bridge/CLI race, socket-path tests and inventory. Actual measured6.51h adopted by SDLC; original3.75h estimate retained. Close evidence bd3b72585ff673a975b203c853e3d366ba8055d9, mirrored close head07c924a7a22ff4e4c3cc4f20e92b7fbcfd7a27ad pushed.

Fresh recorded check on runtime510d18f8b3893c19c177016774019b57740b2222: lease-1706974001, attempt337bc13a6ec37b4446208f5011bc436a, tool session58500; affected expands to full suite, queued normal FIFO (last observed17). Log in common Git sdlc/test-stats/inflight/337bc13a6ec37b4446208f5011bc436a/run.log. Entered normal sdlc merge --yes after SHIP, tool session41525; position1 behind #797/pid9277. Preserve both runners and their evidence, no duplicate checks, no bypass; verify actual run result before advancing after any restart. Hub report10f48335 sent. Consumer still awaits final landed pin. No continuation/restart requested while this owned check is queued.

Merge session41525 exited1 at pre-admission gates: PR548 CI conformance/merge-check pending, plus required recorded affected/full fallback not yet passing. It explicitly says to retain the already-queued check and retry merge after it passes. No active merge runner now; do not mistake the earlier position1 observation for admission or landing. Retain only suite session58500/lease-1706974001 (normal FIFO, last17). Runtime remains510d18f8; subsequent commits are Log/evidence only. PR updated with this durable state; next action is read that existing suite’s actual result, wait for CI, then re-enter sdlc merge --yes. No duplicate verification or main integration before the landing gate.
