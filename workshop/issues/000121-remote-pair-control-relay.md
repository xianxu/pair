---
id: 000121
status: working
deps: []
github_issue:
created: 2026-07-26
updated: 2026-07-26
estimate_hours: 24.52
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
- Implement `pair remote login` as the local enrollment flow. It opens the
  browser for the same OAuth/OIDC provider used by the relay, verifies the
  authenticated issuer+subject locally, registers the local daemon to that
  account, creates a daemon keypair, and stores the daemon private key only on
  the local machine.
- Implement a pairing/authentication model for one or more trusted browser
  devices: short-lived pairing secret, durable per-device credential, command
  allowlist, and auditable command records.
- Use OAuth/OIDC for relay admission and account ownership, but keep Pair-local
  grants as the final authority. The relay verifies that a logged-in user may
  reach a registered daemon; the daemon verifies that the signed device command
  is allowed for this machine and session.
- Bind identity to OIDC issuer+subject, not mutable email alone. Email may be
  displayed but is not the durable authorization key.
- Condition authorization on request class/origin:
  `local-cli` can configure grants, revoke devices, rotate keys, and start the
  daemon; `relay-browser` can only use capabilities granted locally; relay-side
  components may route envelopes but never grant Pair actions; unknown origins
  are denied.
- Provide an account-level nuclear reset: from an authenticated relay session
  for the same OAuth/OIDC issuer+subject, the user can revoke every browser
  session, paired browser credential, daemon registration, pending command, and
  refresh token known to the relay for that account. The relay records a
  monotonically increasing account revocation epoch; local daemons must compare
  it on every successful relay contact and disable remote access until
  `pair remote login` re-enrolls locally. This is Pair's own revocation plane;
  Google/OIDC logout is not enough because it cannot erase Pair-local keys on
  offline machines.
- Implement explicit v1 capabilities: `session.list`, `session.new`, and
  `session.renew`. `session.new` maps to the fresh-session restart gesture
  (`Alt+Shift+N`): keep the Pair tag/workbench identity, drop native agent resume
  config, and start a clean agent session. `session.renew` maps to the context
  refresh gesture (`Alt+Shift+C`) but owns the whole transaction locally: run the
  continuation writer, verify the continuation file, restart fresh, seed the new
  session with a fixed "read this continuation and continue" first message, and
  send it.
- Implement a remote session list using existing Pair identity/state sources
  rather than a second session database (`ARCH-DRY`): repo scope, display tag,
  agent, live/detached/exited state, last activity when available, and current
  context signal when available.
- Implement remote lifecycle actions using Pair's existing restart/continue
  machinery. The daemon may invoke Pair's public lifecycle entrypoints or
  extracted shared launcher logic, but it must not create a parallel restart
  protocol (`ARCH-DRY`).
- Treat `session.renew` as more privileged than `session.new`: the browser sends
  intent only, not arbitrary prompt text. The daemon generates the first message
  locally from a fixed template and a verified continuation path under the
  expected repo continuation directory.
- Before remote control ships, improve the local Alt+Shift+C compaction flow into
  the exact local primitive `session.renew` needs: continuation generation,
  verified continuation-backed fresh restart, locally generated first prompt, and
  at-most-once automatic send in the new session. If Pair cannot prove the send
  is safe after a crash or restart, it must leave the seeded draft visible and
  ask the user to send manually rather than risk duplicate agent input. Manual
  `pair continue <slug>` should keep the current inspect/edit-before-send
  behavior.
- Keep v1 out of scope for terminal streaming, artifact browsing, arbitrary file
  reads, arbitrary shell commands, and annotation.
- Keep core decisions testable as pure functions (`ARCH-PURE`): request
  validation, session/action authorization, relay envelopes, and command routing
  should be unit-testable without network, zellij, or the filesystem.

## Done when

- A local daemon can connect outbound to a relay and publish the current Pair
  session list to an authenticated browser.
- `pair remote login` authenticates the local daemon with the same OAuth/OIDC
  identity the relay uses and stores a local daemon keypair.
- Local authorization grants can allow or deny `session.list`, `session.new`, and
  `session.renew` separately for a paired browser device.
- An authenticated relay account can trigger a nuclear reset that logs out all
  Pair web sessions for that OAuth/OIDC subject, revokes all relay-side device
  and daemon registrations, clears pending commands, bumps the account
  revocation epoch, and causes every subsequently reconnecting local daemon to
  disable remote access until local re-login.
- The browser can trigger approved `session.new` and `session.renew` actions for
  a selected session, and the local daemon routes both through the existing Pair
  lifecycle.
- Relay and daemon reject unauthenticated requests, stale commands, unknown
  actions, mismatched device/session IDs, and replayed command IDs.
- The daemon rejects remote prompt text for `session.renew`; the sent first
  message is generated locally from a fixed template and verified continuation
  file.
- The relay stores no repo contents, file contents, transcripts, scrollback, or
  shell output.
