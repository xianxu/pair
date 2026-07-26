# Remote Pair Control Relay Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a secure remote Pair control surface that lets an authenticated tablet/browser list sessions and trigger local Pair session renewal without exposing local files, terminal output, or arbitrary command execution.

**Architecture:** The local daemon is the authority; the Oracle-hosted relay is an authenticated, short-lived mailbox. OAuth/OIDC admits users to the relay and enrolls the local daemon, while Pair-local grants and signed command envelopes authorize each local action. The first implementation milestone upgrades the existing Alt+Shift+C compaction flow into the local primitive that future remote `session.renew` calls invoke (`ARCH-DRY`, `ARCH-PURE`, `ARCH-PURPOSE`).

**Tech Stack:** Go standard library HTTP/crypto, existing `cmd/internal/launcher`, `cmd/internal/continuationcmd`, existing Pair dispatcher, Neovim Lua startup hooks, local fake relay for tests.

---

## Core Concepts

### Pure Entities

| Name | Lives in | Status |
|------|----------|--------|
| `ContinuationPrompt` | `cmd/internal/launcher/renew.go` | new |
| `VerifiedContinuation` | `cmd/internal/launcher/renew.go` | new |
| `RenewIntent` | `cmd/internal/launcher/renew.go` | new |
| `RemoteIdentity` | `cmd/internal/remotectl/identity.go` | new |
| `RemoteGrant` | `cmd/internal/remotectl/grants.go` | new |
| `CommandEnvelope` | `cmd/internal/remotectl/envelope.go` | new |
| `RelayMailbox` | `cmd/internal/remotectl/mailbox.go` | new |
| `RelayStoreRecord` | `cmd/internal/remotectl/relay_store.go` | new |
| `RemoteSession` | `cmd/internal/remotectl/session.go` | new |

**ContinuationPrompt** — renders the fixed first message used after a continuation-backed renewal.
- **Relationships:** 1:1 with `VerifiedContinuation`; owned by launcher renewal code and later remote `session.renew`.
- **DRY rationale:** The current launcher writes a hardcoded draft string in `createflow.go`; renewal, remote control, and tests should share one renderer.
- **Future extensions:** If terminal streaming later offers editable first prompts, this remains the safe default renderer and widens only with explicit local grants.

**VerifiedContinuation** — validates that a continuation doc is local, expected, and structurally usable before any prompt is rendered from it.
- **Relationships:** 1:1 with a continuation file under a repo's `workshop/continuation/`; consumed by `ContinuationPrompt`, `RenewIntent`, and `session.renew`.
- **DRY rationale:** The path/frontmatter/NEXT ACTION checks are a security property, so they must not be duplicated in the launcher and remote lifecycle code.
- **Future extensions:** Can add tag/session provenance when continuation frontmatter grows that metadata.

**RenewIntent** — pure decision record for whether a continuation restart should seed only, or seed and auto-send at most once.
- **Relationships:** 1:1 with a restart marker carrying a continuation slug.
- **DRY rationale:** Both Alt+Shift+C and future `session.renew` need the same semantics: continuation generation, fresh restart, fixed prompt, and at-most-once one-shot send.
- **Future extensions:** Can add progress phases for remote UI without changing launcher restart mechanics.

**RemoteIdentity** — stable authenticated identity `{issuer, subject, display}`.
- **Relationships:** 1:N with daemon registrations and browser devices.
- **DRY rationale:** Keeps OAuth subject binding centralized; avoids mutable email as an authorization key.
- **Future extensions:** Multiple IdPs can be supported by adding issuers, not by changing grants.

**RemoteGrant** — local authorization matrix for request class, device, daemon, and capability.
- **Relationships:** N:N between paired browser devices and capabilities (`session.list`, `session.new`, `session.renew`).
- **DRY rationale:** Relay admission and Pair-local authorization stay separate rather than duplicated across handlers.
- **Future extensions:** Terminal streaming and artifact access become new capabilities, not new auth models.

**CommandEnvelope** — signed command request with command ID, timestamp, expiry, issuer+subject, device ID, daemon ID, capability, target session, and body hash.
- **Relationships:** 1:1 with a local command execution attempt.
- **DRY rationale:** Replay, staleness, and signature validation are one shared path for every remote command.
- **Future extensions:** Can become DPoP-like proof-of-possession later while preserving the same command model.

