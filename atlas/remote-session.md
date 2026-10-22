# Remote session security

The #759 implementation is under development and defaults off. M2 wires the
protected security surface, qualified and closed SHIP at its milestone review;
M3 now implements document routes, owned dispatch/audit handoff and broker
provenance; qualification and operator acceptance remain pending.
#758 supplies the real document/nvim adapter after interface integration; private
stateful backends qualify that seam without being exposed publicly.

`cmd/internal/remotesession/state.go` decides lifecycle transitions without IO.
A durable arm acknowledgement precedes opening. Kill fences admission before
teardown; stale successful opens are disposed, while duplicate acknowledged
completions do not close the active owner. Reset remains a private event, reserved
for the owned host-TTY key path. An idle kill also waits for durable publication
before reset; poisoned storage/cleanup completions become blocked.

`store.go` publishes the private latch through durablefile. On restart, armed
means killed; an on-disk latch never grants access. Any uncertain publication
blocks the current writer. A failed directory sync may leave the renamed record
on disk: reopening reads that evidence rather than claiming rollback. Off left
by an already-authorized console reset or confirmed normal teardown does not
automatically enable anything.

`session.go` owns at most four cookie hashes and revocation signals. Access JWT
expiry supplies the time bound; no Couch idle or absolute session timer applies.
`replay.go` keeps a sequence high-water mark and at most256 results per session;
uncertain/in-flight entries remain pinned, and evicted requests cannot execute
again. The supervisor owns lifecycle serialization and document dispatch
against its exact admission mutex.

`access.go` uses coreos/go-oidc and go-jose for signature/issuer/audience validation.
`access_keys.go` pins the Cloudflare team's HTTPS key URL, bounds response size,
key count and cache lifetime, coalesces refresh and limits unknown-key refresh.
`policy.go` checks exact HTTPS origin and trusted-proxy client IP. The JWT does
not sign Cf-Connecting-IP; its provenance depends on the exclusive protected
named connector and private Unix origin. Same-UID host compromise is outside
this boundary.

`origin_store.go` retains conservative remote-edit markers by stable registration
ID across editor and Couch lifetimes. The message service initializes its single
origin writer before publishing sockets and lends it to remote mode; local
submits consult it even while remote mode is off. `submit_seen` is persisted
before remote or marked-local submit handoff and never cleared in M3. Such IDs
cannot be forgotten yet: the refusal points to pair#795, which owns lifetime
settlement proof beyond receipt/audit retention. Never-submitted registrations
use the local Couch owner, a durable forget intent, exact-ID registry removal
and an active-operation fence; offline forget says start Couch and retry.
Replacement registrations at a reused path are not removed by the old request.
A missing marker does not prove a legacy document was local. Supported records
retain opaque JSON fields through rewrite.

For these new retained-raw stores, strictjson exposes the additive
`RejectDuplicateKeysPreserveNumbers` check: arbitrary valid JSON numbers remain
raw, without a float64 range restriction. Known members still receive exact typed
validation. Existing `Decode`/`RejectDuplicateKeys` and their equivalence tests
retain their previous behavior. This does not migrate older readers.

See [the approved plan](../workshop/plans/000759-remote-security-plan.md) for the
complete threat model, bounds, provenance contract and operator acceptance.

`documents.go` defines the typed #758 seam and an opaque one-use Permit. Its
EffectDoor uses the supervisor's exact lifecycle/admission mutex to linearize a
nonblocking effect handoff against kill. Navigation receives no effect target
until the supervisor resolves and authorizes a destination. Document adapters
must recheck registration, generation and revision at that door. Dispatch itself
is a nonblocking owned-queue handoff; workers call EffectDoor afterward, never
recursively under the caller's admission lock. Join quiesces remote work rather
than ending nvim. `protocol.go` accepts only read/edit/submit/navigation, opaque
registered IDs and bounded typed fields; it has no path, command or reset input.

M1 review clarified that Sessions is an integration owner (entropy and mutex),
with pure identity checks in policy.go. Publication uses one reusable stage per
record, derived by artifactpath.RemotePublicationStage. Openers first validate
committed authority, then reclaim only exact bounded regular stages; symlink,
nonregular and oversized slots refuse. Stages never grant authority. Across the
latch, origin store and two audit segments, published files plus all staging
slots have a fixed aggregate maximum66MiB+8KiB; normal successful writes remove
their stages. The leased Couch namespace supplies sole-writer ownership.

`supervisor.go` owns persistent command ordering separately from IO: accepted
reset/off/kill publications execute in order, while kill revokes the current
epoch before waiting for any write or connector cleanup. Confirmed teardown
joins the protected mount and remote requests. `broadcast/protected.go` reuses
the guarded named connector and lease with a private Unix origin, required
Access ingress and one opening per owner. There is no anonymous/TCP fallback,
public readiness probe, shared remote mount or automatic reconnect. Teardown
uncertainty permanently blocks that Host instance.

`server.go` serves authenticated security status, handshake, logout and kill;
`server_documents.go` adds the closed document routes below when #758's typed
backend is connected. Without a backend, they report unavailable; no private
fake backend is exposed. Every request verifies Access and trusted-proxy IP. Same-origin
GET need not carry Origin; mutations require exact Origin plus JSON and a custom
header. Kill has a separate bounded lane independent of session capacity and the
per-enable passcode. Cookie resume preserves the original assertion expiry and
never returns the HttpOnly credential as JSON. Failed assertions revoke a
presented session. Request bodies have size and socket-read time bounds.

