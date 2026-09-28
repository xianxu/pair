---
id: '000147'
status: punt
created: 2026-08-21
updated: 2026-09-02
---

# couch: cluster transport and queries

## Problem

Actors that cannot talk are just supervised processes. The cluster needs message
passing that works when the recipient is usually offline, and a way for one actor
to ask another about its state without the caller reaching into the callee's repo
and without spending an LLM turn or a human approval.

The Claude-native peer roster is not that channel: it covers only live ∧ Claude ∧
reachable, missing codex sessions entirely and every parked thread, and a single
status query was held for human approval on both ends.
