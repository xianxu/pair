---
id: 000495
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '59b078705f661d2dc4becd3878160b022ee9db7b' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T12:59:16-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
---

# broadcast: a one-time code locks every cast link (terminal LIVE, web view, web drive); delivered out of band

## Problem

A cast link (the terminal broadcast's LIVE link, the web cast's view link, its drive link) is a bearer secret in a URL: it ends up in chat logs and browser history, and chat apps fetch it to build link previews. Anyone holding the link sees the cast (and, with the drive link, can ask for control of the operator's browser). The operator (10-12) wants a one-time code delivered out of band (said aloud on the call) as a second factor.

## Spec

Decided with the operator, 10-12:
1. **Locked first:** a cast link opens a lock screen asking for the code; it combines the code with the name prompt #470's viewer already shows on a viewer's first note, asking both at once.
2. **The server enforces it:** entering the right code issues a session token; without it the server sends nothing (no record or event stream, no assets, no note/stroke/drive routes). The lock screen is never just a cover over streaming content.
3. **`URL#NNNN`:** the code may ride in the URL fragment for convenience (browsers never send fragments; link-preview bots don't run the page's script); the page reads it and submits it like typed entry.
4. **Guessing:** about 5 wrong tries per client, then the code rotates; an alert in Couch when that happens. 4 digits for view links; consider 6 for the drive link (it controls the operator's browser).
5. **Separate codes per link:** view and drive codes differ; the view code never grants driving. The terminal LIVE link has its own.
6. **Rotation:** random per new cast; clicking the code in Couch's tab bar rotates it, which cuts off everyone holding the old one (sessions revoked); the operator announces the new one out of band.
7. **Display:** the code beside its cell: `LIVE ⏸ 4821`, `Chrome ⏸ 4821` (the drive code in the drive menu, or on Alt+right-click copy). Note: the tab bar is itself what LIVE casts, so terminal viewers can see the code; acceptable (they're already viewers), document it.
8. Atlas: the threat model in one paragraph (what the code protects against: leaked links, previews, history; what it doesn't: a viewer who already has the code).

## Done when

- Every cast link is locked until the right code; the server refuses streams and routes without a session; wrong-code limits and rotation work, with the Couch alert; view and drive codes are separate; clicking the code rotates it and disconnects existing viewers.
- Tests for each guard (including a direct stream request without a session); the operator smoke-tested it on desktop and iPad.

## Plan

Durable plan: `workshop/plans/000495-broadcast-a-one-time-code-locks-every-cast-link-terminal-live-web-view-web-drive-delivered-out-of-band-plan.md` (decisions D1–D6 and their answers are under Revisions).

- [ ] M1 — `castlock` core + Host gate (session seam) + lock page (code + name) + unlock route; terminal LIVE and pointer links locked; revoke ends live streams; every route refuses without a session (direct `GET events` included).
- [ ] M2 — web cast view and drive locks (6-digit drive); lock-page name replaces #470's note-dialog prompt; terminal notes gain the author; headless-Chrome tests unlock first.
- [ ] M3 — Couch: operator-only presenter layer (never tapped); hover tooltip with codes, Alt+click Codes menu with per-link rotate and "rotate all", guess-rotation alert, the three-flash on rotation — all in that layer; atlas threat model; follow-up issue for replay links; operator smoke test on desktop and iPad.

## Log

### 2026-10-12
