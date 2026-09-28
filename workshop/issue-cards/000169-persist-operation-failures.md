---
id: '000169'
status: punt
created: 2026-09-01
updated: 2026-09-03
---

# Persist Couch operation failure diagnostics

## Problem

Couch start and resume failures are reported only through a transient panel
notice. The notice can disappear during refresh/navigation and there is no
durable operation diagnostic to inspect afterward. In the observed Pair
failure, the operator saw the action silently fail, while post-hoc evidence
showed a rolled-back thread reservation and a partially launched Pair/Zellij
session; the original subprocess error was unrecoverable.
