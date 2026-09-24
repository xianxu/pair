---
type: pensive
date: 2026-09-22
topic: couch qoder registry staleness
mode: eureka
description: A couch process must restart to observe a newly registered Pair agent; path preferences choose an agent but cannot extend an already-running binary's registry.
references: [cmd/internal/launcher/agent_defaults.go, cmd/internal/couchcore/couch.go, workshop/issues/000300-integrate-qoder-harness-into-pair.md]
---

# Pensive: Couch qoder registry staleness

Qoder's failure under Couch was not a second Couch-specific agent list or a missing configuration toggle. Pair's shared launcher registry already contains `qoder`, and Couch derives both its start form and its launch validation from it. The unsupported-agent error came from a Couch process started before the M1 registry change; restarting that process made qoder appear and launch normally.

A path preference selects an agent profile; it cannot authorize a name that the running executable does not know. Manual edits to the preference JSON were therefore a useful probe but could never repair the stale process.

## Open questions

The live→park→cold-resume and switch-agent round-trip remain to be verified with the restarted Couch process. The adapt-log restart/quit teardown finding is separate agent-agnostic infrastructure work and should not be disguised as a qoder integration defect.
