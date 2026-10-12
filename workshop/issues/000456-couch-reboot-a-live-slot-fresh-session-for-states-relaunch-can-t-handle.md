---
id: 000456
status: open
deps: []
github_issue:
created: 2026-10-11
updated: 2026-10-11
estimate_hours:
card_mirror: '814f3692ab148cb7d2806fe6b86da05921dde7f0' # card fields mirrored from issue-cards; edit via sdlc
---

# couch: reboot a live slot (fresh session), for states relaunch can't handle

## Problem

`couch --reboot` refuses a live slot (`not-offered: … is live`), and relaunch (Alt+n, `couch --relaunch`) needs a conversation to resume. Some live slots fit neither: a fresh session that never completed a turn (`resume-binding-unbound`), or a Codex slot on an old wrapper that stays `resume-binding-provisional` even after a completed turn. On 10-11 parley.nvim:0 and parley.nvim:3 hit this during the fleet relaunch; the TL had to send a throwaway turn to one, and the operator relaunched the other by hand. The operator needs a way to restart a live slot from scratch.

## Spec

- **Reboot a live slot:** `couch --reboot repo:N --confirm` also accepts a live slot: stop its agent and wrapper, then start a fresh session on the current binary in the same pane and worktree. The conversation is not resumed (that's relaunch's job); the worktree and branch are untouched.
- **Guards**, the same vocabulary as relaunch: refuse `busy` (a turn running), and `busy-unknown` unless `--force-unknown`. A dirty tree does not block (reboot changes no files), but the result reports it.
- **In Couch:** a reboot action next to Alt+n for the thread shown or highlighted, with a confirmation, since it discards the conversation. The key or menu entry is the operator's choice; propose one that doesn't collide (check keyhelp).
- **Relaunch's refusals point at it:** `resume-binding-unbound` and `resume-binding-provisional` name reboot as the next step.
- Atlas and `couch --help` updated.

## Done when

- A live slot with a never-finished fresh session (Claude and Codex) reboots by `couch --reboot … --confirm` and from Couch; the new session runs on the current binary.
- Busy slots are refused; `--force-unknown` overrides busy-unknown.
- Tests cover the guard decisions; a live check reboots a real slot.
- The operator smoke-tested the Couch action before close.

## Plan

- [ ] Reboot of a live slot in couchcore/dispatcher, with guards
- [ ] Couch action and confirmation; keyhelp
- [ ] Relaunch refusals name reboot
- [ ] Tests, live check, atlas, help; operator smoke test

## Log

### 2026-10-11