Remote setup reuses COUCH_BROADCAST_TUNNEL/COUCH_BROADCAST_HOSTNAME and requires
COUCH_REMOTE_ACCESS_TEAM, COUCH_REMOTE_ACCESS_AUDIENCE and comma-separated exact
COUCH_REMOTE_ALLOWED_EMAILS. Account, DNS and Access policy setup belongs to the
operator. `couch --remote on|off|status` runs through the owner-only local socket
and prints action/status plus secret-free recovery detail. Same-UID code is trusted, so local enable/disable
is allowed; no IPC reset operation exists. CLI is asynchronous: starting and
stopping describe work still in flight.

The console's separate “👽 remote” control starts/stops remote mode. Capability
text and a locally generated QR appear solely in the Presenter operator layer;
they never enter the ordinary chrome, broadcast frame or child transcript.
Remote enable refuses when COUCH_CAPTURE_DIR is configured: unset it and restart
Couch, or keep capture and skip remote. The QR encoder is bounded to1024 input
bytes and185×93 terminal cells, includes a quiet zone and explicit black/white,
and caches by link/geometry; small screens keep the text link. The phone scan is
still an operator acceptance check. Ctrl+Alt+R clears a settled kill latch only
from a decoded host keypress, without enabling. Paste, release, repeat, terminal
replies and child broker keys cannot reset. The pair-local remote-console-reset
AST catalog pins capture/calls and rejects alias escapes.


Failure projections preserve recovery across the supervisor, CLI and console.
A busy named transport reports “stop the existing casts, then retry” alongside
the existing host-reset-after-cleanup instruction; it never bypasses the latch.
Configuration and uncertain-cleanup details remain visible on both operator
surfaces. Connector errors are classified into fixed safe text rather than
printing arbitrary stderr, links, credentials or a raw response.


Protected opening failure carries `broadcast.ErrProtectedCleanup` whenever
transport teardown is unconfirmed, independently of whether a Mount exists.
The supervisor retains that classified error before acknowledging opening and
includes it in the epoch cleanup result. A missing mount never proves cleanup.
Cancellation also joins an already-closing Host attempt; failed connector or
listener cleanup keeps the existing blocked state and denies console reset.


M3 dispatch and provenance

`dispatch.go` reserves per-session replay, owns completion after HTTP departure,
and persists origin evidence and audit intent before the nonblocking backend
handoff. Durable work runs outside the lifecycle mutex. Backend workers use the
one-use Permit at the final enqueue, sharing kill's exact fence. Audit intent
failure has zero effects; result audit failure revokes and retains uncertainty,
including any already-known receipt ID. Reset still waits for joined work.
Per-epoch active operations are capped at eight. A canceled observer before
admission cannot start work; departure after admission does not assert rollback.

The closed API under the current capability is GET `v1/documents` and POST
`v1/read`, `v1/stream`, `v1/edit`, `v1/submit`, `v1/navigate`. POST bodies use the
strict typed DocumentRequest and every POST requires session CSRF. Streams emit
NDJSON snapshots, one per session, at most 1 MiB per encoded frame with bounded
writes; JWT/session/epoch ends them, not the ordinary five-second exchange timer.
The backend owns its bounded subscription queue and must return subscriptions
promptly with all remote work canceled/joined through its existing lifetime.
#760 owns the phone editor; #758 owns registered eligibility and actual nvim RPC.

`couchmessage.Broker.SendRemote` assigns Origin remote and comparable
RemoteSource Email/IP facts. `Envelope` emits `[remote <email> <IP>]` before the
canonical `submitted: <path>` body, preserving SubmitPath parsing. Message and
receipt carry RemoteEdited independently plus stable DocumentID. Marked local
submits have no remote display prefix; ordinary unmarked peer messages retain
the old wire. All new provenance requires receiver `remote-provenance-v1`;
unknown/legacy receivers refuse with relaunch recovery before admission. Generic
request JSON cannot set these fields. Same-UID host code remains trusted.

Delivery recorder, retained-raw Entry codec, trim/reopen and owed projections
preserve the facts and unknown nested fields. New DocumentID-bearing rows cannot
be attributed to a replacement registration sharing a path. Recognized malformed
provenance causes refusal/gap, never clean-local inference. Rollback to an old
writer is unsupported for preserving the added facts; disable remote and retain
provenance/transcripts before rollback.

Local submit callers and document owners may be different slots; provenance
follows the registration ID while existing sender/recipient routing remains
intact. ID-bearing ledger rows retain attribution across rename/admission races;
legacy rows still use historical pathname bindings. Ledger append preflights
retention before crossing its byte threshold, so invalid evidence or failed
retention cannot allow repeated growth, including after owner recreation.

The agent's native transcript is the second paper trail. Private Claude/Codex
native-format fixture producers through actual wrapper PTY input and production
readers retain the exact header/body beyond turn eight; this is fixture
qualification, not real-agent/iPhone acceptance. The existing Pair peer recorder
remains best-effort correlation, capped at eight turns / 256 KiB per launch. It
does not prove every submit, and there is no new unbounded transcript format.
Qoder native fixture coverage is unavailable; live acceptance is reported
separately. No fixture transcript contains JWTs, session credentials or the
remote link/passcode.
