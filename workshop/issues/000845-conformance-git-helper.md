---
id: 000845
status: working
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
collaboration-mode: collaboration
card_mirror: '133bf0c3f17519c01515307c7b92c4d6387232f5' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-23T14:16:27-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "95191eac", done: "f7ff37f9"}
---

# Native Couch conformance isolation omits Git upload-pack helper

## Problem

Pair's native Couch conformance bridge fails before readiness on macOS because isolated workspace provisioning cannot execute git-upload-pack for its local private upstream. This blocks ariadne#577 native Couch service conformance evidence; no TL/worker replay or successful service qualification was claimed.

## Spec

The isolated native conformance owner must provision its own local workspace using the selected Git installation without adding ambient PATH directories or inheriting live Couch authority. Include the Git subprocess/helper capability required by the actual local fetch path in the owned tool inventory and private runtime environment. Keep replacement-environment isolation and native cleanup ownership intact.

Observed source seams: cmd/internal/testcouch/owner_build.go selects git in toolNames and assigns Binaries.Git; cmd/internal/testcouch/runtime_files.go installs the selected git in the private runtime bin, but the isolated environment does not make its git-upload-pack helper available. The demonstrated failure is Git's shell helper lookup during local upstream fetch, rather than network access or an unavailable source remote.

Reproduction (2026-10-23, macOS, Apple Git 2.54.0):

```sh
make -C /Users/xianxu/workspace/pair couch-conformance-bridge
cd /Users/xianxu/workspace/worktree/ariadne-slot4/ariadne
BENCH_COUCH_CONFORMANCE_BRIDGE=/Users/xianxu/workspace/pair/bin/couch-conformance-bridge go test ./cmd/sdlc -run '^TestBench(BridgeReplacementEnvironment|CouchNativeServiceConformance)$' -count=1 -timeout 7m -v
```

The Ariadne test uses the native bridge's fresh owned fixture, a physical short /Volumes/TestScratch parent, and the bridge's complete replacement client environment. It sends no live fleet messages and makes no live fleet mutations. Pair's bridge selects its checkout from its physical executable and builds its native candidate normally.

Exact failure at native bridge stage 1:

```text
provision slot 1: couch [--internal provision-workspace /Volumes/TestScratch/pc-3987325977/fleet/fixture --slot=1 --remote=upstream]: exit status 1:
git-upload-pack '/Volumes/TestScratch/pc-3987325977/fleet/upstream.git': git-upload-pack: command not found
fatal: Could not read from remote repository.
fetch remote main: git fetch --verbose --porcelain --no-tags --no-recurse-submodules --no-write-fetch-head --refmap= upstream +refs/heads/main:refs/remotes/upstream/main: exit status 128
```

## Done when

- A regression reproduces the selected Git installation's local upstream fetch through the complete isolated native environment and proves the necessary helper executes without ambient PATH access.
- The native couch-conformance-bridge reaches its bounded readiness line and permits its supported read-only service requests under the advertised replacement environment.
- EOF shutdown joins native children, exits zero, and removes owned fleet/candidate storage; failure cases retain honest cleanup diagnostics.
- Ariadne can rerun the command above to qualify native service/lifetime behavior; actor/model/root replay remains separately operator provisioned.

## Plan

- [x] Pin the local-fetch failure through the complete replacement runtime environment, using an owned bare upstream with a real main commit and Apple's physical selected Git on Darwin. Assert fetched main equals the upstream commit and PATH stays private; poison ambient Git helper settings to prove they do not leak.
- [x] In the existing candidate-build IO seam, query the selected absolute Git's --exec-path using the replacement build environment. Resolve and validate only its git-upload-pack executable; carry it in Binaries and install one symlink via the existing all-sources-before-effects runtime inventory. Missing/invalid helper fails with an actionable selection error and the existing partial cleanup owner. Do not inherit GIT_EXEC_PATH or append ambient directories.
- [x] Extend injected builder/runtime tests for same-installation selection, missing/unsafe helper, pre-effect validation and symlink basename dispatch. Reuse current installation and cleanup machinery (ARCH-DRY/PURE); no new lifetime, persisted format or hot-path loop.
- [x] Qualify focused testcouch/bridge behavior, inventory, native readiness/read-only requests and EOF cleanup, retaining exact results and diagnostics. Recorded suite, close, landing and downstream rerun remain explicit external gates in the Log.


## Log

### 2026-10-23

