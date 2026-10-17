---
id: 000624
status: working
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
card_mirror: 'ecb3f43df4a6c8b9fcf920b477891760c3e7d958' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-16T20:02:10-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "e9385e2f", done: "e8c56ece"}
---

# Something rebuilds pair:0's bin/couch during other slots' work, deleting it for seconds (sends fail)

## Problem

On 2026-10-16, `~/workspace/pair/bin/couch` (pair:0, the machine's installed Couch CLI that every slot and the TL use)
**disappeared for several seconds, twice**, and came back rebuilt at the same revision (9fc035e7, vcs.modified=false):
around 15:40 and at 16:04–16:06. During the gap every `couch --send-to` failed with "no such file or directory".
Nobody asked for a pair:0 rebuild. At 16:04:17 pair:4 (Qoder) started `sdlc milestone-close --issue 583 --milestone M1`
from its slot; the 16:06 rebuild followed. Hypothesis: a review prepare step (pair#549-era "prepare runtime assets
before boundary reviews") or a dev alias builds into pair:0's `bin/` instead of the worktree's own. The same mechanism
would clobber a staged branch build on pair:0.

## Spec

- Find what writes pair:0's `bin/` outside an explicit `make build` / `sdlc stage` in pair:0 (fs_usage or a guard).
- A slot's build, test or review preparation writes only inside its own worktree. pair:0's `bin/` changes only via an
  explicit build or stage there.
- Builds that replace a shared binary replace it atomically (write to a temp file, then rename), never delete first.

## Done when

- A test (or guard) shows a milestone-close / review prepare in a slot worktree leaves pair:0's `bin/` untouched.
- `make build` in pair:0 replaces binaries atomically (no window where `bin/couch` is missing).

## Plan

- [ ] Red first in private fixture: paused compiler exposes delete/partial publication; cover direct binary targets and full make build, failure/cancellation cleanup, concurrent publishers and foreign PAIR_HOME isolation.
- [ ] Use one same-directory temporary-build/rename publisher for pair, couch and pair-launch-helper; opt owned binaries out of the generic direct-output scanner (ARCH-DRY). Keep old executable on failed builds and clean owned temporaries.
- [ ] Verify review asset preparation and worker build stay in their own fixture tree under foreign live-environment values; document observed writers versus unproven historical caller, run recorded full suite, close and land.

## Log

### 2026-10-16 — writer investigation and reproduction

Read-only scan found two direct Couch writers: Makefile.local's explicit bin/couch target and the shared Makefile.workflow build scanner. The latter is an ariadne symlink; Pair can opt its owned binaries out through the existing .skip-make-build mechanism rather than editing the peer. Direct Pair and launch-helper targets share the same unsafe output publication. Current installed couch is a symlink to primary bin/couch, not a rebuilding wrapper. The current interactive alias generator already builds privately and renames, and skips existing binaries in COUCH_TREE. runtimebundle-generate invokes only its generator (whose subprocess is tic), not a Pair build.

The installed Go source (cmd/go/internal/work/shell.go moveOrCopyFile/CopyFile) falls back to removing/truncating its final destination before copying when rename is unavailable or a cached artifact is copied. First private probe was inconclusive because the output was already current. Alternating two valid builds forced replacement: exit 0, 2753 observations, one missing destination and 418 partial-size observations, with Go -x explicitly showing cp to the final path. Evidence retained in /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair624-probe.r7aBha/result.json and cached-replacement.trace. No build or clipboard/GUI action ran in pair:0.

The exact historical initiator at 15:40/16:06 remains unproven. PAIR_DEV plus primary PAIR_HOME can request a primary rebuild through dev_rebuild, but current hosted markers suppress it; do not label review preparation as the culprit without evidence. The proposed fixture guard covers current review preparation/build routing despite foreign environment values. Asking the hub whether this bounded diagnosis satisfies acceptance, rather than silently dropping the historical-caller clause.

Start-plan refreshed dependencies and required two reruns before creating this branch; brew reported existing go/cue/uv, no new machine packages chosen.

## Revisions

### 2026-10-16 — bounded writer diagnosis approved

TL approval fdfe17db-5144-4cc2-8ad9-902345e90249 supersedes the original requirement to identify a non-explicit historical writer: acceptance is the proven direct-output writer class, one temp-sibling-plus-rename publisher, skip sentinels eliminating duplicate generic writers, deterministic atomicity tests and the foreign-PAIR_HOME isolation guard. The exact historical caller remains unknown. The TL reports running make build in pair:0 around that period while rebuilding for #618; this is a likely candidate, not an observed process-to-failure correlation. The TL owns filing the ariadne generic-build half; this issue edits Pair-owned files only.
