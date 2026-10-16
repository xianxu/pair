---
id: 000591
status: working
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-16
estimate_hours:
card_mirror: 'f68a8a74f1cac58af501b6cb5e612e451489b250' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-16T00:59:12-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
---

# Diagnose legacy review resume reconstruction failure

## Problem

Recorded full suite on unchanged source74ea2d2aacf1232af3d0ae38f86353e596ae6208 failed in tests/review-resume-test.sh. The initial legacy empty-agent-round case returned placed=false hl=0 diag=0, producing three failed assertions. Later explicit resolved-identity and ambiguous-history cases passed. No identity status, diagnostic, command, stdout/stderr or timing was preserved for the first case, so the cause cannot be established from the failing artifact.

## Spec

Diagnose the implicit identity-resolution/reconstruction boundary without weakening assertions or increasing timeouts. The first case invokes reconstruct_on_open without identity; nvim/review/init.lua calls identity.resolve and, for missing history, retries resolution with the selected file/head. identity.lua chooses PAIR_HOME/bin/pair when inherited, otherwise the checkout binary, and waits2500ms. Later cases call the shell PAIR_BIN explicitly then supply the resolved identity. This is an observed asymmetry, not proof of an environment or timing cause. #586 concerns another review-observation fixture and may share this seam; keep separate until evidence links them.

Capture actionable evidence on the first failure: exact binary/root, resolver input/output/status/diagnostic, both fallback observations, reconstruction admission reason and Neovim messages/exit. Add a deterministic regression for the demonstrated cause and fix its owning boundary. No sleep, timeout increase, retry loop, skip or assertion weakening.

## Done when

- First-case failures retain enough evidence to distinguish wrong binary, incomplete/invalid observation and genuine unsupported history.
- A deterministic red/green regression verifies the demonstrated cause.
- Legacy empty-round reconstruction and scoped/ambiguous authorization assertions remain exact; isolated and loaded suite results are recorded.

## Plan

- [ ] Add first-case diagnostics before changing its admission: retain the chosen executable/root, ordered real resolver completions (including legacy fallback), exit/signal or spawn error, bounded output metadata/stderr, returned identity/status/diagnostic, reconstruction outcome and Neovim messages/exit. Reuse the existing fixture observation trace rather than a parallel observer (ARCH-DRY); flush evidence at shutdown or fixture failure, not during resolution.
- [ ] Prove diagnostics with deterministic foreign-resolver, malformed/nonzero and silent early-editor-exit cases, plus a positive control retaining both actual missing→selected/resolved observations. Keep exact legacy decoration and scoped/ambiguous authorization assertions; do not add another resolver or change production scheduling (ARCH-PURE, ARCH-STATE).
- [ ] Fix the demonstrated owning boundary: isolate the shell fixture through ReviewTestEnvironment before any Git/editor subprocess, pin checkout PAIR_HOME and executable selection, and preserve caller-owned artifact sentinels. Turn the controlled inherited-PAIR_HOME probe into a red/green regression. Production identity/reconstruction behavior and its 2500ms wait remain unchanged (ARCH-PURPOSE, ARCH-SECURE).
- [ ] Record an isolated run and the required recorded full suite, update Atlas for fixture diagnostics, then request the mandatory close review and land on SHIP after the hub's turn.

## Revisions

### 2026-10-16 — controlled fixture-contamination cause, historical cause unknown

Read-only controlled runs demonstrate that inherited PAIR_HOME alone routes the first case to another executable and reproduces the exact three-failure signature while later explicit-binary cases pass. The clean control passes. This is sufficient to correct the fixture's owning environment boundary; it does not identify the original uninstrumented run's cause. Keep diagnostics so scheduling, malformed responses and unsupported history remain distinguishable rather than claiming they were ruled out.

The work remains fixture-only and quick flow: real Git/Neovim/CLI positive controls plus controlled adverse subprocesses (ARCH-MOCK), existing phase deadlines and bounded diagnostic projections with no document bodies (ARCH-CONSTRAINTS), ephemeral owned fixture artifacts removed after their bounded report is emitted (ARCH-FUNERAL). No production durable state or external event protocol changes. The first-case missing→selected resolution and every return/refusal are observed in order without introducing callbacks or another observer request.

## Log

2026-10-15: observed while verifying #579; diagnostics IO source does not change nvim/review/init.lua, identity.lua or this fixture. Required run9c8da6fdf45067e5d6386cddc8344aea waited804s for quota then failed after3m26s before broad Go. Full log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/9c8da6fdf45067e5d6386cddc8344aea.log, SHA256 prefix1881fe7440d7. Exact first-case failures: no reconstruct (placed=false hl=0 diag=0), no change-highlights on resume, no diagnostics on resume. No isolated retry or source change made. Hub9a25379a-3879-4824-a314-9c396a4df10d requests filing unassigned and authorizes one unchanged recorded-suite rerun for #579; independent fixture fix waits for a free slot.


### 2026-10-16 — claim and read-only controlled binary-selection measurement

- Started from main `6d9da348fa94f86383c6d263abc4607f9dbf9706` after #590 landed (git rev-parse). Claimed #591, ran start-plan and rebuilt existing source with `make -k -B build`. Binary vcs.revision matches; vcs.modified=true reflects only the start-plan issue card mirror, with no production/test changes.
- Probe uses ReviewTestEnvironment to protect all caller artifacts, then invokes the unchanged complete resume fixture twice. Both use checkout PAIR_BIN; only PAIR_HOME changes between checkout and a controlled foreign executable. Checkout home: exit 0, 7.132s, all assertions pass. Foreign home: exit 1, 5.887s, exactly the initial three failures (`placed=false hl=0 diag=0`); every later explicit-identity/scoped/ambiguous/read-only assertion passes. Foreign argv records exactly the implicit readiness resolution, proving which binary produced the invalid observation. No timeout increase, sleep, retry or source edit was involved.
- Evidence `/var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair591-probe.Vz9GPH/`: checkout-home.json, foreign-home.json, foreign-calls.jsonl and results.json; probe `/tmp/pair591-measure.py`. The historical broad failure remains unclassified because it retained none of these observations. Two subsequent #590 broad runs passed this fixture, so those do not establish a historical cause either.
- Proposed diagnostics-first plan fixes a demonstrated inherited-binary isolation defect after preserving evidence. Await the hub's plan decision before change-code; no product change proposed.
