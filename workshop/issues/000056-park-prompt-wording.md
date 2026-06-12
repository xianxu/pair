---
id: 000056
status: done
deps: []
github_issue:
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

## Spec

1. **Reword the prompt + success message in `cleanup_quit_marker` (`bin/pair`):**
   - Prompt → `pair: preserve "pair-3" scrollback to distill into a continuation
     later? [y/N]:` (uses `$SESSION`; says *preserve … later*, not *create*).
   - On Yes → print the preserved `.raw` path on its own line, then the distill
     hint. Success message to `/dev/tty` (the prompt's channel; was stdout).
   - No behavior change — still `park_scrollback` move-mode; text + path + tag only.
2. **Add `:PairTTYRawPath()` (`nvim/init.lua`):** an `_G.PairTTYRawPath()` +
   `:PairTTYRawPath` user command that prints the live raw-scrollback path for
   the current session (`$PAIR_DATA_DIR/scrollback-<tag>-<agent>.raw`) with its
   size, copies it to the `+` register, and warns if not inside a pair session.
   Reuses the existing `pair_data_dir()` helper (ARCH-DRY).

## Done when

- Prompt shows `pair-<tag>` and says "preserve … to distill … later" (not "as a continuation").
- On Yes, the absolute preserved-file path is printed.
- `:PairTTYRawPath` prints the current session's raw-scrollback path (+ copies to `+`).
- `bash -n bin/pair` clean; `luac -p nvim/init.lua` clean.

## Plan

- [x] `bin/pair` `cleanup_quit_marker`: reword prompt (use `$SESSION`); print preserved path on Yes; success msg → `/dev/tty`.
- [x] `nvim/init.lua`: `:PairTTYRawPath` / `_G.PairTTYRawPath()` — print live raw-scrollback path + copy to `+`.

## Log

### 2026-06-12
- 2026-06-12: closed — bash -n bin/pair + luac -p nvim/init.lua clean. Park prompt reworded (shows pair-<tag> via $SESSION; "preserve … distill later" not "as a continuation"; prints preserved .raw path on Yes). :PairTTYRawPath prints live raw-scrollback path + copies to +. Atlas updated. Runtime (interactive Alt+x prompt, :PairTTYRawPath in a live pane) operator-manual; no logic change to park_scrollback.; review verdict: SHIP
- Reworded the Alt+x park prompt (`bin/pair:1425`) to `preserve "$SESSION" … to distill into a continuation later?` (shows `pair-<tag>`, drops the misleading "as a continuation"); success message now prints the preserved `.raw` path on its own line to `/dev/tty`.
- Added `:PairTTYRawPath` / `_G.PairTTYRawPath()` (`nvim/init.lua`) per operator request: prints the live raw-scrollback path for on-demand grabbing during a session (the complement to the quit-time preserve, which exists because quit deletes the `.raw`). Copies to `+`, warns outside a pair session, reuses `pair_data_dir()`.
- Verified: `bash -n bin/pair` clean; `luac -p nvim/init.lua` clean. Runtime (interactive Alt+x prompt + `:PairTTYRawPath` in a live pane) is operator-manual.
