---
id: 000653
status: open
deps: []
github_issue:
created: 2026-10-17
updated: 2026-10-17
estimate_hours:
collaboration-mode: collaboration
card_mirror: '41fc27fc634128c669e6baab1758d1a44290fe2b' # card fields mirrored from issue-cards; edit via sdlc
---

# Context-used figure missing for many slots since the 10:48 restart onto the overnight build (regression; bindings are fine)

## Problem

## Spec

Operator report, 2026-10-17 ~12:20: the per-slot "context used" figure (the agent's context-window usage, shown per slot in Couch) is missing for many slots, ops:0 included. It's been missing since the 10:48 Couch restart onto pair:0's build of the night's landings (15d84ef7, later rebuilds up to 145f38cd).

What's known:
- It isn't a binding problem, at least for ops:0. Its ledger (`repos/aef274ee2fbbcabd/ledger-1-ops-33.jsonl`) has the relaunch's launch record (resume, launch ordinal 32) and a binding record (correlation, root_native_id 20b1e863-53a4-4cba-8682-3489e12d52cc), and `--resume-candidates` reports an established native binding.
- It affects many slots, Claude and Codex both, and the relaunched and the merely restarted alike. So it looks like a regression in that build, not one slot's state.

Suspects among the landings in that build, unverified:
- pair#626 (96a28cca): the shared JSONL framer now searches only newly appended bytes. If the context reading goes through it, a record split across reads, the offset bookkeeping or the first record could be dropped.
- #625 (8013b93c): fading from Settled and last-work time.
- #632 (1606ded7): BackgroundWork on hello-v4.
- #641 (74fdb8e2): the erased-cell projection in terminalRowText, if context is read from the footer.
- #618 M2: peer turns now written into the Pair log.

## Done when

- Root cause named with file:line, and the exact build where it regressed; check against a pre-restart build if possible.
- A regression test that fails on the broken code: a slot whose transcript (or whatever the source is) grows across reads still shows its context figure.
- Fixed and landed; after the next restart, every bound slot shows its context figure. Checked live.

## Plan

- [ ]

## Log

### 2026-10-17
