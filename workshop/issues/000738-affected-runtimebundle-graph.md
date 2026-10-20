---
id: 000738
status: open
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: '13a01cfe07c9ccfff6904a289c0acffd526003fa' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ]

## Log

- Observed on #726's pushed noon-integrated head a9afd1079e958431e4efec7c485ec645e7ac9b3f by `sdlc local-check run --name affected`. It printed full selection due to the missing embed path and launched the suite in /Volumes/TestScratch/local-checks/sdlc-local-4051440168. That suite remains the verification path for #726; this filing is unclaimed follow-up to the affected-runner work in #725.
