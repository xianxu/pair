# Remote session security

The #759 implementation is under development and defaults off. M2 wires the
protected security surface, locally qualified with its milestone review pending;
document routes, dispatch/audit handoff and broker provenance remain M3.
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
again. The supervisor owns lifecycle serialization; M3 adds document dispatch
against its exact admission mutex.

`access.go` uses coreos/go-oidc and go-jose for signature/issuer/audience validation.
`access_keys.go` pins the Cloudflare team's HTTPS key URL, bounds response size,
key count and cache lifetime, coalesces refresh and limits unknown-key refresh.
`policy.go` checks exact HTTPS origin and trusted-proxy client IP. The JWT does
not sign Cf-Connecting-IP; its provenance depends on the exclusive protected
named connector and private Unix origin. Same-UID host compromise is outside
this boundary.

`origin_store.go` retains conservative remote-edit markers by stable registration
ID across editor and Couch lifetimes. Only settled, explicitly forgotten IDs may
be retired locally. A missing marker does not prove a legacy document was local.
The supported records retain opaque JSON fields through rewrite.

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

`server.go` currently serves only authenticated security status, handshake,
logout and kill. Every request verifies Access and trusted-proxy IP. Same-origin
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