**RelayMailbox** — pure model of daemon registration, queued commands, response slots, TTL expiry, dedupe, and revocation epoch.
- **Relationships:** 1:N account to daemons; 1:N daemon to pending commands.
- **DRY rationale:** The relay's behavior is deliberately dumb and testable independent of HTTP.
- **Future extensions:** Long-poll can be replaced by WebSocket/SSE while the mailbox semantics stay the same.

**RelayStoreRecord** — durable relay metadata records for account revocation epoch, daemon/device registrations, invalidated sessions/tokens, command tombstones, and audit records.
- **Relationships:** Owned by the relay; keyed by issuer+subject, daemon ID, device ID, command ID, and revocation epoch.
- **DRY rationale:** Nuclear reset must survive relay restart; the relay still stores no repo content, transcripts, scrollback, or shell output.
- **Future extensions:** Starts as SQLite/file-backed storage and can move to a managed store without changing the pure mailbox core.

**RemoteSession** — browser-facing projection of existing Pair session state.
- **Relationships:** Derived from existing launcher list/session identity surfaces.
- **DRY rationale:** Reuses Pair's session identity model instead of creating a remote session database (`ARCH-DRY`).
- **Future extensions:** Terminal stream and artifact links can attach to the projection later.

### Integration Points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `pair remote login` | `cmd/internal/remotecmd/login.go` | new | OAuth/OIDC browser login + local config |
| `pair remote daemon` | `cmd/internal/remotecmd/daemon.go` | new | local long-poll loop, session scanner, lifecycle dispatch |
| `pair remote relay` | `cmd/internal/remotecmd/relay.go` | new | Oracle-friendly HTTP server |
| `RemoteConfigStore` | `cmd/internal/remotectl/store.go` | new | `$PAIR_DATA_DIR/remote/*.json` local state |
| `RelayDurableStore` | `cmd/internal/remotecmd/relay_store.go` | new | durable relay metadata |
| `LifecycleRenewDispatch` | `cmd/internal/remotecmd/lifecycle.go` | new | existing launcher restart/continue commands |
| `NvimRenewSendHook` | `nvim/init.lua` | modified | one-shot draft send after continuation restart |
| `FakeRelayServer` | `cmd/internal/remotectl/fakerelay_test.go` | new | process-level HTTP fake for daemon/browser tests |

**`pair remote login`** — local enrollment command. It starts a localhost callback, opens the browser to the configured OIDC provider with PKCE, verifies the returned issuer+subject, registers a daemon public key with the relay, and writes local grants.
- **Injected into:** `RemoteConfigStore`; pure OIDC state/subject validation helpers.
- **Future extensions:** Device management UI and multiple account support.

**`pair remote daemon`** — local authority. It connects outbound to the relay, publishes session projections, receives command envelopes, verifies local grants/signatures/revocation epoch, and dispatches approved lifecycle actions.
- **Injected into:** `CommandEnvelope`, `RemoteGrant`, `RemoteSession`, `LifecycleRenewDispatch`.
- **Future extensions:** Terminal streaming and artifact browser handlers.

**`pair remote relay`** — internet-facing mailbox. It verifies relay OAuth sessions, enforces account ownership, queues short-lived envelopes, carries responses, rate-limits, and owns account nuclear reset state.
- **Injected into:** `RelayMailbox`.
- **Future extensions:** Replace long-poll with WebSocket/SSE without changing command semantics.

**RemoteConfigStore** — local Pair remote state: daemon key, enrolled identity, relay URL, paired browser device keys, grants, audit log, seen command IDs, and last observed revocation epoch.
- **Injected into:** login, daemon, grant checks, and reset handling.
- **Future extensions:** Migrate secret material to OS keychain.

**RelayDurableStore** — relay-side durable metadata store. V1 can use SQLite or an atomic JSONL/file store; it must persist revocation epochs and invalidation state across relay restarts.
- **Injected into:** relay HTTP shell and `RelayMailbox`.
- **Future extensions:** Move to Redis/Postgres if multi-process relay ever matters.

**LifecycleRenewDispatch** — effect seam that maps `session.new` and `session.renew` to existing Pair lifecycle entrypoints.
- **Injected into:** daemon command execution.
- **Future extensions:** Exposes progress phases to the web UI.

**NvimRenewSendHook** — draft-pane startup hook that claims a one-shot auto-send sidecar and sends the seeded continuation prompt through the existing draft send path at most once.
- **Injected into:** existing nvim initialization and draft send helpers.
- **Future extensions:** Statusline indicator while a renewal first prompt is pending.

