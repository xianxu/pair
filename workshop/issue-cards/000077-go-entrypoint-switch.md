---
id: '000077'
status: done
started: 2026-06-30T12:42:11-07:00
created: 2026-06-26
updated: 2026-06-30
estimate_hours: 2.6
actual_hours: 0.75
---

# pair Go entrypoint switch

## Problem

At some point the public `pair` command must become Go-owned. The next safe step is to make the Go-owned `pair-go launch ...` path exercise the real launcher contract while leaving the existing `pair` and `pair-dev` entrypoints stable.
