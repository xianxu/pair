---
id: 000749
status: working
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: '7ddead28aaabe2d49d99d15d20ef7e5cc3f1fdaf' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T18:48:49-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flake_tests: [github.com/xianxu/pair/cmd/internal/clipcmd.TestCopyOnSelectShellFixture]
flow: {kind: quick, provenance: inferred, spec: "b7b60c55", done: "a0bbe98b"}
---

# flake: TestCopyOnSelectShellFixture

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/clipcmd.TestCopyOnSelectShellFixture: First seen 2026-10-20 about18:27 PDT during pair#746. Same exact head a4f0b6ae0c9dd679c3829679485b43239f2de7a4: full suite attempt7562be4ed11412edecae702be7f460e4 failed only TestCopyOnSelectShellFixture at cli_admission_test.go:108 after3.54s: detached copy orchestrator did not finish. Complete failure log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/7562be4ed11412edecae702be7f460e4.log hash6a8eb50de62d, run12m11s; every other test passed. Unchanged isolated detached checkout of that exact commit, make runtimebundle-generate then go test ./cmd/internal/clipcmd -run ^TestCopyOnSelectShellFixture$ -count=1 -v, passed3.82s/package4.022s exit0. Retained actual targeted output /Users/xianxu/workspace/pair/.git/manual-verification/pair746-clip-a4f0b6ae.log. Fixture tests/copy-on-select-test.sh:88 waits3s on select for detached completion. Load12.67/13.20/16.84 at observation; scheduling/deadline sensitivity is a hypothesis, not measured cause. No production, test or timeout edits. Independent unassigned follow-up; do not fold into746.

## Spec

Fix the demonstrated copy-on-select fixture/process ownership failure without a retry loop or longer timeout. Keep private clipboard and native-command fakes, join the exact detached work before assertions and cleanup, and preserve hook/orchestrator behavior coverage. Diagnose the actual blocked boundary before selecting the fix (ARCH-PURPOSE, ARCH-FUNERAL).

## Done when

- The root cause is fixed (never a retry loop or a longer timeout); closing this issue unregisters the tests.
- The shared helper preserves inode identity on same-volume and forced-EXDEV paths; copy-on-select and shortcut fixtures reject a per-run executable copy.
- Targeted checks and recorded verification pass before landing.

## Plan

- [x] Reproduce with a fresh test binary; capture the hook/orchestrator boundary and establish root cause.
- [x] Pin shared-inode placement, including forced EXDEV, before replacing raw fixture copies and the shared helper copy fallback with aliases.
- [x] Run targeted copy/shortcut/shared-helper tests, queue recorded verification and request close review; recorded pass remains required before landing and resuming #733.

## Log

### 2026-10-20
- 2026-10-20: closed — AMFI logs prove the fresh copied pair-real spent 3.007s in security assessment before CLI startup. Shared inode assertions failed against both original fixture copies and forced EXDEV fallback, then passed after shared aliases. Full clipcmd/testpair packages and actual shell copy/shortcut/helper contracts passed with unchanged deadlines. Audited reader hash/preflight passed. Recorded affected verification queued on PR 499; final recorded pass remains required before landing. Test-fixture-only repair, no UI stage.; review verdict: SHIP

- TL dispatched #749 as the top Pair blocker; preserve #733 SHIP branch and re-enter after #749 lands. Per explicit TL instruction stopped #733 merge runner cleanly and retained its failure/evidence in its Log, then claimed/started this branch from main in canonical pair:4.
- #733 integration suite attempt 0082d1dc13f7dff65c18d760ca43ba06 at 13bd7df51123cc82c79953ca2e8423511c0a5b49 failed tests/copy-on-select-test.sh: detached copy orchestrator did not finish; 4m46s, complete log sha256 8b7c811b0f82. Fresh TestCopyOnSelectShellFixture on that source also reproduced hook timeout=3.
- Pair:6 supplied #706 attempt 11c40c358b4966a8b6d78660cd2dab3d at 63d530dbaffb52fea561cbbdbdeff9581c1fa66b: private pair-real clip copy-on-select timed out after 3s; exit 2 in 4m46s. Complete log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/11c40c358b4966a8b6d78660cd2dab3d.log, sha256 826acd5c3f915860d5b7c0dc547427583f584e7b09749e57af901a2c037f4c89. Archive diff touches no clip/fixture files.
- Diagnostic script built a fresh binary via the shared builder. One instrumented fresh-binary run passed (hook first launch 0.497s; later 0.039/0.043s). An earlier instrumented probe accidentally used the stale checkout bin/pair; discarded its cross-case results and made the candidate explicit. No tracked test or production edits yet.

## Revisions

### 2026-10-20 — measured root cause and alias design

- Reason: TL approved the measured AMFI cause and expanded the sweep to all fixtures that copy freshly built executables. No timeout increase.
- Delta: same-filesystem aliases remain hard links; the central helper uses a symlink across filesystems instead of copying bytes. Both shell binary-copy sites use the helper, and the clipboard broker fixture uses testpair.At rather than a manual copy fallback. Inode-identity assertions defend actual consumers and a forced-EXDEV regression defends the fallback (ARCH-DRY, ARCH-FUNERAL). Product state and persisted formats are unchanged.
- Unified log at 18:32:53.810 records AMFI assessment of GpEDlx/home/bin/pair-real; 18:32:56.817 reports SystemPolicy refusal after 3.007s. syspolicyd then finds the file missing because timeout cleanup removed the fixture. This is executable launch/security assessment, before the fake clipboard pipeline.
- Fresh candidate symlink alias probe passed all three pane cases, preserving sibling executable resolution on macOS. Five targeted runs and ten two-at-a-time instrumented copy runs passed afterward; these do not erase the retained failures. Root cause is backed by OS diagnostics, not a blind pass.
- Sweep: copy-on-select, terminal-shortcuts and clipboard broker fallback copy real candidates. Emitter-health uses marker text; dev-rebuild copies shell; fleet diagnostic retention copies evidence only. These unrelated copies remain.

- Red/green: both real shell fixtures rejected their original raw copies with the new samefile assertions; forced EXDEV rejected the central helper byte-copy fallback. All three passed after alias-only placement. Clipboard broker now uses testpair.At and verifies samefile too. Its first converted run built after replacing Go environment, creating a private read-only module cache; restored pre-environment Binary resolution and the full clipcmd/testpair targeted packages passed.
- Reviewed cli_admission_test.go repository inputs: the sole checkout input remains ../../../tests/copy-on-select-test.sh; new Stat checks touch only fixture binary aliases. The existing non_go_full_fallback disposition remains correct; refresh only its content hash.

- Targeted green: go test ./cmd/internal/clipcmd ./cmd/internal/testpair -count=1 passed; shared shell contract, copy-on-select fixture and terminal-shortcut fixture passed with an explicit shared candidate on another volume. Reader audit initially rejected the changed source as expected; re-audited its unchanged repository input and updated only the non_go_full_fallback hash; fixed inventory/preflight test passed. The remaining seven inventory regression tests passed before that hash update.
- Verification plan: push implementation and queue sdlc local-check affected; overlap close review if queued, as tl-worker permits, with a recorded final-head pass required before landing. Fixture-only repair needs no operator UI stage.

### 2026-10-20 — verification/review overlap

- Reason: tl-worker permits the close review to overlap a queued recorded check; changes from review requeue verification on the changed code.
- Delta: PR #499 opened and affected verification queued on pushed implementation 814eb8d40. Close review may proceed now, but landing requires recorded success on the final code.
