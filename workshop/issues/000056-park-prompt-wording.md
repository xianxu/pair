---
id: 000056
status: working
deps: []
github_issue:
created: 2026-06-12
updated: 2026-06-12
estimate_hours: 0.2
---

# Park prompt: reword (preserve, not create), show pair-<tag>, print preserved file path

## Problem

The Alt+x park-nudge (`cleanup_quit_marker`) has two UX issues (operator,
post-#55 dogfood):

1. **Inconsistent tag.** The prompt printed `$PAIR_TAG` (`3`) while the very
   next message + the resume/park commands use `pair-3` (`$SESSION`).
2. **Misleading wording.** "park "3" as a continuation? (preserve its
   scrollback)" implies it *creates* a continuation. It does not — at quit
   there's no live agent to distill. It only **preserves the raw scrollback**
   (`.raw` VT bytes, in the XDG data dir, NOT the repo) for a live agent to
   render + distill into a committed continuation doc *later*.

Also: on Yes, the preserved file's location should be printed clearly (the
operator couldn't tell where it lived).

## Spec

Reword the prompt + success message in `cleanup_quit_marker` (`bin/pair`):

- Prompt → `pair: preserve "pair-3" scrollback to distill into a continuation
  later? [y/N]:` (uses `$SESSION`; says *preserve … later*, not *create*).
- On Yes → print the preserved `.raw` path on its own line, then the distill
  hint (`open a session and "park pair-3" …`). Success message to `/dev/tty`
  (consistent with the prompt; the success line was previously on stdout).
- No behavior change — still `park_scrollback` move-mode; just text + the path
  print + the tag display.

## Done when

- Prompt shows `pair-<tag>` and says "preserve … to distill … later" (not "as a continuation").
- On Yes, the absolute preserved-file path is printed.
- `bash -n bin/pair` clean.

## Plan

- [ ] `bin/pair` `cleanup_quit_marker`: reword prompt (use `$SESSION`); print preserved path on Yes; success msg → `/dev/tty`.

## Log

### 2026-06-12
