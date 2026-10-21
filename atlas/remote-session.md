# Remote session security

The #759 foundation is under development. It does not expose a remote editor or
start a tunnel yet. #758 supplies the real document/nvim adapter after its
interface integration; private stateful backends qualify the security boundary.

`cmd/internal/remotesession/state.go` decides lifecycle transitions without IO.
A durable arm acknowledgement precedes opening. Kill fences admission before
teardown; stale successful opens are disposed, while duplicate acknowledged
completions do not close the active owner. Reset remains a private event, reserved
for the owned host-TTY key path when console wiring lands.

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
again. The supervisor will own serialization and atomic mutation admission.

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
