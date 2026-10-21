---
id: 000761
status: working
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours: 2.3535
collaboration-mode: collaboration
card_mirror: '232644e80b60b6602d3d547fcca3bb513bb475b8' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-21T00:55:41-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# Couch native detach intermittently loses post-client ownership proof

## Problem

During #744 native qualification, real Couch leave-detach refused after stopping a worker client: `Couch: leave couch: detach 1-fixture-2: observe Pair session before retiring its incarnation: session "📁 1-2" ownership unresolved`. The remaining diagnostic was clipped by the terminal row. This preserves a real post-client-exit ownership refusal; no owner check was weakened.

## Spec

Diagnose the post-client-exit SessionOwnerUnknown result at couchcore/detach.go:191 and launcher/session_owner_os.go. Retain the exact refusal and both authoritative native pane replies plus server PID/generation/socket evidence. Fix a demonstrated product defect if found; otherwise identify the invalid fixture condition with a regression. Do not grant ownership from cached layout, pane titles or absence of evidence.

## Done when

- Reproduce or positively classify the observed refusal using an isolated real Zellij/Couch fleet.
- Preserve strict pre/post ownership fences and add a regression at the actual failing boundary.
- Recheck real detach/restart without leaving running owned generations.
- Recognize only the approved false-miss signatures within the existing three-attempt/deadline bound; retain typed bounded stdout/stderr/exit/version in the same operation failure evidence on unresolved pane parsing.

## Plan

Durable full-flow plan: [native pane evidence](../plans/000761-native-pane-evidence-plan.md). Product implementation waits for the expanded change-code gate.

- [x] Capture and classify native false-selection output, preserving original failures and exact cleanup proof.
- [x] Implement both approved bounded signatures and operation-owned typed raw evidence, with adversarial boundary and persistence tests.
- [x] Document native counter/control and focused checks, push the candidate, and request its recorded suite for close review; a passing final-head suite and SHIP still gate publication.

## Log

Observed on clean #744 head5aa2570ca7f9cb1be8771903df943618759ca492, native Zellij0.45.1 and installed SDLC8d34e76c815fe389f7979c966418dcdb2a04aaa5. One genuine Claude steward plus three non-hub workers registered; concurrent CLI queries and submitted/accepted ordinary message passed before detach refused. Full log /Volumes/TestScratch/tmp/pair-slot5/pair-744-native-xyneu9_b/native.log, SHA2569030df57986142026b443b6bd53158a0f1a29a049bb378813f4a94f457bfb2e1. Every owned process generation was later confirmed absent and the private root removed.

A deliberate diagnostic rerun at dcdffcea8f14b4465f025c3d40839968082e2a32 added public copy-notice capture only; its Claude subcase completed detach/restart30.60s and cleanup. That run did not reproduce the ownership refusal; it later failed the independent Codex fixture composer check. Full diagnostic log /Volumes/TestScratch/tmp/pair-slot5/pair-744-native-diagnostic-phd4gpjx/native.log, SHA256c83886c74d8553e3a44fef3082f2d198410bcdb94baad557602a71b69aff2dcb. This is not registered as a known flake: there is no same-exact-head fail/pass evidence. No product fix or root cause is claimed.

### 2026-10-21 — recurrence and initial trace

Final #744 clean candidate bfa769c1dec8ee80eabba775220d7084ae1c4983 failed48.08s after genuine Claude four-actor membership and submitted/accepted ordinary message. Public refusal: detach1-fixture-3, observe Pair session before detach, session 📁1-3 ownership unresolved: invalid live pane JSON, operation-392b33c981ed0bbda69e1a8a9d6763ed. Full log /Volumes/TestScratch/tmp/pair-slot5/pair-744-native-final.nEqfD0/native.log SHA2561d4b86f5c114a28e372bb025ee7c5eaf30355f05d301802d0f1ecddc420ce7b5. Exact owned generations absent, IO joined, root removed; no Codex attempt after failure. The authoritative bytes were not retained, so truncation, wrong reply and concatenation remain unproved.

