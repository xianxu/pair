---
id: 000572
status: working
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: 'aabfeeed20ee12fe03a322a6a126d37bf118d2fc' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-15T10:52:12-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "d5f0ba4c", done: "7f5d6529"}
---

# Build and test with -trimpath so worktrees share the Go build cache

## Problem

On 2026-10-15 the shared Go build cache (`GOCACHE=/Volumes/AgentScratch/go-build`) filled the 186 GB scratch volume
(184 GB), failing every slot's suite. Main multiplier: builds embed each worktree's absolute source path, so the same
package is cached separately for each of ~20 slot worktrees; after a `go clean -cache`, ~30 GB came back within
minutes. The operator approved building with `-trimpath`.

## Spec

- Run this repo's full suite with `GOFLAGS=-trimpath`; fix every test that depends on absolute source paths (for
  example locating fixtures via `runtime.Caller(0)` or embedding `__FILE__`-like paths), using module-relative or
  `go list`-derived roots instead.
- Then make `-trimpath` the default for this repo's builds and its recorded suite (Makefile and `.sdlc` suite
  command, as allowed by the local-gates guard), not via a global `go env -w`.
- Note the cache effect: build the same package in two worktrees and confirm one cache entry (or measure the cache
  growth for two worktrees before and after).

## Done when

- The recorded full suite passes with `-trimpath` on by default.
- A measurement shows cache entries shared across two worktrees for an unchanged package.

## Plan

- [ ] Execute [trimpath plan](../plans/000572-trimpath-cache-plan.md): baseline, path-independent discovery, defaults, cache proof, full qualification and operator gate approval.

## Log

### 2026-10-15

- 2026-10-15: claimed in pair:1. Read-only audit identified ten test files,
  shared recording fixtures, and ten probes using runtime.Caller for assets.
  Main has no .sdlc/local-checks.json; SDLC supplies implicit make test.
  The plan makes that declaration explicit and preserves one Make-owned flag.
- Baseline started with -trimpath added to the host's existing -p=3. A first
  queued command that replaced GOFLAGS was cancelled before admission and
  restarted preserving the concurrency cap. Shared cache was not cleared.
- Local-gates label must be applied by the operator after reviewing the PR.

- Baseline `GOFLAGS='-p=3 -trimpath' make -k test` completed (exit 2): 12
  Go packages failed through Caller-derived test/fixture paths; shell/Lua
  prerequisites and couchcore passed. Full output: /tmp/pair572-trimpath-baseline.log.
- Added module-validated cwd discovery (foreign nested modules rejected),
  converted all Caller-based asset consumers, and routed probes through the
  existing zellijprobe adapter with explicit PAIR_PROBE_REPO. No live probes.
  ARCH-DRY: both use internal/reporoot; no compiled-path fallback or cached cwd.
- Root tests failed before implementation, then passed both with and without
  trimpath. The recording fixture regression failed at three cwd depths before
  its fix. Targeted former failures passed (/tmp/pair572-path-tests.log).
  Probe tests and compilation passed with trimpath.
- Make flag regression failed before the default, then passed: preserved
  environment/command-line GOFLAGS, no duplicate trimpath in nested Make, and
  explicit PAIR_GO_TRIMPATH= opt-out. `make build` passed; go version -m bin/pair
  confirms -trimpath=true. Required missing oracle assets fail rather than skip.
- The existing artifact inventory now covers top-level internal/ and its root
  resolver. Probe discovery stays in the existing shared probe adapter.
