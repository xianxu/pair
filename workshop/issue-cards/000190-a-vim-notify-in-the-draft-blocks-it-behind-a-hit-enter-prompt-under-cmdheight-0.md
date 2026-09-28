---
id: 000190
status: open
created: 2026-09-05
updated: 2026-09-05
estimate_hours:
github_issue:
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