Discovered while implementing ariadne#577's native service conformance test. The bridge build succeeded. The native attempt failed before readiness and returned exit status 1; this was recorded as unavailable conformance, never a pass. Diagnostics reported native fleet cleaned, all owned generations absent, direct child and IO joined. Independently checked /Volumes/TestScratch/pc-3987325977 and /Volumes/TestScratch/couch-candidate-59051363 absent after return. No Pair production source changes or ambient-tool workaround were made. Filed unclaimed for a separate Pair slot.

### 2026-10-23 — claimed; root cause and proposed quick-flow plan

Verified #819 merge runner EXIT0: PR550 landed in train318 (train PR564), cleanup returned this checkout to unchanged main-slot4. Claimed845 and start-plan created its independent branch from current main. No implementation changes.

A bounded disposable probe reproduced the exact failure with /Library/Developer/CommandLineTools/usr/bin/git invoked through the private bin symlink: local upstream fetch exited128 with git-upload-pack command-not-found. The same fetch with only a private git-upload-pack symlink to that installation's physical helper exited0 and fetched main. All scratch repos were private and removed on probe exit. The /usr/bin/git launcher alone did not reproduce this lookup boundary, so the regression must cover Apple's physical executable, not assume its launcher models helper lookup. Existing resolveTool resolves symlinks; helper git-upload-pack itself resolves to Git's multicall executable, so preserving the installed basename is essential.

Recommended narrow capability: selected Git --exec-path -> validated physical git-upload-pack -> Binaries -> existing private runtime inventory. Alternative: append selected Git's whole exec directory to PATH or export it as GIT_EXEC_PATH; rejected because this expands runtime capability beyond the observed local-fetch need. Ambient helper lookup is rejected because it can select a different installation. Existing command runner bounds and CandidateOwner cleanup cover failed selection; runtime inventory validates every source before installation effects. Additional state/action table is unnecessary: this adds one immutable selected tool, not mutable user state. Existing startup/shutdown owners and protocols stay unchanged (ARCH-ORDER/FUNERAL/EVOLVE); trust remains selected executable provenance (ARCH-SECURE). Work is startup-only and adds one bounded metadata query plus one symlink, with no new scan/growth dimension (ARCH-CONSTRAINTS). Native fetch plus service lifetime proof covers the issue's actual purpose and external dependency (ARCH-PURPOSE/MOCK).

Plan committed and sent to TL as requested; hold change-code until the hub responds. No live fleet calls, environment widening, timeout change or code edits.

## Revisions

### 2026-10-23 — TL approval (40e2eb78)

Approved the selected Git --exec-path helper, validation, Binaries ownership and single named private symlink. No helper directory or ambient PATH. Proceed; notify ariadne:4 when landed.

### 2026-10-23 — implementation; red/green and native lifetime proof

change-code inferred quick flow and accepted the recorded design. Native regression first failed on private clone's missing upload-pack; isolated setup to the selected absolute Git so it then failed at the precise private-PATH fetch (exit128, package0.432s). Implemented selected --exec-path query under the replacement build environment, strict clean absolute output, physical executable resolution, Binaries.GitUploadPack and one named runtime symlink. The injected builder test caught an initially unwired selection result; wired the real builder before acceptance. Focused selection/invalid-helper/pre-effect inventory/native-fetch tests PASS0.585s. Helper aliases resolve to Git's multicall executable; the private git-upload-pack basename is retained for dispatch.

Native bridge opt-in TestNativeCouchBridge PASS34.147s: Claude default candidate22.40s and Codex explicit candidate11.65s. Both advertise bounded readiness/documents/socket and serve read-only calls; EOF cleanup reports all owned generations absent, direct child and IO joined, root_absent=true. Candidate roots /Volumes/TestScratch/couch-candidate-2907532016 and /Volumes/TestScratch/couch-candidate-336179200 were removed by their owners. This is deterministic fixture conformance, not live actor/model replay or ariadne#577 qualification. The standalone bridge binary also builds successfully.

Portable run found the legacy fake Git did not model --exec-path (returned selected-head). Extended that same fake with an installation-local helper; no product workaround. Corrected full portable package set PASS: testcouch38.497s, agentfixture12.497s, fixture command0.359s, bridge0.288s. Artifactpath10.050s and repolint11.237s PASS. Atlas documents the exact selected helper and no environment widening. No timeouts, worker state, shutdown policy or live fleet changed.