**FakeRelayServer** — test server that speaks the relay HTTP protocol.
- **Injected into:** daemon and browser-flow integration tests.
- **Future extensions:** Reused by terminal stream/artifact issues.

## Chunk 1: Local Renew Substrate

### Task 1: Extract the continuation first-message renderer

**Files:**
- Create: `cmd/internal/launcher/renew.go`
- Test: `cmd/internal/launcher/renew_test.go`
- Modify: `cmd/internal/launcher/createflow.go`

- [ ] Write failing unit tests for `VerifyContinuation(root, docPath, raw)` covering valid doc, path outside `workshop/continuation`, path traversal, symlink escape where observable through injected path facts, missing frontmatter, wrong `type`, and missing NEXT ACTION.
- [ ] Implement `VerifiedContinuation` so callers render prompts only from a verified value.
- [ ] Write a failing unit test for `ContinuationPrompt(verified)` using `/repo/workshop/continuation/20260726T120000-demo.md`.
- [ ] Assert the exact output is `Read workshop/continuation/20260726T120000-demo.md and continue from its NEXT ACTION.\n`.
- [ ] Replace the hardcoded draft write in `createflow.go` with `ContinuationPrompt(verified)`.
- [ ] Run `go test ./cmd/internal/launcher -count=1`.

### Task 2: Carry auto-send intent through continuation restart

**Files:**
- Modify: `cmd/internal/launcher/markers.go`
- Modify: `cmd/internal/launcher/compaction.go`
- Test: `cmd/internal/launcher/compaction_test.go`
- Test: `cmd/internal/launcher/lifecycle_test.go`

- [ ] Add `AutoSend bool` to `RestartMarker`.
- [ ] Extend `serializeRestartMarker` / `parseRestartMarker` with `auto_send=1`.
- [ ] Update round-trip tests so compaction markers include `auto_send=1` only for continuation compaction.
- [ ] Make `runCompaction` write `{NewSession: true, Continue: slug, AutoSend: true}`.
- [ ] In restart re-entry, convert `AutoSend+ContinueDoc` into a one-shot auto-send state path under the scoped data dir.
- [ ] Run `go test ./cmd/internal/launcher -count=1`.

### Task 3: Add a one-shot auto-send state machine

**Files:**
- Modify: `cmd/internal/launcher/scoped_paths.go`
- Modify: `cmd/internal/launcher/runtime.go`
- Modify: `cmd/internal/launcher/createflow.go`
- Test: `cmd/internal/launcher/scoped_paths_test.go`
- Test: `cmd/internal/launcher/lifecycle_test.go`

- [ ] Add `ScopedPaths.RenewAutoSend()` returning `renew-autosend-<tag>.json`.
- [ ] Define JSON shape with state `pending|claimed|sent`, session, tag, continuation basename, prompt digest, created timestamp, claimed timestamp, and sent timestamp.
- [ ] Write `pending` atomically after seeding the draft and before launching zellij.
- [ ] Define safety semantics as **at-most-once send with visible failure**: if nvim crashes after `claimed` and before `sent`, the hook must not duplicate-send; it should leave the draft intact and surface the claimed-but-unsent state for manual action.
- [ ] Ensure non-auto-send `pair continue <slug>` still only seeds the draft.
- [ ] Run `go test ./cmd/internal/launcher -count=1`.

### Task 4: Teach draft nvim to consume auto-send once

**Files:**
- Modify: `nvim/init.lua`
- Test: `nvim/*_test.lua` if an existing test seam can cover the pure decision; otherwise add a focused headless shell test.

- [ ] Extract a small pure Lua decision helper: given sidecar JSON, current tag/session, current draft text, return send/skip/error.
- [ ] On `VimEnter`, read `renew-autosend-<tag>.json` when present.
- [ ] Validate tag/session and prompt digest against the draft content.
- [ ] Atomically transition `pending` to `claimed` before sending to avoid duplicate sends on crash/reopen.
- [ ] Transition `claimed` to `sent` after the existing send path reports success.
- [ ] Surface `claimed` on startup as a warning rather than sending again; the user can inspect/send the seeded draft manually.
- [ ] Use the existing draft send path so key mapping/queue/history behavior remains canonical (`ARCH-DRY`).
- [ ] Run `make test-lua` and the relevant terminal shortcut smoke tests.

## Chunk 2: Authentication And Grants

### Task 5: Add remote command family skeleton

