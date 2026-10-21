---
id: 000738
status: working
deps: [ariadne#517]
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'b43a4c223b737ec0123547c12006cd55be7cea1b' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T19:58:34-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
---

# Affected package graph falls back to full suite before runtime bundle preparation

## Problem

During #726's first recorded affected check after integrating #725, a fresh local-check worktree could not load the Go package graph:

`cmd/internal/runtimebundle/embed.go:8:41: pattern assets/runtime/files: no matching files found`

`sdlc local-check run --name affected` selected the full suite instead, with reason `package graph unavailable`. The checkout had not generated the runtime bundle before the Go graph query. This is safe (the suite still runs) but prevents the affected runner from selecting the intended smaller check in that fresh-checkout case.

## Spec

Trace the generated-runtime-bundle preparation prerequisite for affected package-graph collection in a fresh local-check worktree. Establish that prerequisite before graph discovery, or supply a correct equivalent graph input without depending on artifacts in the operator's existing checkout. Preserve the full-suite fallback for genuinely unavailable or unclassifiable graphs. Keep this separate from #726's admission work.

## Done when

- A fresh disposable checkout with the generated runtime files initially absent can compute the intended Go package graph for an otherwise eligible affected change.
- A regression reproduces the original missing-embed-path failure before the fix, without borrowing generated files from another checkout.
- Genuine graph failures still select the full suite and name their cause; no false passing affected result is recorded.

## Plan

- [x] Reproduce missing embed assets in a private clean checkout and trace the graph owner.
- [ ] Wait for ariadne#517's bounded preparation seam to land.
- [ ] Declare the existing runtime generator, qualify clean graph and broken-graph fallback, then obtain operator local-gates verification before landing.

## Log

- Observed on #726's pushed noon-integrated head a9afd1079e958431e4efec7c485ec645e7ac9b3f by `sdlc local-check run --name affected`. It printed full selection due to the missing embed path and launched the suite in /Volumes/TestScratch/local-checks/sdlc-local-4051440168. That suite remains the verification path for #726; this filing is unclaimed follow-up to the affected-runner work in #725.

### 2026-10-20 — measured prerequisite and parked work

A disposable git archive of this issue's main-based checkout contained no
generated runtime assets. Native go list -test ./... failed at embed.go:8:41
with pattern assets/runtime/files: no matching files found. Running the existing
runtimebundle generator in that same checkout made the graph load successfully
(408 entries). No output was borrowed from another checkout. The private tree
was removed; retained measurement log is .git/manual-verification/pair738-cold-graph.log.

Ariadne productionAffectedDeps creates a detached HEAD checkout and immediately
calls LoadGoGraph. Pair's selected command executes later, so adding generation
there cannot repair discovery. Filed unclaimed ariadne#517 for the provider seam.
The hub approved this dependency and parked #738 while assigning #736. No code,
dummy embed, partial -e graph, or local-gates label was introduced.

## Revisions

### 2026-10-20 — prepare at the graph owner

The measured ordering requires ariadne#517: an optional bounded argv declared
by the repo runs once in the same head checkout before LoadGoGraph. Preparation
failure, timeout or cancellation retains conservative full-suite fallback and
its cause. ARCH-DRY reuses make runtimebundle-generate rather than duplicating
the generator or accepting an incomplete graph. After that provider lands,
#738 declares that command in .sdlc/affected.json and tests a genuinely clean
graph plus a broken-graph control. This is a local-gates path: the operator
verifies and labels it; the worker never self-labels. No staging is requested.
