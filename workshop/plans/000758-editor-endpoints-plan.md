---
collaboration-mode: discussion
---

# Registered editor endpoints implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 for execution strategy. The SDLC binary owns mandatory fresh-context reviews. Do not implement before the hub approves this plan and change-code admits it.

**Goal:** Give parley.nvim#405 a confirmed, incarnation-fenced editor endpoint producer before building the remote document adapter.

**Architecture:** The existing Couch singleton owns one durable endpoint registry per canonical Couch namespace. A pure transition model chooses the current editor by stable document ID; IO adapters validate the actual private nvim endpoint and publish transitions before acknowledgements. Parley advertises asynchronously, receives bounded replacement notifications and never polls or automatically reclaims a superseded binding.

**Tech stack:** Go, existing sessionsync registry, procutil.Source, durablefile and private Couch IPC; a pinned standard Neovim Go RPC client behind an injected typed EditorRPC seam (no hand-written MessagePack codec or shell-expression interpolation). No public HTTP route, Cloudflare change or #759 code dependency in M1.

## Scope and alternatives

#758 is now the independently landable producer prerequisite, per TL79aac81f. Adapter and security integration moved to unclaimed #798, depending on #758, parley.nvim#405 and #759. This issue has one whole-issue close and no milestone-close; it does not claim the phone editor works.

Recommended: Couch's single owner retains endpoint authority and retired-attempt evidence, publishing acknowledgements only after persistence. Alternatives rejected: a registry file independently rewritten by nvim duplicates ownership and permits stale retry takeover; discovery by periodic socket/process scanning cannot identify explicit desktop intent and adds polling. Neither solves lost acknowledgements after supersession.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| EndpointBinding, RegisterRequest, UnregisterRequest, EndpointReply | cmd/internal/editorendpoint/model.go | new |
| EndpointState and ApplyEndpointEvent | cmd/internal/editorendpoint/state.go | new |
| DecodeEndpointState and EncodeEndpointState | cmd/internal/editorendpoint/codec.go | new |

Namespace + DocumentID is one logical document. It has at most one current binding. A process incarnation owns many attachment attempts, at most 256 active, and each process/document pair retains one latest AttachID/status. A replacement terminally retires the displaced attempt: no fallback resurrection. Explicit reopen advances AttachID. These values, decision rules and codec tests perform no IO.

### Integration owners

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| EndpointOwner | cmd/internal/editorendpoint/owner.go | new | serialized decisions, publication, pending operations and join |
| EndpointStore | cmd/internal/editorendpoint/store.go | new | portable namespace-local durablefile storage |
| EditorRPC | cmd/internal/editorendpoint/nvim.go | new | bounded fixed nvim callback queries/notifications |
| EndpointProcessSource and ClassifyEndpointProcess | cmd/internal/editorendpoint/process.go | new | injected error-preserving procutil.Source.ReadIdentity observations |
| Source.ReadIdentity | cmd/internal/procutil/source.go | new | expose existing native per-PID identity plus error without changing Identity |
| Stateful editor fixture | cmd/internal/editorendpoint/owner_test.go | new | current/pending attachments, delayed RPC, death, replacement and retained channels |
| endpoint CLI/service adapter | cmd/internal/couchcmd/editor_endpoint.go, editor_endpoint_service.go | new | bounded stdin JSON, private broker socket and runtime lifecycle |
| Registration namespace projection | cmd/internal/sessionsync/registry.go; couchcmd/session_sync.go | modified | existing confirmed document register/lookup replies |

Concrete names are the intended declarations, not claims of delivered implementation. Owner/store/RPC are INTEGRATION even when invoked from pure decisions. Reuse durablefile, namespace resolution, the existing singleton lease and shared process observations; do not create a second daemon or per-buffer poller.

## Producer wire contract (consumer agreement before code)

Parley coordination: dea1b235, bcc436ae, fc5fd55f, 8b7dcc76 and 66ccd77d. Final consolidated contract confirmed by parley.nvim:1 receipt7d43957e, with the established-channel ownership clarification below. No consumer incompatibilities remain; Parley separately takes crash-orphan runtime-directory cleanup policy to the hub.

