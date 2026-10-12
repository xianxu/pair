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
4. **Guessing:** one budget per code (about 8 wrong tries across all clients; a client can't be identified behind the tunnel), then the code rotates and Couch alerts; existing sessions survive a guess-rotation. Unlock attempts are rate-limited per lock. 4 digits for view and pointer links, 6 for the drive link.
5. **Separate codes per link:** view and drive codes differ; the view code never grants driving. The terminal LIVE link has its own.
6. **Rotation:** random per new cast; clicking the code in Couch's tab bar rotates it, which cuts off everyone holding the old one (sessions revoked); the operator announces the new one out of band.
7. **Display:** no codes inline in the tab bar. Hovering a cast cell shows its codes as a tooltip (`view 4821 · pointer 7730` on LIVE, `view 5512 · drive 903114` on Chrome); Alt+click on the cell opens a Codes menu (one rotate item per link, plus rotate all); on rotation the new code flashes three times. All of these draw in an operator-only presenter layer that the LIVE broadcast never carries. One lock page asks for the code and the viewer's name for every link; terminal notes gain the author.
8. Atlas: the threat model in one paragraph (what the code protects against: leaked links, previews, history; what it doesn't: a viewer who already has the code).

## Done when

- Every cast link (terminal LIVE, terminal pointer, web view, web drive) is locked until the right code; the server refuses streams and routes without a session; the per-code wrong-try budget rotates with a Couch alert; codes are separate per link; rotating from the Codes menu disconnects that link's viewers; codes show only in the operator-only layer (tooltip, menu, flash), never in the broadcast frame; the lock page's name attributes notes on every link. Replay links are a filed follow-up.
- Tests for each guard (including a direct stream request without a session); the operator smoke-tested it on desktop and iPad.

## Plan

Durable plan: `workshop/plans/000495-broadcast-a-one-time-code-locks-every-cast-link-terminal-live-web-view-web-drive-delivered-out-of-band-plan.md` (decisions D1–D6 and their answers are under Revisions).

- [ ] M1 — `castlock` core + Host gate (session seam) + lock page (code + name) + unlock route; terminal LIVE and pointer links locked; revoke ends live streams; every route refuses without a session (direct `GET events` included).
- [ ] M2 — web cast view and drive locks (6-digit drive); lock-page name replaces #470's note-dialog prompt; terminal notes gain the author; headless-Chrome tests unlock first.
- [ ] M3 — Couch: operator-only presenter layer (never tapped); hover tooltip with codes, Alt+click Codes menu with per-link rotate and "rotate all", guess-rotation alert, the three-flash on rotation — all in that layer; atlas threat model; follow-up issue for replay links; operator smoke test on desktop and iPad.

## Log

### 2026-10-12
