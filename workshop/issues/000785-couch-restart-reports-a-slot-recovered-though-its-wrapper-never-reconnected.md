---
id: 000785
status: open
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'b8eb04eba3425243fc24989032d7442490a12ba9' # card fields mirrored from issue-cards; edit via sdlc
---

# Couch restart reports a slot 'recovered' though its wrapper never reconnected

## Problem

## Spec

After a Couch restart, the lifecycle notice can report a slot **"recovered"** although its wrapper never reconnected:
- 10-21 (ops-echo:0): echo:1 reported recovered by the 13:xx restart pass, but its wrapper socket was missing
  (`prepare: dial unix ... no such file`) until the next restart.
- 10-21 13:58 (ops:0): after the operator's restart, pair:1 was absent from the restart list and had no connected
  wrapper ("no wrapper is connected for this thread"); `couch --resume pair:1` fixed it.

Fix: "recovered" only after the slot's wrapper has actually connected (a confirmed hello); otherwise report it as
**"not reattached"** with the one recovery command (`couch --resume SLOT`), in the lifecycle notice and in
`couch --actors`.

## Done when

- A restart where one slot's wrapper never connects reports that slot as not reattached (with the resume command),
  not recovered; a test covers the missing-socket case.

## Plan

- [ ]

## Log

### 2026-10-21
