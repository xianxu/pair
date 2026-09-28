---
id: 000270
status: open
created: 2026-09-16
updated: 2026-09-16
estimate_hours:
github_issue:
---

# Draft hit-enter prompt has no trigger, no recovery

## Problem

Operator report with screenshot (2026-09-16): the draft nvim pane is sitting at

    Press ENTER or type command to continue

with the draft text intact above it and the custom statusline replaced by the
prompt. Every keystroke queues behind that prompt, so `Alt+Return` — and Alt+q,
Alt+h, and ordinary typing — all appear dead. **The draft cannot submit.**

It is intermittent: "this doesn't always happen, but not sure how I get into
this bad state." The session that reported it was *already* in the state when
noticed, and the only known way out is restarting the session (park/resume, or
Alt+n) — which is also the thing that destroys the evidence.

**#190 owns the prevention half and already diagnosed the mechanism**:
`cmdheight = 0` (`nvim/init.lua:247`) leaves no command line for a message to
land in, so anything that does not fit forces nvim's hit-enter prompt; its fix
is one non-blocking `pair_notify` seam replacing the `vim.notify` sites. What
#190 could *not* do is name the trigger — "which one actually fired cannot be
recovered — nvim's message history died with the process" — and it says nothing
about recovery or detection. This issue is those three: **which path fires, how
the operator gets out without restarting, and how pair notices at all.**

Two things have changed since #190 was written (2026-09-05):

- The `vim.notify` count in `nvim/init.lua` is now **31**, up from the 22 that
  issue counted. The class is growing, not shrinking.
- #190 assumes the trigger is a `vim.notify`. **It may not be.** An nvim *error*
  — a failing autocmd or callback, `E5108` and friends — also forces the prompt
  under `cmdheight=0`, and a notify seam would not touch that path. If that is
  what is happening here, #190 can land in full and this symptom survives it.

Constraints found while filing, which bound the possible designs:

- **The draft nvim is not remotely addressable.** Nothing passes `--listen` or
  calls `serverstart`, so there is no socket to run `:messages` through or to
  send a dismissing `CR` to. Keys can reach the pane only the way the draft send
  path already reaches it — `zellij action write-chars` / `send-keys` at the pane
  id — which is therefore the only transport a recovery gesture can use.
- **A recovery chord cannot be handled by the stuck nvim**, since that is what is
  blocked. It has to be handled by something above it (a zellij/couch-level
  binding, or a pair-side command), the way couch already intercepts its six
  chords ahead of the pane.
- `vim.opt.more` is not set anywhere in `nvim/init.lua`.

**Ordering note worth acting on:** the trigger can only be identified from a
*live* stuck pane. If #190 lands first and the trigger was a notify site, the
state stops happening and the question becomes unanswerable — which is fine for
the operator and bad for knowing whether the error path (above) was ever
involved. Capture the evidence before #190 ships, not after.
