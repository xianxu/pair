---
id: '000056'
status: done
created: 2026-06-12
updated: 2026-06-12
estimate_hours: 0.2
actual_hours: 0.08
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

**Operator follow-up:** "why preserve, vs just print the path?" — answered: at
Alt+x quit `cleanup_quit_marker` *deletes* the `.raw` (`[ "$_parked" = 1 ] ||
rm -f …`), and the next same-tag launch `O_TRUNC`s it; preserve = rename it out
of the recyclable namespace so it survives. So you can't merely print the path
*at quit*. But *during* a live session the `.raw` exists — so add an on-demand
`:PairTTYRawPath()` to print it (the complementary half).