### CLI

- `couch --editor-endpoint register --json`: one JSON object on stdin, maximum 8 KiB.
- `couch --editor-endpoint lookup --namespace CANONICAL --document-id ID --json`: no stdin, bounded observation of current endpoint; never acquires or takes over an editor.
- `couch --editor-endpoint unregister --json`: one batch JSON object on stdin, maximum 256 KiB, 1..256 entries for one namespace/process incarnation.
- Existing `couch --session-sync register FILE --owner SLOT --json` and `lookup FILE --json` gain `Registration.Namespace` inside a non-null Registration. This is the canonical absolute physical Couch store path, not cwd, slot path, session name or a new document identity. Existing null Registration means confirmed absence. The additive namespace is a projection, not duplicated persisted registry authority.
- The selected existing Couch namespace must match supplied Namespace exactly after ExistingCouchNamespace validation. A CLI outside a Couch pane can use the confirmed document lookup result; it does not need slot identity and cannot select another namespace through inherited identity. Offline endpoint commands return unavailable; no CLI writer or new Couch launch.

Register request (all fields required, exact known types, duplicate/unknown request fields refused):

```json
{"Version":1,"Namespace":"/physical/couch/store","DocumentID":"32-lowercase-hex","EditorIncarnation":"32-lowercase-hex","AttachID":"1","PID":1234,"Socket":"/physical/private/run/doc.sock","Buffer":7,"Mode":"desktop"}
```

EditorIncarnation is 16 random bytes encoded lowerhex, allocated once per nvim process. AttachID is a positive canonical decimal string, monotonically increasing per incarnation, allocated only on explicit attach/reopen and retained unchanged across retry. Bound it to unsigned 64-bit and refuse overflow; no Lua number precision dependence. Same process/document retains only its highest observed AttachID plus state/payload, so repeated reopening does not grow tombstones per attempt. Cross-document counter gaps are allowed.

DocumentID is the existing immutable sessionsync registration ID. Producer resolves its current path by ID. Rename preserves ID; a buffer still naming the old path refuses stale until confirmed move and fresh explicit attachment. No path-based re-registration or implicit acquisition. Mode is exactly desktop or headless; headless is only a fallback when no live current desktop exists. A new explicit desktop attachment wins over any current binding; a retry never becomes a new explicit attach.

### Success and refusal acknowledgements

Every parsed operation prints one bounded JSON envelope, at most 1 MiB, with Version=1, Code, Error and EditorEndpoint (explicit null for no current binding). Successful register returns Code=ok, Changed=true for a new publication or false for an exact current retry, and the complete current binding:

```json
{"Version":1,"Code":"ok","Error":"","Changed":true,"EditorEndpoint":{"Namespace":"/physical/couch/store","DocumentID":"32-lowercase-hex","EditorIncarnation":"32-lowercase-hex","AttachID":"1","PID":1234,"ProcessIdentity":"opaque-native-start-identity","Socket":"/physical/private/run/doc.sock","Buffer":7,"Mode":"desktop","BindingGeneration":"32-lowercase-hex","File":"/physical/session.md"}}
```

BindingGeneration is a fresh opaque random 16-byte lowerhex token per accepted replacement, persisted before reply, unchanged by exact retry. It is never an arithmetic timestamp. ProcessIdentity is Couch's opaque native start identity next to PID; Parley copies it verbatim and never manufactures a platform-specific stamp. Socket/path/buffer are immutable within an attachment attempt. A same-AttachID different payload is conflict, never mutation in place.

Exit 0: Code ok. Exit 2: invalid input. Exit 3: unavailable (Couch, process inspection, endpoint probe, capacity or storage unavailable). Exit 4: superseded, stale, conflict, not-registered or desktop-active; fixed recovery text distinguishes them. Exit 5: uncertain publication; retry identical intent, never invent success or rotate AttachID automatically. A malformed CLI invocation before JSON parsing prints usage on stderr. Unknown reply fields are tolerated by clients, but known types and required fields must validate.

