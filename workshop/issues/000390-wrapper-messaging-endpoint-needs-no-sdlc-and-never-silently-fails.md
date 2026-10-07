---
id: 000390
status: open
deps: []
github_issue:
created: 2026-10-04
updated: 2026-10-04
estimate_hours:
card_mirror: 'f4ad78619250953cf395791c86c69760e673fe51' # card fields mirrored from issue-cards; edit via sdlc
---

# Wrapper messaging endpoint needs no sdlc and never silently fails

## Problem

A `pair wrap` wrapper sets up its Couch messaging endpoint once, at startup, and
gives up silently on any error. Before serving anything it runs `<agent>
--version` (2s limit), resolves its slot address by running `sdlc workspace
--json` (`couchcore.NewOSSlotCatalog(...).ResolveWorkspace`), and reads its process
identity (`wrapcmd/peer_runtime.go`, `startPeerRuntime`). Any failure skips the
whole runtime for the agent's life. Observed 2026-10-04 in pair#367's smoke test:
`pair:0`'s codex wrapper started at 10:24:29 during an `sdlc move` branch switch,
registered nothing (no endpoint socket, no registry session), while the eight
other wrappers reconnected to a restarted Couch fine (the registry session
already reconnects with backoff). The slot showed live in `couch --list` yet could
not receive `--send-to` messages; Alt+n (relaunch) fixed it.

It also breaks the layer rule: `pair wrap` is a Pair process, and Pair must work
without ariadne/sdlc (a vanilla coding-agent setup). Only Couch may depend on
sdlc. The slot address is Couch's concept, derived from Couch's own layout
(`couchcore/slot.go`) and records, never from `sdlc workspace`.

## Spec

Captured for operator review; no implementation is authorized by this issue
creation.

- **Serve first, from the environment alone.** The wrapper serves its endpoint at
  a deterministic per-thread path computed only from values both sides already
  hold (the thread's scope and tag from `COUCH_THREAD_SCOPE`/`COUCH_THREAD_TAG`,
  plus the incarnation identity), before any subprocess. No `sdlc`, no slot
  address, no agent `--version` on the critical path (the version is
  informational and can be reported later).
- **Couch dials and verifies.** Couch derives the endpoint from its own thread
  record and checks identity in a handshake (wrapper pid, launch nonce, agent)
  against its records before handing over a message, keeping the property that
  only the current incarnation receives (a stale socket fails to connect or fails
  the check, and shows as not receivable). The push session for activity/idle
  events (#365) stays as an optimization, not the delivery precondition.
- **Never silent.** Anything that can still fail retries with backoff, and Couch
  shows "live, messaging not connected" in `--actors`, the switcher and pair#367's
  recovery report instead of leaving the gap invisible until a send fails.
- **Dependency audit:** remove every Pair-layer call to `sdlc`/`weave` (`pair
  wrap`, `pair term`, launcher, nvim); Couch-layer calls stay and stay explicit.

## Done when

- A wrapper started while `sdlc` is unavailable or failing (stateful fake / PATH
  without sdlc) still serves its endpoint and receives messages; pair works with
  no ariadne installed.
- A stale endpoint (crashed or replaced wrapper) is refused by Couch's identity
  check; only the current incarnation receives (tested with a replaced
  incarnation and a leftover socket file).
- A transient setup failure recovers without relaunch, and the unrecovered state
  is visible as "messaging not connected" in `--actors` and the switcher.
- The audit finds no Pair-layer `sdlc`/`weave` invocation, enforced by a test.

## Plan

Implementation plan to be designed after issue claim and start-plan.

## Log

### 2026-10-04

Filed from pair#367's smoke test and operator discussion: the registration
failed once and silently, and the operator set the layer rule (couch → sdlc only;
pair independent of ariadne).
