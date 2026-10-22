---
collaboration-mode: discussion
---

# Registered editor endpoints implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 for execution strategy. The SDLC binary owns mandatory fresh-context reviews. Do not implement before the hub approves this plan and change-code admits it.

**Goal:** Give parley.nvim#405 a confirmed, incarnation-fenced editor endpoint producer before building the remote document adapter.

**Architecture:** The existing Couch singleton owns one durable endpoint registry per canonical Couch namespace. A pure transition model chooses the current editor by stable document ID; IO adapters validate the actual private nvim endpoint and publish transitions before acknowledgements. Parley advertises asynchronously, receives bounded replacement notifications and never polls or automatically reclaims a superseded binding.

**Tech stack:** Go, existing sessionsync registry, procutil.Source, durablefile and private Couch IPC; a pinned standard Neovim Go RPC client behind an injected typed EditorRPC seam (no hand-written MessagePack codec or shell-expression interpolation). No public HTTP route, Cloudflare change or #759 code dependency in M1.

## Scope and alternatives

M1 is the producer prerequisite explicitly dispatched by TL35c71a59. M2 supplies the registered-document backend after parley.nvim#405 and pair#759 land; M3 qualifies the integrated real editor and remote flow. Those later slices require their own detailed execution refinement and approval before implementation. M1 must not claim the phone editor works.

Recommended: Couch's single owner retains endpoint authority and retired-attempt evidence, publishing acknowledgements only after persistence. Alternatives rejected: a registry file independently rewritten by nvim duplicates ownership and permits stale retry takeover; discovery by periodic socket/process scanning cannot identify explicit desktop intent and adds polling. Neither solves lost acknowledgements after supersession.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| EndpointBinding, EndpointRequest, EndpointReply | cmd/internal/editorendpoint/model.go | new |
| EndpointState and endpoint transition function | cmd/internal/editorendpoint/state.go | new |
| Endpoint codec with retained unknown fields | cmd/internal/editorendpoint/codec.go | new |

Namespace + DocumentID is one logical document. It has at most one current binding. A process incarnation owns many attachment attempts, at most 256 active, and each process/document pair retains one latest AttachID/status. A replacement terminally retires the displaced attempt: no fallback resurrection. Explicit reopen advances AttachID. These values, decision rules and codec tests perform no IO.

### Integration owners

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| EndpointOwner | cmd/internal/editorendpoint/owner.go | new | serialized decisions, publication, pending operations and join |
| EndpointStore | cmd/internal/editorendpoint/store.go | new | portable namespace-local durablefile storage |
| EditorRPC | cmd/internal/editorendpoint/nvim.go | new | bounded fixed nvim callback queries/notifications |
| EndpointProcessSource | cmd/internal/editorendpoint/process.go | new | injected procutil.Source identity-bracketed observations |
| Stateful editor fixture | cmd/internal/editorendpoint/fixture_test.go | new | current/pending attachments, delayed RPC, death, replacement and retained channels |
| endpoint CLI/service adapter | cmd/internal/couchcmd/editor_endpoint.go, editor_endpoint_service.go | new | bounded stdin JSON, private broker socket and runtime lifecycle |
| Registration namespace projection | cmd/internal/sessionsync/registry.go; couchcmd/session_sync.go | modified | existing confirmed document register/lookup replies |

Concrete names are the intended declarations, not claims of delivered implementation. Owner/store/RPC are INTEGRATION even when invoked from pure decisions. Reuse durablefile, namespace resolution, the existing singleton lease and shared process observations; do not create a second daemon or per-buffer poller.

## M1 wire contract (consumer agreement before code)

Parley coordination: dea1b235, bcc436ae, fc5fd55f, 8b7dcc76 and 66ccd77d. Consumer accepted the following pins; the final consolidated text is sent for confirmation before change-code.

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

- `return require("parley.remote_session").endpoint(...)`, args=[query]. Query contains Namespace, DocumentID, EditorIncarnation, AttachID, Buffer. Return exact tuple plus PID, Socket, File only for that pending/current live attachment; otherwise nil. Couch brackets with native identity reads and validates the private socket and canonical registered path. This proves initial endpoint identity even across PID/socket reuse; the producer supplies ProcessIdentity in ack.
- `return require("parley.remote_session").superseded(...)`, args=[notice]. Notice contains exact Namespace, DocumentID, EditorIncarnation, AttachID, BindingGeneration. Send only to the displaced endpoint after replacement publication, outside authority locks. Failure never rolls back current authority or grants an old binding a retry. No periodic Parley watcher.

Parley validates all five notice identifiers, ignores stale/malformed notices, fences the matching attachment and existing proxy channels, stops only its owned listener and notifies once. Notice-before-register-ack is retained as unconfirmed and compared against eventual ack before showing active. Notification is advisory: subsequent Couch adapter operations must check current binding generation and the editor's typed methods must reject retired attachments even on established RPC channels. A canceled/missing notice cannot grant old authority. M2 must linearize its final nonblocking enqueue against replacement; M1 does not expose remote dispatch.

A live PID with endpoint(query)=nil positively retires only that exact attachment (buffer wiped, detached, moved or superseded). A timeout/transport error does not prove absence. Probe completions carry the complete original binding generation and owner revision; late nil/death/notice outcomes never remove a successor.

## Lifecycle, ownership and persistence

One EndpointOwner lives under Couch's existing lifetime singleton lease. The CLI only calls it. Startup reads a separate `editor-endpoints.json` under the canonical namespace; store/stage paths are classified by artifactpath. Durable state retains current bindings and per-process/document high-water attempt evidence across Couch restart. After restart no binding is usable until process/socket/attachment validation succeeds; no retry revives a superseded or retired attempt. A publication error pins uncertainty and blocks further mutation until exact evidence is reconciled or Couch is restarted and the committed record validated.

