---
id: 000330
status: open
deps: []
github_issue:
created: 2026-09-25
updated: 2026-09-25
estimate_hours:
---

# Carry binding_stale diagnostics into COUCH_TRACE

## Problem

Since #328, `QuerySessionContext` records a `binding_stale` diagnostic when a
binding's proof no longer validates. But nothing in Couch reads it:
`SessionInventoryNativeBindingResolver.ResolveEstablished`
(`cmd/internal/couchcore/resume.go`) drops the query's diagnostics, and the
relaunch refusal shows only the generic provisional sentence. The operator
asked for this reason in a debugging log, not on the UI.

## Spec

- Carry the query's diagnostics on the resume refusal as debug-only detail,
  kept out of the rendered UI text.
- When `COUCH_TRACE` is on, write the detail to the trace file.
- Keep the `--owner` CLI unchanged: `nvim/init.lua:838` captures its stdout
  and stderr together.

## Done when

- With `COUCH_TRACE` set, a relaunch refused by a failed proof writes one trace
  line naming the `binding_stale` detail. The UI refusal text is unchanged.

## Plan

- [ ]

## Log

### 2026-09-25

- Follow-up from #328's close review (Minor: the diagnostic has no reader).
