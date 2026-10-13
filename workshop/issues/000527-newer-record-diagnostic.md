---
id: 000527
status: open
deps: [pair#517]
github_issue:
created: 2026-10-13
updated: 2026-10-13
estimate_hours:
card_mirror: 'f6a2fa53e3b649d614006dbb5c16ed8c3b955282' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-13T07:43:55-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
---

# couch: distinguish newer slot record formats from damaged records

## Problem

An older Couch mislabels a valid record written by a newer build as damaged.
On 2026-10-13 the operator ran main without #517 after testing its staged
build. pair:5 retained `previous_session` in its slot-local `.couch/thread.json`.
The older UI reported “damaged slot record has unresolved terminal binding;
inspect before recovery.” The operator must not be pushed toward reboot or
record repair merely because the reader predates the record format.

## Spec

Distinguish unsupported/newer persisted formats from malformed data at the
record decoder and surface that distinction through slot inspection/recovery.
Tell the operator “written by a newer Couch; update Couch” (or equally clear
wording) when supported by format evidence. Preserve strict validation: unknown
fields alone must not silently become accepted or erase terminal authority.
Choose the compatibility/versioning contract during design; existing released
binaries cannot be retroactively given better diagnostics.

## Done when

- A newer-format record produces an actionable update diagnostic instead of
  the generic damaged/unresolved-terminal message, without rewriting it.
- Truly corrupt records remain distinguishable and fail closed.
- Tests cover a newer writer/older reader fixture and unchanged record bytes;
  recovery does not offer destructive repair based only on version mismatch.

## Plan

Not started; follow-up only.

## Log

### 2026-10-13
- 2026-10-13: unclaimed: Follow-up filed and scoped; #517 landed as PR304. Exact pair:5 snapshot also passes decoder verification on landed main 86d206ccc. No implementation started; available for scheduling.

- TL receipt 07e52450-7047-4d71-b1e8-cb36b2c35e03 requested this follow-up.
  Exact pair:5 snapshot: 4,251 bytes, schema 2, revision 291, tag 1-pair-20,
  scope 23e53ad8c4dfb803, SHA256
  6bf4e1024df57a7f8f77822af900696d672e6cea596e12eaf50d482c8e6e62f7.
  #517 DecodePersisted accepts it with real launcher validators. Overlaying
  origin/main805031105's pre-#517 Record/decoder rejects the same bytes with
  `json: unknown field "previous_session"`: strict unknown-field decoding at
  unchanged schema 2 is the immediate compatibility boundary. Live bytes were
  verified unchanged; no reboot, rewrite or recovery was performed.
