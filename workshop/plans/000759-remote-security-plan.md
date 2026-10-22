---
collaboration-mode: discussion
---

# Remote-session 👽 security implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach. Use superpowers-subagent-driven-development for bounded independent tasks or superpowers-executing-plans for work that depends on session context. Steps use checkbox syntax.

**Goal:** Expose only the remote-session document interface through verified Cloudflare Access, with bounded remote sessions, truthful provenance, and a persistent kill latch reset only through Couch's own host-terminal key path.

**Architecture:** One remote supervisor owns authorization, the latch, sessions and the remote-only HTTP mount. It reuses the existing guarded cloudflared transport and named-tunnel lifetime lock, with strict Access configuration and no anonymous or loopback fallback. #758 supplies the registered-document/nvim adapter; its operations require a live, server-created remote permit.

**Tech stack:** Go HTTP/context, existing Couch console and message broker, durablefile publication, guarded cloudflared, an established Go JWT/OIDC verifier (Cloudflare's Go example uses coreos/go-oidc/v3), fake issuer/JWKS and existing stateful FakeTunnel. No handwritten signature implementation.

**Status:** Operator APPROVED with changes via TL receipt 7c5b26ce-a932-4fb1-8d97-21c12f04fa05 (2026-10-21 09:04). The approved current contract below supersedes earlier proposals retained in Revisions. Record revision, then run change-code; account/network changes remain unauthorized. Source: ops `teams/devinfra/projects/remote-session/{project,design}.md`, pair#759, and TL receipt 793b0f54-b454-4226-a0da-227c22d7a76b. The operator reviews before code; later changes append Revisions.

## Approved scope and implementation decisions

1. MVP is registered hub session files and their registered side docs. No terminal frames, browser drive, arbitrary files, raw nvim RPC, shell, or slot lifecycle API. The terminal cast row gains the 👽 control; this does not turn the terminal viewer into the phone editor.
2. Require an explicitly configured named tunnel/hostname, Access team, application AUD and exact operator identity allowlist. No quick-tunnel or LocalOnly fallback. Validate JWT signatures/issuer/audience/time locally as well as requiring cloudflared's Access enforcement.
3. Use the existing named tunnel **exclusively** for remote mode. Starting while terminal/Chrome/replay casts hold it refuses with “stop the existing casts, then retry”; never stop them implicitly. Ordinary cast starts cannot share the active remote mount. This is a proposed usability constraint to keep the kill boundary small: killing remote closes its entire exposure owner, not merely one of several mounts. A later ordinary cast cannot route to the retired remote server.
4. Clicking 👽 at the console starts remote mode and shows its link; clicking again stops it. `couch --remote on|off` is equivalent. Each enable creates a fresh 256-bit per-enable URL capability (the link passcode); off destroys it, and re-enable makes every old link invalid. On/off/passcode rotation revoke all sessions. This ordinary toggle is separate from the remote kill latch. Off at every clean startup. `couch --remote on` can request an ordinary start only while unlocked/off; it cannot clear a killed/uncertain latch. Any authenticated remote user may invoke kill (off only), without needing to be the initiating session. Remote has no enable/reset action. Reset is a key in Couch's own UI, then a separate explicit start. CLI/status/API have no reset verb. Prefer a dedicated Remote security panel reached from the 👽 cell: its reset action requires a decoded key press, not mouse/paste/terminal replies.
5. Sessions have no Couch absolute or idle timeout. They last until remote off, kill, logout or Access JWT expiry, bounded by Cloudflare's configured Access lifetime. A valid Access assertion permits a silent fresh handshake without a login prompt. Keep four concurrent sessions; passcode rotation revokes all.

6. **Trust limit:** “at the machine” means through Couch's owned host-TTY key reader. It proves separation from HTTP, broker IPC, child PTYs and `couch --keys`; it does not prove a physical human against arbitrary code with the same macOS UID, filesystem access or terminal-injection privileges. Host OS/Couch binary/user account and Cloudflare configuration are trusted. Stronger isolation needs a separate OS security principal or hardware-backed approval and is not claimed here.
7. #758 is currently unowned/unimplemented. #759 owns the security wrapper and trusted submit path; #758 owns document eligibility, editing/reflow/navigation and nvim lifecycle. An absent document adapter returns unavailable, never a permissive fallback. Security can be tested with the same interface's stateful document backend; actual nvim edit/submit acceptance requires integration with #758 before calling the remote-session MVP usable. TL settled sequencing in receipt e205ff19-08b2-4909-be86-ea8e4d728d82: build/test #759 against the stateful document backend; #758 follows parley.nvim#405, and the real-nvim edit/submit drill belongs to their integration. Do not block the security foundation on #758 or claim the editor MVP works before that integration.

## Existing evidence and alternatives

`couchcmd/broadcast.go` defaults ordinary casts to quick tunnels when the environment is unset. `broadcast/cloudflared.go` generates a named ingress without Access validation and can fall back from Unix socket to loopback. These are unsuitable defaults for remote instructions.

Production ordinary cast links also have numeric-code locks (#495), despite stale comments calling the URL token the only credential. View is read-only; pointer links allow annotations, and Chrome drive links additionally require enable/grant. Tokens live in URL paths for the cast lifetime; locks have no wall-clock expiry. These are existing-cast findings reported to TL separately, not authority for this API.

`broadcast.Host` shares one endpoint between terminal/Chrome/replay; last unmount closes it. Simply adding a remote mount and unmounting it on kill would leave exposure running. Existing `Host.Mount` also expects an unauthenticated public 200 probe, incompatible with Access.

Options considered:

| Approach | Benefit | Rejection or recommendation |
|---|---|---|
| Reuse cast bearer/code sessions and mount beside every cast | Smallest apparent diff | Reject: wrong identity/lifetime/capability contract; kill does not close shared tunnel |
| Share the host and add a global all-mount kill/mode migration | Concurrent cast use | Reject for MVP: extends the security state machine through every cast/replay start and ownership path |
| Remote-only host using the existing named transport and exclusive lock | One exposure owner, existing process cleanup | Recommend: explicit busy refusal instead of automatic cast migration; no second connector or duplicated tunnel process implementation |

## Threat model

Protected assets: live session-buffer contents, submit authority over bypass-capable agents, nvim's local sockets, Cloudflare credentials, and audit/provenance integrity within the trusted account.

| Adversary/input | Boundary and required defense |
|---|---|
| Leaked URL or unauthenticated Internet request | Access edge and cloudflared required=true; application independently verifies JWT before document or static-app dispatch |
| Forged identity header, JWT for another app/team, expired token, attacker `jku` | Only configured issuer/JWKS URL and exact AUD; signature/time/algorithm validation, human subject/email allowlist; never trust email headers or token-supplied key URLs |
| Stolen authenticated phone session | Access JWT lifetime is the time bound; no additional Couch idle/absolute limit. Least API and epoch kill/off/logout revocation; residual power until revocation/Access expiry is acknowledged |
| Cross-site request or hostile document text/link | Exact Origin/Host, Secure HttpOnly SameSite=Strict cookie, CSRF token on mutations, JSON-only bodies, CSP/no-store/no-referrer, text rendering; no HTML/script execution from markdown |
| Path traversal, old registry ID, renamed/forgotten doc, exposed nvim methods | #758 resolves eligible registered IDs at use time, matches editor generation, never accepts caller paths, commands or arbitrary RPC; #759 accepts only typed actions |
| Delayed start, stale session, duplicate submit, kill racing an edit | Single lifetime owner, epoch-fenced permits, serialized per-document mutation admission, idempotency key/revision; late completions cannot restore authority |
| Agent using CLI/IPC/`--keys`/pasted reset chord | Reset callback exists only in host-TTY decoded-key branch; child writer has no route into that input channel; real broker-to-wrapper negative test |
| Host process or same-UID code already compromised | Outside this boundary. It can alter binaries/state or inject host input. Do not describe the latch as a sandbox against the agents' OS account |

An allowed remote instruction can ask the TL to do anything it could do locally. API least capability does not sandbox the agent. ariadne#520 owns the confirmation rule for irreversible/out-of-flow remote requests; remote provenance is data that rule consumes, never permission to skip confirmation.

### Access verification and secrets

Operator config supplies non-secret team name/AUD/hostname/allowed identity. cloudflared locates existing credentials; Pair never creates credentials, changes DNS/Access/account settings or records credential bytes. Generated per-run config remains private and is removed with the owned handle.

Require a Unix listener in a 0700 directory (socket 0600); long socket path or bind failure refuses, never silently uses TCP. Generated ingress has `originRequest.access.required: true`, configured `teamName` and exactly the expected `audTag`, plus final 404. Keep the existing guarded process and named-lock lifetime.

Application verifier consumes `Cf-Access-Jwt-Assertion`, validates configured issuer, signature against bounded team JWKS, expected AUD, expiry/not-before, allowed human identity and supported algorithm; rejects service-token/missing-user claims. Never follow token-supplied `jku`/`x5u`, HTTP redirects or issuer discovery to arbitrary hosts. Use a reviewed verifier, not just presence of the header. Test key rotation with a local real-signature issuer. JWKS refresh uses the configured HTTPS team endpoint, one in-flight refresh, time/size/key-count limits, and no network work on Couch's reducer. Existing cached valid keys may serve within their configured lifetime; unavailable keys refuse. JWTs/cookies/CSRF values never enter logs or URLs.

Session cookies contain fresh256-bit random material; hash server-side, bind to issuer+subject+AUD+remote epoch/passcode generation, Secure/HttpOnly/Strict. The link capability uses a fresh opaque `/remote/<passcode>/` prefix per enable; session cookies and HTTP routes are scoped beneath it. Do not log or persist the passcode. Off invalidates the old route before teardown; only a fresh start publishes a new link. Every request/stream and streamed mutation rechecks Access/session/epoch. There is **no15-minute absolute or5-minute idle deadline**. One session lives until off, kill, passcode rotation, logout or its verified Access assertion's expiry. At that expiry retire the old session/streams and silently re-handshake if Access supplies a currently valid assertion; otherwise require Access's normal sign-in flow. Do not let silent renewal revive a retired epoch or exceed four concurrent sessions. App-level renewal never creates or refreshes Access credentials.

A new handshake restores the view, not prior mutations. Keep unsent phone text, read current document/revision and reconcile prior uncertain operations by their recorded result/receipt. Retained receipts can be read by the same verified principal/document without dispatch authority; expired evidence means unknown and explicit inspection. Off/kill/rotation revoke every session. Logout revokes that session. Fake-clock tests cover long idle and elapsed time without expiry while Access remains valid, Access expiry, silent handshake, cap-four races, off/kill/rotation and delayed stream callbacks.

**Identity/IP provenance:** Access email comes from the verified JWT, not a client text field or email header. Client IP comes only from one strictly parsed `Cf-Connecting-IP` supplied through the protected Cloudflare -> named cloudflared -> private Unix-origin chain; no X-Forwarded-For/RemoteAddr substitution. Parse with net/netip, reject missing/duplicate/list/control-containing values, and preserve normalized IPv4/IPv6. A JWT signature does not sign this header: its provenance rests on the trusted ingress. Reject Worker-subrequest indicators for this MVP and ask the operator to confirm no Worker/transform/pseudo-IP override rewrites the hostname's IP path during live setup. Same-UID origin forgery remains outside the stated boundary. Bind the email/IP fact to each admitted HTTP request or established stream and never accept replacements inside an action frame; a later phone-network change obtains a fresh verified request fact. IP is audit/provenance, not an authentication factor.

Static app assets, status, kill and stream routes all use the remote auth boundary; no public health/document bypass. Replace the Host's public-200 mount readiness for this mode with bounded local server-start plus connector-ready evidence. Call it “waiting for authenticated phone” until the first verified handshake; do not claim public delivery from connector readiness. Ordinary casts retain their existing probe.

## Document integration contract (settle before M1)

Proposed ownership: #759 owns the remote HTTP listener/routes, auth/session admission and trusted submit delivery; #758 supplies a typed document adapter behind that boundary, not a second server. This refines #758's current server wording and requires TL coordination before code; neither branch edits the other's implementation. The adapter contract is agreed first, not invented in M3:

- Resolve an eligible DocumentRef by opaque registered ID and editor generation, returning text/revision/highlights through bounded subscribe/read operations.
- Mutations accept a typed action, base revision, request ID and server-only permit. The adapter exposes a nonblocking handoff into its owned operation queue. The handoff shares the supervisor's admission mutex with kill: while holding it, recheck epoch/session, record the dispatched identity and enqueue exactly once. No audit/storage/network wait holds this mutex. Kill either wins and prevents handoff, or handoff wins and the operation is in flight and may commit/return uncertain. A held audit intent is followed by a fresh atomic handoff check, never by a stale permit read.
- The adapter rechecks permit cancellation and document/editor identity at its effect door; queued work not yet at that door is discarded after kill. An effect already handed to nvim is treated as in-flight, not retrospectively rolled back. Reset waits for adapter workers/streams to join. Never hide a second unbounded queue behind the handoff.
- Navigation is resolve -> authorize destination -> bounded navigate, never execute a raw nvim link/jump command and inspect the result afterward. Targets, parent links and jump-list entries must resolve to eligible registered document/editor identities before nvim opens them, and be rechecked at handoff. Test unregistered targets, symlinks, stale history and target rebinding between resolve/handoff.
- Persist remote-edit provenance before mutation handoff as described below. Remote submit passes through #759's origin-aware path after #758's save semantics, carrying exact document/revision and request identity; no shell-based send-to fallback.
- Outcomes are confirmed result, confirmed refusal-before-effect, or uncertain with retained operation/submit identity. Adapter stateful tests and real-nvim tests share the interface. Missing adapter fails closed.

## Core concepts (ARCH-PURE / ARCH-DRY)

| Pure entity | Lives in | Status |
|---|---|---|
| remoteState, remoteEvent, remoteEffect, ReduceRemote | `cmd/internal/remotesession/state.go` | delivered M1 |
| AccessIdentity, remotePolicy; validAt/samePrincipal value checks | `cmd/internal/remotesession/policy.go` | delivered M1 |
| RemoteAction, DocumentRef, DocumentEdit, NavigationTarget, DocumentRequest | `cmd/internal/remotesession/protocol.go` | delivered M1 |
| DocumentSummary, DocumentSnapshot, DocumentResult, DocumentSource | `cmd/internal/remotesession/documents.go` | delivered M1 |
| Replay, replayEntry, replayAdmission (value-only sequence decisions) | `cmd/internal/remotesession/replay.go` | delivered M1 |
| RemoteStatePaths, ResolveCouchRemote, RemotePublicationStage | `cmd/internal/artifactpath/remote.go` | delivered M1 |
| Message, Origin, RemoteSource; ValidateProvenance/SameProvenance and envelope projection | `cmd/internal/couchmessage/model.go`, `provenance.go`, `batch.go` | implemented M3, qualification pending |

The reducer owns legal transitions; policy decides identity, origin and bounds from values. Replay changes owned in-memory values without IO. One supervisor has one epoch and at most four sessions; each session has many allowed document operations, each tied to current document/editor identity. One action enum is the allowlist shared by decoder, dispatcher and tests. The future phone UI consumes this contract; no general “execute” extension slot.

| Integration point | Lives in | Status | Wraps |
|---|---|---|---|
| Sessions | `cmd/internal/remotesession/session.go` | delivered M1 | OS entropy, synchronized cookie/revocation ownership; pure identity predicates remain in policy.go |
| Store | `cmd/internal/remotesession/store.go` | delivered M1 | latch publication and bounded stage recovery |
| OriginStore | `cmd/internal/remotesession/origin_store.go` | delivered M1 | durable conservative edit evidence |
| Audit | `cmd/internal/remotesession/audit.go` | delivered M1 | metadata append, durability, rotation/retention |
| AccessVerifier / accessKeys | `cmd/internal/remotesession/access.go` / `access_keys.go` respectively | delivered M1 | JWT library, bounded HTTPS JWKS |
| DocumentBackend, admissionGate, Permit | `cmd/internal/remotesession/documents.go` | delivered M1 seam | typed adapter interface and synchronized one-use effect handoff; stateful private backend only |
| Supervisor / remoteEpoch; sessionOperations | `cmd/internal/remotesession/supervisor.go`, `dispatch.go` | M2 reviewed SHIP; M3 dispatch implemented | lifecycle effects, ordered persistent writes, synchronized replay/rate state, sessions, joined requests |
| RemoteServer | `cmd/internal/remotesession/server.go`, `server_documents.go` | M2 security surface reviewed SHIP; M3 document routes implemented | protected HTTP, signed Access, bounded request lanes, cookies/CSRF |
| ProtectedTunnel / protected Host; remote configuration | `cmd/internal/broadcast/protected.go`, `cloudflared.go`, `host.go`; `couchcmd/remote.go` | M2 reviewed SHIP | existing cloudflared handle/guard/lock, strict remote config |
| Console remote control / remoteConsoleView | `cmd/internal/couchtty/console_remote_session.go`, `terminal_input.go`, `console.go` | M2 reviewed SHIP | local decoded key event, status and operator-only link/QR |
| Trusted remote submit | `cmd/internal/couchcmd/remote_submit.go`, `couchmessage/broker.go` | planned M3 addition/modification | broker, receipt, batching and delivery accounting |

Effects receive the pure transition's attempt/epoch identity and return typed outcomes. Store, issuer, clock, tunnel and document adapter are replaceable with portable private-root stateful backends. Reuse FakeTunnel and guarded cleanup; add fake JWKS rotation and fake document revisions/held operations, not function-call-only mocks.

## Authority, durable states and recovery (ARCH-ORDER)

One remote supervisor owns the sole admission lock for its lifetime; existing named-tunnel lock excludes other connectors. Permits retain the exact supervisor epoch and revocation channel. State is private and changes only through ReduceRemote; an exported HTTP or CLI enum can never represent ConsoleReset. The console holds the only reset closure, installed during makeConsole wiring and called only by its host-key branch. Do not export a generic setter that future callers can invoke.

Durable state is small versioned JSON: `off`, `armed`, or `killed`, with generation. `armed` is written and file+directory synced **before** listener/tunnel opening. It grants nothing on disk: only the current in-memory owner may use that epoch. Restart finding `armed` treats it as killed/unclean, so a disk-full error during a later kill cannot make reboot silently re-enable. A clean ordinary stop may persist `off` only if no kill won; reset of `killed` is console-only. Missing file on a new install is off; truncation, unknown version/state, symlink/nonregular file or unreadable storage blocks remote access visibly. Preserve unknown JSON members on supported versions; unknown authority values fail closed. No GC/automatic repair removes a kill latch.

| State/event | Next state and effects |
|---|---|
| Boot + missing/off | Off; no listener, credentials lookup, session or tunnel started |
| Boot + armed/killed | Killed; no authority restored; collect only owned orphan transport using existing identity-fenced cleanup |
| Boot + invalid/unreadable latch | Blocked; do not infer off or overwrite evidence |
| Off + explicit on | Preparing(attempt); verify config/owner lock, persist armed; no session admission yet |
| Preparing + persistence/config failure | Off or Blocked according to durable certainty; report why, no exposure |
| Preparing + durable armed and open success | Ready(epoch); admit verified handshakes, publish link only locally |
| Preparing + stop/kill or late old completion | Revoke attempt first; dispose every late handle. Kill is sticky; completion cannot promote it |
| Ready + authorized request | Check session/epoch/doc generation; begin bounded operation; return confirmed failure/success or explicit uncertain result |
| Preparing/Ready + ordinary console/CLI off | Stopping; discard link passcode and revoke sessions/queued admissions; join exposure; persist off only after confirmed closure and no kill event |
| Any + kill from any authenticated allowed remote user, or local kill | Killed/closing; revoke in memory immediately; persist killed; close listener/streams/transport regardless of write error |
| Killed/closing + confirmed teardown | Killed; retain latch. Failure/timeout stays “disabled; cleanup unconfirmed,” never off |
| Killed + CLI on, remote on, broker keys, old completion | Refuse/no grant; no implicit reset |
| Killed + ConsoleReset after confirmed cleanup | Persist off; only confirmed publication yields Off. Failed/uncertain publication remains blocked/killed |
| Ready + tunnel/server/audit fatal failure | Revoke and stop; retain armed/killed requiring local reset, no automatic reconnect |
| Ready + Access JWT expiry | Revoke old session/streams; fresh valid Access permits silent handshake, never old-epoch resurrection or mutation replay |
| Ready + Access assertion rejected/unrenewable | Revoke affected session/streams; sign-in-required; remote mode may remain ready for a newly verified identity |
| Any + Couch shutdown/process death | No automatic restart; existing guard owns child cleanup; armed/killed survives crash |

Kill acceptance is an immediate authority fence, not a claim of instantaneous external teardown or rollback. A mutation already committed before the fence may remain; no operation waiting for admission may start after it. An RPC already handed to nvim can have an uncertain result: cancel, retain request ID and require re-read/reconciliation, never resend automatically. Reset is blocked until requests and handle are joined; if cleanup remains uncertain, the screen says so. Kill response can be lost when its own tunnel closes: the phone reports unconfirmed until reconnect/console evidence, and retry of the same disable intent is idempotent.

Actions: **start**, **kill**, **reconnect/read status**, **inspect current document**, **local reset**, **repair local configuration/storage**.

| Internal state/failure | Visible state | One recovery action | Possible loss |
|---|---|---|---|
| Clean off | Off | start | none |
| Opening, verified configuration | Starting | read status | none; no edits admitted |
| Ready with valid session | Remote on / 👽 | kill | already-applied work remains |
| Access assertion expired, fresh Access valid | Reconnecting (normally silent) | reconnect | unsent phone edit remains; no automatic mutation replay |
| Access expired/rejected | Sign in required | reconnect | unsent phone edit remains; never claim saved |
| Kill or unclean armed state, cleanup confirmed | Disabled at machine | local reset | sessions revoked; unsent phone work retained locally |
| Cleanup still in flight/unconfirmed | Disabled; cleanup pending | read status | in-flight write may have committed |
| Invalid config/storage/uncertain latch publication | Remote blocked, reason | repair local configuration/storage | retained evidence; no remote grant |
| Mutation handed off but reply missing | Outcome unknown | inspect current document | possible prior commit; no automatic retry |

## Allowlist, provenance and audit

Decode strict typed JSON under the current `/remote/<passcode>/v1/` capability: session handshake/status, eligible-document list, subscribe, paragraph edit, submit, follow eligible link/parent/back/forward, and logout. A separate `/remote/kill` route permits any Access-authenticated allowed user to kill without owning a current session/passcode or consuming the normal four-session capacity; it requires exact same Origin, JSON and a custom kill-action header (no CORS/simple-form path), but no session-owned CSRF token. All other mutations require the session CSRF token. It can only revoke/disable. No generic operation name passed to Couch's IPC switch, no proxying of request paths to files, no browser-drive routes. #758 owns the document action semantics; #759 owns outer authorization and per-operation permits. Registered does not alone mean remotely eligible: only the selected hub session roots and eligible registered side docs. Revalidate ID membership and editor generation at use time; stale/forgotten/foreign refs fail before nvim IO.

Remote submit generates canonical `submitted: <registered-current-path>` on the server and carries an immutable broker origin `remote` in the trusted envelope, receipt, batches and delivery ledger. The visible remote-submit header is exactly `[remote <Access email> <verified Cf-Connecting-IP>]` before the message, generated from typed server facts. The enclosing Couch delivery header retains its delivery ID for receipt verification. Console submits have no remote display prefix, including when RemoteEdited is true; the receipt field still preserves that provenance for #520. Never append the tag to the path line: SubmitPath treats the entire first line as the path. Do not use OriginSystem, which skips delivery obligations. The phone cannot supply an origin, caller binding, routing owner or arbitrary message body. #758 calls the trusted path through its admitted permit after parley's save/submit semantics; it must not shell out to ordinary --send-to and lose provenance. Extend endpoint validation and compatibility tests; older wrapper unable to carry provenance refuses remote submit and asks for relaunch, rather than silently downgrading to local.

Approved conservative provenance policy: persist a monotonic “remote edited” marker keyed by stable registered document ID **before** the first remote mutation handoff, even if that mutation later fails. #759 owns this small origin store; #758 supplies the exact document identity and never clears it on save, revision changes, headless/desktop handover or nvim restart. The ordinary local submit admission consults the same store even when remote mode is off, so Couch restart cannot launder provenance. A local submit retains RemoteEdited=true in authoritative metadata without adding the remote identity/IP display header. Unknown/unreadable origin evidence never yields an unqualified local provenance claim.

**Receipt contract for ariadne#520:** `Receipt.Message.Origin = "remote"` identifies a direct phone submit. The independent JSON key `Receipt.Message.RemoteEdited` is a server-assigned boolean; `true` means this registered document has the durable remote-edited marker. The Go field is `Message.RemoteEdited bool` with JSON key `RemoteEdited` (omit when false). A local submit can therefore have peer/local Origin and `RemoteEdited: true`; a direct phone submit has Origin remote and also carries RemoteEdited true if the marker exists. Persist the same `RemoteEdited` key alongside Origin in the submit/delivery and owed record projections. Only trusted broker admission sets it from the origin store; neither request JSON, webpage content nor worker message text can select it. Keep it unchanged through receipt, batching and endpoint delivery. Verification for #520 treats **either** Origin remote **or** RemoteEdited true as remote provenance. A missing/false key is not proof of clean local provenance, especially in legacy or unavailable evidence; do not use omission to waive a required confirmation.

For simplicity there is no content-diff proof or “all remote text removed” heuristic. The conservative marker remains for the registration lifetime; the existing explicit local forget/re-register workflow is the reconciliation boundary, after pending submits for the old ID settle. Remote API cannot forget/register or clear markers. Before clearing/re-registering, the operator reviews the document; this intentionally costs convenience rather than silently clearing evidence. Markers survive rename/alias changes; bounded cleanup removes old-ID markers only after registry absence and no pending operation/receipt. Proposed limits:4096 markers/1 MiB total, private versioned JSON preserving supported unknown fields, capacity/error refuses remote edits; no silent eviction. This policy is covered by the operator approval retaining Origin/RemoteEdited; same-UID local modification remains outside its guarantee.

Each edit/submit uses bounded `(session epoch, monotonically increasing request sequence, document ID, base revision)` replay protection. Retain a256-result ring plus the accepted sequence high-water mark; an older evicted sequence refuses with result-expired/inspect, never re-executes. Uncertain in-flight entries are pinned and capped; if all slots are pinned, refuse new mutations until reconciliation. There is no forced session renewal at256 operations. Same ID/same payload returns the retained result without a duplicate effect; same ID/different payload refuses. Uncertain effects are retained as uncertain and reconciled by document revision/submit receipt, not re-run. Killed/expired sessions cannot use retry to act again.

Audit metadata only: timestamp, event/action, opaque principal/session IDs, epoch, document ID, request ID, before/after revision, outcome, delivery receipt ID. Never log raw JWT/cookies, URLs containing secrets, document bodies, absolute paths, nvim commands or query strings. Persist an intent before mutation and a result afterward; intent without result means uncertain. An intent-write failure refuses before effects. A post-effect audit failure revokes remote mode and reports uncertain; it never changes the operation into “not applied.” Kill never waits for an audit slot before revoking. No audit log is an authorization source. Keep the normal agent/session transcript as a **second paper trail**: the full emitted remote prefix plus canonical submission reaches the existing delivery/agent-input transcript recording path. Do not replace the metadata audit with transcript parsing or create a second unbounded transcript format. Qualify retained transcript round-trip through the real wrapper/log seam using private roots. The approved identity/IP is visible in the submit/transcript, while the security audit can retain opaque principal/session references; never log JWT/cookie/passcode material.

## Persisted-format contract (PQ-1, ARCH-EVOLVE)

Use existing `strictjson.RejectDuplicateKeys` at the full-document/row boundary (including nested objects), before typed projection. Follow `storagegc/runtime_codec.go`'s retained raw-object plus owned-field merge pattern: each evolving object retains a cloned `map[string]json.RawMessage`; decode known fields with exact case/type/null rules, validate required fields and authority enums, and merge updated known fields back over the retained map on write. Keep nested remote-source/registration metadata as their own retained raw objects, so a parent remarshal cannot erase nested future members. Unknown fields never participate in authorization. Known-key case aliases, trailing JSON, malformed/truncated JSON and oversized/deep objects refuse; supported version unknown fields survive semantically, unsupported authority versions refuse without rewriting. Do not introduce a generic schema framework.

| Changed format / codec | Known-field validation and actual rewrite test | Mixed-version boundary |
|---|---|---|
| `LatchRecord` / `decodeLatch`, `encodeLatch` in remotesession/store_codec.go | Required Version, Phase(off/armed/killed), Generation; exact types/no null; retain unknown root objects. Seed future nested JSON, call Store.Arm -> Kill -> ConsoleReset -> reopen, verify every opaque member survives; crash/failed rename preserves old bytes | Version1 only; unknown version/state blocks remote. Old Couch has no remote latch reader/clearer; no claimed legacy remote writer support |
| `OriginStoreRecord` / `decodeOrigins`, `encodeOrigins` in remotesession/origin_codec.go | Required version and bounded document map; validate document IDs, marker generations and fixed booleans. Retain unknown envelope and per-document nested members; MarkRemote, rename-reference read, GC forgotten settled IDs and reopen must preserve surviving unknown data | New private family, old Couch does not touch it; unknown version/ref identity blocks remote edits and yields unknown provenance, never clean local |
| `AuditRecord` / `decodeAuditRecord`, `encodeAuditRecord` in remotesession/audit_codec.go | Required version, bounded IDs, action, UTC time and finite intent/result enum; raw unknown nested metadata retained. Drive Append -> size/age rotation -> reread surviving future rows with known results and opaque values intact | New private family; unsupported version blocks append/mutation before effects. Do not compact undecodable existing rows as if absent |
| `sessionsync.Entry` JSON codecs in sessionsync/ledger_codec.go | Legacy required-ID behavior retained for legacy rows; rows with new provenance require typed Origin, RemoteEdited and valid RemoteSource when Origin remote. Preserve raw unknown root and RemoteSource members through readRows -> Ledger.trim -> reopen, then OwedForDocument projection; inject wrong known types/duplicate nested keys/missing remote identity and assert gap/refusal, not local inference | Current Couch is the sole ledger writer under existing supervisor singleton; no concurrent old writer supported. Legacy local rows stay readable; missing provenance stays unknown. Old standalone CLI reads cannot rewrite ledger. An intentional rollback to an old supervisor is unsupported for retaining new provenance: disable remote, retain/export store and transcripts before rollback; unknown/missing provenance never authorizes bypass of #520 |
| `RemoteSource` / broker receipt, batch and ledger projection | Typed JWT email plus parsed IP, server assigned; JSON field `RemoteSource` contains Email and IP. Remote Origin requires it; peer/local Origin cannot carry a fake remote display source. Local RemoteEdited remains metadata only. Store and trim preserve nested future fields; wire tests verify the known security facts and reject unsafe types | Receiver wrapper must advertise remote-provenance-v1; refusing old/unknown receivers before dispatch prevents silently stripping tags. OriginSystem is not reused. Existing legacy peers keep their contract |

The receipt and OwedEvent JSON are projections, not independent writable authority stores: propagate validated Origin/RemoteEdited/RemoteSource from the decoded delivery Entry. Tests must traverse real writer paths, not just encode/decode a standalone sample. Bound all raw objects using the format's stated byte cap and depth32. Distinguish a legacy torn tail from a recognized malformed provenance row; the latter marks a recording gap and cannot disappear into a clean-local answer. Single-writer rollout plus explicit unsupported rollback is the mixed-version limit, not a claim that we can retrofit preservation into an old executable.

## Bounds, cost and lifetime (ARCH-CONSTRAINTS / ARCH-FUNERAL)

These are initial engineering proposals for operator review, not measured performance claims.

| Growth driver | Bound / operation cost | Excess behavior |
|---|---|---|
| Remote mode | One supervisor, one named lock/transport, four sessions | busy/refuse; no second connector |
| Request/frames | 64 KiB mutation, 8 KiB assertion; 5-second ordinary exchange | 413/timeout; uncertain if handed off |
| Key fetch | 64 KiB JWKS, 8 keys, 5-second fetch, one refresh; 5-minute refresh TTL | fail closed; rate-limit unknown-kid refresh to avoid fetch per request |
| Per-session traffic | 10 mutation requests/sec, burst20; 8 concurrent requests process-wide, reserved kill admission | 429; kill bypasses normal operation queue |
| Streams | One active document stream/session; bounded 1 MiB queued output each | close/resubscribe; no unbounded slow-client queue |
| Replay table |256-result ring/session + monotonic sequence high-water,1 KiB result cap; up to256 pinned uncertain entries | reject evicted/old sequences without effects; refuse new mutations if uncertainty fills capacity; no artificial session expiry |
| Origin evidence | <=4096 markers / 1 MiB; stable document IDs, lookup once per submit | refuse remote edits at capacity; retire only forgotten, settled IDs |
| Audit | <=2 KiB/record; 16 MiB active + one 16 MiB previous, max30 days | rotate under writer; remove superseded old segment; failure blocks remote mutation |
| Latch | One <=4 KiB private file plus lifetime lock; fixed read at boot, writes at lifecycle transitions only | malformed/oversized blocks remote; never auto-delete |
| Registry/documents | Reuse #758 and existing registry/file limits, no scan per edit | unsupported/oversized document refusal |
| Console response | Revoke epoch without disk/network wait; effects on joined worker | show pending state; do not block key reducer |
| Shutdown | existing ~5-second transport close budget, bounded joined RPC cleanup | disabled with explicit cleanup uncertainty; no permission restored |

Hot paths: one cached-JWKS signature verification plus O(1) session/replay lookup per request; O(1) permission check per mutation/stream frame. At N=8/16 requests, count backend dispatches and audit intents/results; reject duplicate dispatches and quadratic scans. Count one JWKS fetch for the same valid-key batch, one coalesced/rate-limited refresh under 8/16 unknown-kid requests, and one tunnel open per epoch. Kill work is O(active sessions + admitted requests) within the stated caps. No transcript scan, liveness subprocess or full registry scan on each key/HTTP frame.

Lifecycle: cookies expire/delete on termination; hashed sessions/replay tables die with epoch; key cache dies with supervisor; subscriber goroutines cancel and join with session; handles/config/socket files end with owned transport. Latch persists until successful local reset/clean normal stop. Audit rotation is part of append, and startup sweeps its own expired segments. No remote operation deletes another store's files.

## Implementation and review boundaries

Operator approval received; run change-code before any task implementation. TL has settled the #758 acceptance sequencing; agree the typed adapter contract with its eventual owner before implementation of that seam. Then run central `sdlc change-code --issue 759`; its plan-quality gate owns the mandatory review and estimate.

### M1 — Auth policy, lifecycle and durable authority

Files: new `cmd/internal/remotesession/{state,policy,protocol,session,store,origin_store,access,audit,documents}.go` and colocated tests; `cmd/internal/artifactpath/remote.go`, `manifest.go`; dependency files if the reviewed verifier is new.

- [x] Use the agreed #759-security/#758-document adapter contract and stateful backend; coordinate the exact Go seam with its eventual owner. No real-nvim dependency for this security foundation.
- [x] Write the failing tests for the ReduceRemote and policy rows of the function-strategy table below; that table is the single unit-test obligation inventory.
- [x] Run `go test ./cmd/internal/remotesession -run 'TestRemoteState|TestRemotePolicy' -count=1`; observe red before implementation, then green.
- [x] Implement the typed reducer, durable store and session policy using the Store, codec and Sessions function-strategy rows; run their red/green tests.
- [x] Implement the stateful signed issuer/key backend and AccessVerifier.Verify strategy row; run `go test ./cmd/internal/remotesession -count=1` red/green.
- [x] Implement Audit, OriginStore and Replay using their named function-strategy rows, including the counted checks specified there.
- [x] Update `atlas/remote-session.md` and `atlas/index.md`; commit coherent steps with agent trailers; `sdlc milestone-close --issue 759 --milestone M1` with evidence.


### M2 — Exclusive exposure, local reset and revocation

Files: `broadcast/cloudflared.go`, `host.go` and tests; new `couchcmd/remote.go`, `remote_cli.go`, tests; new `couchtty/console_remote_session.go` and tests; console wiring/input/status files; local lint catalog guard.

- [x] Extend FakeTunnel and implement the ProtectedTunnel.Open / Host.Mount function-strategy row test-first.
- [x] Wire remote-only Host to protected named transport, existing guard and lifetime lock. Ordinary cast/replay holders produce explicit busy; no shared mounts, no unauthenticated public readiness probe, no quick/loopback fallback.
- [x] Implement the M2 security portion of the RemoteServer.ServeHTTP function-strategy row using the real private HTTP server, protected Unix origin and stateful signed issuer; document routes/backend dispatch qualify in M3.
- [x] Wire 👽 click toggle/show-link and CLI on/off/status plus authenticated remote kill; no CLI/HTTP reset opcode. Every re-enable rotates link passcode and revokes old cookies/streams. Add an unexported host-key reset closure. Paste/mouse/reply/release and child-output bytes never invoke it.
- [x] Implement the Console.routeRemoteSessionKey row through the real broker/wrapper and owned-console fixture paths.
- [x] Implement the matchRemoteConsoleReset function-strategy row as pair-local catalog rule remote-console-reset, citing #759.
- [x] Complete the ReduceRemote and Store lifecycle/session sequence strategies against the integrated exposure owner, including counted8/16 open/close controls; Supervisor.Dispatch and audit-handoff strategies qualify in M3.
- [x] Run targeted remote/broadcast/couchtty/couchcmd/wrapcmd tests plus relevant race tests, update README/atlas/CLI help and commit the local M2 qualification checkpoint.
- [x] Milestone-close M2 after the hub lifts the DNS/push hold; mandatory fresh review finalized SHIP, with BR-5/6/7/8 disposed.

### M3 — Trusted submits, document integration and acceptance

Files: extend existing `remotesession/documents.go`; new `couchcmd/remote_submit.go`; `couchmessage/{model,batch,broker}.go` and endpoint/accounting consumers; session-sync provenance ledger/owed projections; #758 adapter at its agreed owner path; tests and docs. The M3 execution slice below fixes the concrete ordering and qualification against the delivered M1/M2 seams.

- [x] Implement the M1-agreed document adapter seam and its stateful backend using the DocumentBackend and Supervisor.Dispatch rows; leave #758's RPC/editor implementation with its owner.
- [x] Exercise the RemoteServer.ServeHTTP, DocumentBackend and Supervisor.Dispatch strategies end to end before connecting the actual broker.
- [x] Add broker-origin handling without reusing OriginSystem; carry remote provenance through endpoint validation, batches, receipts, delivery accounting and preserved supported persisted fields. Keep the canonical submitted path line unchanged. Local legacy submits stay compatible; older incompatible wrappers visibly refuse remote delivery.
- [x] Complete the Broker.SendRemote / Envelope / deliveryRecorder.record / OwedForDocument and proxy.dispatchPeer / peerTurnRecorder function-strategy rows, with the native-transcript/correlation distinction below. The operator dropped ariadne#520; implement provenance, not an agent confirmation policy.
- [x] Record the #758 adapter contract and qualify #759 against its stateful backend now. Per TL sequencing, real-nvim edit/submit acceptance runs after #758/parley.nvim#405 integrate; it does not block this security foundation. The #759 security harness is not real-nvim acceptance.
- [ ] Run targeted packages and reader inventory audit on committed HEAD; push; recorded affected/full check per current gate contract; required final close review. Do not weaken deadlines or bypass failed gates.
- [ ] Operator-assisted live conformance only after separate authorization for setup/staging/network: approved named tunnel + Access policy; iPhone login as allowed user, refused unapproved identity, raw forged/missing/wrong-audience assertion refused before handler, no artificial idle/absolute expiry, silent Access-bound re-handshake, explicit sign-in-needed on invalid Access, exact email/IP header and transcript paper trail, console untagged, kill by a second authenticated session, off/on old-link revocation, old tab/session dead, restart still killed, CLI/--keys unable to reset, console reset then fresh login succeeds. Check no owned stage/connector remains after teardown. Record exact build and outcomes without secrets. Before #758, this live pass covers Access/status/expiry/kill/reset only; tagged-submit behavior is verified against the private stateful backend and actual broker. The real document edit/submit and nvim handover portion is deferred to #758 integration per TL; use the stateful backend here and make that limit explicit. Do not expose a test document backend on the operator's public tunnel to manufacture acceptance.
- [ ] After operator acceptance and SHIP, run `sdlc merge --yes`, verify the actual car admission, final Git/PR/tracker outcome, and report to TL. One week of daily use belongs to the project, not a false claim in this close.

## Risk functions and mechanical defenses

| Named risky function / location | Adversarial class | Mechanical guard |
|---|---|---|
| `AccessVerifier.Verify` / remotesession/access.go | forged/expired/not-yet-valid/rotated/wrong-app or wrong-algorithm JWT, missing human identity, attacker key URL/redirect/SSRF, oversized claims and request-IP spoof | real-signature stateful issuer + generated claims/header inputs; pinned issuer/AUD; count one coalesced refresh at N/2N |
| `ReduceRemote` / state.go | kill during every lifecycle phase, parent death/restart, late success, duplicate effects, normal cast/replay open while remote owns transport | generated event sequences against independent no-grant-after-kill/old-epoch invariants; deterministic held-effect rendezvous |
| `Store.Arm`, `Store.Kill`, `Store.ConsoleReset` / store.go | process death/IO failure after each write/sync/rename | stateful fault filesystem and reopen after every publication boundary; assert armed exists before any transport open |
| `decodeLatch`, `decodeOrigins`, `decodeAuditRecord`, `Entry.UnmarshalJSON` / named codecs | duplicate/wrong-type/truncated/unknown nested fields and unsupported authority version | fuzz bounded decode; newer-record fixture through Arm/Kill/reset, MarkRemote/GC, audit rotation and actual Ledger.trim; independent raw-field equality oracle |
| `Sessions.Handshake`, `Sessions.Admit`, `Sessions.Logout` / session.go | cap race, revoked cookie, valid long idle, JWT expiry, stale passcode and delayed stream | fake-clock generated sequences; four-session race rendezvous; no fifth admission; off/kill/rotation close all streams, silent valid-Access handshake |
| `Replay.Admit`, `Replay.Complete` / replay.go | duplicate/conflicting/evicted/out-of-order sequence, uncertain commit | generated monotonic sequences and independent dispatch-count oracle; evicted requests cannot dispatch, result ring bounded at N/2N |
| `RemoteServer.ServeHTTP` / server.go | unauthenticated/forged-assertion dispatch, arbitrary method/path and attempts to reach cast/drive/IPC/file/nvim routes | real private HTTP server + stateful issuer/doc backend; generated method/path requests assert zero unauthorized backend calls |
| `matchRemoteConsoleReset` / internal/repolint | future reset callback escape or overbroad test exemption | AST catalog positive/negative corpus and every tracked production call site; a seeded foreign-call mutation must fail |
| `Supervisor.Dispatch` / supervisor.go | kill during audit wait/between permit check and backend handoff | shared linearization mutex; hold each boundary in stateful backend; assert kill-before-handoff has zero effects and handoff-before-kill remains in flight |
| `ProtectedTunnel.Open`, `Host.Mount` / broadcast transport | anonymous fallback, another mount/late open/reopen after kill | existing exclusive lease + FakeTunnel lifecycle count tests, private socket and strict ingress golden; no token route reaches remote adapter |
| `Console.routeRemoteSessionKey` / couchtty/console_remote_session.go | IPC/--keys/paste/replies/mouse treated as reset | actual broker->wrapper negative route plus owned host-input positive control; AST call-site guard forbids reset authority escaping that key branch |
| `DocumentBackend.ResolveNavigation`, `DocumentBackend.Dispatch` / documents.go contract | foreign/stale/forgotten/renamed ID or editor generation, revision conflict, link/jumplist traversal, rebind at handoff and ambiguous commit | stateful ID/revision backend and generated invalid refs; assert no nvim open before destination authorization; #758 real adapter integration later |
| `Broker.SendRemote`, `Envelope`, `deliveryRecorder.record`, `OwedForDocument` | forged source, exact remote email/IP prefix vs untagged console, tag stripping/batch loss, restart/editor-handover/local-submit laundering and local forgotten-ID retirement | real broker/endpoint/ledger/trim/owed round trip; exact remote header + receipt fields; restart/handover origin-store tests and incompatible-wrapper refusal |
| `proxy.dispatchPeer`, `proxy.advancePeer`, `peerTurnRecorder`, `pairlog.PersistPeerTurn` / wrapcmd and pairlog | remote prefix absent from second paper trail, raw-credential leakage | deliver actual remote envelope in private fixture, reopen the normal transcript and assert exact prefix/body with no JWT/cookie/link passcode |
| `Audit.AppendIntent`, `Audit.AppendResult`, `OriginStore.MarkRemote` | intent failure, committed effect with missing result, disk-full/rotation/capacity | injected boundary errors + preserved uncertain request; no blind replay; audit intent/result/backend counts at8/16 requests |

## Architecture cross-check and sources

- ARCH-DRY: reuse transport/guard/lease, registry identity, broker origin and parley submit; no duplicated nvim or Cloudflare client stack.
- ARCH-PURE: policy/state/admission are value functions; effects injected and returned as events.
- ARCH-PURPOSE: all routes, sessions, edits/submits and recovery respect the same fence; an absent adapter never becomes an exemption.
- ARCH-MOCK: stateful tunnel, signed issuer/JWKS, portable store and document backend; operator live conformance before product acceptance, repeat after relevant cloudflared/Access changes.
- ARCH-CONSTRAINTS: bounds and counted tests above; disk/network work outside console loop.
- ARCH-SECURE: trust table, typed inputs, secret exclusions, local verification and explicit same-UID limitation.
- ARCH-ORDER: reducer, epoch permits, ordering, uncertain results and restart semantics above.
- ARCH-FUNERAL: every cache/session/worker/process/artifact has an owner, end or retention bound.
- ARCH-EVOLVE: supported persistent records preserve unknown fields; unknown authority versions block. Remote origin is additive but cannot be silently lost by old wrappers.

Cloudflare primary documentation checked 2026-10-21 (read-only public browsing):

- [Origin Access enforcement](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/configure-tunnels/origin-parameters/#access): ingress Access JWT validation before proxying.
- [Validate Access JWTs](https://developers.cloudflare.com/cloudflare-one/access-controls/applications/http-apps/authorization-cookie/validating-json/): team JWKS, issuer/AUD and Go verifier example.
- [Cloudflare client-IP header provenance](https://developers.cloudflare.com/fundamentals/reference/http-headers/#cf-connecting-ip): proxy chain and Worker caveats; IP is not JWT-signed.
- [Session management](https://developers.cloudflare.com/cloudflare-one/access-controls/access-settings/session-management/): Access application/global sessions are distinct; Couch's short session is independently bounded.

## Approval record

Operator approval: received with changes, TL7c5b26ce-a932-4fb1-8d97-21c12f04fa05; current contract above and Revisions below record them. TL acceptance sequencing: stateful document backend now, real-nvim drill after #758/parley.nvim#405 integration (e205ff19-08b2-4909-be86-ea8e4d728d82). Account/network/staging authorization: none. No product implementation yet; plan and issue are checkpointed on the claimed branch. Prior unrelated detach note was reverted; investigation evidence stays in ops.


## Revisions

### 2026-10-21 — advisory fresh-eyes review before operator submission

The initial draft left backend ownership/handoff, durable edit origin and navigation prevalidation implicit. Refined the whole contract before implementation: #759 owns HTTP/security, #758 is a typed adapter agreed before M1; kill and nonblocking mutation handoff share one linearization boundary; conservative per-registration origin survives process/editor lifetimes and has an explicit local retirement policy; navigation validates its destination before nvim opens it. This is one advisory editing pass, not approval. The operator reviews the proposed convenience/security tradeoffs and TL settles #758 scope ownership before change-code.


### 2026-10-21 — TL expiry-UX clarification and integration sequencing

Per receipt e205ff19-08b2-4909-be86-ea8e4d728d82, state the expiry behavior explicitly for the operator's pending review: Couch session replacement is silent within valid Access authorization (absolute expiry immediately; idle expiry on next user activity), with no old-session resurrection or mutation replay. Access rejection requires its normal sign-in flow; kill never renews. The bounds rotate a Couch credential and do not require a human prompt every15 minutes. TL also settled #758 sequencing: stateful document backend for #759 now, real-nvim drill at integration after parley.nvim#405 and #758. Code remains held pending the operator's answer.


### 2026-10-21 — exact remote-edited receipt key

TL d191c1f3-8552-4c95-8488-16bd73e45ca4 requested a stable JSON key for ariadne#520. Named `Receipt.Message.RemoteEdited` (boolean, server-assigned; true is positive durable remote-edit evidence), independent of `Receipt.Message.Origin = "remote"` for direct phone submits; mirrored as `RemoteEdited` in submit/delivery and owed projections. Missing/false is not proof of a clean local source. No implementation or approval status changed.


### 2026-10-21 09:04 — operator APPROVED with changes (go to change-code)

Authoritative receipt:7c5b26ce-a932-4fb1-8d97-21c12f04fa05. This approval supersedes the earlier pending-review/session-timer/header proposals retained above in revision history:

1. 👽 click at the console toggles ordinary remote mode, shows the link on start and stops on the next click; CLI on/off equivalent. Re-enable rotates the link passcode so old links die. Every off/rotation revokes all sessions. The persistent remote-kill latch remains distinct and only a host keypress resets it.
2. Any authenticated allowed remote user may kill, off only; no initiating-user privilege, enable or reset route.
3. Exact visible remote prefix is `[remote <Access email> <client IP from verified Cf-Connecting-IP>]` before the message; console submits have no remote prefix. Receipt Origin and RemoteEdited stay as designed. Identity is JWT-verified; IP is parsed from the trusted named-tunnel/private-origin proxy chain, not claimed to be cryptographically bound to JWT. Preserve actual submitted identity/IP in the transcript as a second paper trail.
4. Remove proposed15-minute absolute and5-minute idle Couch expiries. Sessions last until off/kill/logout, bounded by Access JWT and its Cloudflare-configured lifetime; valid Access permits silent re-handshake. Cap remains four. Replay storage is a bounded monotonic-sequence window, not a reason to force session expiry.
5. Keep Access app/account/network setup with the operator. No configuration changes, live tunnel, staging or deployment authorized yet; TL will arrange setup when requested. Implementation may proceed after change-code gates.


### 2026-10-21 — plan-quality round1 disposition

PQ-1 addressed across every changed persisted format: named retained-raw-object codecs, shared duplicate-key rejection, exact known-field validation, nested preservation, actual rewrite-path future-record tests and the honest single-writer/unsupported-rollback boundary. PQ-2 addressed by replacing the broad risk table with named function-level adversarial classes and mechanical strategies; generated invariants, held interleavings and operation counts remain mandatory. No operator-facing security policy changed.


### 2026-10-21 — plan-quality round2 full test-obligation sweep

PQ-1 closed. PQ-2 remained because milestone prose duplicated adversarial test inventories and the transcript row lacked actual function names. Removed those duplicate unit-case inventories across all M1–M3 steps; steps now reference the single function-strategy table. Added RemoteServer.ServeHTTP and matchRemoteConsoleReset rows and named proxy.dispatchPeer/advancePeer, peerTurnRecorder and pairlog.PersistPeerTurn for the actual transcript seam. Expanded the single rows to retain the displaced adversarial classes. Manual operator acceptance remains explicit; generated sequence and counted-operation obligations remain unchanged.


### 2026-10-21 — additive raw-number validation approved during M1

TL87a8195f approved a shared preserve-number duplicate-validation entrypoint used only by the new retained-raw remote stores. Existing strict readers and their pinned acceptance remain unchanged. This resolves the observed valid-JSON unknown1e999 rejection without broadening legacy readers; known authority fields still undergo exact typed validation. Atlas records the distinction. Publication uncertainty is interpreted from actual durable evidence: current writers block after errors; an off record already published by an authorized host reset or confirmed ordinary teardown may recover off, with no automatic enable and no claim that failed sync rolls back rename.


### 2026-10-21 — M1 document seam checkpoint

The Go contract now lives in remotesession/documents.go and protocol.go: opaque registration/editor references, typed read/edit/submit/navigation, destination resolution before navigation, nonblocking owned-queue Dispatch and one-use Permit.EffectDoor sharing the supervisor's lifecycle mutex. The stateful private backend qualifies rebind/kill/forgotten-target ordering; it is not exposed and does not represent real-nvim acceptance. Join quiesces remote work, not the editor process. M1 foundation and seam are implemented, boundary review pending; all console/CLI/transport/runtime wiring stays in M2. The earlier approval checkpoint's “no implementation” statement describes that historical moment, not this progress.


### 2026-10-21 — M1 review class corrections

BR-1: the actual Sessions owner is INTEGRATION (OS entropy plus mutex-owned sessions and revocation), not a nonexistent pure SessionState. The Core concepts tables above now name actual types; AccessIdentity.validAt/samePrincipal are pure value checks, while session lifecycle/entropy stays in the integration table. BR-2 retains the planned generated fake-time sequences and controlled admission rendezvous; these now exercise revoke/logout/expiry while entropy completion is held, plus independent capacity and historical-cookie invariants. BR-3 replaces every unique remote publication temp family with one canonical reusable stage per record, resolved by artifactpath. Validated openers reclaim only their exact regular bounded slot, refusing unsafe/oversized stages. Latch/origin stages have the record caps; each of two audit stages has the16MiB segment cap, so published data plus all possible crash-left slots are bounded by66MiB+8KiB. No stage is authority, and no glob or automatic repair of invalid committed evidence runs. The single leased Couch writer owns cleanup before publication/on reopen. These are implementation/lifecycle corrections, not an operator-policy change.


### 2026-10-21 — M1 repeated concept-map finding, complete delivered-row audit

BR-4 exposed a second instance in the core-concept-classification family: DocumentResult is declared in documents.go, not protocol.go. Applied the whole rule: every delivered row now names actual declarations at their actual files, with IO/entropy/mutex owners classified as integration and future wiring explicitly marked planned. Split protocol values from document values; included the real pure replay and artifact-path entities; separated Store/OriginStore/Audit; mapped AccessVerifier/accessKeys to their respective files; classified Permit/admissionGate with the integration seam. Verified every delivered row against the declarations and imports. No runtime change; prior BR-1/2/3 were all disposed addressed by round2.


### 2026-10-21 — M1 SHIP, capture/link transfer refinement and project coordination

M1 closed SHIP at4f9bb9b371b5 with all four findings disposed. During M2 read-only wiring inspection, confirmed opt-in terminal capture records final host-write bytes even for the Presenter operator-only layer. TL7b706dfa-f54f-499d-8d0f-6e0456653c49 approved refusing remote enable whenever COUCH_CAPTURE_DIR is configured, with exact recovery: “unset COUCH_CAPTURE_DIR and restart Couch, or keep capture and skip remote”. CLI on/off prints action/status only; link appears solely in Couch's own console, never a Pair child transcript. Add a bounded locally generated QR alongside the text link where geometry fits; retain text/copy fallback in small terminals. No network QR service or generated image artifact. Terminal-app scrollback stays within trusted-account scope; Pair's own capture does not persist the capability.

The TL-owned remote-session project records the operator dropping ariadne#520 at10:00. This issue preserves the approved origin/RemoteEdited and email/IP transcript provenance but implements no confirmation policy; historical references above describe the earlier project state. #760 owns the phone editor web app; #759's live pre-#758 surface remains authenticated security status/kill/logout, never a public fake document backend.


### 2026-10-21 — M2 lifecycle and protected security surface checkpoint

Implemented protected transport with one opening per reservation, confirmed-only teardown, private Unix origin and existing named lease/guard. A held lifecycle fixture reproduced repeated connector openings (2–4) after a lost owner; the protected generation fence now refuses that reopening. Supervisor serializes persistent commands in acceptance order while revoking immediately under the shared admission mutex. Reset waits for transport, requests and persistence; even an idle kill requires its durable join. Unknown publication or cleanup stays blocked. Lifecycle state transitions remain in ReduceRemote, including an explicit poisoned-completion event.

The initial RemoteServer exposes only signed-Access security page/status/handshake/logout and fixed session-independent kill, with bounded request lanes and socket-body deadlines. M3 document operations and broker origin are still pending. Browser same-origin GET may omit Origin; any present foreign Origin and every mutation without exact Origin refuses. The console decodes Ctrl+Alt+R only from a non-repeat host keypress and renders capability text/locally generated QR exclusively in its operator-only layer. CLI remote on/off/status has a closed reset-free IPC shape and prints only action/status. A pair-local AST guard pins reset capture/invocation and rejects alias escape/foreign callers; production inventory qualification is next. No public exposure or account/configuration change.


### 2026-10-21 19:55 PDT — M2/M3 boundary and local qualification

TL04dd04f3 approved explicitly separating the overlapping checklist obligations: M2 closes on protected exposure, lifecycle and session controls; document dispatch and audit-handoff qualification remain M3. This records the existing implementation/atlas boundary, not removal of an obligation. The original M2 rows named the complete RemoteServer strategy with a document backend and Supervisor.Dispatch sequences; those document-operation portions remain mandatory under M3's first two rows and the shared strategy table. No document/edit/submit route is claimed implemented in M2.

TL616b174f resumed M2 after #776 landed. Operator retraction4d0da020 keeps the original separate console 👽 remote control; cast drawing still uses annotation permission. The implemented click/CLI toggle, link/passcode rotation, authenticated off-only kill, host-key-only reset, four-session cap and capture refusal are retained. M2 tests/docs/reader audit completed locally; mandatory milestone review remains pending. TLa9bb056d explicitly holds milestone-close and pushes until DNS returns because the verb automatically pushes. Commit locally now, with no remote transport/account/network changes.


### 2026-10-21 — M2 recovery projection correction (BR-7)

Review round2 disposed socket-budget/help BR-5/6 with independent RED controls, then identified failure-recovery-visibility: busy transport recovery was collapsed internally and operator surfaces dropped safe Detail. Restore the existing explicit recovery contract across configuration, admission/startup, persistence and cleanup projections. CLI may print secret-free status detail along with action/status; it still never prints the link/passcode or arbitrary connector output. Console notices retain the same classified detail plus host-key recovery. Busy still revokes and joins under the existing latch; no new refusal, reset authority or lifecycle state. Document operations remain M3.


### 2026-10-21 — BR-8 transport cleanup confirmation for every opening outcome

Review round3 disposed BR-7 but found a Critical missing outcome: killed opening can return no mount after failed connector cleanup, and the supervisor incorrectly inferred successful teardown. TLbe526c06 approved requiring teardown evidence independently of mount presence and joining the same attempt when cancellation meets an already-closing owner. Existing blocked state and reset refusal remain the recovery; no new remote state or reset authority.

Implementation retains the Host's cleanup error through the explicit broadcast.ErrProtectedCleanup classification, including subsequent calls on an uncertain Host. An epoch stores that outcome before its started acknowledgment; its cleanup join consumes it even when no Mount was returned. Ordinary busy remains separately classified. The shared unwanted-owner rule returns the existing close attempt instead of returning early. Lifecycle qualification includes canceled held-open/failed-close, failed-open/listener-close failure and held-close cancellation; tests assert Join error, blocked status, reset/on refusal and one exact owned close attempt. No raw connector error reaches operator status.


### 2026-10-21 — M3 execution refinement after M2 SHIP

TL41f4e736 requested dispatch/audit/provenance planning while the retained M2 suite ran. TLb16a99b4 confirmed the two source-discovered boundaries: the existing eight-turn/256 KiB peer recorder remains a correlation aid, while the native agent transcript is the operator's second paper trail; forget uses Couch's sole origin writer and a durable intent in the existing origin record, with offline refusal “start Couch and retry”. No registry tombstone, extra origin writer or second unbounded transcript. This refinement preserves the approved security policy, M3 review boundary and operator-last acceptance sequencing. Earlier claims about the generic transcript seam are qualified by the distinction below.

## M3 execution slice — dispatch, durable origin and trusted delivery

This slice is implementation planning, not a claim that these routes exist. M1/M2 are closed; M3 remains one review boundary. The existing risk-function table owns the adversarial classes; tasks below name concrete producer/consumer paths and additional source-discovered cases. No new Cloudflare configuration, network exposure or operator-session action is authorized.

### Concrete concepts and ownership

| Concept | Exact location | Classification / status |
|---|---|---|
| `Message`, new `RemoteSource`, remote origin and rendered envelope | `cmd/internal/couchmessage/model.go`, `batch.go` | PURE values/projection; planned modifications; keep Message comparable |
| `Replay` and typed document request/result | `cmd/internal/remotesession/replay.go`, `protocol.go`, `documents.go` | PURE decisions/values already delivered |
| `Supervisor.Dispatch`, owned operation completion | `cmd/internal/remotesession/supervisor.go`, new `dispatch.go` | INTEGRATION; planned session, audit, backend and join wiring |
| `RemoteServer` document routes and streams | `cmd/internal/remotesession/server.go`, new `server_documents.go` | INTEGRATION; planned verified HTTP admission and bounded streaming |
| `OriginStore` and pending-forget record fields | `cmd/internal/remotesession/origin_store.go`, `origin_codec.go` | INTEGRATION owner and PURE retained-raw codec respectively; extend existing family |
| Trusted submit and local provenance admission | new `cmd/internal/couchcmd/remote_submit.go`, existing `message_service.go`, `remote_service.go` | INTEGRATION; exact registry identity, service lifetime and broker enqueue |
| Local forget routing and transaction | `cmd/internal/couchcmd/session_sync.go`, `cmd/internal/sessionsync/registry.go` | INTEGRATION; coordinate existing registry lock with sole origin owner |
| `Broker.SendRemote` and receiver capability | `cmd/internal/couchmessage/broker.go`, `endpoint.go`, `endpoint_batch.go`; `cmd/internal/wrapcmd/peer_runtime.go` | INTEGRATION; planned trusted admission and compatibility refusal |
| Delivery `Entry`, retained-raw codec and owed projection | `cmd/internal/sessionsync/ledger.go`, new `ledger_codec.go`, `owed.go`; `cmd/internal/couchcmd/message_deliveries.go` | ledger/recorder INTEGRATION; codec/value projection PURE; planned changes |
| Existing peer turn correlation and actual agent input | `cmd/internal/wrapcmd/peer_turn_log.go`, `peer_delivery.go`; `cmd/internal/pairlog/runcli.go` | INTEGRATION; preserve correlation bounds and qualify native transcript separately |

One service-lifetime OriginStore instance is shared by remote operations, ordinary local submits and local forget even while remote mode is off. The existing Couch singleton is its lifetime resource (ARCH-DRY / single writer). No CLI opener may independently rewrite its cached record. Registry transactions retain their existing lock; no filesystem, audit or broker wait runs under the admission mutex. Resolve under registry transaction, release it, then revalidate the pinned ID/path/editor generation at effect admission; do not introduce inverted lock nesting.

### Task 0 — correct the executable documentation example

Files: `cmd/internal/couchcmd/skills/couch/SKILL.md`; existing regression `cmd/internal/couchcmd/messages_test.go`.

- [x] Preserve the recorded RED from M2's completed suite: `TestSkillDocumentsRecovery` treats literal `couch --remote on|off|status` as executable and correctly rejects it. Replace the executable example with concrete on, off and status commands; synopsis prose may describe alternatives outside executable examples. Do not relax the parser or scanner.
- [x] Run `go test ./cmd/internal/couchcmd -run '^TestSkillDocumentsRecovery$' -count=1`, expect PASS. Commit the correction with its source failure. This is a branch defect, not a flake or resume candidate.

### Task 1 — origin ownership, local submit and exact-ID forget

Files: origin owner/codec and Couch service/registry files in the concept table; add `cmd/internal/couchcmd/remote_submit_test.go` and extend origin/registry tests.

- [ ] First add failing integration tests for local submit after remote edit, off/restart/rename/editor handover, unreadable evidence, concurrent forget, path reuse and crashes at each forget publication boundary. Use real private stores and the registry, not function-call-only mocks.
- [ ] Open the sole origin owner with the message service and inject it into the remote supervisor. A local `submitted: PATH` must resolve the registered stable ID/effective owner and consult this owner before broker admission; peer request JSON cannot supply origin fields. Never infer clean provenance from missing/false legacy evidence.
- [ ] Route local forget through this Couch owner. Persist bounded `forget_pending: true` in the existing per-ID retained-raw origin object before registry removal; it is a local intent, not a remote-edit assertion. Adjust `Marked` to inspect validated `remote_edited` rather than mere map membership, so an intent for a never-remote-edited registration cannot create false edit evidence. Preserve unknown envelope/per-ID fields and the existing version, 4096-entry/1 MiB bounds; capacity/publication failure refuses before removal.
- [ ] Fence new work for that exact ID, cancel/join owned document operations and settle its submit receipts. Remove the registration only through its existing identity-checked transaction. Retire the exact absent ID only when its persisted intent and settled work prove the retirement. A replacement at the same path has a different ID and remains untouched. On restart retry the retained intent; arbitrary absence is never authorization to retire. Unknown receipts retain intent/evidence and require inspection, not automatic re-execution or eviction.
- [ ] Offline forget refuses with “start Couch and retry”; no independent origin writer, no new registry tombstone format. Test the actual CLI/server route, including a lost reply and retry after registry removal. Run focused origin/registry/service tests RED then GREEN and commit this unit.

### Task 2 — owned dispatch, audit ordering and typed HTTP routes

Files: `remotesession/{dispatch,server_documents}.go` (new), existing `supervisor.go`, `server.go`, `session.go`, `documents.go`; add corresponding dispatch/server integration tests.

- [ ] Build failing held-boundary tests using the existing private stateful document backend, real stores, signed issuer and fake clock. Cover kill at each admission/audit/handoff/result boundary with independent effect counts, and restart/uncertainty recovery. Record zero effects for refused/expired/stale refs.
- [ ] Authenticate to an internal immutable request context containing Access identity and the normalized verified proxy IP; never decode either from a browser-supplied source object. Pin stable document ID, current path, owner and editor generation before work is queued. Resolve navigation destinations before any open. Missing backend remains unavailable; inject the typed seam for tests/#758 without exposing a public fake backend.
- [ ] Reserve replay and owned completion state, persist conservative edit evidence before mutating handoff, then persist audit intent outside the lifecycle mutex. Recheck the same epoch/session/ref under the admission gate, and hand off via the existing nonblocking `DocumentBackend.Dispatch`. The backend's one-use `Permit.EffectDoor` guards the final nonblocking effect enqueue; never call blocking `Broker.Send` or do IO inside it. A submit's save-and-submit work is one owned typed operation, not an unchecked second request with a copied grant. Coordinate #758 on this exact contract before adapter integration.
- [ ] Keep result/audit completion owned after HTTP disconnect or timeout. Intent failure gives zero effects; post-effect/result-publication failure retains uncertainty and revokes remote mode. Cleanup/reset waits for joined owned work. Replay returns cached/pending results without a second effect; evicted sequences refuse, and conflicting payloads refuse. No automatic replay of an uncertain mutation.
- [ ] Add only the closed list/read/subscribe/edit/submit/navigation routes under the current capability. Preserve Access, same-origin, CSRF and reserved kill admission. Apply the five-second exchange budget to ordinary requests and stream establishment, not the entire subscription lifetime; stream lifetime follows JWT/session/epoch, one stream/session, bounded writes and 1 MiB output. Session traffic is 10 mutations/s, burst20; process request lanes stay capped at eight.
- [ ] Run targeted remotesession dispatch/server/replay tests RED then GREEN, followed by race qualification of these owners. Commit the working slice before broker wiring.

### Task 3 — trusted broker provenance and compatible receiver admission

Files: `couchcmd/remote_submit.go`; `couchmessage/{model,batch,broker,refusal,endpoint,endpoint_batch}.go`; `wrapcmd/{peer_runtime,peer_delivery}.go`; colocated tests.

- [ ] Add failing actual broker/endpoint/batch tests for direct remote versus marked local submission, forged metadata, duplicate IDs with different provenance and old receiver refusal. Retain legacy peer/system controls.
- [ ] `Broker.SendRemote` accepts only trusted typed server facts. Set `Origin="remote"`, comparable `RemoteSource{Email, IP}` and independently derived `RemoteEdited`; leave Body canonical `submitted: <current registered path>`. `Envelope` places exact `[remote <email> <IP>]` before that body. Local submits never acquire this display header just because RemoteEdited is true.
- [ ] Add a closed receiver capability query/response advertising `remote-provenance-v1`; unknown/legacy capability refuses remote before admission and names relaunch as recovery. Do not broaden legacy hello decoders or alter binding/socket identity. A strict old receiver must never receive a downgraded local submit; legacy peer delivery still works.
- [ ] Carry and compare metadata through every Message constructor, refusal receipt, deduplication, retained promotion, singleton reserve/commit/status, prepare/finalize batch and fallback. Keep Message comparable. Do not recover remote authority solely by adopting a wrapper's retained receipt; keep existing peer-only adoption. Apply remote deadline fencing as for privileged system recovery. Count rendered prefix bytes in message/batch bounds.
- [ ] Run couchmessage/wrapcmd provenance, compatibility and batch tests RED then GREEN; inspect all Message constructors/equality sites before committing. Browser and worker text resembling the prefix remains untrusted ordinary text.

### Task 4 — ledger, owed records and the transcript paper trail

Files: sessionsync ledger/codec/owed, Couch delivery recorder and wrapper/pairlog locations above; extend their real-writer tests.

- [ ] Seed future nested fields, then test actual deliveryRecorder.record -> Entry append -> Ledger.trim -> reopen -> OwedForDocument. Add exact known-field validation, duplicate rejection and unknown-preserving root/RemoteSource codecs; raw maps belong in ledger codec state, not comparable Message. Recognized malformed provenance produces a recording gap/refusal, not a silently skipped clean-local row. Preserve current ledger size/age/row caps.
- [ ] Preserve Origin, RemoteEdited, RemoteSource and admission document identity where needed through fresh recorder/owed structs. Keep historical path/time attribution for legacy rows; new identity cannot migrate to a replacement registration merely sharing a path.
- [ ] Qualify actual wrapper PTY delivery with exact prefix, canonical body, delivery ID and receipt metadata. Native transcript producers/readers use `sessioninventory.OSRuntime.NativeRoots` and poisoned-identity private fixtures; no inherited live namespace. Qualify Claude and Codex; Qoder when credits permit, recording unsupported/unavailable cases honestly.
- [ ] Keep `peerTurnRecorders` at eight turns / 256 KiB aggregate and best-effort behavior. Its Pair log is a correlation aid only. Assert the native agent transcript contains a remote submit after turn eight as well as earlier submits; do not claim correlation captures every submit or create another unbounded format. Native transcript fixtures qualify recording behavior; the operator's real native transcript check remains the integration evidence when #758 supports actual dispatch.
- [ ] Run focused ledger/owed/recorder/wrapper/log tests RED then GREEN, then race checks. Commit the qualified provenance path and update atlas with authority, lifetime, forget recovery and transcript limits.

### States and recovery for the added operation paths

Actions: inspect document/receipt/audit, retry the same sequence, resubscribe, start Couch and retry forget, relaunch receiver, or disable remote and request host reset after cleanup. No reset authority changes.

| Internal state / failure | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Audit intent or origin publication refused before effect | not applied | inspect storage/status | request was not applied; conservative edit marker may remain |
| Handoff accepted; result not yet known or HTTP gone | pending/uncertain | inspect document/receipt/audit | result visibility; never infer effect rollback |
| Same sequence with retained complete result | completed | retry same sequence to retrieve result | none; no new effect |
| Replay result evicted or sequence conflict | inspect | inspect document/receipt/audit | cached response; no automatic rerun |
| Result audit fails after effect | remote blocked, uncertain | disable remote and request host reset after cleanup | durable result line; effect may have happened |
| Forgotten ID intent persisted, registry present or absent | forget pending | start Couch and retry forget | no authority/evidence removal until exact-ID settlement |
| Old-ID receipt remains unknown | inspect | inspect document/receipt/audit | registration/evidence retained conservatively |
| Origin evidence unreadable or unsupported | provenance unavailable | inspect storage/status | no claim of clean local provenance |
| Receiver lacks remote capability | relaunch required | relaunch receiver | no submit admitted |
| Stream expires, revokes or overflows | disconnected | resubscribe after valid silent handshake | intermediate snapshots, not mutation replay |

Covered: local submit while off/restarted, remote edit/submit, navigation, stale identity, forget/re-register, delivery uncertainty and bounded streaming. Not covered: arbitrary files/commands, protection from hostile same-UID code, actual editor RPC implementation (#758), phone editor UX (#760), automatic audit repair or unsupported downgrade writers. Their exclusion follows the approved boundary, not a test exemption.

### Cost, qualification and final handoff

- [ ] Count N=8 and16 unique mutations and duplicates: one backend dispatch per accepted unique request, one intent/result pair when completion is known, zero duplicate effects; no full transcript/registry scan per frame. Include origin store serialized publication work under its explicit 1 MiB/4096-entry cap rather than calling persistent writes O(1). Count capability queries per receiver binding/batch, not per delivered byte. Inspect stream goroutine/queue counts across kill/logout/off.
- [ ] Run `go test ./cmd/internal/remotesession -run 'Test.*(Dispatch|Replay|Document|Server|Origin|Forget)' -count=1`; `go test ./cmd/internal/couchmessage ./cmd/internal/wrapcmd -run 'Test.*(Remote|Origin|Batch|Peer)' -count=1`; and `go test ./cmd/internal/sessionsync ./cmd/internal/couchcmd ./cmd/internal/pairlog -run 'Test.*(Remote|Origin|Forget|Ledger|Owed|SkillDocumentsRecovery|Peer)' -count=1`. Every named new regression must actually run (verify -v output, no empty selector success). Broaden to affected package/race checks after those pass.
- [ ] Audit production sources/readers individually before inventory certificate updates; run repolint, artifactpath and affected-reader checks. Update README/atlas and the #758 handoff. Push the corrected code and record the required affected/full check through sdlc on that head. The M2 own-defect failure cannot become passing evidence via resume. No manual stale-head status or guessed process cancellation.
- [ ] Operator-assisted Access/iPhone acceptance is the final functional step, arranged through TL with explicit setup/staging/network authorization. Use the exact current build, security-only public surface until #758 exists, and approved real document/native-transcript checks at adapter integration. No public fake backend. Record what passed and what remains deferred without secrets; keep CLI links suppressed and capture refusal unchanged.
- [ ] Mandatory M3/issue review and landing gates follow functional acceptance; the binary owns fresh-eyes review. Send outcomes to TL; TL owns project updates. This plan refinement does not close M3, grant network changes or assert a passing full suite.


### 2026-10-21 — M3 forget-settlement scope split to #795

TLd789d04d stopped the third refinement of the forget/evidence mechanism. Retained broker receipts expire, delivery-ledger trimming removes old uncertain rows, and audit retention removes unmatched intents: an empty scan cannot prove lifetime settlement. The bounded read-only scan approved in TL6b2915fd is therefore superseded and is not implemented as settlement proof. Filed unclaimed pair#795, workshop/issues/000795-remote-forget-settlement.md, with the proposed bounded per-ID pending-operation evidence in the existing single-writer origin record, publication/crash ordering, uncertainty pinned across retention, capacity and operator reconciliation questions. That mechanism requires its own full plan review.

Current M3 override: persist conservative `submit_seen: true` before every remote or marked-local submit handoff, never clear it in M3, and refuse local forget for that exact ID with “this document had remote submits; forget is not available yet, see pair#795”. The existing origin record may contain a submit-only marker without remote_edited; neither absence nor false means clean local provenance. Capacity/publication failure refuses before handoff. A refused admission may conservatively retain the bit. Surviving unknown fields remain intact. No receipt/audit/ledger scan may waive the block. Exact-ID forget for never-submitted origins keeps the single-writer intent/registry transaction and active-operation fence; remote-edited-only records can retire only after owned work settles. Ordinary untouched local origins keep their existing behavior through the local Couch owner. Offline forget still says start Couch and retry. Explicit CLI retry after an ambiguous reply is a new local action: inspect the registration first; there is no across-invocation at-most-once promise.

The execution slice's previous settlement scans/recovery suggestions and state table apply only as historical design notes where contradicted by this override. Dispatch/audit/kill ordering, broker capability refusal, retained-raw ledger provenance, local-submit provenance, counted tests and operator Access/iPhone acceptance remain M3. No new pending-operation protocol is implemented here.


### 2026-10-21 — M3 delivered concept/adapter map

The delivered concept rows now name actual declarations, including comparable RemoteSource in couchmessage/provenance.go, DocumentSource in remotesession/documents.go, synchronized sessionOperations in dispatch.go and the IO-bearing server_documents.go. Entry's retained-raw codec lives in sessionsync/ledger_codec.go. These are implementation locations, not a claim of M3 close. Subscribe's existing lifetime contract is explicit: a prompt owned channel handoff with initial editor IO on the backend's joined worker; its stream context must not expire at the ordinary exchange deadline. #758 still owns the real editor implementation and must honor the final Permit gate and subscription lifetime.


### 2026-10-21 — full M3 code milestone tonight; operator staging tomorrow

TLc4102caf requests the full M3 milestone close tonight and the operator checklist, with staging tomorrow after #753/#739 and before #737. This supersedes the earlier intent to use final issue-close as M3's only review boundary. M3 now closes its implemented code/private qualification through milestone-close; whole-issue acceptance and landing remain held for the operator's Access/iPhone check. No live result is inferred from a code milestone. The numbered checklist is [operator acceptance](000759-remote-security-plan.side/operator-acceptance.md), including official Cloudflare setup links and the explicit #758/#760 edit/submit dependency. It names native transcript and metadata audit observations without copying secrets. Retain the existing code-identical queued suite on c3634473; docs-only checkpoint does not cancel/requeue it.


### 2026-10-21 — M3 boundary round1 class corrections

Full milestone review returned REWORK (BR-9/10/11), not M3 completion. BR-9: caller, effective document owner and chosen recipient are distinct; preserve existing cross-slot sends. Origin follows stable registration ID; unmarked local submits keep legacy wire and content-read behavior. New production-service regressions must initialize the actual origin namespace and use a different caller/recipient owner for both marked/unmarked files. The original plan's “self-submit/effective owner” wording must not be interpreted as sender=owner.

BR-10 repeats durable-artifact-lifecycle: failed validation/retention cannot leave a durable writer growing indefinitely. Preflight existing over-threshold ledgers before append, preserve damaged bytes and gap reporting, and retain a fixed bound even across recreated Ledger owners. No repair, truncation or new history store. Repeated8/16 refusal tests cover malformed provenance and IO/retention failures, with size/hash equality oracles.

BR-11 repeats deterministic-state-sequence-coverage: enumerate accepted-effect/result/audit publication against observer departure, session logout/kill and teardown. Controlled held results and held result-audit writes must establish one effect, retained replay/receipt uncertainty and no successful cleanup/reset while owned work is held. Existing pre-admission/intent/EffectDoor tests remain necessary but do not stand in for these post-handoff states. Apply the rule across the owner, not one reported branch.

BR-9 attribution sweep also covers a rename between registration capture and broker timestamp: valid new submit rows use their matching stable DocumentID in both owed and round-clear projections; only legacy rows need historical path/time matching. Independent RED caught the old-path race; replacement IDs remain excluded. BR-10 preflight reserves the encoded next row through the existing trim path before append, rather than merely checking an already oversized file.


### 2026-10-22 — approved dedicated remote tunnel revision

TL 795fb6dc and 05e409a2 approve a dedicated tunnel: remote reads only
COUCH_REMOTE_TUNNEL and COUCH_REMOTE_HOSTNAME. The ordinary cast configuration
is passed only to a collision guard. Equal names or canonical UUIDs refuse
before connector spawn, including name/UUID aliases. Names resolve through
read-only cloudflared tunnel list --output json --name NAME; UUID inputs skip IO.
At most two sequential lookups share a five-second deadline; stdout is capped
at 1 MiB and diagnostics are capped by the existing owned ProvisionIO runner.
Unavailable, malformed, ambiguous or expired evidence refuses and tells the
operator to inspect tunnel settings and retry. Access checks, connector guards,
record ownership, capture refusal, kill/reset authority and session policy stay
unchanged. The operator creates couch-remote and changes its DNS route; agents
make no account/network writes. No shared-ingress owner is added.

Implementation uses the existing couchcore.ProvisionIO seam (ARCH-DRY), whose
OS runner joins cancellation and bounded collection; no new process collector.
One optional DistinctFrom tunnel reference on Cloudflared enables the preflight
only for protected remote transport. Pure UUID/metadata validation lives beside
the bounded lookup adapter. Tests cover direct name/UUID collision, aliases,
malformed/ambiguous/unavailable resolution, two-query/deadline bounds, remote
settings independence and simultaneous cast+remote mounts on distinct tunnels.
New temporary configs remain in the existing listener/run-record lifecycle;
no new durable artifact family or cache is introduced (ARCH-FUNERAL).

Current-main integration retained both endpoint and remote protocol additions,
used the single shared preserve-number checker and combined both regression
oracles. Targeted strictjson, couchtty, couchcmd remote/endpoint checks passed;
couchmessage remote/forget/orphan checks passed independently. The first console
check lacked generated embedded runtime assets; generation resolved setup only.

#### Dedicated-tunnel revision tasks (same issue, one final close)

| Pure entity | Lives in | Status |
|---|---|---|
| Canonical tunnel identity and exact metadata validation | cmd/internal/broadcast/tunnel_identity.go | new |
| Remote-only connector configuration | cmd/internal/couchcmd/remote.go | modified |

| Integration point | Lives in | Change |
|---|---|---|
| Bounded read-only identity lookup | cmd/internal/broadcast/tunnel_identity.go | reuse ProvisionIO, five-second shared budget, 1 MiB output |
| Connector admission | cmd/internal/broadcast/cloudflared.go | collision preflight before claim/spawn |
| Remote service setup | cmd/internal/couchcmd/remote_service.go | remote-only variables, separate cast comparison |

- [ ] RED then GREEN identity validation/lookup and before-spawn refusal using a stateful fake behind ProvisionIO; assert at most two queries and shared cancellation.
- [ ] RED then GREEN remote settings without inherited broadcast selection and concurrent distinct-tunnel cast/remote mounts; retain Access/capture and ownership refusals.
- [ ] Update README/atlas/operator exports and source/reader inventories; targeted package and race checks; final full close reviews integration and revision before landing.
- [ ] Stage only on TL go, report actual binary revision and exports; operator Access/iPhone acceptance remains required and unclaimed until observed.


### 2026-10-22 — operator pivot to Tailscale, approved before code

Operator decision a487d0d1 supersedes the entire remote Cloudflare Access path
and the dedicated-tunnel revision. TL f8ad8c19 approved the complete revised
transport plan below. #803 landed as PR544 before this branch resumed. Its code
is integrated; the shared retained-number codec remains reconciled with #758.
Public cloudflared casts remain unchanged. No Tailscale product code or live
Serve mapping has been created. The Homebrew tailscale-app 1.104.2 installation
still needs readiness confirmation from the operator through the TL.

Threat model and authority

- Protect against an unallowed tailnet user, shared/tagged device, LAN or public
  caller, browser script, forged HTTP identity/IP headers, stale session/link,
  another local UID, and late completion after off/kill. Root and malicious
  code running as the operator's UID remain trusted, as in the existing plan;
  this does not protect against a compromised operator account or tailscaled.
- Remote uses HTTPS Serve only, never Funnel, quick tunnels or the public cast
  host. A nonempty COUCH_REMOTE_ALLOWED_LOGINS is the remote allowlist. Remote
  ignores cast/Cloudflare tunnel, hostname, team, audience and email settings.
- Use a private physical 0700 directory and 0600 Unix socket, no TCP listener
  or loopback fallback. Bind the accepted connection to verified proxy-peer
  evidence in server ConnContext; an HTTP header cannot create that evidence.
  Require kernel Unix peer credentials/PID plus identity-bracketed supported
  Tailscale daemon inspection through procutil.Source. Unavailable or mismatched
  inspection refuses. The exact installed Mac daemon/extension identity must
  be measured; do not infer it from an argv substring or assume UID alone proves
  Serve. The first bounded native probe qualifies this before product code.
- Only after that peer check accept one bounded Tailscale-User-Login and one
  canonical IP in X-Forwarded-For. Reject duplicate, malformed, empty, control-
  bearing or unexpected encoded values; compare the decoded login exactly with
  the configured allowlist. Tagged/missing-login and Funnel-marked requests
  refuse. Pin expected forwarded host/protocol and existing Origin/CSRF checks.
  Identity is authenticated anew on every API request, including kill and
  streams; a browser cookie never substitutes for current ingress identity.

Owner and lifecycle

- Keep the existing transport-independent supervisor and its on/opening/on,
  closing, killed and cleanup-blocked states. Keep the 👽 console switch, CLI
  on/off/status, console-only link/QR, capture refusal, four-session cap, replay
  limits, audit-before-effects, dispatch fencing and host-key-only latch reset.
- Preflight the installed CLI/daemon and logged-in Running state, the canonical
  node HTTPS name and HTTPS readiness with bounded read-only CLI commands.
  Missing install/login/HTTPS permission refuses with one concrete recovery;
  no automatic login, policy/certificate enrollment, sudo or machine settings.
- Run one foreground tailscale serve --https=443 unix:<owned-socket>, without
  --bg, --yes, --service or Funnel. The foreground daemon session is the
  lifetime ownership resource. Native admission must atomically refuse an
  occupied port, including a racing foreground owner; preflight inventory is
  diagnostic, not absence authority. Never overwrite another Serve config.
- Reuse the current guarded-child lifetime/EOF collection model and bounded
  metadata runner. Opening has the existing 30-second envelope; each metadata
  exchange has a five-second cap and 1 MiB output cap; closing retains the
  existing five-second join budget. No new periodic process or liveness loop.
  Publish on/link only after the exact foreground mapping and private-origin
  readiness are confirmed. Ordinary remote requests spawn zero children.
- Off/kill first revoke epoch, sessions, passcode and dispatch admission, then
  cancel/join the owned foreground attempt and close the origin. Verify its
  exact mapping is absent; unrelated mappings may remain. Never issue serve
  reset or a generic off that might erase another owner. Unknown teardown keeps
  the existing blocked state and reset unavailable. A child exit, daemon loss
  or profile switch revokes remote mode; no silent connector restart.
- Sessions have no new idle/absolute timeout. They end on off, kill, logout or
  current identity/epoch loss. Same valid authenticated identity can silently
  re-handshake while mode is on. Re-enable rotates passcode/link and revokes
  every prior session. A stream cannot survive identity/transport revocation.
- Provenance stays Origin=remote and independently derived RemoteEdited;
  delivered text is [remote <login> <IP>]. Use an explicitly named Login field
  in the unlanded remote identity contract rather than treating arbitrary
  logins as verified email. Update receipt/audit/ledger/receiver tests together
  and tell the hub the exact JSON key. Preserve unknown fields and conservative
  historical evidence. Native agent transcript remains the second paper trail.

Visible states and recovery

| Failure/state | Operator sees | One recovery | Possible loss |
|---|---|---|---|
| Missing CLI, logged out, HTTPS unavailable or invalid allowlist | unavailable, specific prerequisite | install/sign in/configure as named, then retry on | none; no exposure |
| Occupied Serve endpoint | busy; existing service preserved | stop/reconfigure that service explicitly, then retry | none |
| Socket inaccessible or proxy identity cannot be proved | unavailable; secure origin unsupported | inspect installed Mac variant, then retry | none; no weaker fallback |
| Opening/cancel race | opening or closing, no usable old link | wait for owned attempt | no submit admitted until ready |
| Healthy on | on; private tailnet link on console only | use phone | existing bounded operation uncertainty applies |
| Unallowed/forged/missing identity | request denied | use an allowed personal login | no effect |
| Off or phone logout | off or signed out | enable or re-handshake, respectively | draft saving follows existing explicit result |
| Kill latch set | killed | host physical keypress, then re-enable | revoked sessions; retained unsent work stays local |
| Cleanup outcome uncertain | blocked; reset unavailable | inspect/recover locally and restart | no claim of removed ingress; effects remain revoked |

Implementation and qualification

1. After #803 lands, record this pivot and operator approval. Coordinate one
   bounded private native Serve probe on the actual Mac installation: socket
   access, kernel peer identity, header replacement/IP, foreground owner conflict
   and daemon-session cleanup. No production listener, global reset, public
   exposure or unrelated mapping change. If secure peer proof/socket access is
   unsupported, stop and return the evidence and simplest alternative for plan
   review; do not quietly fall back to localhost.
2. Replace only the remote transport/auth adapter and its configuration. Remove
   remote Access/JWK/JWT and dedicated-tunnel code, retain ordinary cloudflared
   paths/tests. Keep a pure login/IP authorization core separate from IO.
3. Stateful fake daemon/CLI behind the same seam: install/login errors,
   malformed/capped output, held opening/closing, late success, mapping conflict,
   losing the owner, uncertain teardown, port races and unaffected other Serve
   mappings. Count one foreground child/epoch and bounded metadata operations;
   N/2N requests/connections have linear validation and zero per-request spawns.
4. Negative HTTP/Unix tests: forged direct TCP and unrelated Unix peer denied;
   unavailable peer inspection denied; duplicate login/IP headers, public/Funnel,
   tagged identity, wrong login/host/origin/CSRF and stale cookies/links denied
   with zero dispatch. Identity changes revoke existing sessions and streams.
5. Re-run lifecycle/capture/four-session/kill-reset/replay/audit/provenance and
   native-transcript regressions. Include browser logout/re-handshake, off/kill
   while a request or stream is held, daemon exit and profile change. Full
   packages/race, source and reader inventories, full recorded suite, mandatory
   fresh review. No live-network test is represented by a fake result.
6. Update README, atlas and acceptance steps. Operator installs/logs in on Mac
   and iPhone and enables HTTPS if required. Stage through sdlc on TL go; phone
   opens private link, allowed login edits/submits, transcript/audit show login
   and verified IP, phone kills, only host key resets. Check an unallowed login
   and direct-origin spoof control. Public cast still works independently.

Evidence informing the proposal (installed version still must be qualified)

- https://tailscale.com/docs/features/tailscale-serve — identity header handling
  and the direct-origin spoofing caveat.
- https://github.com/tailscale/tailscale/blob/main/cmd/tailscale/cli/serve_v2.go —
  Unix targets and foreground ownership through a daemon session.
- https://github.com/tailscale/tailscale/blob/main/ipn/ipnlocal/serve.go — source-IP
  forwarding, removal of client identity headers, atomic foreground port
  conflict checks and foreground session removal. This is upstream evidence,
  not proof of the operator's installed Mac binary.


### 2026-10-22 — tsnet revision proposed after operator choice A

Operator choice relayed by TL2e7f4b13 supersedes the Serve transport revision.
The failed native socket probes remain evidence, not an implementation route.
This is the plan sent for approval; no tsnet dependency, node or auth key exists.
Public cloudflared cast behavior is unchanged. All remote Cloudflare verification
and Serve/socket/proxy authentication paths are removed from the remote feature.

Core concepts (ARCH-PURE, ARCH-DRY, ARCH-SECURE)

| Entity | Location | Change |
| --- | --- | --- |
| TailnetPrincipal (canonical login, node/user IDs, accepted peer IP, epoch) | cmd/internal/remotesession/tailnet_identity.go | new pure identity validation; replaces AccessIdentity semantics |
| Exact login policy | cmd/internal/remotesession/policy.go | modified; COUCH_REMOTE_ALLOWED_LOGINS only, nonempty explicit entries, no suffix/domain/case-fold match |
| Remote epoch/reducer | cmd/internal/remotesession/state.go and supervisor.go | modified for enrollment/health events; retains existing kill/reset and cleanup authority |
| Remote receipt/audit identity | cmd/internal/remotesession/protocol.go, audit_codec.go, session.go | additive Login; immutable old Email data remains historical, never reinterpreted as newly verified Login |

| Integration | Location | Change / external boundary |
| --- | --- | --- |
| TailnetNode owner + stateful fake | cmd/internal/remotesession/tailnet.go, tailnet_fake_test.go | new narrow adapter over pinned tsnet Start/Up, LocalClient status/WhoIs/watch, ListenTLS, Close |
| Accepted connection authentication | cmd/internal/remotesession/server.go, tailnet_identity.go | modified; trusted ConnContext inserted only by owned tsnet listener |
| Namespace state and console publication | cmd/internal/couchcmd/remote.go, remote_service.go; cmd/internal/artifactpath/remote.go | replaces cast tunnel settings with owned node state/config and console-only enrollment/link |
| Host UI and docs | cmd/internal/couchtty remote controls; README.md; atlas existing remote page | preserve controls, show enrollment/assigned name and recovery |

1. Dependency gate, before integrating the adapter (ARCH-CONSTRAINTS)

- Candidate exact pin is tailscale.com v1.104.2, go.sum verified; no latest/main
  dependency. Source go.mod declares Go1.27.3; Pair currently declares Go1.26.3.
  This is a material compatibility decision, not permission to upgrade Pair's
  toolchain. Report it to TL with the dependency review before accepting the pin;
  if a toolchain change is required, stop for a decision rather than silently
  selecting an obsolete version or editing go directives. v1.104.1 also requires
  Go1.27.1 and is not an assumed compatible escape hatch.
- After plan approval, inspect/download the pinned source in isolated scratch
  without starting a node, changing product go.mod, or running enrollment. Record
  module provenance/checksum, licenses, govulncheck results, actual compiled
  Darwin/arm64 package/module closure (not all monorepo go.mod entries), changed
  shared dependency versions, CGo/toolchain and supported-platform effects.
- Same compiler, architecture, tags and release flags for matched baseline and
  proposed linked Couch binary. Record byte sizes, delta, cold/warm build times
  and retained module/package lists. Use a compile-only measurement fixture;
  importing/building tsnet is never consent to call Start/Listen/Dial/LocalClient.
- Measure baseline idle Couch and candidate remote-OFF RSS; once the operator
  authorizes enrollment, measure remote-ON idle for60s and four-session edit/stream
  load, peak and steady RSS, CPU, goroutines and node-state bytes. Include disabled
  startup and off cleanup. Same workload/host controls, raw timestamped samples;
  no guessed footprint or subtraction presented without a comparable baseline.
- Report actual costs and any toolchain conflict for acceptance before final
  integration/landing. No claim of zero overhead; zero tsnet node instances and
  zero enrollment/network work while remote is off is an enforced lifecycle rule.

2. Connection identity and authority (ARCH-SECURE)

- Only tsnet.Server.ListenTLS on tailnet TCP443 can feed remote HTTP. Never
  ListenFunnel, Serve, host TCP/loopback listener, exposed LocalAPI/Loopback helper,
  generic reverse proxy, automatic route advertisement or exit-node selection.
  Existing public cast transport is separate. RunWebClient=false; no extra
  tailnet management listener. No forwarded header establishes login, host or IP.
- At accept, bind the actual net.Conn.RemoteAddr to unexported ConnContext with
  owner/epoch. HTTP RemoteAddr/header rewrites cannot replace this evidence.
  Use this node's LocalClient.WhoIs with that captured address only. Require
  canonical tailnet peer IP and complete untagged human user/node identity;
  refuse missing login, tags/service identities, malformed address or lookup
  failure. Exact login membership, e.g. xianxu@gmail.com, comes from explicit
  COUCH_REMOTE_ALLOWED_LOGINS. No email-domain or suffix inference.
- Revalidate WhoIs and current node/epoch health on each request, immediately
  before protected handoff, and before each stream delivery; bind sessions to
  verified user/node/login/IP plus epoch. Keep Origin/Host/CSRF/replay/passcode
  checks, four-session cap and capability checking. HTTP/2 connections retain
  the same captured peer; any identity change ends their sessions and streams.
- One owned event watch observes node/netmap changes; membership/expiry/auth
  loss revokes admissions and streams immediately on observation. Re-run WhoIs
  for affected active sessions; no persistent identity cache can override newer
  evidence. Watch loss or uncertain health fails closed and requires explicit
  local recovery. An expiry deadline timer is not an idle-session timer.
- Revocation is bounded by Tailscale's control-plane propagation and what this
  node observes; do not claim instantaneous global revocation during partitions.
  No positive WhoIs from a stale map is advertised as fresh control-plane proof.
  Native acceptance must measure removal/revocation behavior; if the pinned API
  cannot establish the planned expiry/health policy, stop and report the gap.
- Delivered header remains [remote <login> <IP>]; receipt Origin=remote and
  RemoteEdited stay. Login is explicit end-to-end, including broker, ledger,
  audit and native transcript. Console submits remain untagged. Native transcript
  is the paper trail; bounded correlation recorder is not complete delivery proof.

3. Enrollment, state and operator recovery (ARCH-FUNERAL, ARCH-EVOLVE)

- One stable node per leased Couch namespace, one tsnet instance per active
  epoch. Reuse explicit namespace-owned state under the existing remote state
  root, in a new physical0700 tailnet directory; reject symlinks, foreign owner
  or unsafe existing permissions instead of traversing or broad chmod repair.
  Key/state files0600. Namespace lease plus exclusive directory ownership keeps
  two Couch owners from using the same keys. No implicit user config fallback.
- State is tsnet-owned opaque storage, never decoded/re-written by our codecs;
  audit its actual writer permissions/atomicity at the pin. New Pair metadata
  uses the shared retained-raw strict codec, preserving unknown nested fields.
  No secrets in audit/transcript/CLI/status/test fixtures. Enrollment login URL
  is console-only; capture-configured refusal still applies before starting.
- Explicit initial enrollment is a local console action with the operator
  present. Show tsnet's login URL on the console; operator opens it, signs into
  their intended tailnet and approves the new Couch device (including admin
  approval if required). No automatic auth-key/OAuth creation, copied desktop
  credentials or account/policy changes. No node Start before this authorized
  enrollment test. Reject inherited TS_AUTHKEY/TS_AUTH_KEY, OAuth/federation,
  TSNET_FORCE_LOGIN and control-server override inputs rather than consuming
  ambient credentials or mutating process-global environment.
- Use the backend-confirmed assigned FQDN and own node/tailnet identity after
  enrollment; never assume requested couch is available. Verify HTTPS readiness
  and publish the actual https://FQDN link/QR only on Couch's console once ready.
  Record enrolled identity and refuse unexpected tailnet/node replacement until
  a fresh explicit local enrollment decision. Default-off never auto-enrolls.
- Initial local enrollment may wait up to10min with visible status and cancel;
  no remote route/dispatch is live meanwhile. Ordinary enrolled enable retains
  the30s opening budget. WhoIs/status operations cap at2s and active request
  concurrency is bounded by the existing server; overflow refuses. Cancellation
  revokes the epoch before joining the node, with existing5s cleanup budget.
- Off/kill revoke link/passcode/sessions/dispatch FIRST, close HTTP connections,
  cancel the same opening attempt, join watch and tsnet.Close. Late readiness
  cannot publish. Unproved cleanup stays blocked, host reset unavailable. Off
  preserves enrolled keys for explicit re-enable without creating a new device.
  Re-enable rotates capability/passcode; never rotates node identity silently.
- Node key expiry, needs-login, device removal, missing policy access, certificate
  failure, or observed caller revocation denies requests and stops dispatch.
  Recovery names the action: reconnect/check tailnet policy, or local re-login
  and operator approval. No automatic new node or disabled key expiry. Policy
  outages are not repaired by deleting state. Same account reauthentication
  never restores killed latch authority; reset remains physical-host-key only.
- Durable node state has one identity rather than per-epoch/per-session growth;
  inventory/cap upstream caches and logs during dependency review. Keep it across
  off/restarts, remove only through explicit operator retirement after device
  removal and proven Close, never automatically reset/delete to repair errors.
  Enrollment cancelled after possible registration may leave a device: preserve
  state and report exact identity/unknown outcome for operator inspection.

4. Reducer and uncertainty table (ARCH-STATE)

| State/event | Outcome and owned effects |
| --- | --- |
| off + local enrollment approved | opening/enrolling; one owner, console URL; no remote admission |
| opening + enrolled and HTTPS/identity ready | on; publish actual origin under current epoch only |
| opening + timeout/cancel/error | revoke then closing; join same attempt, retain possibly enrolled state |
| on + off or authenticated kill | revoke before close; kill persists existing latch |
| opening/on + expiry, identity loss or watch uncertainty | revoke; close; recovery status identifies login/policy/health cause |
| closing + late open completion | never publish; join exact resource and preserve its cleanup outcome |
| closing + confirmed completion | off or killed; persistent node identity retained |
| closing + uncertain completion | existing cleanup-blocked; reset unavailable; no second owner |
| killed + remote/CLI reset or re-handshake | refuse; only actual host key can clear after cleanup proven |

5. Implementation and qualification sequence (tests first, no extra milestones)

- After TL plan approval, run change-code plan-quality on this revision. Finish
  dependency/compatibility report first; seek the explicit toolchain decision
  if needed. No node or auth key until operator enrollment window.
- Red/green pure login/principal and reducer tests; then stateful TailnetNode
  fake behind the production seam. Model enrollment pending/approved/rejected,
  actual hostname collision, held startup/close, late readiness, expiry, device
  removal, watch loss, WhoIs errors/change/tagged identities and state contention.
  Count N/2N requests and stream events: bounded linear identity operations,
  zero spawned CLI children; one node/watch per epoch and zero while off.
- Add root/header spoof tests proving WhoIs input is captured connection address,
  never HTTP headers. Missing context, forged login/IP/RemoteAddr, unallowed login,
  wrong origin, stale cookie/capability and expired/revoked identity yield zero
  document effects. Deterministic off/kill during held WhoIs/dispatch and both
  existing failed-cleanup opening/closing regressions retain the same authority.
- Replace Cloudflare-only remote transport/verifier and tests, reconcile shared
  host coupling narrowly, retain public cast tests. Sweep Login/provenance across
  server, sessions, broker, IPC receipt, audit, ledger and transcript. Preserve
  old audit unknown fields and historical Email bytes. Forget split remains.
- Operator-approved enrollment test: isolated explicit0700 state, no auth key;
  operator sees console URL and approves one identified device. Confirm actual
  HTTPS name, phone WhoIs login/IP, disallowed identity and header-forgery cases,
  off/re-enable identity reuse and link rotation, kill from phone, host reset,
  expiry/revocation response, joined cleanup and unchanged desktop app/public cast.
  Native test node is retired only with operator authorization; no silent delete.
- Measure full footprint above, update README/atlas/operator acceptance, targeted
  packages/race plus recorded suite, then required fresh-context boundary review.
  Stage only on TL go. Operator iPhone edit/submit checks native transcript and
  audit provenance before landing; fakes never replace live acceptance.

References checked 2026-10-22: tailscale.com/docs/reference/tsnet-server-api;
pkg.go.dev/tailscale.com@v1.104.1/tsnet (API/enrollment/ambient credentials);
raw.githubusercontent.com/tailscale/tailscale/v1.104.2/go.mod (candidate pin and
Go1.27.3 requirement). API/Go requirements must be rechecked at selected pin.
