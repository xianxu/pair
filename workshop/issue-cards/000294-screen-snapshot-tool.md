---
id: 000294
status: open
created: 2026-09-19
updated: 2026-09-19
estimate_hours:
github_issue:
---

# Screen snapshot for debugging: chord for the operator, pair screen for the agent

## Problem

When pair or couch draws something wrong, the operator has to describe the screen
to the agent in words, and the agent can't check its own rendering changes. Since
#255, pair and couch keep an exact model of each child's screen:
`terminal.Endpoint.Snapshot(now)` (`cmd/internal/terminal/endpoint.go:275`)
returns a `Frame` with every cell and its style, the cursor and the modes. A
screenshot of what pair drew is one call away, and nothing exposes it.
