# Registered editor endpoints

The #758 producer advertises a live private nvim endpoint for an existing registered document. It does not expose remote editing or arbitrary RPC. parley.nvim#405 consumes it; #798 owns the later typed document adapter/security integration.

The existing Couch singleton owns one EndpointOwner in `cmd/internal/editorendpoint/owner.go`. `couchcmd/editor_endpoint_service.go` publishes a separate private framed-JSON endpoint via the existing Couch transport, and joins service/owner/maintenance at shutdown. Invalid retained metadata leaves this service unavailable while ordinary Couch remains usable. The CLI never writes the authority file or launches Couch.

Commands:

- `couch --editor-endpoint register --json`: at most8KiB JSON stdin; Version, Namespace, DocumentID, EditorIncarnation, AttachID, PID, Socket, Buffer and Mode.
- `couch --editor-endpoint lookup --namespace PATH --document-id ID --json`: observe the current validated binding; no editor acquisition.
- `couch --editor-endpoint unregister --json`: at most256KiB JSON stdin, one namespace/process incarnation and up to256 exact DocumentID/AttachID/BindingGeneration entries. No editor RPC during exit.

The PascalCase envelope includes Version, Code, Error, Changed, EditorEndpoint (explicit null when absent) and optional Results. A binding carries all input identity plus File, native ProcessIdentity and random BindingGeneration. The document CLI supplies canonical `Registration.Namespace`; DocumentID survives rename. A stale old buffer path refuses until the editor confirms its move.

EditorIncarnation is a per-process32lowercase-hex random token. AttachID is a monotone positive uint64 decimal string: allocate on explicit attach/reopen, retain across lost-ack retry. Namespace+DocumentID has one selected binding; a new explicit desktop attach wins, and headless is fallback only. Same-current retries preserve generation without new publication. Superseded/retired equal-or-older attempts never reacquire. A refused headless high-water can coexist with an older still-selected desktop from the same process; CurrentBinding is the single projection for that finite state.

The pure model/strict retained-raw codec live in model.go/state.go/codec.go. `store.go` publishes one bounded namespace snapshot and reusable stage, named by artifactpath.EditorEndpoints. Unknown root/record/binding/current members survive rewrites, including numeric lexemes outside float64 range. The additive strictjson preserve-number checker/SplitObject/EncodeObject entry points validate retained-raw structure without converting numbers; owned fields remain typed and existing strict readers keep their acceptance contract. Damaged authority is not reset to empty; a refused encoded-size candidate returns capacity without changing or poisoning healthy authority, while an invoked writer failure poisons that owner until committed evidence is inspected/reopened. Up to4096 retained process/document pairs,256 active per process and8MiB encoded metadata are enforced before publication. Evidence is collected only for the exact confirmed dead/replaced process incarnation. No age eviction can let a delayed retry reclaim authority.

Source.ReadIdentity preserves native per-PID errors without changing existing Identity callers. `process.go` classifies positive death/replacement separately from unsupported/permission/malformed uncertainty. Native identity brackets an exact typed endpoint query. A valid nil reply proves only the queried attachment absent; RPC error/timeout never does. Lookup can compare-retire the exact stale generation, and the minute maintenance pass fairly checks at most64 process identities without idle nvim RPC polling. No Parley watcher is added.

`nvim.go` runs the installed nvim client through OSProvisionIO with literal --server/--remote-expr argv, fixed luaeval callback expressions and separately JSON-decoded data. One-second execution budget preserves the existing joined TERM/KILL grace; parent stdout/stderr are byte-limited. No custom MessagePack codec or new module dependency. Native client memory is not bounded by the parent's output cap.

The fixed Lua callbacks are parley.remote_session.endpoint(query) and superseded(notice). Notice matches Namespace, DocumentID, EditorIncarnation, AttachID and BindingGeneration. It follows replacement publication and is advisory; failure cannot restore old authority. Parley retires only its own advertisement/listener and ignores stale notices. Built-in Neovim RPC on an existing connection cannot be revoked by the plugin: #798 must gate each operation by current generation and retire old channels, with typed editor methods rejecting retired tuples.

Sockets have an owned physical0700 parent, Unix type and same-user ownership. Couch never unlinks editor-owned sockets/directories. Parley owns normal and positive-dead crash cleanup of its runtime paths. Same-UID host code is trusted; this API is not browser-accessible.

Exit0 is ok;2 invalid input;3 unavailable/capacity;4 superseded/stale/conflict/not-registered/desktop-active;5 uncertain. Retry uncertain intent unchanged; explicitly reopen after supersession; close unused editors at capacity; inspect storage/restart Couch for blocked authority. A lookup's null confirms absence only with Code ok; unavailable is not absence. Exit batches can time out without a false release claim.

See the [approved plan](../workshop/plans/000758-editor-endpoints-plan.md) for sequence/count tests and producer qualification. Private fixed-callback native conformance is not evidence of the future phone editor working.
