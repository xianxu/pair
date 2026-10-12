---
id: 000495
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '4099207e5613f51e822ecb757cfa54b04f6a225c' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ]

## Log

### 2026-10-12