**Files:**
- Modify: `cmd/internal/dispatcher/dispatcher.go`
- Modify: `cmd/pair-go/main.go`
- Test: `cmd/internal/dispatcher/dispatcher_test.go`
- Test: `cmd/pair-go/main_test.go`
- Create: `cmd/internal/remotecmd/runcli.go`

- [ ] Register `remote login`, `remote daemon`, `remote relay`, `remote devices`, `remote revoke`, `remote reset`, and `remote status` in `Families()`.
- [ ] Mark long-running or stdio routes as streaming where needed.
- [ ] Add runner stubs that parse flags and return clear "not implemented" errors until later tasks wire behavior.
- [ ] Run `go test ./cmd/internal/dispatcher ./cmd/pair-go -count=1`.

### Task 6: Implement identity and grant pure core

**Files:**
- Create: `cmd/internal/remotectl/identity.go`
- Create: `cmd/internal/remotectl/grants.go`
- Test: `cmd/internal/remotectl/identity_test.go`
- Test: `cmd/internal/remotectl/grants_test.go`

- [ ] Model `RemoteIdentity{Issuer, Subject, Display}` and validate non-empty issuer+subject.
- [ ] Model request classes: `local-cli`, `relay-browser`, `relay-daemon`, `unknown`.
- [ ] Model capabilities: `session.list`, `session.new`, `session.renew`.
- [ ] Implement grant decisions: local-cli may configure; relay-browser may only use locally granted capabilities; relay/unknown cannot grant Pair actions.
- [ ] Add tests for mutable email not being authorization identity.
- [ ] Run `go test ./cmd/internal/remotectl -count=1`.

### Task 7: Implement signed command envelopes

**Files:**
- Create: `cmd/internal/remotectl/envelope.go`
- Test: `cmd/internal/remotectl/envelope_test.go`

- [ ] Define envelope fields: command ID, issued-at, expires-at, issuer, subject, daemon ID, device ID, capability, target session, body hash, signature.
- [ ] Use Ed25519 for v1 device signatures.
- [ ] Verify signature over canonical JSON or a stable binary/string canonical form.
- [ ] Reject expired, future-issued, mismatched target, mismatched body hash, unknown capability, and replayed command IDs.
- [ ] Keep replay tracking behind an injected `SeenCommandStore` interface.
- [ ] Run `go test ./cmd/internal/remotectl -count=1`.

### Task 8: Implement local remote config store

**Files:**
- Create: `cmd/internal/remotectl/store.go`
- Test: `cmd/internal/remotectl/store_test.go`
- Modify: `cmd/internal/launcher/scoped_paths.go` only if shared path helpers are needed.

- [ ] Store remote state under `$PAIR_DATA_DIR/remote/`.
- [ ] Write daemon identity/key metadata atomically with `0600` permissions for private material.
- [ ] Store device grants, seen command IDs, audit records, and last relay revocation epoch.
- [ ] Add tests for atomic writes, permission mode where portable, corrupt JSON rejection, and no silent grant widening.
- [ ] Run `go test ./cmd/internal/remotectl -count=1`.

### Task 9: Implement browser device pairing

**Files:**
- Create: `cmd/internal/remotectl/pairing.go`
- Test: `cmd/internal/remotectl/pairing_test.go`
- Modify: `cmd/internal/remotecmd/login.go`
- Modify: `cmd/internal/remotecmd/relay.go`
- Test: `cmd/internal/remotecmd/login_test.go`

- [ ] Add local CLI generation of a short-lived pairing secret bound to issuer+subject, daemon ID, expiry, nonce, and requested capabilities.
- [ ] Add browser redemption under the same relay OAuth account; reject wrong issuer+subject, expired secrets, replayed secrets, and daemon mismatch.
- [ ] Bind browser device public key to daemon/account after redemption.
- [ ] Persist local grants only after local CLI approval; remote browser cannot widen grants.
- [ ] Write audit records for pairing created, redeemed, expired, rejected, and revoked.
- [ ] Run `go test ./cmd/internal/remotectl ./cmd/internal/remotecmd -count=1`.

### Task 10: Implement `pair remote login`

**Files:**
- Modify: `cmd/internal/remotecmd/login.go`
- Test: `cmd/internal/remotecmd/login_test.go`

- [ ] Implement OIDC config flags/env: relay URL, issuer URL, client ID.
- [ ] Use Authorization Code + PKCE with exact redirect matching to a localhost callback.
- [ ] Verify ID token issuer+subject and nonce.
- [ ] Generate daemon keypair and register daemon public key with relay.
- [ ] Present local capability choices for v1 grants and persist them.
- [ ] Add tests with a fake OIDC provider/relay process.
- [ ] Run `go test ./cmd/internal/remotecmd ./cmd/internal/remotectl -count=1`.