- Tests cover auth envelope validation, session-list projection, command
  allowlisting, local grant decisions by request class, stale/replayed command
  rejection, and the daemon's lifecycle command dispatch seam.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. `sdlc estimate-source` reports the calibration
source as stale, so the number is provisional but uses the required method.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.80 impl=0.08
item: lua-neovim design=0.40 impl=0.40
item: smaller-go-module design=0.06 impl=0.16
item: cross-cutting-refactor design=0.10 impl=0.20
item: greenfield-go-module design=0.40 impl=0.32
item: greenfield-go-module design=0.40 impl=0.32
item: greenfield-go-module design=0.40 impl=0.32
item: api-integration design=0.60 impl=0.60
item: greenfield-service design=3.00 impl=3.20
item: greenfield-service design=3.00 impl=3.20
item: tui-screen design=0.40 impl=0.40
item: api-integration design=0.40 impl=0.40
item: cross-cutting-refactor design=0.80 impl=1.12
item: atlas-docs design=0.25 impl=0.20
item: milestone-review design=0.08 impl=0.12
item: milestone-review design=0.08 impl=0.12
item: milestone-review design=0.08 impl=0.12
item: milestone-review design=0.08 impl=0.12
item: milestone-review design=0.08 impl=0.12
design-buffer: 0.15
total: 24.52
```

The two service-scale items are distinct: the local daemon/control-plane
orchestration and the Oracle-friendly relay/web service. The three greenfield
modules cover the auth/grant core, signed envelopes/replay, and relay mailbox /
session projections.

## Plan

- [ ] M1 — Local renew substrate: improve Alt+Shift+C so continuation compaction
  restarts fresh, seeds the verified continuation prompt, and sends it at most
  once in the new session, with visible manual recovery if the one-shot state is
  ambiguous after a crash. Keep manual `pair continue <slug>` as seed-only.
- [ ] M2 — Authentication and grants: design and implement `pair remote login`,
  OAuth/OIDC issuer+subject binding, daemon key registration, paired browser
  device credentials, browser pairing secrets, request-class-aware local grants,
  local revocation, and signed command envelope validation.
- [ ] M3 — Relay and session list: implement the Oracle-friendly relay as a
  dumb authenticated mailbox, local daemon long-poll connection, authenticated
  web session list, short TTLs, replay rejection, durable relay security
  metadata, account-wide relay nuclear reset semantics, reset tombstones, and
  audit records. Reuse existing Pair session identity/state sources (`ARCH-DRY`).
- [ ] M4 — Remote lifecycle actions: implement locally authorized `session.new`
  and transactional `session.renew` over the relay. `session.new` maps to the
  fresh-session restart path; `session.renew` runs continuation generation,
  verifies the continuation, restarts fresh, and sends the locally generated
  continuation prompt as the new session's first message.
- [ ] M5 — Hardening and smoke: add end-to-end local fake-relay tests, stale and
  replay command tests, grant downgrade/revocation tests, relay non-retention
  checks, account nuclear-reset tests, reconnect-after-reset tests, and a manual
  smoke path for using an iPad/browser against a real local Pair session.

## Log

### 2026-07-26

Seeded from remote Pair brainstorming. First slice is intentionally remote
control only: a dead-simple proxy plus local daemon so an iPad can refresh a
Claude session near context-full without exposing local files or terminal output.

### 2026-07-26

Refined security model: `pair remote login` authenticates the local daemon with
the same OAuth/OIDC identity as the relay, but OAuth is admission rather than the
final Pair authority. Pair-local grants and signed device commands decide whether
`session.list`, `session.new`, or `session.renew` can run. `session.renew` is a
single local transaction: continuation generation, verified continuation file,
fresh restart, and locally templated first prompt send.

### 2026-07-26

Added the account-level nuclear reset requirement. Relay-side OAuth login can
initiate "log all Pair sessions out" for the same issuer+subject, but the durable
security property is Pair-owned: the relay revokes every known web session,
device credential, daemon registration, pending command, and token for that
account, bumps a revocation epoch, and local daemons disable remote access on
next contact until re-enrolled with `pair remote login`.

### 2026-07-26

Added the local renew substrate milestone and durable implementation plan at
`workshop/plans/000121-remote-pair-control-relay-plan.md`. The remote
`session.renew` command will call a local behavior first proven through
Alt+Shift+C, instead of teaching the remote layer how to synthesize prompts.

### 2026-07-26

Fresh plan review found six issues: reset state was not durable, reset could
hide the revocation epoch from revoked daemons, browser pairing was
underspecified, M2/M3 reset scope was misaligned, exactly-once auto-send was not
crash-coherent, and continuation validation was not named as a pure entity.
Revised the issue and durable plan to make relay security metadata durable in
v1, add reset tombstones, add explicit browser pairing, move reset semantics to
the relay/session-list milestone, define auto-send as at-most-once with visible
manual recovery, and add `VerifiedContinuation`.
