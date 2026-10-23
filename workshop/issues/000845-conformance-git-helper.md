---
id: 000845
status: open
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
collaboration-mode: collaboration
card_mirror: '52ba9ee0c05ea78237974a7d4817360bb4f77e06' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ]

## Log

### 2026-10-23

Discovered while implementing ariadne#577's native service conformance test. The bridge build succeeded. The native attempt failed before readiness and returned exit status 1; this was recorded as unavailable conformance, never a pass. Diagnostics reported native fleet cleaned, all owned generations absent, direct child and IO joined. Independently checked /Volumes/TestScratch/pc-3987325977 and /Volumes/TestScratch/couch-candidate-59051363 absent after return. No Pair production source changes or ambient-tool workaround were made. Filed unclaimed for a separate Pair slot.
