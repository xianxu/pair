---
id: 000190
status: open
deps: []
github_issue:
created: 2026-09-05
updated: 2026-09-05
estimate_hours:
---

# A vim.notify in the draft blocks it behind a hit-enter prompt under cmdheight=0

## Problem

Operator report: "did alt+return break?" — in a couch session on the parley.nvim
workspace, the draft stopped sending. Park/resume fixed it, and it was the first
time it had happened.

**Alt+Return was not broken, and neither was any keymap.** The screenshot shows
the draft sitting at nvim's hit-enter prompt:

    Press ENTER or type command to continue

Every keystroke is queued behind that prompt until a plain Enter dismisses it, so
`<M-CR>` — and Alt+q, Alt+h, ordinary typing — all appear dead. Park/resume
"fixed" it by restarting nvim, which is why the fix looked unrelated to the
symptom.

Ruled out on the way, because both were plausible: couch intercepts only six
chords (`ctrl-space`, `ctrl-backspace`, `alt+x`, `alt+d`, `alt+n`,
`ctrl+alt+n`) and no Enter-family sequence, so it forwards `<M-CR>` untouched;
and although `parley.nvim` binds `<M-CR>` for respond/define/review-menu, the
draft launches as `nvim -u <pair>/nvim/init.lua` with no `.nvim.lua` and no
`exrc`, so the user plugin is not loaded there.

**The cause is `cmdheight=0` plus `vim.notify`.** `init.lua:241` sets
`cmdheight = 0`, so there is no command line for a message to land in; anything
non-trivial forces the hit-enter prompt. The draft has **22 `vim.notify` calls**
and **2 `flash_at_cursor` calls**.

The image-paste path (Alt+i) contains both styles, in one function:

    flash_at_cursor('[no image in clipboard]', 'WarningMsg', 1000)   -- non-blocking
    vim.notify('pair: PAIR_TAG unset — not inside a pair session?', ERROR)
    vim.notify('pair: exact image-capture paths unset — restart the pair session', ERROR)
    vim.notify('pair: pair-wrap pid missing — restart the pair session (Alt+n)', ERROR)
    vim.notify('pair: pair-wrap (pid N) not running — placeholder left in place; restart the pair session (Alt+n)', ERROR)

That path is the likely trigger here: the operator had been pasting screenshots,
and three of those four messages advise restarting the session, which is exactly
what park/resume did. The last is ~100 characters and overflows on any realistic
draft width. Which one actually fired cannot be recovered — nvim's message
history died with the process — so this issue is about the CLASS, not that
instance.

`flash_at_cursor` (`init.lua:1102`) already exists and is the right shape:
inline virtual text at the cursor, auto-clearing after a duration, blocking
nothing.

**The author already knew messages misbehave here.** `init.lua:243-248` suppresses
`:w`'s file-info output with `shortmess:append('WF')` because "every autosave or
send-and-clear write briefly pops the cmdline up under `cmdheight=0`". The write
path was fixed; the error paths were not.

## Spec

An error in the draft must not take the draft hostage. The draft is where the
operator types; a modal prompt there costs them their input surface and gives no
hint that Enter is what clears it.

Route operator-facing draft diagnostics through the non-blocking affordance that
already exists. `vim.notify` at ERROR/WARN should survive only where a message
genuinely must stop the operator — and the bar for that inside a text buffer they
are mid-sentence in is very high.

**Decided (2026-09-05, with the operator): ONE SEAM, not four conversions.**
Add a single `pair_notify(message, level)` that renders through the flash and
blocks nothing, and point every `vim.notify` site at it. Converting only the four
image-paste sites fixes the reported instance and leaves eighteen others able to
freeze the draft the same way — the hand-maintained-list shape that `pair#182`'s
review found five times over. One seam also means a site added later cannot
reintroduce the block without deliberately reaching past it.

`vim.notify` may stay where a path is genuinely NOT draft-reachable (some
`PairReview` and spell-check paths may not be), but the default flips: blocking
is the exception that must be argued, not the idiom.

Two things to decide rather than assume:

1. **`flash_at_cursor` is inline virt_text, so a 100-character message will wrap
   or clip at the cursor.** The messages need shortening, which is an improvement
   rather than a compromise: most of their length is the same advice repeated —
   "restart the pair session (Alt+n)" is the actionable half, and three of the
   four say it. If a failure genuinely needs more than a flash can carry, write
   it to the pair log where it can be read later and flash a pointer.
2. **Which sites are draft-reachable at all.** Enumerate before converting; a
   site that can never run in the draft is not evidence for or against the rule.

## Done when

- No draft-reachable code path can leave nvim at a hit-enter prompt, proved by a
  test that drives the failing condition and asserts the editor still accepts
  input.
- The image-paste failures report through a non-blocking affordance.
- The remaining `vim.notify` sites are enumerated, with each either converted or
  recorded as deliberately blocking with its reason.
- `cmdheight = 0`'s interaction with messages is documented beside the setting,
  which currently explains only the `:w` half.

## Plan

- [ ] Reproduce first: force one image-paste failure (unset
      `PAIR_IMAGE_CAPTURE_PATH`) in a draft and confirm the hit-enter prompt, so
      the fix has a red state to be measured against. Without this the fix is
      unfalsifiable — a draft that accepts input looks identical whether the
      message was non-blocking or never fired.
- [ ] Add `pair_notify`, routing through the flash. Shorten the four
      image-paste messages to their actionable half as they move.
- [ ] Enumerate the remaining `vim.notify` sites, convert the draft-reachable
      ones, and record each survivor with the reason it must block.
- [ ] Document the `cmdheight=0` message rule beside the setting, which today
      explains only the `:w` half.

## Log

### 2026-09-05

Found from an operator report of "alt+return broke", which it had not. Worth
recording how the diagnosis went, because the reported symptom pointed at three
wrong layers before the right one: a chord that stopped working looked like
couch interception (it intercepts no Enter-family sequence), then like a keymap
collision with `parley.nvim` (which does bind `<M-CR>`, but is not loaded in a
`-u`-launched draft), and was neither. The screenshot contained the answer in
plain text at the bottom of the pane — the hit-enter prompt — which no amount of
reading the keymap tables would have produced.