Pair collection uses distinct stdout/stderr buffers and joins process/pipe collection; parser input is not limited by the observer's64KiB copy. Upstream taggedv0.45.1 emits pane JSON via serde pretty serialization into one Log; IPC reads a length-prefixed payload with read_exact. Numeric connection-ID reuse and late Log routing are established #748 context, but #761's offending bytes are still needed. Existing #748 recovery intentionally handles only empty or fully validated crossed kinds; malformed nonempty output remains a refusal. No product behavior changed during this trace.


### 2026-10-21 — native reproduction and classified false misses

Minimal native stimulus: start three private shell-only Zellij sessions; run four workers issuing1000 total actual read commands, alternating zellij --session NAME action list-panes --json --command (and --state), zellij --session NAME action list-clients, and zellij list-sessions --no-formatting. The existing production read boundary performs its own bounded attempts. Reproduction source and exact offending stdout/stderr are retained under workshop/evidence/000761; the source runs as a Go overlay in cmd/internal/launcher with TestNativeMixedCapture761. Each fixture owns private HOME/XDG/socket/config roots; no real agent, Pair wrapper or Couch broker is launched.

Mixed run14.86s:1000logical reads/1003attempts,11failedfinal (2paneJSON/9client parses),3empty attempts recovered. Both malformed pane responses exited0 with186bytes of complete ANSI-colored native session listings; stderr said the requested target was not found while stdout listed that exact live target. Taggedv0.45.1 src/commands.rs send_action_to_session repeats get_sessions; after a false miss it calls list_sessions(false,false,true), whose exit0 prevents the following exit1. sessions.rs assert_socket classifies any unexpected ConnStatus response as false. This establishes native false-selection/fallback output, not truncation or interleaved JSON. The existing plain crossed-reply validator rejects the colored listing and misses this signature. Full /Volumes/TestScratch/tmp/pair-slot5/pair-761-mixed-13d5hzyj/queries.jsonl SHA2561f71a30b666ac7bf0088d23f57d535c5fb3ac811f3fdac24d698f3b6dd37d65c; all three exact server generations absent, root removed.

The first one-session baseline had5exit1 final failures/1000 and no per-failure liveness observations. The focused bracketing control then retained24original exit1 failures/1000reads (2.4%), all exactly empty stdout/There is no active session! stderr. Each of24 diagnostic requeries succeeded under the original5s proof deadline; all24 kept PID11560 generation1792570266.40064 and its socket present before/after. Original failures remain counted. Focused JSONL /Volumes/TestScratch/tmp/pair-slot5/pair-761-exit1-errn9bsu/queries.jsonl SHA256aa179dbda866eaad83e59d7494694ccf840595fd9d6507f313544818aa8b0ed2. Exact cleanup passed. Baseline complete captures remain in pair-761-native-oe7dr7za; queries hash5974d31ba419fe9a0d9f201a774c248c91e11b8c4d7090c00be465f9aa9a19b2.

TL a9a79b15 approves validated colored false-miss recovery and persisted evidence; TL3aacb9e7 additionally approves exact exit1 signature only while target socket is present, under unchanged3-attempt/deadline bounds. Full plan records both signatures and counter-run acceptance. No product fix or counter-run pass is claimed yet.


### 2026-10-21 — implementation and adversarial verification checkpoint

Expanded full-flow change-code passed after naming function-specific adversarial strategies; estimate-quality INFO accepted the calibrated derivation with coverage caveats. Retry integration includes its stateful boundary tests; the cross-cutting item covers consumers/persistence wiring while the typed-data item covers cloning/codec/bounds; real-API discovery covers native counters/owner/detach controls and evidence processing. Focused checks are part of their implementation primitives; suite/review active verification belongs to the single review item, excluding queue idle. Estimate remains provisional, actual measured at close.

Both exact retry signatures now use the original allowlist/three-attempt/deadline owner. Captured colored186byte responses were RED before the classifier; strict parser/fuzz and stateful schedules now reject wrong stderr/target, exited-only targets, arbitrary escapes, missing/non-socket/symlink roots, collector failures and non-read commands. A negative regression also caught the old generic crossed-listing branch retrying nonempty native stderr after the new classifier refused it; that older branch now requires empty stderr, leaving native lookup errors to the validated signature. Final targeted recovery/748 races passed36.306s, parser fuzz12664executions and classifier fuzz51599executions.

