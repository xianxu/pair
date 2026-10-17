---
id: 000643
status: open
deps: []
github_issue:
created: 2026-10-17
updated: 2026-10-17
estimate_hours:
collaboration-mode: collaboration
card_mirror: '0cfc7685c8642ae4535679eaab68b25edcfb3ffe' # card fields mirrored from issue-cards; edit via sdlc
---

# couch session-sync: frontmatter-scoped read/write that moves the diff baseline only for the frontmatter

## Problem

## Spec

From ariadne#424 M2 (ariadne:2), 2026-10-17. `couch --session-sync read --all` and `write --all` reset the diff baseline for the whole file. Measured with the live conformance test: after a `read --all`, `couch diff` prints only `lock: none`, even for an operator edit made just before. A writer that needs to change only the frontmatter (ariadne#424 maintains the `discussions:` index there) therefore consumes the owner's unread operator edits in every section.

Proposal: frontmatter-scoped verbs that move the baseline only for the frontmatter, the way the section verbs do for a section:
- `couch --session-sync read --frontmatter FILE`
- `couch --session-sync write --frontmatter BODYFILE FILE --if-hash H`

The hash is the frontmatter's; a stale hash refuses. The operator's lock refuses as other writes do.

## Done when

- Reading or writing the frontmatter leaves the diff baseline of the body unchanged. A test makes an operator edit in a section, writes the frontmatter, and the edit still shows in `couch diff`.
- A stale frontmatter hash refuses, and the operator's lock refuses.
- The verbs are in `couch --help` with their cost tags. tl-lead (ariadne#424 or a follow-up) switches to them.

## Plan

- [ ]

## Log

### 2026-10-17
