---
id: 000635
status: open
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
card_mirror: '3d35c7f49eb24dcfd84b24e65f7e67bb322c2232' # card fields mirrored from issue-cards; edit via sdlc
---

# couch --message-status with a short ID reports a submitted message as unretained

## Problem

`couch --send-to` prints the message's full ID. A sender who checks it with the 8-character prefix the
fleet uses in reports (`couch --message-status 97fcc5ef --json`) gets
`{"Code":"refused","Error":"receipt unavailable: no connected recipient retains this message"}`, even while the
broker holds the receipt as `submitted`. Querying the full ID returns the real status. Observed 2026-10-16 on pair:2:
three hub reports read as lost were in fact delivered, and the TL had replied to one of them. A false
"unretained" answer leads senders to resend, or to treat delivery as broken.

## Spec

- `--message-status` with an ID prefix resolves it against the receipts the caller may see (its own sends and
  receipts addressed to it).
- Exactly one match: answer for that full ID.
- Several matches: refuse as ambiguous and name the candidates' full IDs.
- No match: refuse as unknown, worded so it can't be mistaken for an answer about a real message.
- Never answer "no connected recipient retains this message" for an ID the broker holds.

## Done when

- A test sends a message, queries its 8-character prefix and gets `submitted`.
- A test with two receipts sharing a prefix gets an ambiguity refusal that lists both.
- A test with an unknown prefix gets an unknown refusal, not "unretained".

## Plan

- [ ]

## Log

### 2026-10-16