Raw-field reader tests were RED for unbounded9000/10000byte streams and malformed formerly-unknown fields; now exact4KiB prefixes, original counts, nullable exit, strict types and unrelated unknown members survive normalization/rewrite. An actual NamedPairSessionContext refusal through OperationAttempt/Save/reopen was RED without cause propagation and now preserves exact raw bytes/exit/version under its nonce without exposing raw data in operator text. Clone/enrichment and64/128-attempt retention checks pass. Root ownership/persistence race subset passed1.397s.

Launcher capture qualifies the full original parser result, owns its prefix/argv before calling a preexisting observer, retains custom-IO raw bytes with unknown exit/version, and carries evidence through generation/cancellation fences without changing Unknown authority. Each gap had a focused RED regression. The first full launcher/core run exposed old tests counting diagnostic version calls as pane reads; explicit version frames and separate exact ownership-read counts preserve the retry cap. The shared stateful Zellij fake now models --version as an observed effect-free read (RED no-session response before its adapter fix). Recovery-native negatives and fake checks then passed2.657s/8.878s. Full affected-package rerun passed: launcher117.501s, couchcore326.412s, artifactpath14.341s and repolint20.356s. Evidence-bounds fuzz passed (9861 executions initially; final seeded run116), and the affected-reader inventory and diff whitespace checks passed. Final native counters/owner/detach checks and recorded suite remain pending.

### 2026-10-21 — fixed native counter qualification

Clean implementation head895bb09b113b7b1b9375350a13869560860bd824 was pushed and both prepared controls ran serially once, with no test-level retry. Single-session control:1000reads/1024attempts,0final failures (0% observed versus the focused baseline24/1000,2.4%);12socket-present exit1 misses,11empty and1crossed reply recovered. Mixed three-session control:1000reads/1010attempts,0final failures (baseline11/1000);4validated colored exit0 misses (three clients, one pane+state) and6empty replies recovered. Maximum2attempts/read; all intermediate failures remain retained. These samples establish observed residuals, not a guarantee of zero native failures.

Native0.45.1 binary /opt/homebrew/Cellar/zellij/0.45.1/bin/zellij SHA256604c684330fe097bc1363ada020d1b9b0ad17caa48d633e838869dfe9fc49e9c. Single runtime root1510417027/server76646 and mixed root755324766/three exact server generations were absent after joined cleanup; both roots removed and cleanup errors nil. Durable summaries, source and logs are workshop/evidence/000761/fixed-{single,mixed}-*. Full per-attempt files remain in pair-761-fixed-single-kfn404h_ and pair-761-fixed-mixed-57pdmvnf under /Volumes/TestScratch/tmp/pair-slot5. JSONL hashes respectively091e639ca25619afe76ff9ff7207728a69296d87c49196022e9c825f8b187c80 and afae5cc279beb84d9b24a913973650e49836d27568151c065383f66e9207d49d. Commands are in the durable summaries. PR511 opened for CI overlap; recorded suite admitted after35s capacity wait. Actual native owner/managed detach acceptance remains pending. Full supervisor restart belongs to #744 after #761 lands; no #744 helper is stacked here.

### 2026-10-21 — managed client acceptance and coverage gap

A once-only private overlay of main's existing TestStartupManagedAttachDetachTimingLive completed managed attach/detach and retained-session reattach/detach within the unchanged45s operation budget (3.58s). Its command FAILED solely on the existing helper's second PTY Close: file already closed. Exact server42939/generation1792572890.63797 was independently absent afterward; fixture and isolated socket roots removed. The full failing log is retained as workshop/evidence/000761/managed-reattach-native.log, SHA25649fdd7e0a1e461926b7d13edfa342319d278f28531fea2c8d86e36f29bffac92, with source/trace in /Volumes/TestScratch/tmp/pair-slot5/pair-761-native-reattach.Id4Vb5. Filed independent unclaimed pair#764 for that double-close and discarded-error class; no fixture fix is stacked here.

