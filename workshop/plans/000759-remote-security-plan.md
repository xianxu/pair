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
| RemoteState, RemoteEvent, ReduceRemote | `cmd/internal/remotesession/state.go` | new |
| AccessIdentity / RemotePolicy | `cmd/internal/remotesession/policy.go` | new |
| SessionState and mutation admission values | `cmd/internal/remotesession/session.go` | new |
| RemoteAction / DocumentRef / RemoteResult | `cmd/internal/remotesession/protocol.go` | new |
| Remote submit origin and envelope projection | `cmd/internal/couchmessage/model.go`, `batch.go` | modified |

The reducer owns legal transitions; policy decides claims, identity, origin, action and bounds from values. One supervisor has one epoch and at most four sessions; each session has many allowed document operations, each tied to current document/editor identity. One action enum is the allowlist shared by decoder, dispatcher and tests. The future phone UI consumes this contract; no general “execute” extension slot.

| Integration point | Lives in | Status | Wraps |
|---|---|---|---|
| Remote supervisor | `cmd/internal/remotesession/supervisor.go` | new | lifecycle effects, sessions, joined requests |
| Latch store, origin evidence and audit | `cmd/internal/remotesession/store.go`, `origin_store.go`, `audit.go` | new | injected private directory, durablefile, lifetime lock |
| Access verifier / key source | `cmd/internal/remotesession/access.go` | new | JWT library, bounded HTTPS JWKS |
| Protected named transport | `cmd/internal/broadcast/cloudflared.go`, `host.go`; `couchcmd/remote.go` | modified/new | existing cloudflared handle/guard/lock, strict remote config |
| Host-TTY security control | `cmd/internal/couchtty/console_remote_session.go`, `terminal_input.go`, `console.go` | new/modified | local decoded key event, status and notices |
| Trusted remote submit | `cmd/internal/couchcmd/remote_submit.go`, `couchmessage/broker.go` | new/modified | broker, receipt, batching and delivery accounting |
| Document backend seam | `cmd/internal/remotesession/documents.go` | new | #758's registry/nvim adapter, stateful test backend |
| Artifact ownership | `cmd/internal/artifactpath/remote.go`, `manifest.go` | new/modified | store paths, latch/audit retention classification |

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

- [ ] Use the agreed #759-security/#758-document adapter contract and stateful backend; coordinate the exact Go seam with its eventual owner. No real-nvim dependency for this security foundation.
- [ ] Write pure transition/policy failing tests for all rows above; generated event sequences independently assert no grant after kill, no restart grant from disk and no old-epoch completion adoption.
- [ ] Run `go test ./cmd/internal/remotesession -run 'TestRemoteState|TestRemotePolicy' -count=1`; observe red before implementation, then green.
- [ ] Implement typed reducer, durable armed-before-exposure store and session policy. Fault-inject write/rename/file sync/directory sync, corrupt/unknown/truncated state and restart at every publication boundary.
- [ ] Add fake signed JWT/JWKS issuer with rotating keys and requests for wrong AUD/issuer/algorithm, forged claims, no subject/email, expired/not-yet-valid, oversized input, redirect/SSRF and refresh stampede; run `go test ./cmd/internal/remotesession -count=1` red/green.
- [ ] Add bounded audit/replay tests, including failing intent, committed effect with missing result, disk full at kill, capacity/rotation and no secret/content leakage. Count N/2N operations.
- [ ] Update `atlas/remote-session.md` and `atlas/index.md`; commit coherent steps with agent trailers; `sdlc milestone-close --issue 759 --milestone M1` with evidence.

### M2 — Exclusive exposure, local reset and revocation

Files: `broadcast/cloudflared.go`, `host.go` and tests; new `couchcmd/remote.go`, `remote_cli.go`, tests; new `couchtty/console_remote_session.go` and tests; console wiring/input/status files; local lint catalog guard.

- [ ] Extend the existing fake tunnel to retain exclusive owner/open/close state and inject late-open, ignored cancellation and blocked close. Test strict ingress config and Unix-only listener before implementation.
- [ ] Wire remote-only Host to protected named transport, existing guard and lifetime lock. Ordinary cast/replay holders produce explicit busy; no shared mounts, no unauthenticated public readiness probe, no quick/loopback fallback.
- [ ] Add real HTTP server tests with signed fake Access assertions: invalid requests invoke zero document handlers; `/remote/` cannot route to cast tokens, drive, IPC, filesystem or nvim sockets.
- [ ] Wire 👽 click toggle/show-link and CLI on/off/status plus authenticated remote kill; no CLI/HTTP reset opcode. Every re-enable rotates link passcode and revokes old cookies/streams. Add an unexported host-key reset closure. Paste/mouse/reply/release and child-output bytes never invoke it.
- [ ] Exercise actual broker -> wrapper `--keys` delivery containing the reset chord and confirm outer Couch remains killed. Positive control drives the owned console fixture's host stdin, resets after joined teardown and starts only with a separate request.
- [ ] Add pair-local structural guard `remote-console-reset` to stop reset authority escaping the host-input file; cite #759. Test every production call site and permissive test fixture exclusions.
- [ ] Sequence/fuzz kill during preparation, final mount admission, session handshake, held mutation and teardown; normal cast/replay start during remote ownership; parent death and stale completion. Reopen private portable storage and prove still killed.
- [ ] Run targeted remote/broadcast/couchtty/couchcmd/wrapcmd tests plus relevant race tests, update README/atlas/CLI help, commit and milestone-close M2.