Audited only the named changed readers before regenerating: owner_build_test.go keeps non_go_full_fallback because its fake builder exercises source/build inputs; legacy_build_test.go keeps non_go_full_fallback because public Build traverses the real repository builder while observed/calls/supplied reads remain private; runtime_files_test.go keeps local_or_literal_fixture because all reads are its own private executable/config fixtures. New git_helper_test.go uses private repos and the existing registered environment helper, and inventory requests no new certificate. No bulk refresh. Dependency capability enforcement belongs to the native selected-Git fetch regression plus builder/inventory seams; it is not a generic syntactic misuse suited to a new Go lint rule.

Next: checkpoint/push exact candidate, queue one recorded suite, then required inventory immediately before SDLC close (review may overlap the queued suite under worker rules). Landing requires that suite pass and SHIP. Notify ariadne:4 on landing per TL40e2eb78.

Recorded suite queued on candidate5eed4896f: retained session17766, lease-3557552526, attemptc51313d2cb7b23109d72c5b36587d9e8, immutable snapshot /Volumes/TestScratch/local-checks/sdlc-local-1123206548. Initial FIFO4. PR568 is open. No pass claimed. Required inventory passes before close; review may overlap this queued snapshot, but landing still needs its pass. Preserve this one run; no duplicate, no new stage.

### 2026-10-23 — reader admission corrected before review

The final inventory after tracking the new test correctly requested a new certificate for git_helper_test.go; the earlier untracked-file scan did not see it. My first close incorrectly described inventory as passed before inspecting its final result. Close refused at its plan-check guard, before review dispatch, because the implementation checkbox also bundled its own future close/landing; no review verdict or success was recorded. Canceled the queued, not-started suite using supported local-check cancel --run lease-3557552526; command exited with context-canceled admission and no test evidence. No PID kill or blind passing retry.

Audited the new reader: it selects actual Git metadata and calls runtimeEnvironment through the selected repository tests/lib/pair-test-env.sh; private upstream/target repos are its own, but that repository helper requires non_go_full_fallback. Added exactly that certificate and reran inventory: PASS. This supersedes the earlier 'no new certificate' assumption. Reworded the final plan checkbox to completed implementation/native qualification; external suite/close/landing gates remain here in the Log rather than a self-referential checkbox. No product scope or Done-when change. Next: push corrected certificate head, record one new suite, and close only with this actual inventory result.

Corrected certificate head12631235e301 is pushed on PR568. Its sole recorded suite is retained session82030 / lease-578785275 / attempt3f6c94550fbc62f2930785427c6fc71e, snapshot /Volumes/TestScratch/local-checks/sdlc-local-2120957593, initial FIFO4. Previous canceled run is not evidence. Inventory completed EXIT0. Next: close review may overlap this exact isolated snapshot; preserve it, and land only after verified pass and SHIP.

### 2026-10-23 — BR-1: reviewer native watchdog and residue

Close review21860 EXIT1/REWORK round1, only BR-1 Important native-qualification-evidence. No code defect identified; reviewer portable testcouch38.997s, bridge0.398s and bridge command0.192s passed. Reviewer independently launched the opt-in native test with -timeout4m, shorter than the existing300s StartupBudget and630s test context. It was killed in Claude WaitAgent, before readiness/cleanup evidence; Codex not reached. This does not establish a product timeout defect or invalidate the earlier complete34.147s native run, but the interrupted owner now needs reconciliation before another fleet. No timeout or source change proposed.

Readonly inspection found reviewer-age residue /Volumes/TestScratch/pc-606023944 with marker ef01be422ae9a6ab6d38abd4c34ce4d2 and ownership.json recording Couch75819 generation1792790855.429547, now absent by ps. Associated candidate /Volumes/TestScratch/couch-candidate-1262126714 and auxiliary pc-268856685 (copied explicit candidate) and empty pc-515603984 remain. No process command referenced those paths; ps environment scan found no exact run marker. These observations are not full native ownership reconciliation, so no signal, cleanup or new fleet was attempted. Sent hub ae385483 asking coordinated proof/removal. Suite session82030/lease-578785275 is independent and preserved.

Exact next step: after confirmed residue reconciliation, run pinned-source native qualification with the original12m command watchdog covering630s plus cleanup; retain full output, readiness/read-only/EOF-or-TERM join and owned-root/candidate absence. Then satisfy BR-1 via the normal re-close with required inventory and preserve the exact-source suite. Tell ariadne:4 when landed, not before actual landing. No new implementation work is needed absent actual failing evidence.