## Chunk 3: Relay And Session List

### Task 11: Implement durable relay store and mailbox core

**Files:**
- Create: `cmd/internal/remotectl/mailbox.go`
- Create: `cmd/internal/remotectl/relay_store.go`
- Test: `cmd/internal/remotectl/mailbox_test.go`
- Test: `cmd/internal/remotectl/relay_store_test.go`

- [ ] Model daemon registrations, browser sessions, pending commands, command responses, TTL expiry, dedupe, and revocation epoch.
- [ ] Persist relay metadata needed for security across relay restarts: account revocation epoch, daemon registrations, device registrations, token/session invalidation state, command tombstones/dedupe, and audit records.
- [ ] Ensure nuclear reset removes browser sessions, device registrations, daemon registrations, pending commands, responses, and refresh tokens known to the relay.
- [ ] Ensure nuclear reset bumps the account revocation epoch monotonically.
- [ ] Ensure reset leaves a durable tombstone that can be returned to a previously registered daemon even after its active registration is revoked.
- [ ] Run `go test ./cmd/internal/remotectl -count=1`.

### Task 12: Implement relay HTTP shell

**Files:**
- Modify: `cmd/internal/remotecmd/relay.go`
- Create: `cmd/internal/remotecmd/relay_test.go`

- [ ] Expose endpoints for login/session validation, daemon register, daemon poll, browser command enqueue, command response fetch, session snapshot fetch, and nuclear reset.
- [ ] Verify OAuth/OIDC session on browser endpoints.
- [ ] Verify daemon signatures on daemon endpoints.
- [ ] When a reset daemon polls with a revoked registration, return a signed reset tombstone/auth-failure response containing the current account revocation epoch so the local daemon can disable remote access.
- [ ] Add account/daemon/device rate limits and short TTLs.
- [ ] Ensure responses do not persist repo contents, transcripts, scrollback, or shell output.
- [ ] Run `go test ./cmd/internal/remotecmd ./cmd/internal/remotectl -count=1`.

### Task 13: Implement session projection

**Files:**
- Create: `cmd/internal/remotectl/session.go`
- Test: `cmd/internal/remotectl/session_test.go`
- Modify: `cmd/internal/launcher/list.go` only if an exported pure helper is needed.
- Modify: `cmd/internal/remotecmd/daemon.go`

- [ ] Derive browser-facing `RemoteSession` from existing Pair session/list state.
- [ ] Include repo display, tag, agent, live/detached/exited state, last activity when available, and context signal when available.
- [ ] Exclude raw paths, transcript paths, scrollback contents, and local absolute repo roots from the browser projection unless explicitly approved by the spec.
- [ ] Run `go test ./cmd/internal/remotectl ./cmd/internal/launcher -count=1`.

### Task 14: Implement local daemon long-poll loop

**Files:**
- Modify: `cmd/internal/remotecmd/daemon.go`
- Test: `cmd/internal/remotecmd/daemon_test.go`
- Create: `cmd/internal/remotectl/fakerelay_test.go`

- [ ] Connect outbound to relay with daemon signature.
- [ ] Publish session snapshots on interval and after lifecycle actions.
- [ ] Poll for command envelopes.
- [ ] On every relay response, including reset tombstone/auth-failure responses, compare account revocation epoch; if remote epoch is newer, disable local remote access until `pair remote login` re-enrolls.
- [ ] Add fake-relay integration tests for reconnect, stale commands, and epoch reset.
- [ ] Run `go test ./cmd/internal/remotecmd ./cmd/internal/remotectl -count=1`.

## Chunk 4: Remote Lifecycle Actions

### Task 15: Implement lifecycle command authorization

**Files:**
- Create: `cmd/internal/remotecmd/lifecycle.go`
- Test: `cmd/internal/remotecmd/lifecycle_test.go`

- [ ] Map `session.list`, `session.new`, and `session.renew` to typed command handlers.
- [ ] Require local grant, valid device signature, non-replayed command ID, matching session identity, and fresh timestamp for every command.
- [ ] Reject arbitrary prompt text in `session.renew`.
- [ ] Record an audit entry before and after each accepted command.
- [ ] Run `go test ./cmd/internal/remotecmd ./cmd/internal/remotectl -count=1`.

