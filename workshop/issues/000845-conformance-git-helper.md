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

- [ ] Pin the local-fetch failure through the complete replacement runtime environment, using an owned bare upstream with a real main commit and Apple's physical selected Git on Darwin. Assert fetched main equals the upstream commit and PATH stays private; poison ambient Git helper settings to prove they do not leak.
- [ ] In the existing candidate-build IO seam, query the selected absolute Git's --exec-path using the replacement build environment. Resolve and validate only its git-upload-pack executable; carry it in Binaries and install one symlink via the existing all-sources-before-effects runtime inventory. Missing/invalid helper fails with an actionable selection error and the existing partial cleanup owner. Do not inherit GIT_EXEC_PATH or append ambient directories.
- [ ] Extend injected builder/runtime tests for same-installation selection, missing/unsafe helper, pre-effect validation and symlink basename dispatch. Reuse current installation and cleanup machinery (ARCH-DRY/PURE); no new lifetime, persisted format or hot-path loop.
- [ ] Run focused testcouch/bridge tests, inventory, native bridge readiness/read-only requests/EOF cleanup, then a recorded suite and close. Ask ariadne#577's owner to rerun its qualification against the candidate bridge; retain exact results and cleanup diagnostics. Land through SDLC after review.


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
