---
id: '000300'
status: done
started: 2026-09-20T18:58:25-07:00
created: 2026-09-20
updated: 2026-09-22
estimate_hours: 5.35
actual_hours: 4.42
---

# integrate qoder harness into pair

## Problem

pair hosts four harness CLIs — `claude`, `codex`, `agy`, `muse` — and the
operator's next harness, **qoder**, is registered nowhere: no TTY profile
(Return remap / overlay detection), no session scanner, no resume binding, no
slug/glyph support, invisible to couch. Separately, the bring-up guide
(`atlas/how-to-bring-up-a-new-harness-cli.md`) predates couch: it documents the
eight integration aspects as if Zellij+`pair wrap` were the only hosting path,
and never says what couch requires, what it consumes automatically, or that the
launcher agent registry is the single choke point both hosts share.