Lookup returns Code=ok and EditorEndpoint=null only on confirmed no eligible binding. Unknown process/RPC evidence returns unavailable, not null. Endpoint lookup may cause the Couch owner to retire an exactly verified stale binding; this is stale-authority cleanup, not a document write or editor acquisition. The existing document lookup remains read-only.

Unregister request includes Version, Namespace, EditorIncarnation, PID, and Entries containing DocumentID, AttachID, BindingGeneration. ProcessIdentity may be echoed from ack for validation; omission does not weaken comparison because all exact tokens are required. No callback to nvim during unregister (VimLeavePre blocks its loop). One durable transaction compares every entry to current authority and returns Results with DocumentID, AttachID, BindingGeneration and Status removed|stale|absent. A stale entry never touches a newer binding. Duplicate IDs/oversize batch refuse before any write. Lost reply retries return absent/stale safely; they cannot acquire authority. Client can wait at most 500 ms at exit, then stop its own listeners and leave; an unacknowledged release remains unknown until producer cleanup.

### Fixed RPC callbacks and established-channel fencing

Use only nvim_exec_lua with constant code, parameters passed in the API argument array, never interpolated Lua/Ex strings or shell commands. The [official Neovim API](https://neovim.io/doc/user/api/#nvim_exec_lua()) defines separate code/args and NIL results. Bound decoded native replies to 1 MiB before typed validation; close/join the owned connection on deadline, including partial frames. This API is consumed only through EditorRPC, not a generic browser-supplied call:

- `return require("parley.remote_session").endpoint(...)`, args=[query]. Query contains Namespace, DocumentID, EditorIncarnation, AttachID, Buffer. Return exact tuple plus PID, Socket, File only for that pending/current live attachment; otherwise nil. Couch brackets with error-preserving Source.ReadIdentity reads and validates the private socket and canonical registered path. This proves initial endpoint identity even across PID/socket reuse; the producer supplies ProcessIdentity in ack.
- `return require("parley.remote_session").superseded(...)`, args=[notice]. Notice contains exact Namespace, DocumentID, EditorIncarnation, AttachID, BindingGeneration. Send only to the displaced endpoint after replacement publication, outside authority locks. Failure never rolls back current authority or grants an old binding a retry. No periodic Parley watcher.

Parley validates all five notice identifiers, ignores stale/malformed notices, retires the matching advertised attachment, stops only its owned listener and notifies once. It cannot revoke built-in Neovim RPC methods on already-open channels. The future Couch adapter must gate every operation by binding generation and retire its old channels; any future typed editor methods must reject retired tuples. Notice-before-register-ack is retained as unconfirmed and compared against eventual ack before showing active. Notification is advisory: subsequent Couch adapter operations must check current binding generation and the editor's typed methods must reject retired attachments even on established RPC channels. A canceled/missing notice cannot grant old authority. #798 must linearize its final nonblocking enqueue against replacement; #758 does not expose remote dispatch.

A live PID with endpoint(query)=nil positively retires only that exact attachment (buffer wiped, detached, moved or superseded). A timeout/transport error does not prove absence. Probe completions carry the complete original binding generation and owner revision; late nil/death/notice outcomes never remove a successor.

## Lifecycle, ownership and persistence

One EndpointOwner lives under Couch's existing lifetime singleton lease. The CLI only calls it. Startup reads a separate `editor-endpoints.json` under the canonical namespace; store/stage paths are classified by artifactpath. Durable state retains current bindings and per-process/document high-water attempt evidence across Couch restart. After restart no binding is usable until process/socket/attachment validation succeeds; no retry revives a superseded or retired attempt. A publication error pins uncertainty and blocks further mutation until exact evidence is reconciled or Couch is restarted and the committed record validated.

Transitions: absent+new valid desktop => current; current+exact retry => same generation; current+new desktop => publish new, retire old, then notify; desktop+headless => desktop-active; superseded/retired+equal-or-lower AttachID => superseded; fresh higher explicit AttachID => candidate; current+exact unregister or confirmed stale observation => retired; stale completion => ignored; unknown observation => retain evidence/unavailable. Observer departure never undoes a committed transition.

Durable high-water evidence is retained until the exact process incarnation is positively dead/replaced; missing/unreadable identity is uncertainty. Cleanup never depends on age alone. The 4096 retained namespace-wide process/document-pair cap and 8 MiB encoded file cap refuse new admission before publication; no live evidence eviction. Maximum 256 active endpoints per editor process. Confirmed dead cleanup frees capacity. Forgotten/replaced documents cannot regain endpoints via old IDs. Generation tokens prevent ABA even when the last dead row is collected.

Couch owns stale binding/evidence cleanup, not another application's filesystem. Parley creates one process-owned physical 0700 runtime directory and private Unix sockets, unlinks only its own exact socket after listener shutdown, and removes its own empty directory at exit. Couch validates ownership/type/private parent, never recursively removes runtime directories or unlinks a registered path that might now belong to a newer editor. Crash-orphan runtime directory reclamation is outside endpoint authority: follow the existing application runtime-dir lifetime/OS cleanup; do not add arbitrary path deletion. Confirmed native endpoint liveness is not inferred from a leftover file.

Persisted codec preserves unknown root/nested fields through register/replacement/unregister/cleanup/reopen, with strict known types, required fields, duplicates and byte bounds. Invalid or truncated authority refuses visibly, never resets to empty. Same-UID host programs are trusted; private IPC/socket permissions prevent other users, but do not sandbox hostile same-UID code. No browser registration endpoint or arbitrary RPC proxy is added.

## Process-evidence seam (PQ-1)

Existing procutil.Source.Identity at source.go:46 deliberately drops the native error; do not use it to infer death. Add the additive `Source.ReadIdentity(pid int) (string, error)` which returns the existing injected reader/nativeIdentity result unchanged for a valid positive PID; invalid PID is an explicit validation error, never native absence. Existing Identity/StrictIdentity consumers and equivalence tests retain their semantics; no host snapshot/table scan is needed.

Pure `ClassifyEndpointProcess(expected string, observed string, err error)` produces alive (matching nonempty identity, nil error), replaced (different nonempty identity, nil error), dead (errors.Is ESRCH or os.ErrNotExist), or unknown (all other errors or empty identity without positive absence). Darwin nativeIdentity already performs its own signal-zero check for an empty sysctl result and returns ESRCH for a zombie; Linux's exact /proc/PID/stat reader returns os.ErrNotExist for absent/zombie. Preserve errors, including a nonempty birth plus terminal error; EPERM/EACCES, EIO, EINVAL, unsupported and malformed evidence remain unknown. Do not use broad command-read IsProcessGone for identity reads. Positive replaced/dead evidence retires only the captured process incarnation; application nil retires only the exact captured attachment. Before/after observations must both confirm the candidate birth to publish a new binding.

## Operating envelope and visible recovery

This is editor attach/close IO, never keystroke IO. Register/lookup CLI deadline 3 s including bounded request transport; native identity and RPC probes/notification each at most 1 s within owned operation deadlines. Unregister does no RPC and fits the consumer's 500 ms wait when possible; expiry remains unknown, not a claimed hard latency guarantee. All started work is canceled/joined at service shutdown. At most 4 validating requests, 64 queued requests; overload returns unavailable before admission. Publication is serialized; slow process/RPC probes never hold the authority mutex. Probe results apply only if their captured revision/identity still matches.

One named owner cleanup task runs at startup and on explicit endpoint operations, and on a bounded Couch-owned maintenance cadence (one minute, serial, at most 64 distinct process identities per pass with fair continuation). No full host process table, per-buffer ticker or Parley polling. Within a read/pass, identity observations are shared once per distinct PID, bracketed around endpoint verification when needed. A process with N endpoints costs O(N) endpoint checks over a complete sweep, not N process-table scans. Count N8/16 publications, retries, replacements, exact removals and sweeps; assert linear store mutation/probe/notification counts and constant state size per repeated reopen of the same process/document pair. Each mutation rewrites one bounded snapshot, O(S) encoded bytes for S retained pairs; M new registrations therefore cost O(M*S) bytes, up to the explicit 8 MiB per publication, rather than a claimed linear byte budget. Batch unregister publishes once for its entire batch. Tests count publication bytes as well as calls and reject extra rewrites. This attach-only tradeoff avoids a second journal/compaction protocol.

Actions: explicit open/reopen; retry identical uncertain operation; close editor; inspect/restart Couch when blocked.

| Internal state/failure | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| validated durable current | available | use normally | none introduced |
| displaced/retired attempt | superseded | explicit reopen | old editor stops serving remote work; buffer retained |
| desktop already current, headless candidate | desktop active | use desktop | no new headless binding |
| process/RPC/namespace unavailable | unavailable | retry identical operation | no success claimed; buffer retained |
| publication outcome unknown | needs inspection | inspect/restart Couch | previous/new binding may be committed; no rollback claim |
| capacity exhausted with live evidence | unavailable | close unused editors, retry identical operation | no evidence eviction |
| proven stale current | unavailable | explicit reopen | stale endpoint removed, document content untouched |

## Architectural decisions

- ARCH-DRY: one Couch authority, existing document IDs and namespace resolver; no second nvim registration database.
- ARCH-PURE: replacement/retry/release rules are values and transitions; process, RPC and durable publication are injected owners.
- ARCH-PURPOSE: M1 explicitly unblocks #405; full adapter/phone purpose remains named in M2/M3, not silently dropped.
- ARCH-MOCK: stateful editor/process/storage seams plus installed-nvim private conformance, portable explicit roots.
- ARCH-CONSTRAINTS: attach/close workload, request/reply/store/capacity/concurrency/time limits above, counted operations and snapshot bytes.
- ARCH-SECURE: namespace/document/socket/process boundaries are typed and checked; no arbitrary RPC, same-UID trust explicit.
- ARCH-ORDER: native probe -> revision compare -> durable publication -> ack/notice; lost ack, delayed results, supersession and exit sequences are controlled tests.
- ARCH-FUNERAL: retained high-water evidence ends only on confirmed process incarnation death; fixed writer caps, bounded stages and joined handles; socket/runtime removal stays with its owning editor.
- ARCH-EVOLVE: persisted root/nested unknown fields survive all rewrites; malformed authority refuses rather than becoming an empty registry.

## Function-level test strategy (PQ-2)

| Risky production function | Adversarial class and mechanical test strategy |
|---|---|
| ValidateEndpointRequest (model.go) | Fuzz bounded typed input/decimal counters and identifiers; invalid shapes cannot reach the transition function or publication seam. |
| ApplyEndpointEvent (state.go) | Seeded/generated event sequences with an independent latest-authority oracle; inject retry, supersede, release, rename, restart and late completion, asserting at most one current binding and no terminal-attempt resurrection. |
| DecodeEndpointState / EncodeEndpointState (codec.go) | Fuzz truncation/duplicates/known-field type errors; retain opaque root/nested sentinel values through actual decode/rewrite/reopen, and reject input/encoded writer limits. |
| Source.ReadIdentity (procutil/source.go) / ClassifyEndpointProcess (process.go) | Exhaust typed native evidence including birth+terminal-error, PID reuse and permission/unsupported/empty/malformed uncertainty; assert errors survive and no Table call occurs; native child conformance confirms platform mapping. |
| EndpointOwner.Register (owner.go) | Stateful storage/editor/process doubles hold each validation/publication/ack boundary; race replacement and cancellation against completion, assert exact generation/current authority and unchanged idempotent retry effects. |
| EndpointOwner.Unregister / Lookup / Sweep | Compare exact captured tuple at retirement after held RPC/process observations; generated stale exits and successor installs cannot erase new authority; batch release performs one publish and no editor RPC. |
| EndpointOwner.Close and EditorRPC.Describe / Superseded | Controlled partial frames, delayed completion and cancellation with owned joins; typed callbacks only, reply-byte/deadline caps, notification never grants authority or rolls back a published successor. |
| runEditorEndpointCLI / handleEditorEndpoint (couchcmd integration) | Real private IPC and bounded stdin/reply fixtures validate all JSON exit codes and namespace isolation, including one outside-pane registered lookup; unchanged submit remains a legacy control. |
| EndpointStore.Load / Publish | Portable real directory with controlled failed writes/rename/sync; exact retained bytes and blocked uncertainty across reopen; N8/16 publication counts/bytes and repeated-reopen state plateau defend declared bounds. |

Nvim-native conformance and integration cases remain the separate final task below; they qualify the stateful seam rather than replace pure adversarial tests. Each behavior change starts with an observed failing test, then minimal implementation and GREEN/race verification.

## Tasks — single whole-issue producer boundary

- [x] Pin consolidated CLI/ack/callback contract with parley.nvim:1, record receipt and get hub plan approval. No implementation before change-code.
- [x] Implement the named pure model/codec/classifier functions from the strategy table, with RED/GREEN at each function and independent authority/opaque-field oracles; commit each coherent unit.
- [x] Add error-preserving Source.ReadIdentity through the existing injected native reader; preserve existing public helpers and test exact positive-death versus unknown evidence without a host-table scan.
- [x] Implement the named portable store, owner, EditorRPC and CLI/service functions behind the stateful seams after their controlled-order tests fail. No IO under pure transition/authority locks; started operations are owned, canceled and joined.
- [x] Exercise N8/16 operations/publication bytes and fixed-state repeated reopening; then run focused race suites for editorendpoint, procutil, couchcmd and sessionsync. Preserve discovered failure evidence and fix the whole class.
- [x] Qualify fixed RPC calls against installed nvim in explicit empty-base private roots: two editor processes, several buffers, desktop supersession, old detach, live-PID buffer wipe, process death and cleanup. Qualify held/failed/canceled notification delivery through the production owner seam and stateful editor double. Shared disposable Couch harness only for fleet scenarios; no operator sessions, live socket roots or #759 stage changes.
- [ ] Update atlas/README, artifactpath source/family classifications and affected-reader certificates individually; run repolint, artifactpath and make test-affected-inventory. Commit checkpoint, push, record required suite and close #758 through the normal whole-issue close.
- [ ] On SHIP, enter normal merge; report the landed producer contract/commit to parley.nvim:1 and hub. #405 and #798 consume the landed producer; never stack on #759.

## Deferred scope

#798 preserves the document backend, headless-editor provisioning and integrated real editor/phone acceptance. It depends on #758, parley.nvim#405 and #759 and requires its own detailed plan before code. #760 owns phone UI. No milestone-only partial landing or unlanded branch dependency.

## Revisions

### 2026-10-22 — producer first, consumer contract before code

TL35c71a59 dispatches M1 producer because no endpoint API exists. #759 remains on its separately pushed branch, code SHIP with passing suite, held for operator acceptance. New #758 branch is based on main, not on #759. Consumer coordination adds monotone attachment attempts, batch exit, an exact fixed endpoint query and callback fencing, so lost acknowledgements, restart and live-PID buffer retirement have explicit evidence. No implementation yet.

### 2026-10-22 — approved whole-issue split and final consumer confirmation

TL79aac81f approves producer-only scope and whole-issue landing. Filed unclaimed #798 with dependencies #758, parley.nvim#405 and #759 for the original adapter/security-integration purpose. Removed the former M1/M2/M3 review rows; #758 closes once as an atomic producer. The original milestone decomposition above was a planning proposal, never implemented. #759 operator acceptance takes priority when called.

Parley final contract confirmation7d43957e accepts the fields, monotone uint64 decimal-string AttachID, bounds/codes/deadlines, namespace/private-root ownership, live-PID nil retirement and notice-before-ack handling. Corrected the shared boundary: Parley retires advertised state and listener, but cannot revoke native built-in RPC on established connections. Generation-gated operations and channel retirement belong to the future Couch adapter (#798), with typed editor methods also rejecting retired tuples. No arbitrary-RPC revocation claim. Native SIGKILL leaves runtime directories/sockets; Parley is reviewing that cleanup separately with the TL. Couch never deletes those external paths.

### 2026-10-22 — plan-quality round1 evidence and strategy corrections

PQ-1: Source.Identity suppresses errors, so explicitly add Source.ReadIdentity returning the existing injected native result and ClassifyEndpointProcess with narrow identity-specific positive absence. All permission, unsupported, empty/malformed and ambiguous errno evidence remains unknown; no full-host snapshot or new native implementation. PQ-2: name risky functions and a mechanical adversarial strategy for each, compressing the former case inventories into generated invariants, fuzz/opaque round trips and controlled IO ordering. These refine execution of the approved producer scope without changing the pinned consumer contract. Parley3cfbb244 confirms its plan pins aa24764a and separately owns stable runtime-parent/PID+start marker cleanup after positive-dead evidence.


### 2026-10-22 — approved implementation refinements

The fixed callback seam now invokes installed nvim --server/--remote-expr as a bounded owned child rather than adding the Go RPC client, per TLe0091eff. Fixed Lua callbacks receive separately JSON-decoded data; no arbitrary code or browser-controlled method is exposed. A 1s operation budget is followed, when needed, by the existing 2s process-group cleanup grace. CurrentBinding projects the selected older desktop when a newer same-process headless attempt is terminally refused; the finite Current metadata combination is validated and unknown-preserving.

TL6bf10860 approves additive RejectDuplicateKeysPreserveNumbers, SplitObjectPreserveNumbers and EncodeObjectPreserveNumbers as this producer's codec prerequisite. Only new endpoint state uses them; existing strict readers/equivalence tests remain unchanged. Decode the exact owned subset with typed json.Unmarshal after structural validation, so nested RawMessage children retain valid numeric lexemes without float conversion while known types stay strict. Test unknown over/underflow exponents, precise fractions, huge integers and negative zero at root/record/binding/Current, store reopen, duplicate aliases and known numeric overflow. #759 has a sibling checker: whichever lands second must reconcile to one shared implementation (ARCH-DRY), never permanent duplication.

### 2026-10-22 — whole-issue review correction plan

BR-1 fences the negative response and retirement together under the authority lock; a changed selected generation returns stale. BR-2 separates deterministic candidate encoding (size exhaustion returns capacity) from an invoked durable writer (failure is uncertain and poisons). Tests keep healthy lookup/unregister/cleanup usable after no-write refusal. BR-3 adds controlled pending/current editor attachment and held/error/canceled notification behavior; previous native tests qualified tuple/identity/death, but the checked held-notification claim was premature and is now pending. Test publication before notice, failure preserving successor, and shutdown cancellation/join. These correct the approved contract rather than introduce new states. #801 separately owns the external shared-fleet bridge; #758 remains producer-only.

### 2026-10-22 — qualification correction completed

The fixed-callback native tests cover real tuple/identity/death and owned child cleanup. Held/error/canceled notification controls belong to the stateful owner RPC seam; that qualification now proves durable publication before delivery, failed-delivery successor/retry preservation and Close cancellation/join. The checklist separates those evidence kinds instead of claiming a held native callback. BR-1's delayed-negative matrix and BR-2's near-capacity/continued-use tests pass, including same-sentinel writer-error classification. Slow native probes/notices remain outside the authority mutex; one bounded durable publication remains serialized inside it. No new authority, reset path or external fixture is introduced.

### 2026-10-22 — BR-4 actual byte evidence pending

Round2 disposed the behavior/notification findings but found the checked N8/16 byte claim unsupported: existing fixtures counted only writes and records. Add actual writer-payload length/cumulative counters and assertions for bounded snapshot growth, O(M*S) cumulative growth, zero retry/no-op bytes, one batch publication, and retained-byte plateau through actual repeated reopen. Keep this task pending until measured tests pass. This completes the original bound, without changing the snapshot architecture or claiming linear total bytes.

### 2026-10-22 — BR-4 qualification completed

Actual writer-ledger bytes now back the N8/16 claim. Sixteen rounds with actual owner/store reopen measure snapshot upper growth, cumulative O(M*S), one batch publication and zero retry/no-op/reload bytes. Reopen size plateau permits only bounded canonical decimal widths; 64-bit counter widths are finite. Negative controls reject duplicate publications and growing retained-attempt metadata. Root race passed3.218s: 648152B/144 publications atN8 and2415268B/272 atN16, final snapshot increases10B/17B. No production change, journal or incremental-writer mechanism is introduced.
