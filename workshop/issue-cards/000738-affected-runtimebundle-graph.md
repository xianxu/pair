---
id: 000738
status: open
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
github_issue:
---

# Affected package graph falls back to full suite before runtime bundle preparation

## Problem

During #726's first recorded affected check after integrating #725, a fresh local-check worktree could not load the Go package graph:

`cmd/internal/runtimebundle/embed.go:8:41: pattern assets/runtime/files: no matching files found`

`sdlc local-check run --name affected` selected the full suite instead, with reason `package graph unavailable`. The checkout had not generated the runtime bundle before the Go graph query. This is safe (the suite still runs) but prevents the affected runner from selecting the intended smaller check in that fresh-checkout case.