### Task 16: Dispatch `session.new`

**Files:**
- Modify: `cmd/internal/remotecmd/lifecycle.go`
- Test: `cmd/internal/remotecmd/lifecycle_test.go`

- [ ] Implement `session.new` as the remote equivalent of fresh-session restart.
- [ ] Use existing Pair lifecycle command path (`pair restart --new-session` or shared launcher seam).
- [ ] Ensure it targets the selected session/tag only and does not accept arbitrary Pair args.
- [ ] Add fake lifecycle-dispatch tests for success, wrong session, missing live session, and command expiry.
- [ ] Run `go test ./cmd/internal/remotecmd -count=1`.

### Task 17: Dispatch `session.renew`

**Files:**
- Modify: `cmd/internal/remotecmd/lifecycle.go`
- Test: `cmd/internal/remotecmd/lifecycle_test.go`
- Test: existing launcher/continuation tests if shared behavior changes.

- [ ] Implement `session.renew` as one local transaction with progress phases.
- [ ] Ask the live agent to run the existing continuation flow, or call the same local path when the command originates locally and can safely do so.
- [ ] Verify the created continuation file through `VerifiedContinuation`: under `workshop/continuation/`, no traversal/symlink escape, belongs to the selected repo/tag context where metadata exists, valid continuation frontmatter, and non-empty NEXT ACTION.
- [ ] Trigger the continuation-backed fresh restart with auto-send intent.
- [ ] Ensure the first prompt is rendered locally with `ContinuationPrompt`.
- [ ] Add timeout/failure states for continuation not written, restart failed, auto-send sidecar not consumed, and daemon reconnect mid-transaction.
- [ ] Run `go test ./cmd/internal/remotecmd ./cmd/internal/launcher ./cmd/internal/continuationcmd -count=1`.

### Task 18: Add minimal web dashboard

**Files:**
- Create: `cmd/internal/remotecmd/web/` or embedded static assets under the chosen relay package.
- Modify: relay handler tests.

- [ ] Render authenticated session list.
- [ ] Show available actions based on relay/account routing and local grant-reported capabilities.
- [ ] Provide buttons for `session.new` and `session.renew` with clear confirmation text.
- [ ] Show command progress and final status.
- [ ] Avoid displaying transcripts, scrollback, file contents, raw paths, or prompt text.
- [ ] Run relay/browser handler tests.

## Chunk 5: Hardening, Docs, And Smoke

### Task 19: Add nuclear reset end-to-end tests

**Files:**
- Test: `cmd/internal/remotecmd/relay_test.go`
- Test: `cmd/internal/remotecmd/daemon_test.go`

- [ ] Prove browser reset revokes all relay sessions/devices/daemon registrations/pending commands for the account.
- [ ] Prove relay increments revocation epoch.
- [ ] Prove local daemon disables remote access on next contact after observing a newer epoch, including the case where its daemon registration was already revoked and the relay returns only a reset tombstone/auth-failure response.
- [ ] Prove local `pair remote login` is required before access resumes.
- [ ] Run `go test ./cmd/internal/remotecmd ./cmd/internal/remotectl -count=1`.

### Task 20: Add operator docs

**Files:**
- Modify: `atlas/architecture.md`
- Create or modify: `atlas/remote-control.md`
- Modify: `atlas/index.md`

- [ ] Document trust boundary: local daemon authority, relay as dumb mailbox.
- [ ] Document `pair remote login`, grants, revocation, nuclear reset, and v1 capability scope.
- [ ] Document that Google/OIDC logout is not sufficient without Pair reset because local keys can exist offline.
- [ ] Cite `ARCH-DRY`, `ARCH-PURE`, and `ARCH-PURPOSE` decisions where relevant.

### Task 21: Verification sweep

**Files:**
- Modify as needed based on test fallout only.

- [ ] Run `go test ./cmd/internal/launcher ./cmd/internal/continuationcmd ./cmd/internal/remotectl ./cmd/internal/remotecmd ./cmd/internal/dispatcher ./cmd/pair-go -count=1`.
- [ ] Run `go test ./... -count=1`.
- [ ] Run `make test-lua`.
- [ ] Run any existing terminal shortcut tests touched by the nvim startup hook.
- [ ] Run `git diff --check`.
- [ ] Perform manual smoke: local fake relay, `pair remote login` test IdP, daemon session list, `session.new`, `session.renew`, nuclear reset, daemon reconnect-after-reset.