The trace also establishes that current warm attach/detach does not invoke SessionOwnerProbe, so this run is client behavior evidence only. A distinct owner-control schedule is required before close: explicit production Probe/Revalidate before resume and after both detach boundaries, each requiring Owned, the same exact server generation and counted native pane queries. No blind retry of the previous acceptance is used. Full Couch supervisor restart stays with #744 after this prerequisite lands.

### 2026-10-21 — explicit native ownership boundary qualified

The distinct owner-control overlay on unchanged source HEAD830f29d9ef0892b6fb8d06869a4840eed28f7ad9 completed in3.56s. Actual SessionOwnerProbe.Probe and Revalidate succeeded before the first Resume and after each of two Detach calls; each required Owned and exact server86965/generation1792573021.266006. Each boundary emitted2actual list-panes queries,6total, all stdout3848bytes/stderr0/exit0, with zero failure-only version queries. Both managed attach/detach cycles succeeded inside the same45s caller budget. Whole command still FAILED solely on the preserved #764 already-closed PTY diagnostic, not on ownership or client behavior. Independently observed exact server absent and both private roots removed; no residue or test retry.

Durable log/source are workshop/evidence/000761/owner-reattach-{native.log,source.go.txt}; full native trace/overlay in /Volumes/TestScratch/tmp/pair-slot5/pair-761-native-owner.fhLcYT. Log SHA256b06f0141da241ecc8cc6564d39a41d9ca31188cbe2c03701b0c18580b4bddb86; source SHA256aa974d1b81a16f32630c7626bf4b8d338bb883e89299b485f2a9b9e907b96eae. This supplies actual owner/managed-client acceptance, while honestly retaining the independent fixture failure; it does not claim a full supervisor restart. That final installed three-worker matrix remains #744's required post-landing qualification under the TL's prerequisite sequence.

### 2026-10-21 — complete suite found missing fixture-producer registration

Recorded suite on implementation895bb09b finished FAILED11m6s, with exactly one failing Go test: zellijtest.TestEveryZellijFixtureReplyUsesProductionDecoder rejected new launcher/zellij_query_evidence_test.go as an unregistered CLI reply producer. All other packages and non-Go groups completed successfully; no abort or passing retry concealed the failure. Retained failed log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/0b7f92c1bacfa84d218f99423185fa89.log (recorded hash prefix645c0a14fdc0). The focused fake-version subset had omitted the full producer inventory, so the missing generated producer cases escaped preflight. Fix the actual producer through the shared reply generator and production decoder, not an exclusion; rerun full zellijtest and the required suite on the changed head.

PR511 local-gates-guard also stopped before path classification with unexpected end of JSON input from gh api (run39573338651). No protected file changed and no operator label was requested or applied. Reran the failed CI run as a classified API/infrastructure failure; the guard still must decide successfully before landing.

### 2026-10-21 — producer-contract repair qualified

Missing producer registration reproduced RED6.807s. The new evidence producer now uses zellijtest.Case/ShellCase/Reply with producer-specific catalog entries in tests/lib/zellij_fixture_evidence_cases.json: exact malformed/empty/canceled/custom-IO/generation bytes are explicit rejected faults, and a valid owned-pane seed uses the production decoder. The large valid reply uses the shared semantic serializer with an irrelevant field expanded beyond70KiB. No scanner exclusion or production behavior changed. Full zellijtest passed12.916s, affected launcher evidence subset passed1.416s; reader disposition is now non_go_full_fallback because it genuinely reads the repository generator/catalog. CI rerun returned both local-gates-guard and its verdict green, no protected paths touched. Required replacement suite will run on the committed test fix; native observations remain on identical product code.

### 2026-10-21 — BR regressions and full long-temp class sweep