Transitions: absent+new valid desktop => current; current+exact retry => same generation; current+new desktop => publish new, retire old, then notify; desktop+headless => desktop-active; superseded/retired+equal-or-lower AttachID => superseded; fresh higher explicit AttachID => candidate; current+exact unregister or confirmed stale observation => retired; stale completion => ignored; unknown observation => retain evidence/unavailable. Observer departure never undoes a committed transition.

Durable high-water evidence is retained until the exact process incarnation is positively dead/replaced; missing/unreadable identity is uncertainty. Cleanup never depends on age alone. The 4096 retained namespace-wide process/document-pair cap and 8 MiB encoded file cap refuse new admission before publication; no live evidence eviction. Maximum 256 active endpoints per editor process. Confirmed dead cleanup frees capacity. Forgotten/replaced documents cannot regain endpoints via old IDs. Generation tokens prevent ABA even when the last dead row is collected.

Couch owns stale binding/evidence cleanup, not another application's filesystem. Parley creates one process-owned physical 0700 runtime directory and private Unix sockets, unlinks only its own exact socket after listener shutdown, and removes its own empty directory at exit. Couch validates ownership/type/private parent, never recursively removes runtime directories or unlinks a registered path that might now belong to a newer editor. Crash-orphan runtime directory reclamation is outside endpoint authority: follow the existing application runtime-dir lifetime/OS cleanup; do not add arbitrary path deletion. Confirmed native endpoint liveness is not inferred from a leftover file.

Persisted codec preserves unknown root/nested fields through register/replacement/unregister/cleanup/reopen, with strict known types, required fields, duplicates and byte bounds. Invalid or truncated authority refuses visibly, never resets to empty. Same-UID host programs are trusted; private IPC/socket permissions prevent other users, but do not sandbox hostile same-UID code. No browser registration endpoint or arbitrary RPC proxy is added.

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

## Tasks and review boundaries

### M1 — producer contract and endpoint registry

- [ ] Pin consolidated CLI/ack/callback contract with parley.nvim:1, record receipt and get hub plan approval. No implementation before change-code.
- [ ] Add pure model tests first: desktop/headless order; exact retry, conflict, lower/equal superseded retry; explicit higher reopen; old unregister after new attach; path rename retains ID; generation/AttachID overflow and capacities. Run RED, implement minimal transitions, run GREEN and commit.
- [ ] Add portable store tests first for unknown nested fields, truncate/duplicates/known-type errors, lost reply/restart after replacement, failed publication and fixed bounds. Run RED/GREEN; never heal damaged state by deleting it.
- [ ] Add stateful process/editor fake through the same EditorRPC seam: PID reuse, socket reuse, attachment nil with live PID, timeout/unknown identity, held probes then replacement, held notice then newer attach, notice-before-ack and live channels after listener close. Assert independent effect and state oracles.
- [ ] Implement owner integration and bounded command/notification lifetimes after the tests fail. No IO under pure transition/authority locks and no hidden unbounded goroutines.
- [ ] Add actual private IPC/CLI tests: one-time outside-pane document lookup and Namespace, register+lost-ack retry, root mismatch, full EditorEndpoint ack, 256-item compare-unregister, offline/unknown refusals, one namespace cannot touch another. Existing outside-pane submit unchanged. Run RED/GREEN.
- [ ] Add N8/16 counters and repeated explicit reopen plateau tests, positive-dead versus unavailable cleanup, exact generation fencing and crash/restart cases. Run focused race suites for editorendpoint/couchcmd/sessionsync.
- [ ] Qualify fixed RPC calls against installed nvim in explicit empty-base private roots: two editor processes, several buffers, desktop supersession, old detach, live-PID buffer wipe, held notification, process death and cleanup. Shared disposable Couch harness only; no operator sessions, live socket roots or #759 stage changes.
- [ ] Update atlas/README, artifactpath source/family classifications and affected-reader certificates individually; run repolint, artifactpath and make test-affected-inventory. Commit checkpoint, push, record required suite and close M1 through milestone-close.
- [ ] Report M1 contract/commit to parley.nvim:1 and hub so #405 can build against the landed producer. M1 has its own reviewed boundary; arrange normal landing of producer before dependent consumer code if the issue needs splitting (ask hub; do not stack on #759).

### M2 — document adapter (held)

After producer/consumer and #759 interfaces land, refine/approve a detailed slice for eligible-document read/subscribe, revision-safe paragraph edits, save+trusted submit, navigation resolution and headless fallback lifecycle through the existing DocumentBackend. No generic nvim RPC route. Preserve Permit kill gate plus current endpoint generation at final effect enqueue; stale channels cannot bypass them. Close its own milestone review.

### M3 — integrated real editor acceptance (held)

Qualify actual desktop/headless handover, phone edits/reflow/submit/native transcript and eligible navigation with #759/#760. Operator arranges live Access/iPhone check. Close whole issue only when this implemented scope works; M1 alone is a producer prerequisite, not remote-session completion.

## Revisions

### 2026-10-22 — producer first, consumer contract before code

TL35c71a59 dispatches M1 producer because no endpoint API exists. #759 remains on its separately pushed branch, code SHIP with passing suite, held for operator acceptance. New #758 branch is based on main, not on #759. Consumer coordination adds monotone attachment attempts, batch exit, an exact fixed endpoint query and callback fencing, so lost acknowledgements, restart and live-PID buffer retirement have explicit evidence. No implementation yet.
