---
id: 000591
status: done
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-16
estimate_hours:
card_mirror: 'd9a88f13a998e3fc56ab48dd44272bab8a7a29f7' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-16T00:59:12-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "d8334f87", done: "e8023d2b"}
actual_hours: 0.65
---

# Diagnose legacy review resume reconstruction failure

## Problem

Recorded full suite on unchanged source74ea2d2aacf1232af3d0ae38f86353e596ae6208 failed in tests/review-resume-test.sh. The initial legacy empty-agent-round case returned placed=false hl=0 diag=0, producing three failed assertions. Later explicit resolved-identity and ambiguous-history cases passed. No identity status, diagnostic, command, stdout/stderr or timing was preserved for the first case, so the cause cannot be established from the failing artifact.

## Spec

Fix the demonstrated fixture isolation defect: the first reconstruction case implicitly selects PAIR_HOME/bin/pair, while later cases use PAIR_BIN. A controlled inherited foreign PAIR_HOME reproduces the exact first-case-only failure signature. Run the complete fixture through the existing ReviewTestEnvironment before any Git/editor subprocess, pinning the checkout home and executable and protecting caller-owned artifacts.

Preserve every reconstruction/decoration, scoped/ambiguous authorization and read-only assertion, and all existing deadlines. Add a deterministic inherited-foreign-home red/green regression. No production resolver change, extended diagnostics trace, sleep, retry, skip or larger budget. The historical run lacked diagnostics; the controlled defect is proven, but attribution of that old run remains unknown. The TL explicitly defers additional diagnostics unless the failure recurs after isolation (#586 already supplies seam diagnostics).

## Done when

- The complete fixture passes with inherited foreign PAIR_HOME and PAIR_BIN; neither foreign executable is invoked and caller artifact sentinels remain unchanged.
- The inherited-foreign-home regression fails with the old fixture and passes with isolation.
- Every existing assertion and deadline is preserved; isolated and recorded broad-suite results are logged.

## Plan

- [x] Add the real-fixture regression with a foreign executable and caller artifact sentinels, and observe red before fixing the fixture.
- [x] Reuse ReviewTestEnvironment's shell re-exec to pin checkout PAIR_HOME/executable and isolate artifacts before any subprocess (ARCH-DRY, ARCH-SECURE). Preserve the complete fixture body and deadlines.
- [x] Wire the regression into the suite, update Atlas and record targeted plus required recorded broad-suite verification.

Remaining workflow checkpoints: mandatory close review, then land on SHIP in the hub's order.

## Revisions

### 2026-10-16 — controlled fixture-contamination cause, historical cause unknown

Read-only controlled runs demonstrate that inherited PAIR_HOME alone routes the first case to another executable and reproduces the exact three-failure signature while later explicit-binary cases pass. The clean control passes. This is sufficient to correct the fixture's owning environment boundary; it does not identify the original uninstrumented run's cause. Keep diagnostics so scheduling, malformed responses and unsupported history remain distinguishable rather than claiming they were ruled out.

The work remains fixture-only and quick flow: real Git/Neovim/CLI positive controls plus controlled adverse subprocesses (ARCH-MOCK), existing phase deadlines and bounded diagnostic projections with no document bodies (ARCH-CONSTRAINTS), ephemeral owned fixture artifacts removed after their bounded report is emitted (ARCH-FUNERAL). No production durable state or external event protocol changes. The first-case missing→selected resolution and every return/refusal are observed in order without introducing callbacks or another observer request.

### 2026-10-16 — TL-approved isolation-only revision

The TL approved with a narrower scope (receipt 86755f4e-dd9a-4fd5-b861-d6abacdd23e2): the measured inherited-binary defect is sufficient; omit the proposed expanded diagnostics and their new adverse-response machinery. Reconcile the active Spec/Done-when/Plan to isolation and its regression only. #586 diagnostics remain available, with further instrumentation deferred until a recurrence after isolation. Production behavior, deadlines and all existing assertions remain unchanged. The existing helper owns environment and ephemeral artifacts (ARCH-DRY/ARCH-FUNERAL); real Git/Neovim/CLI exercise the same boundary (ARCH-MOCK/PURPOSE). No new state protocol or scheduling mechanism is introduced (ARCH-STATE/PURE); two bounded fixture runs exercise the two inherited executable selectors without performance assertions (ARCH-CONSTRAINTS).

## Log

2026-10-15: observed while verifying #579; diagnostics IO source does not change nvim/review/init.lua, identity.lua or this fixture. Required run9c8da6fdf45067e5d6386cddc8344aea waited804s for quota then failed after3m26s before broad Go. Full log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/9c8da6fdf45067e5d6386cddc8344aea.log, SHA256 prefix1881fe7440d7. Exact first-case failures: no reconstruct (placed=false hl=0 diag=0), no change-highlights on resume, no diagnostics on resume. No isolated retry or source change made. Hub9a25379a-3879-4824-a314-9c396a4df10d requests filing unassigned and authorizes one unchanged recorded-suite rerun for #579; independent fixture fix waits for a free slot.


### 2026-10-16 — claim and read-only controlled binary-selection measurement
- 2026-10-16: closed — TL-approved isolation-only scope: four entry lines reuse ReviewTestEnvironment before Git/editor subprocesses, pin checkout PAIR_HOME/PAIR_BIN and protect caller artifacts. Both inherited-selector subcases failed before the fix and passed afterward (12.080s); standalone original assertions pass. Existing fixture body/assertions and 30s/10s plus production 2500ms deadlines are unchanged; no production resolver or extended trace change. Clean make -k -B build reports vcs.revision 980d581cfebb5b3fcc9c1a3de60694caaf240144, modified=false. Required recorded suite ab98062bbfcf7d6b44a710d2c0a9a80e passed all recipes/packages in15m9 on that source, including the new regression in8.565s; log SHA256 a12c6621bf5cf5cd861fad86b9de939ae950a0c2977aa3bb29c425ce35732929. Later changes are issue evidence only. Atlas maps the coverage. TL owns ops/test-economy, so project update is reported to hub rather than written here.; review verdict: SHIP

- Started from main `6d9da348fa94f86383c6d263abc4607f9dbf9706` after #590 landed (git rev-parse). Claimed #591, ran start-plan and rebuilt existing source with `make -k -B build`. Binary vcs.revision matches; vcs.modified=true reflects only the start-plan issue card mirror, with no production/test changes.
- Probe uses ReviewTestEnvironment to protect all caller artifacts, then invokes the unchanged complete resume fixture twice. Both use checkout PAIR_BIN; only PAIR_HOME changes between checkout and a controlled foreign executable. Checkout home: exit 0, 7.132s, all assertions pass. Foreign home: exit 1, 5.887s, exactly the initial three failures (`placed=false hl=0 diag=0`); every later explicit-identity/scoped/ambiguous/read-only assertion passes. Foreign argv records exactly the implicit readiness resolution, proving which binary produced the invalid observation. No timeout increase, sleep, retry or source edit was involved.
- Evidence `/var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair591-probe.Vz9GPH/`: checkout-home.json, foreign-home.json, foreign-calls.jsonl and results.json; probe `/tmp/pair591-measure.py`. The historical broad failure remains unclassified because it retained none of these observations. Two subsequent #590 broad runs passed this fixture, so those do not establish a historical cause either.
- Proposed diagnostics-first plan fixes a demonstrated inherited-binary isolation defect after preserving evidence. Await the hub's plan decision before change-code; no product change proposed.

### 2026-10-16 — approved trimmed scope

TL approved the isolation fix and inherited-foreign-home red/green only. Extended tracing is deferred; no production change or assertion/deadline change is authorized. Updated the active contract and retained the original proposal above as historical revision context. Proceeding to change-code.

### 2026-10-16 — isolation red/green

- New real-fixture regression controls PAIR_HOME and PAIR_BIN independently, supplies caller artifact sentinels, and requires all existing success messages plus zero foreign-executable invocations. Red before the fix: both subcases failed (9.942s). PAIR_HOME produced the exact three initial failures; PAIR_BIN left the first case passing but failed the later scoped/ambiguous cases, proving both selectors need isolation.
- The fix is exactly four entry lines reusing the existing ReviewTestEnvironment shell re-exec before any Git/editor subprocess. The original fixture body, assertions and 30s/10s watchdogs are byte-for-byte unchanged; the production 2500ms resolver deadline is untouched. No extended trace was added. The shared helper pins PAIR_HOME, removes inherited PAIR_BIN and moves writable artifacts to fixture storage.
- Green: python3 -B tests/review-resume-isolation-test.py passed both independent selector cases in 12.080s; neither foreign executable ran, and every caller sentinel remained unchanged. Standalone bash tests/review-resume-test.sh passed all six assertion groups. bash -n and git diff --check passed. The regression is wired into test-review; Atlas maps the fixture isolation coverage. Required recorded broad-suite verification and close review remain pending.

### 2026-10-16 — clean build and recorded full suite

- Source verified by git rev-parse: `980d581cfebb5b3fcc9c1a3de60694caaf240144`. `make -k -B build` passed; bin/pair reports that exact vcs.revision and vcs.modified=false.
- Required `sdlc local-check run --name suite`, run `ab98062bbfcf7d6b44a710d2c0a9a80e`: PASS, exit 0 in 908.941525584s (15m9), separate queue wait 242.824364917s. Every shell/editor recipe and Go package completed; couchcore 562.718s and couchcmd 101.019s. No retry or budget change was needed.
- The unchanged original resume assertions passed in the broad context, and the inherited PAIR_HOME/PAIR_BIN regression passed both cases in 8.565s. Observation/fresh restoration and startup-cancellation checks also passed.
- Retained log `/Users/xianxu/workspace/pair/.git/sdlc/local-checks/980d581cfebb5b3fcc9c1a3de60694caaf240144/suite.log`; SHA256 `a12c6621bf5cf5cd861fad86b9de939ae950a0c2977aa3bb29c425ce35732929`. Implementation and verification are ready for the mandatory close review. The TL owns ops/test-economy; this worker will report its tick rather than edit the project. #607 was still codecomplete/not-landed at the last ordering check; coordinate final landing in the hub's order.
