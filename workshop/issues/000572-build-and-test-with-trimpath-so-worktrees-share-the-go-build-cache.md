---
id: 000572
status: open
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: '513e86566c52758a50757958c33381db6e5eda0d' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ]

## Log

### 2026-10-15