BR-1 positive/negative filesystem regressions were RED0.391s for actual wrong admission outcomes and now GREEN0.409s; recovery/path/colored-list races passed17.359s. BR-2's new long-TMPDIR regression was RED with bind invalid argument and its fixed focused targets passed0.620s, then0.597s normally/0.525s under an inherited160-character leaf. A full launcher run under219-byte TMPDIR completed63.515s and found only the older same-class WorkbenchPerformanceOwnerPass socket fixture failing for callers1/3/21. That prevents claiming full long-temp portability yet. Sweep all launcher Unix listener sites through one budgeted test-directory allocator, keeping counted owner-fence assertions and exact cleanup. No production authority delta beyond BR-1's native path spelling.

### 2026-10-21 — BR-1/BR-2 class corrections verified

The final test-only allocator covers all3launcher Unix listener sites; BR-1 budgets its longest actual bind path directly, lookup fixtures budget contract_version_1/name, and workbench budgets max session-N. Optional scratch allocation failures fall back without changing inherited TMPDIR; listener/removal errors stay explicit. Full launcher under219-byte inherited TMPDIR passed63.054s after the older fixture RED was retained; private long root removed. Final BR-1 focus passed0.418s; artifactpath/repolint passed7.645s/8.112s (and independently8.372s/9.156s). Reader inventories and staged whitespace checks pass. Durable BR-2 artifacts: workshop/evidence/000761/br2-{launcher-long-green.log,workbench-red.log,run.json}; green log SHA256f34ddcec12df8b870c2e4cb2e92fec64491ea77bbf1a71581f94715df15b6b78. Native counters, a replacement suite and the close review will qualify this committed correction before landing.

### 2026-10-21 — final corrected native counters

Clean committedee5caeb55b40f309cfab60cd401dfa27397cc443 ran both original control schedules serially once in fresh evidence directories. Single9.16s:1000logical/1042attempts,0final failures or malformed/context results;30socket-present exit1,11empty and1crossed replies recovered. Attempt histogram961one/36two/3three, so the unchanged three-attempt cap was exercised. Mixed15.13s:1000/1010,0final failures/malformed/context;7validated exit0 misses and3empty replies recovered, max2attempts. The observed single residual is0/1000 versus focused baseline24/1000; mixed0/1000 versus11/1000. No test retries or failures were discarded.

