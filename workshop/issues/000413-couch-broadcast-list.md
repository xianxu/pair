---
id: 000413
status: open
deps: []
github_issue:
created: 2026-10-08
updated: 2026-10-08
estimate_hours:
card_mirror: '8fc1040079f19d6019f72679c0cffea18fbfa6cd' # card fields mirrored from issue-cards; edit via sdlc
---

# couch --broadcast-list: print the current broadcast session and its viewer count

## Problem

A couch broadcast (#395, `atlas/broadcast.md`) streams the screen view-only to
browser viewers through a token link. Ctrl+Alt+b starts and stops it, and the
status row shows LIVE. From a shell, though, nothing says whether a broadcast is
running or how many people are watching. To find out, the
operator has to look at the couch screen itself.

## Spec

`couch --broadcast-list` asks the running couch and prints its current
broadcast session, if any, and the number of connected viewers.

- **Routing.** It is a live-owner query, like `--actors` (`couchcmd/cli.go:92`):
  answered by the running console over its socket. With no couch running it
  says so and exits non-zero. With no broadcast running it says
  "no broadcast" and exits zero.
- **Output.** One line per session (today there is at most one):
  - state (starting, live, ending);
  - since when;
  - mode (tunnel or local-only);
  - the viewer count.

  `--json` gives the same fields for scripts.
- **The link is never printed** (operator decision, 2026-10-08: "don't show
  link for privacy reasons"). The token in the path is the broadcast's only
  access control, so the command prints neither the link nor the token, in
  text or `--json`, and has no flag to reveal it. The link stays where it is
  shown today: in couch, to the operator who started the broadcast.
- **Viewer count.** `broadcast.Hub` holds its subscribers inside its run
  goroutine and exposes no count. Add a read that answers through the hub's own
  loop (`h.do`), or a counter the hub updates as viewers join and leave, so the
  answer comes from memory with no per-viewer probe on the request path (see
  the "deterministic commands answer from memory" rule). Count viewers
  currently subscribed, including ones marked for resync.
- **No new trust boundary.** The socket is already owner-only; the query is
  read-only and starts, stops or changes nothing.

## Done when

- `couch --broadcast-list` prints the session state, start time, mode and live
  viewer count from the running couch, and "no broadcast" when there is none.
  `--json` prints the same fields.
- The viewer count is read from the hub's own state. A test joins and leaves
  viewers and checks the count, including a viewer that is resyncing.
- The output never contains the link or its token, in text or `--json`, and
  there is no flag to reveal it. A test checks both formats against a live
  session's token.
- With no couch running, the command refuses with a clear message and a
  non-zero exit.
- `atlas/broadcast.md` documents the command.

## Plan

- [ ] Hub viewer count read (through the hub loop), with a test.
- [ ] Console and session expose a broadcast snapshot to the socket handler.
- [ ] `--broadcast-list` CLI routing, text and `--json` output, and the
      no-couch / no-broadcast cases.
- [ ] Atlas, verification, close.

## Log

### 2026-10-08

Filed at the operator's request: "make a task to have a couch command, `couch
--broadcast-list` to print out current session, and how many viewers are
there." The details stay local until the operator asks to publish them.
