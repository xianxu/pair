---
id: 000121
status: working
deps: []
github_issue:
created: 2026-07-26
updated: 2026-07-26
estimate_hours:
started: 2026-07-26T11:20:10-07:00
---

# Remote Pair control relay

## Problem

Pair sessions are useful from a tablet when the local workstation is already
running the coding environment, but Pair currently requires local terminal access
for session discovery and lifecycle gestures. The immediate pain is context
refresh: when a Claude session is near the context limit, the user wants a simple
remote web control to see the active Pair session and trigger the existing
continuation/restart flow without being at the keyboard.

The first slice should prove remote control without widening the trust boundary:
the local machine remains authoritative for repos, files, zellij, Pair sidecars,
and lifecycle commands. A small Oracle-hosted service is only a dumb relay that
lets a browser reach the local Pair daemon through an outbound connection.

## Spec

- Add a local Pair remote-control daemon that runs inside the local Pair trust
  boundary and exposes a small, explicit command surface for remote clients.
- Add a minimal Go relay service suitable for Oracle free tier. The relay accepts
  an outbound long-poll connection from the local daemon and authenticated browser
  requests from a paired device. It does not read repos, execute shell commands,
  inspect Pair sidecars directly, or persist sensitive state.
- Implement a pairing/authentication model for one or more trusted devices:
  short-lived pairing secret, durable per-device credential, command allowlist,
  and auditable command records.
- Implement a remote session list using existing Pair identity/state sources
  rather than a second session database (`ARCH-DRY`): repo scope, display tag,
  agent, live/detached/exited state, last activity when available, and current
  context signal when available.
- Implement one remote lifecycle action: refresh a session from an existing or
  just-created continuation using Pair's existing restart/continue machinery.
  The daemon may invoke Pair's public lifecycle entrypoints or extracted shared
  launcher logic, but it must not create a parallel restart protocol.
- Keep v1 out of scope for terminal streaming, artifact browsing, arbitrary file
  reads, arbitrary shell commands, and annotation.
- Keep core decisions testable as pure functions (`ARCH-PURE`): request
  validation, session/action authorization, relay envelopes, and command routing
  should be unit-testable without network, zellij, or the filesystem.

## Done when

- A local daemon can connect outbound to a relay and publish the current Pair
  session list to an authenticated browser.
- The browser can trigger the approved refresh/restart action for a selected
  session, and the local daemon routes it through the existing Pair lifecycle.
- Relay and daemon reject unauthenticated requests, stale commands, unknown
  actions, mismatched device/session IDs, and replayed command IDs.
- The relay stores no repo contents, file contents, transcripts, scrollback, or
  shell output.
- Tests cover auth envelope validation, session-list projection, command
  allowlisting, stale/replayed command rejection, and the daemon's lifecycle
  command dispatch seam.

## Plan

- [ ] Define the daemon/relay/browser protocol envelope and auth model.
- [ ] Add pure session-list and command-authorization cores that derive from
  existing Pair state.
- [ ] Add the local daemon IO shell and lifecycle dispatch seam.
- [ ] Add the Oracle-friendly relay IO shell with long-poll transport.
- [ ] Add a minimal web dashboard for session list + refresh action.
- [ ] Add unit/integration tests and a local smoke path that does not require
  Oracle deployment.

## Log

### 2026-07-26

Seeded from remote Pair brainstorming. First slice is intentionally remote
control only: a dead-simple proxy plus local daemon so an iPad can refresh a
Claude session near context-full without exposing local files or terminal output.
