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
- [ ] Implement both approved bounded signatures and operation-owned typed raw evidence, with adversarial boundary and persistence tests.
- [ ] Document and complete native counter/control, focused checks, recorded suite and close readiness; publication remains gated by close and merge.

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
