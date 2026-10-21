---
id: 000748
status: codecomplete
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: '526b666c043328a38919d4f599a2377c9012958c' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T17:33:54-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
actual_hours: 1.11
---

# flake: TestRecoveryRealHelperAndSessionConformanceLive/warm (zellij list-panes no reply after helper exit) holds the pair landing line

## Problem

`couchcore.TestRecoveryRealHelperAndSessionConformanceLive/warm` in the required `couch-zellij-conformance` job fails
intermittently on GitHub macOS: "recovery session state could not be checked; retry: observe live session owner:
zellij action list-panes --json --command returned no reply" (helper exited before recovery; the final query got no
reply). It failed on pair#706's heads twice on 10-20 (run 39501347923 attempt 1; run 39521409698 job 118965862227,
log SHA256 cd823c3894b2ef291a8c4ae4be3b0c70c062aa9ada5fa3d8b436538e393c4f69) and passed on rerun once; #706 touches
no couchcore, launcher or zellij code. Registered in pair#636. Because conformance is required, each occurrence
holds the pair landing line.

## Spec

Root-cause it (no timeout bump, no retry loop): reproduce with the exact CI geometry and timing (a fresh HOME, pinned
Zellij 0.45.1), find why list-panes gets no reply after the helper exits (session teardown racing the query? a
socket that's gone?), and fix the test or the product, whichever is wrong. Record the capture as a fixture.

## Done when

- A deterministic reproduction exists (red before the fix), the fix passes it, and 20 consecutive local runs of the
  case pass; the pair#636 entry is closed with the cause.

## Plan

- [x] Add red deterministic read sequences for empty/crossed replies, exhaustion, cancellation, nonzero exit, malformed data and non-read argv. Reuse the stateful CLI-frame fake and existing strict reply parsers.
- [x] In launcher's query boundary, allow at most three total attempts for allowlisted idempotent reads: full unformatted session listing, list-clients, JSON list-panes with existing command/state flags. Retry only exit-zero empty replies or a validated different reply kind. Share the original five-second/caller deadline across attempts; no sleeps. Observe each attempt and retain final typed refusal.
- [x] Update existing immediate-refusal tests to exhaust the three-attempt policy; retain no-effects and healthy-recovery assertions. Verify counted bounds at N/2N, cancellation before later effects, and no retries for mutation/unrecognized argv, nonzero/collector errors, malformed same-kind payload or conflicting valid observations.
- [x] Run native control through the production query boundary for 2,000 logical requests and retain first-attempt plus final-answer counts; run exact park→warm geometry and 20 consecutive warm cases. Update atlas, lesson/prevention guard, and #636 with the supported cause.

Verification: recorded affected/suite and boundary review precede landing.


## Log

### 2026-10-20
- 2026-10-20: closed — BR-1 fixed: count every child before schedule validation; exact typed error/output assertions; unsafe retry gate mutant deterministically fails nonzero/mutation/unknown-flags; all frame-fake consumers swept and targeted launcher27.422s passes. BR-2 corrected native control validates all2000 final pane payloads with production required command/state/geometry/tab fields and nonempty inventory:2003 attempts,3 empty-attempt errors recovered,0 invalid/error finals in46.28s. BR-3 comments distinguish logical reads/attempts/attempt-local observer errors.20 consecutive native warm cases and final exact park-warm pass; changed-contract race23.976s/15.040s; source-reader audit passes; recorded final-head full suite required before landing. (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP
- 2026-10-20: flow upgraded quick → full — 103 added lines in code files (limit 100)

### 2026-10-20 — native cause investigation checkpoint

Claimed on pair:3 and started a separate branch at main `127001eae6379ce571b6097f4573deb57ca90416`; #743 remains claimed and its pushed branch preserved. No production or compiled test changes. No image, installation, operator session or machine settings changed.

Downloaded completed job 118965862227 through the GitHub job-log API (the containing workflow is still running, so the run-log command refuses). Raw downloaded log SHA256 `db34d56cac77ff292937babcfeecedc1b944a209751528a2bc34220843cd10e4`; this differs from pair:6's extracted-log hash because the downloaded representation includes the job's timestamps/escapes. The original recovery observer capture is retained at `workshop/evidence/000748/ci-failed-query-capture.json`. Local client exit/helper wait occurred at 00:30:13.694; four pane replies plus two session listings/header-only clients succeeded afterward. The final owner query at 00:30:14.004690 exited 0 after 21.787 ms with zero stdout/stderr and no cancellation. Native log shows the CLIENT's stdout-EIO panic on PTY closure at 13.694; server/plugin shutdown begins only at fixture cleanup (14.049), after the refused query. Helper death or client exit is not evidence of server death.

Fresh-HOME current-main control with installed Zellij 0.45.1: exact park→warm package selection passed once (park 1.50 s; recovery 1.84 s). This sample is not acceptance or a fix. Retained local log `/Volumes/TestScratch/tmp/pair-slot3/pair748-uv7wouog/baseline.log`.

Reconstructed #640's raw-protocol control with the current private-root/startup fixture and fresh HOME, no proxy: 2,000 independent sequential ConnStatus/ListPanes requests, 45 bad FIRST replies including 5 crossed reply kinds; test fails in 1.85 s. Every request is independent; no failed request is retried. Retained source, result and bounded first-response records are under `workshop/evidence/000748/`. Initial diagnostic setup attempt lost an output-path environment variable to the fixture scrub and failed opening an empty path; it is excluded, and the corrected diagnostic uses an explicit owned path. No product behavior was changed.

ARCH-ORDER: pinned v0.45.1 frees the lowest numeric client ID before all duplicate cleanup/queued replies have finished. A later connection can reuse it; old cleanup deletes the replacement's sender, and pending replies can go to another replacement. Source: lib.rs:655–672,1617–1619; route.rs:2676–2678,2941; os_input_output.rs:448–503 at release commit `efd8fd5a89a20c07a111d248ad7fce53848d2c18`. Raw crossed replies establish actual native misrouting, but do not identify every internal instruction in the particular CI failure. The CLI maps premature Disconnect/Unblock to successful empty output (existing #640 protocol controls and launcher empty-reply regressions).

A detach-only change cannot establish the required cause fix: stock CLI actions run ConnStatus discovery even with an explicit session, recreating the cleanup race independently. Serialized native requests already fail. No stock retirement acknowledgement exists. Keeping typed unknown/refusal is correct; replacing it with empty inventory, retries, a sleep or a timeout increase would be wrong.

Verified actual upstream PR5630 server diff (retained): monotonic client IDs and initial ConnStatus fast path before ID registration; latest published release remains 0.45.1. The mixed PR's title/body do not mention these changes, but its actual diff does. Prior #640 operator disposition declined a local Zellij build pending release. A cause fix therefore needs hub scope/dependency-policy guidance before coding: qualify a pinned patched native dependency, or explicitly revise the healthy-native success contract. Current Done-when remains unmet; no fix or deterministic forced native schedule is claimed.

## Revisions

### 2026-10-20 — hub-authorized bounded product mitigation

Authenticated receipt `d8569808-2a30-4c3d-847f-84bbcc7455d5` supersedes the original prohibition on all read retries: the hub explicitly chooses a narrow live-product workaround for unreleased upstream PR5630. No test rerun or unknown-as-success exemption is authorized. The original native binary remains 0.45.1. Interpret the bound as three total subprocess attempts, not three extra attempts. A retry is a fresh idempotent observation only; mutations and the existing two-observation agreement fence retain their semantics.

ARCH-DRY/PURE: reuse `validateSessionListing`, `zellijreply.ParseClients`, `ParsePanes`, and `ParseTab` for pure reply-kind classification; keep the bounded loop at `ZellijSource.runContext`, with the current process collector as one-attempt IO. Do not introduce a second IPC transport. Only the exact enumerated read argv are eligible. Valid same-kind replies remain subject to all existing caller validation/independent confirmation; malformed/duplicate-key or missing-decision-field payloads remain errors without automatic replay unless they form a complete different supported reply kind.

ARCH-ORDER/CONSTRAINTS: one logical read owns one unchanged caller/five-second budget, at most three sequential children, no concurrent children and no backoff. At 8 and 16 logical requests, subprocess count is at most 24 and 48. Caller cancellation stops before the next spawn; each completed attempt emits its own bounded observer record. No persistent runtime state is added (ARCH-FUNERAL); existing fixture temp roots and child joins own cleanup. PR5630 removal note belongs next to the workaround.

Actions: perform read, cancel, explicitly invoke a later recovery after refusal.

| Internal state / failure mode | Visible state | Recovery action | Possible loss |
| --- | --- | --- | --- |
| Valid expected reply on attempt 1–3 | Existing healthy recovery result | Perform read | No new loss; existing agreement/ownership checks apply |
| Exit-zero empty or complete other-kind reply, attempts remain | Read still pending within original budget | Cancel | No mutation dispatched by this read |
| Three signature failures, deadline/cancel, collector/nonzero error, malformed payload or later agreement mismatch | Existing typed unknown/refusal | Explicit later recovery | No unsafe absence/ownership authority; operation remains refused |

The hub separately authorized #743's image pull/build in receipt `3708133b-f037-4efe-bdb2-4bd40dd5e457`, only AFTER #748 finishes. Preserve pair-test; verify digest; remove only that interrupted pull's identified partial state; delete nothing else in ~/.tart without asking. #743 remains parked meanwhile.

### 2026-10-20 — implementation and qualification

Implemented the authenticated hub's product mitigation: exact read allowlist; three TOTAL sequential attempts; empty or completely decoded other-kind signature only; original caller/five-second deadline; each attempt observed; no sleeps or retry of mutations, nonzero/collector errors, malformed same-kind payloads, or later confirmation disagreement. Existing strict admission and server-generation checks remain. Pair-local golint rule `zellij-read-attempt-bypass` prevents new callers from bypassing the bounded owner. Runtime additions remain within the quick-flow shell (90 added production lines; tests/docs excluded).

Red evidence: the new deterministic empty→crossed→valid and exhaustion regressions failed on the original implementation (one call instead of three); retained local `/tmp/pair748-red.log`. Green targeted launcher/couchcore contracts passed (31.818s/8.859s). Broader package run exposed old one-attempt fake assumptions in attachment, owner and park refusal tests; supplied three corrupt frames and retained all no-effects assertions. Focused changed contracts with race passed (launcher23.976s, couchcore15.040s); latest focused non-race launcher1.253s/couchcore6.930s/repolint1.955s. Ad hoc independent review of the query wrapper/classifier found no actionable finding. Mandatory close review remains pending.

Initial native control (superseded by BR-2 below: this error-only oracle did not prove valid final replies): 2,000 logical real-CLI pane reads through the production boundary on unchanged0.45.1,2,001 attempts,1 empty failed attempt,0 bad final answers;32.76s. This is a different path from the raw-protocol45/2,000 first-reply diagnostic, not a measured45→1 improvement. Exact source, erroneous attempt evidence, result and reproduction instructions are retained under workshop/evidence/000748. Twenty consecutive fresh-fixture native warm cases passed without discarding any failure; their result summary/raw-log digest is retained there. Earlier exact park→warm baseline and the raw-native diagnostic are retained separately. No images, native installs or operator sessions touched.

Audited all six changed source-reader inventory entries: same fixture/native CLI inputs, changed attempt counts only; original dispositions and fixed-preflight ownership unchanged. Refreshed only their digests and the real inventory audit passes. Initial inventory unittest run began before refresh and correctly rejected stale hashes; it is not treated as a pass. Recorded affected check and SDLC close are next.

Final exact CI park→warm selection on the final implementation also passed (retained local park-warm-final.log under the native warm evidence root). Added direct pure reply-kind tests for malformed/duplicate/missing-fact payloads and complete different kinds; they pass. PR500 is open. The first recorded affected request fell back to full suite because the detached package graph lacks generated runtimebundle assets before Make generation; suite owns generation. It was still queued when final pure tests were added, so cancel/requeue on the final pushed head rather than spending a suite on superseded tests.

### 2026-10-20 — close review corrections (BR-1/BR-2/BR-3)

Boundary review found no production correctness defect, but two important verification gaps. BR-1: the frame fake counted only declared matching frames, hiding extra children when forbidden replay was introduced. It now increments before schedule/argv validation. Tests assert original typed errors/output as well as process count; an unsafe retry-gate mutation now deterministically fails nonzero, mutation and unknown-flag cases (retained unsafe-retry-mutant-result.txt). This also exposed one generation-bracket fixture that had silently exhausted frames on its second confirmation read; it now supplies both valid reads and asserts both were consumed. Existing generation refusal is unchanged.

BR-2 supersedes the earlier initial error-only native control's “bad final answers” claim: the corrected2,000-read control validates every returned payload with production ParsePanes, required command/state/geometry/tab fields, and nonempty controlled-session inventory. Result:2,003 attempts,3 empty-attempt failures recovered,0 invalid/error final answers,46.28s. Current retained source/result/events/README use that oracle/count. A different first-attempt rate from the raw control is not an improvement claim. The prior result was insufficient proof and is not current acceptance evidence. BR-3 swept ZellijSource comments to distinguish logical reads, bounded subprocess attempts and attempt-local observer errors. No production mechanism or timeout change.

BR-1 class sweep also found two older client/listing foreign-reply schedules that had passed only because an unexpected child was not counted. Both now provide all three declared corrupt frames and assert the typed crossed-reply error plus exact exhaustion count. All frame-fake consumers were enumerated; valid-listing and malformed-payload one-attempt checks remain unchanged. Reader digests retain existing dispositions/inputs.

### 2026-10-20 — SHIP acceptance, binary provenance and residual qualification

Full close review returned SHIP with no open findings; all BR-1/2/3 disposed. The independent reviewer observed2,016 native attempts/17 attempt errors/1 failed final, then2,007/7/0. This is residual native failure, not universal success. Authenticated hub receipt0fe3af03-7588-42d1-9403-b7eaf6037fd8 accepts bounded mitigation with3attempts/no growth and directs landing after gates; removal remains a supported Zellij release containing upstream PR5630.

Provenance follow-through: installed Homebrew0.45.1 hashes604c684330fe097bc1363ada020d1b9b0ad17caa48d633e838869dfe9fc49e9c, unlike CI's official asset. Downloaded the official0.45.1 asset only into a private temporary directory and verified ca5f9333735bdbc59a621f1d8ed8e24798845302a28ff175d253d4793d5a4a2c against upstream's release checksum; installed binary unchanged. Official exact park→warm and20consecutivewarm passed. Its complete strict2,000-read control yielded2,012 attempts/14 attempt errors/2 failed finals in56.23s; both failures were exit1 “There is no active session!”, correctly fail-closed outside the approved exit-zero retry signature. No failed test/control was retried as passing evidence. Retained separate official results/events/warm summaries; hub informed. Do not infer an exact reduction rate by comparing the different raw/CLI geometries.

Follow-up for #743 image: use the checksum-verified OFFICIAL native0.45.1 asset like CI, not an unverified Homebrew binary bearing the same version string. Its image/pull authorization remains sequenced after #748 lands, preserving pair-test and deleting only the identified interrupted pull state. Recorded full suite still queued; no gates waived.
