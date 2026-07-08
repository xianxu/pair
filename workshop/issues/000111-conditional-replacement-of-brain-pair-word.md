---
id: 000111
status: working
deps: []
github_issue:
created: 2026-07-07
updated: 2026-07-07
estimate_hours: 0.40
started: 2026-07-07T23:38:22-07:00
---

# conditional replacement of brain/pair word

In pair, when we set cmux workspace title, we replace brain for 🧠 and pair for another unicode. we should not do this when brain or pair is the single word. 

## Problem

The cmux workspace title convention currently replaces `brain` with `🧠` and
`pair` with `♋` unconditionally. That is useful for compound Pair session names
such as `pair-brain`, but it is wrong when the whole title is just `brain` or
`pair`: a plain shell cwd/workspace title should stay readable as the literal
repo name.

There are two active implementations of the convention:

- `cmd/internal/launcher.EmojiTitle`, used by launcher cmux renames.
- `cmd/internal/titlepoller.cmuxWorkspaceTitle`, used by the title poller heat
  ramp.

## Spec

- Replace `brain`, `book`, and `pair` only when the input title contains a
  separator/compound session form, e.g. `pair-brain` → `♋-🧠`.
- Preserve literal single-word titles: `brain` → `brain`, `pair` → `pair`,
  `book` → `book`.
- Keep the launcher and title poller on one shared pure substitution rule
  (`ARCH-DRY`, `ARCH-PURE`), so the cmux rename paths cannot drift again.

## Done when

- `EmojiTitle("pair-brain-book") == "♋-🧠-📗"` still holds.
- `EmojiTitle("brain") == "brain"` and `EmojiTitle("pair") == "pair"`.
- `cmuxWorkspaceTitle` follows the same behavior through the title poller.
- Focused Go tests cover both direct launcher formatting and title poller titles.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module design=0.10 impl=0.30
design-buffer: 0.0
total: 0.40
```

## Plan

- [ ] Add RED tests for single-word preservation and compound replacement in
  `cmd/internal/launcher` and `cmd/internal/titlepoller`.
- [ ] Extract/share the pure emoji-title rule so launcher and title poller derive
  from one implementation.
- [ ] Verify focused packages and the wider Go suite.

## Log

### 2026-07-07

- Claimed and planned. Root cause: the cmux title convention uses unconditional
  `strings.ReplaceAll` in both launcher and titlepoller. The fix should be a
  shared pure rule, not two parallel conditional edits (`ARCH-DRY`,
  `ARCH-PURE`), and it must preserve both affected consumers (`ARCH-PURPOSE`).
