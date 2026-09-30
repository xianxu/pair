---
id: 000354
status: open
deps: [pair#211]
github_issue:
created: 2026-09-29
updated: 2026-09-29
estimate_hours:
card_mirror: 'ea1055225377173e792d8b6ca5f691a526d31788' # card fields mirrored from issue-cards; edit via sdlc
---

# Verify bracketed draft send ended Claude's middle-read drop

## Problem

pair#211 found Claude Code dropping whole ~1 KiB middle tty reads of
unbracketed draft sends (8 of 11 three-read sends in the 2026-09 audit) and
fixed it by sending the body as one bracketed paste. The drop could not be
reproduced on demand (an idle Claude delivered plain sends whole too), so
whether the bracketed send defeats it on a busy Claude is proven only by
real use after the fix lands.

## Spec

After about a week of normal use with the fix on `pair:0`, run
`scripts/send-audit.py --since <#211 merge date>`. Zero lossy Claude sends in
the 3- and 4+-read rows, over enough such sends to mean something (the
pre-fix rate was 8 of 11), confirms the fix. Any hole means bracketing did not
defeat it; reopen the cause in #211's terms (the hole's offset and size) rather
than patching around it.

## Done when

- The audit covers at least ~10 Claude sends of 3+ reads made after the fix,
  with 0 lossy, recorded in this Log with the command output
- Or: a lossy post-fix send is recorded with its offset/size and a new issue
  carries the diagnosis

## Plan

- [ ] Run `scripts/send-audit.py --since <merge date>` after a week of use
- [ ] Record the table here; close or escalate per Done when

## Log

### 2026-09-29

- Filed from #211's close: the post-ship verification row moved here so #211
  could close on what it proved (pair's path byte-exact, fix pinned by tests).