Every owned server generation absent, both runtime roots removed, cleanup errors nil. Durable summaries/logs/sources: workshop/evidence/000761/final-{single,mixed}-*. Full attempts and manifests remain pair-761-final-single.KqjM3v and pair-761-final-mixed.fSSjNJ under /Volumes/TestScratch/tmp/pair-slot5. Log hashes single2908fd092d9cc212799e8dc14112ffcb78c5056d6fab4ad4d25ff327d84c41be and mixed33318789ef909d6402178043b5038dc73fb9a3c7500bbc2c4a87c6b2e5adc6b7. Earlier baselines and fixed runs are retained separately. Final corrected suite is queued first; CI is green. Re-close now disposes BR-1 and BR-2; final-head suite success still precedes landing.

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md. Method A only; calibration is marked stale by estimate-source, so provisional. Read v2/v2.1 primitive ranges; design picks receive the thorough-spec0.2 factor, implementation picks the v3.1 0.4 scale. Familiarity1.0, single repo,15percent design buffer. Existing Go JSON/strictjson, query runner, fake and lifecycle codec cover the stack; no greenfield library discovery is needed.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.10 impl=0.08
item: api-integration design=0.40 impl=0.40
item: smaller-go-module design=0.03 impl=0.14
item: typed-data-prototype design=0.20 impl=0.14
item: cross-cutting-refactor design=0.12 impl=0.14
item: atlas-docs design=0.02 impl=0.05
item: milestone-review design=0.02 impl=0.14
item: real-api-discovery design=0 impl=0.24
design-buffer: 0.15
total: 2.3535
```

Respectively: issue/spec and reproduction record; native lookup retry integration; pure colored-list parser; bounded diagnostic value; two error consumers plus additive failure persistence; atlas; one boundary review; one external Zellij qualification/control. The unscaled implementation picks are0.2/1.0/0.35/0.35/0.35/0.125/0.35/0.6hours. Design subtotal0.89, implementation1.33;0.89×1.15+1.33=2.3535 focused ship-hours. Queue idle time is not added. Actual remains automatically measured at close.

## Revisions

### 2026-10-21 — TL159f625e: recurrence at the pre-detach fence

The first observation was post-client retirement. The final #744 rerun now refused at the pre-detach ownership probe with concrete diagnostic invalid live pane JSON. Diagnose the shared authoritative pane-query class across both fences, preserving all ownership requirements. #744 is parked/pushed f86a6c6a023ec4a794d0b7e2bdbe202c9c3ab689; this issue starts independently from landed main. No unlanded helper source is imported.

Investigation plan: use the existing #748 bare-native diagnostic as an owned Go overlay, with physical private HOME/XDG/Zellij roots and shell stand-ins only. Capture every completed production query, including successful subprocesses later rejected by ParsePanes, plus independent final validation. Run one bounded mixed pane/client read schedule; retain exact stdout/stderr, argv, exit and byte counts. Check cleanup errors and prove exact private native generations absent. No live slot or Couch broker is opened. After establishing bytes/root cause, add a deterministic failing boundary regression and record the smallest fix plan. Reject blind retries and widened ownership proof (ARCH-SECURE/ORDER).

Observer copies suffice for bounded replies and include truncation flags; an owned forwarding shim is available only if full bytes exceed64KiB. Reusing #744 source here is rejected because it would stack independent issues. Existing private native fixtures avoid that dependency. No product workaround is selected before capture.

### 2026-10-21 — TL19a8db73: operation-owned raw-query evidence

Verified submitted approval of the investigation adds a shipping requirement independent of final diagnosis: when Couch cannot parse authoritative pane JSON, retain bounded stdout, stderr, native exit status and Zellij version in that operation failure evidence. Keep operator text concise. Select capture using the actual full-input parser result, never validation of a truncated observer prefix. Preserve SessionOwnerUnknown semantics and strict ownership fencing. Native diagnostic capture is underway; additive evidence plumbing is sized outside the quick shell and will get a durable full-flow plan before implementation.

### 2026-10-21 — approved full-flow implementation contract

TL3aacb9e7 approves the durable full plan plus socket-present exit1 recovery. The original two prose Plan bullets covered investigation and generic verification; the current checklist names the established reproduction and complete implementation/acceptance handoff. Done when now explicitly includes both approved bounded signatures and typed raw operation evidence. No ownership fence or three-attempt/deadline budget is relaxed.

### 2026-10-21 — close readiness with queued suite

The worker contract explicitly allows close review to overlap a queued recorded suite, while requiring its final-head pass before landing and a replacement suite for any code fix. The readiness checklist now records requesting that check, rather than implying its result has already arrived. Replacement suite on72a985824c8c4053d23ae13204396c71f64cb176 is queued first behind an admitted ariadne suite. Native controls and producer-contract repair are verified; the prior failed suite remains retained. This changes verification scheduling only, not Done when, SHIP or landing requirements.

### 2026-10-21 — BR-1/BR-2: native path identity and socket-fixture budget

Close round1 returned REWORK: BR-1 requires preserving the native root spelling at the socket-present retry check; BR-2 requires fixtures whose socket paths fit regardless of inherited TMPDIR. Neither changes the approved retry signature or authority. Add both false-negative and false-positive symlink/parent traversal regressions through actual retry admission; append validated constant/name path components without cleaning the native root. Use a short private socket fixture with exact cleanup and a regression that deliberately inherits a long temporary directory. The queued72a98582 suite was canceled as obsolete through local-check cancel; it acquired capacity after419s just before cancellation reached it, so no passing/failing code result is claimed.

Class audit: #761 introduces one retry-admission socket derivation; its root must stay in native spelling. Other launcher socket derivations in terminal handoff/recovery concern existing canonical lease identities, not new retry admission; their separate lifecycle behavior is not changed by this correction. Existing ResolveZellijSocketRoot already documents the native spelling contract. BR-1 filesystem RED observed both wrong outcomes: native socket present/admission false, and unrelated cleaned socket present/admission true (a second undeclared fake call proved the false admission).