### M3 — Trusted submits, document integration and acceptance

Files: new `remotesession/documents.go`, `couchcmd/remote_submit.go`; `couchmessage/{model,batch,broker}.go` and endpoint/accounting consumers; session-sync provenance ledger/owed projections; #758 adapter at its agreed owner path; tests and docs.

- [ ] Implement the M1-agreed narrow document adapter/permit integration without implementing #758's RPC/editor subsystem here. Use a stateful fake with IDs, rename/forget, revision conflicts, remote-origin revisions, held commits, ambiguous replies and submit receipts.
- [ ] Write end-to-end fake-server/broker tests before wiring: allowed registered roots/side docs; foreign registered docs, traversal, IDs rebound to another editor and arbitrary methods refused. Kill while audit is held and immediately before atomic handoff prevents dispatch; handoff-before-kill is recorded as in flight. Navigation into an unregistered target is refused before any nvim open.
- [ ] Add broker-origin handling without reusing OriginSystem; carry remote provenance through endpoint validation, batches, receipts, delivery accounting and preserved supported persisted fields. Keep the canonical submitted path line unchanged. Local legacy submits stay compatible; older incompatible wrappers visibly refuse remote delivery.
- [ ] Test exact remote email/IP display prefix, console submits without that prefix, authoritative receipt fields and existing transcript retention; test direct remote submit and remote edit followed by local submit retain provenance across Couch/nvim restart, editor handover, rename and failed edit; explicit local forget/re-register retires only settled old-ID evidence; duplicate retry delivers once, unknown outcomes require reconciliation, and tag reaches the TL's trusted envelope/owed ledger. #520 supplies policy; do not implement its confirmation agent rule here.
- [ ] Record the #758 adapter contract and qualify #759 against its stateful backend now. Per TL sequencing, real-nvim edit/submit acceptance runs after #758/parley.nvim#405 integrate; it does not block this security foundation. The #759 security harness is not real-nvim acceptance.
- [ ] Run targeted packages and reader inventory audit on committed HEAD; push; recorded affected/full check per current gate contract; required final close review. Do not weaken deadlines or bypass failed gates.
- [ ] Operator-assisted live conformance only after separate authorization for setup/staging/network: approved named tunnel + Access policy; iPhone login as allowed user, refused unapproved identity, raw forged/missing/wrong-audience assertion refused before handler, no artificial idle/absolute expiry, silent Access-bound re-handshake, explicit sign-in-needed on invalid Access, exact email/IP header and transcript paper trail, console untagged, kill by a second authenticated session, off/on old-link revocation, old tab/session dead, restart still killed, CLI/--keys unable to reset, console reset then fresh login succeeds. Check no owned stage/connector remains after teardown. Record exact build and outcomes without secrets. Before #758, this live pass covers Access/status/expiry/kill/reset only; tagged-submit behavior is verified against the private stateful backend and actual broker. The real document edit/submit and nvim handover portion is deferred to #758 integration per TL; use the stateful backend here and make that limit explicit. Do not expose a test document backend on the operator's public tunnel to manufacture acceptance.
- [ ] After operator acceptance and SHIP, run `sdlc merge --yes`, verify the actual car admission, final Git/PR/tracker outcome, and report to TL. One week of daily use belongs to the project, not a false claim in this close.

## Risk functions and mechanical defenses

| Risky boundary | Adversarial class | Mechanical defense |
|---|---|---|
| JWT verifier | forged/rotated/unknown claims, attacker key URL | real-signature fake issuer, verifier library, pinned issuer/AUD, fuzz bounded decode |
| ReduceRemote / effect adoption | kill/restart/delayed success/duplicate events | generated event sequences + independent epoch invariants; held-effect rendezvous |
| Store publication | crash/IO failure at each durable boundary | fault-injected filesystem + reopen same private store |
| Host admission/cleanup | remaining mount/reopen/late child | exclusive named lease, FakeTunnel lifecycle counts, no multi-mount remote host |
| Console reset | IPC/keys/paste/replies mistaken for local key | actual negative route test + positive host-input control + AST call-site guard |
| Document dispatch | stale ID/path/editor, method injection | stateful backend, generated invalid refs/actions, strict typed allowlist |
| Submit provenance | tag stripping, batching, local-submit laundering | broker-origin round trip and remote-revision tests; wire compatibility refusal |
| Audit/retry | effect committed but reply/log missing | intent/result sequence tests and retained uncertain request; no blind retry |

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
