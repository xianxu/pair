---
id: 000758
status: working
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-22
estimate_hours:
collaboration-mode: collaboration
card_mirror: '876331e5359be44ec56eb8ff8941349ceca9885f' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-22T04:59:40-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
---

# remote-session: Couch remote server and allowlisted proxy to each session file's nvim (subscribe, edit re-wrapped, submit, navigate)

## Problem
The remote document adapter cannot discover the live nvim buffer: existing registration identifies the document and owner, but advertises no editor socket or incarnation. A per-file socket alone cannot distinguish retry, explicit desktop takeover, stale cleanup or PID reuse. parley.nvim#405 needs the producer contract first.

remote-session (ops project, operator-approved 10-20): the phone talks only to Couch; Couch serves the session views by
proxying to the nvim that has each file open (registered by parley, parley.nvim issue filed alongside).

## Spec
M1 produces Couch-owned editor endpoint register/lookup/compare-unregister keyed by canonical namespace plus immutable document ID, with editor incarnation, attachment attempt, socket/buffer, native process identity and binding generation. Last explicit desktop attach wins; headless is fallback only. Exact current retry is idempotent; superseded retry never reacquires, including after restart. Couch owns replacement notices and exact stale-binding cleanup; Parley owns private listener/runtime files. Pin CLI/ack/callbacks with parley.nvim:1 before code.

Full issue remains the allowlisted registered-document/nvim adapter and its integration with pair#759. M1 does not expose a remote editor or complete the issue. Detailed producer contract, states, bounds, test counts and subsequent held slices: [plan](../plans/000758-editor-endpoints-plan.md).

- A Couch remote server (Go) behind the 👽-mode tunnel (sibling issue). The phone never reaches nvim directly.
- **Allowlisted proxy, not a pipe:** Couch translates exactly these into msgpack-RPC on the registered nvim: subscribe
  to a buffer's text and its highlights/extmarks (`nvim_buf_attach`, `nvim_buf_get_extmarks` with details), apply an
  edit, submit (calls parley's own submit, identical to Alt+Return), follow a link, go to the parent, jump back and
  forward (nvim's jump list is the truth, so desk and phone share history). Anything else is refused before nvim.
- **Edits keep the file hard-wrapped:** an edit inside a reflowed paragraph arrives as a replacement of that
  paragraph's line range and is re-wrapped at the file's width (like `gq`).
- **Headless fallback:** if no nvim has the file, Couch starts a headless nvim with parley for it, and hands over when
  a desktop nvim opens the file.
- Scope: every hub's `session.md` and its side docs. No agent panes or input boxes in v1.

## Done when
- Producer contract is confirmed by parley.nvim:1, with production CLI/private IPC and stateful tests defending retry, replacement, batch release and stale cleanup.
- Unknown fields survive durable rewrites and all authority/refusal paths are bounded and generation-fenced.
- Real document adapter and integrated operator acceptance deliver the issue scope after prerequisite interfaces land; M1 alone is not whole-issue completion.

- Tests against a real headless nvim: subscribe receives text and highlights; an edit round-trips re-wrapped; submit
  produces the same session-sync submit as Alt+Return; a non-allowlisted call (e.g. an Ex command) is refused.

## Plan
- [ ] M1 — endpoint producer and consumer contract, with independent boundary review.
- [ ] M2 — allowlisted registered-document adapter after #405/#759 integration, detailed slice approved before code.
- [ ] M3 — integrated real editor/phone acceptance and whole-issue qualification.

- [ ]

## Log
### 2026-10-22 — claimed and planned producer prerequisite
Card checked unowned, claimed by pair:6. #759 pushed branch preserved; switched to resting main-slot6 before start-plan. Catch-up twice generated the same unrelated merge-check.yml toolchain grep fix; each exact workflow-only diff was retained in a named stash, not mixed into #758. Third start-plan succeeded from current main; no stacked branch. Current code remains untouched.

Read remote-session project/design and existing sessionsync namespace/registry/process seams. Consumer receipts fc5fd55f, 8b7dcc76 and 66ccd77d pin constant endpoint/superseded RPC callbacks, namespace projection, monotone AttachID, bounded batch exit, no polling/auto-takeover and established-channel fencing. Draft durable plan awaits final consolidated contract confirmation and hub approval. No change-code, estimate, native run or implementation yet.

### 2026-10-20
