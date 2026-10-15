# Couch agentic document caller contract

Agreed for pair#573 and parley.nvim#332 (universal-agent-input-interface S1).
The registry and ledger belong to the selected Couch namespace. Registration
does not change the document or submit a message.

```
couch --session-sync register FILE --owner SLOT [--json]
couch --session-sync lookup FILE [--json]
couch --session-sync list [--json]
couch --session-sync forget FILE [--json]
couch --session-sync rename OLD NEW [--json]
```

Save the named markdown file, register it successfully, then send the existing
`submitted: FILE` message to the returned effective owner. A send failure may
leave the file registered; retrying registration preserves its identity,
registration horizon, snapshots, hashes and folds. Different fallback owners
conflict. An explicit `couch-owner:` wins; legacy `owner:` is routing metadata
only on a `type: session-sync` file, so business metadata stays meaningful.

JSON uses the existing capitalized `Code`/`Error` envelope. Successful register,
lookup and rename return `Registration` with `ID`, canonical `File`, effective
`Owner`, fallback `RegisteredOwner` and RFC3339 `RegisteredAt`. Mutations return
`Changed`; an idempotent replay returns false. Registering a new alias spelling
retains it for lock authority and submit history and returns true once, without
resetting the document identity or its metadata. List returns `Registrations`,
an array of persisted bindings (its Owner is the fallback owner, without
reading every document); use lookup for the current effective owner.

Lookup is read-only and independent of the delivery-recording marker. Only
exit 0, `Code: "ok"` and an explicit `Registration: null` prove absence. A
nonzero result, unreadable document/registry, invalid or missing JSON, or a
missing Registration field means unavailable. An older binary can emit usage
text instead of JSON; this also means unavailable. Never auto-register or
disable a lock based on a failed probe.

New verb refusals exit 1 with `Code`/`Error`: `owner-conflict`, `target-conflict`,
`not-registered`, `registry-invalid`, `registry-busy`, `registry-limit`, or
`registry-uncertain` where applicable. Ordinary IO errors also exit nonzero.
Existing writes retain `operator-lock` (exit 3) and `stale` (exit 6).

For an unregistered typed session-sync file, first register OLD successfully
while it and its legacy companions still exist. Registration imports its
horizon, shadow, folds and history. Rename never adopts vanished legacy state.

Rename changes Couch metadata **after** the caller's successful filesystem
operation. OLD can be absent (including a path through symlinked ancestors),
or can still exist after saveas. NEW must exist. The operation keeps the ID,
owner, registration time, snapshots and folded events; old-path submits remain
owed using their historical ownership interval. OLD ceases to be a writable
registered path. An occupied target refuses without changing either binding.
A repeated identical OLD-to-NEW request returns the same effective Registration
and Changed false, without requiring a second lookup.

Parley owns moving the file, its `.side/` folder and backlinks, its `.lock`
sidecar, and its buffer baseline. An agent doing its own filesystem move owns
the same effects. Couch moves none of these: subsequent writes inspect NEW.lock.
Partial filesystem progress remains visible to the caller. `registry-uncertain`
means metadata publication may have happened; retry the exact same command and
arguments to confirm durability. Do not undo a filesystem effect based only on
an uncertain reply.

Agentic status persists until forget. Forget does not remove the document or
the operator's lock and works for a deleted document. Typed session-sync files
continue via legacy metadata after a durable handoff of current state/shadow
and submit history. Repeated forget succeeds. Existing session-sync files need
no registration.

Capacity is 4096 registrations and a 16 MiB registry, with no eviction. Overflow
refuses and advises `couch --session-sync forget FILE`. Rename history is capped
at 256 intervals plus 256 additional admitted aliases per document and is never
pruned merely by age: the delivery ledger can retain old rows below its
size-triggered trim threshold. Registered document/shadow size is bounded at
4 MiB and state at 1 MiB including reserved legacy handoff history;
registration, alias admission, rename, shadow refresh and writes reserve that
complete representation before growth. Prospective writes and legacy imports are
checked before changing authority or document content. Registry waits and diff
execution are bounded at two seconds.

A held operator lock always blocks agent writes. A valid lock timestamp older
than 30 minutes adds its age and advice to ping the operator; it never expires
the lock. Invalid/future timestamps retain the refusal with unknown age.

The TL rebuilds Couch after #573 lands. The combined plain-note live check is
scheduled after parley.nvim#332 is staged, with operator Couch/nvim restarts;
automated Couch verification does not claim that editor acceptance.

For real CLI conformance, use an existing fresh temporary root with
`COUCH_ISOLATED_ROOT` set. Set `COUCH_STORE_DIR`, `COUCH_PAIR_DATA_DIR` and
`COUCH_IDENTITY_DIR` to distinct directories inside that root; clear inherited
`PAIR_DATA_DIR`, `XDG_DATA_HOME`, `COUCH_TREE`, `COUCH_THREAD_SCOPE`,
`COUCH_THREAD_TAG`, `PAIR_SESSION_NAME`, `PAIR_LAUNCH_NONCE` and
`COUCH_CAPTURE_DIR`. Initialize this empty isolated singleton via public
`--adopt-store STORE --pair-data PAIR --identity-dir IDENTITY`, then repeat
with `--apply` and the preview's `digest`. Keep the same environment and roots
for every command. Registry/ledger live under `STORE/session-sync`; no running
supervisor is needed for register/lookup/rename. Status additionally needs a
recording marker/ledger fixture. A store override alone does not isolate the
account-wide singleton authority, and an uninitialized singleton refuses as
UNMIGRATED instead of proving absence.
