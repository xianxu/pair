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

## Collaborative snapshot (#838)

`cmd/internal/editorial/` adapts the shared `pkg/sidedoc` chat scanner for one
collaborative mode. `read --all FILE --json` returns `Read.Editorial` with chat
regions, protected source ranges and the shared compatibility notices verbatim.
`EditorialCoordinates` is `document-byte-offsets`; section reads also use
complete-document coordinates. Run `diff` before the whole read, which
acknowledges the snapshot. There is no runtime mode or processing-brief catalog.

The shared [collaborative editing contract](../.agents/skills/xx-universal-review/collaboration-modes.md)
and [round instructions](../.agents/skills/xx-universal-review/round.md) own tag
and item grammar, question/command routing, edit scope, briefs and side-doc
handling. Direct operator edits are content. The reviewing agent interprets
ready markers: questions and unclear commands branch to side docs; clear
commands act within their scope. Couch does not enforce that semantic choice
with a natural-language classifier. New writes produce clean text and highlight
metadata, including in documents carrying retired presentation keys.

A standalone `@@chat@@` line directly above an ATX heading marks its section,
including descendants, as chat. Tagged sections use numbered asks. Legacy chat
scalars, maps and adjacent heading comments remain readable with their old
bullet/numbered grammar, including when a tag names the same heading. The
innermost chat section governs its items. Misplaced tags remain ordinary text.
Retired mode and instruction keys, including malformed or unknown values,
produce nonfatal notices rather than selecting settings or failing a round.
The agent reports the shared notices once per round, word for word.

Normal writes never add, remove or rewrite legacy settings. The shared edit
guard protects real chat tags, their heading attachment and legacy source bytes
before file, Shadow or round publication. It also preserves chat-item text,
allowing an appended answer link. A syntax-only or code ending refuses when an
append cannot preserve Markdown structure and produce a real answer hyperlink.
Agents pair side-doc links with `discussions:` entries under the shared contract;
new side docs use the month layout without mode keys. Existing flat and `qa/`
answer links and their metadata remain valid.

Section hashes bind the body plus document and ancestor authority. They are
opaque tokens returned by read/write; unrelated prose can change concurrently,
while a changed chat tag or protected legacy source can stale the token even
when the section body is identical. Existing H2 addressing remains unchanged.

Explicit operator migration uses:

```text
couch --session-sync migrate-editorial FILE --json
couch --session-sync migrate-editorial FILE --confirm --if-hash BEFORE_HASH --json
```

The preview returns an exact map-to-tag candidate and complete source SHA256.
Confirmation recomputes that transform under the existing document transaction
and operator-lock guard; it accepts no supplied replacement body. Stale source
requires a new preview. Conversion must preserve heading routing and eligible
asks. It refuses, naming the affected section, when numbered-only tags would
lose legacy bullet asks or whole-document preamble asks. It never renumbers the
operator's text, invents a compatibility flag or fills canonical mode defaults.
Unmatched names or conflicting assignments that prevent lossless conversion
also refuse. Keep the legacy source when conversion is unsafe; ordinary review
continues to read it. Unrelated metadata and source bytes remain intact.
Shadow/state stay unread so the operator change appears on the next round.
This confirmation is a workflow boundary, not authentication against other
code running as the same user.

Historical `Round`/`Span` presentation values retain their direct/highlight/propose
meanings through readers, records, commits and resume; absent span values fall
back to the round, then highlight. Known-field types and duplicate-key checks
remain strict, and unknown nested extensions survive rewrites. Only newly
written edits always use highlight. The persisted compatibility seam and shared
fixture are mapped in [review-workbench.md](review-workbench.md); Parley's
consumer acceptance remains independent of Pair's headless tests.

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
checked before changing authority or document content. Writes prepare their
document, shadow, post-fold state bytes and result hash together; publication
consumes that same bundle. Registry waits and diff
execution are bounded at two seconds.

A held operator lock always blocks agent writes. A valid lock timestamp older
than 30 minutes adds its age and advice to ping the operator; it never expires
the lock. Invalid/future timestamps retain the refusal with unknown age.

## Registered operator acquisition (#777)

The editor acquires and releases through the same registry transaction that
registered `write` and `apply` hold through document, shadow, state and round
publication:

```text
couch --session-sync operator-acquire FILE --id ID --turn TOKEN --json
couch --session-sync operator-release FILE --id ID --turn TOKEN --json
```

ID is the current registration ID. TOKEN is a caller-generated 32-character
lowercase hexadecimal nonce for this operator turn, retained across lost replies.
FILE resolves to the current registered canonical document. Neither command falls
back to an unregistered legacy file. These commands run in-process without a
supervisor; a stopped daemon alone does not make acquisition unavailable.

Success exits 0 with `Code: "ok"`, `Error: ""` and `OperatorLock` containing
`ID`, canonical `File`, `Holder`, `Turn`, `LockPath`, exact `LockText` and `State`.
Acquire states are `acquired` and `already-held`, both with Holder `operator`.
The sidecar has additive `document: ID` and `turn: TOKEN` lines alongside holder
and time. Same-turn retries preserve the exact bytes and timestamp; foreign,
legacy or malformed operator protection is never adopted. Unknown sidecar lines
survive retries and moves verbatim.

Acquire always returns LockPath equal to canonical File plus `.lock`, and those
exact LockText bytes must exist there. Same-token alias-only protection is copied
to that canonical witness under the registry transaction before acknowledgment;
an existing same-turn canonical proof takes precedence over alias copies.
Matching aliases keep their existing move/lock cleanup owner.

Acknowledgment proves earlier registered writers completed before protection
and future writers see it. The editor must then perform its unconditional
freshness check and complete any clean native reread before admitting input
(parley.nvim#408). Existing clients that create sidecars directly retain
conservative lock checks and the writer's best-effort post-write warning; they
do not participate in this new exclusion guarantee.

Release states are `released` and `already-released`, with the requested ID,
canonical File and Turn echoed, empty Holder/LockText and canonical LockPath.
Absence does not assert past ownership. Release removes only that exact turn's
matching sidecars; foreign protection refuses without deletion. Both release
states confirm every admitted containing directory, including already-absent
sidecars, so retry reconciles an unlink followed by failed directory fsync.
Join an outstanding command before releasing; after release, begin a new turn
with a fresh token rather than replaying an old acquisition.

| Outcome | Code / exit | Caller action |
| --- | --- | --- |
| Foreign or unproven protection | `operator-lock` / 3 | Inspect; never delete or infer ownership |
| Registration identity changed | `operator-identity` / 1 | Refresh identity explicitly |
| Missing/unregistered/unsafe document or lock path | `operator-path` / 1 | Inspect the path; no legacy fallback |
| Invalid ID/token | `operator-invalid` / 1 | Correct the request |
| Registry busy or invalid authority | `registry-busy`, `registry-invalid` or other explicit refusal / 1 | Inspect or retry explicitly; never classify as unavailable |
| Namespace/runtime/store access unavailable | `session-sync-unavailable` / 7 | Consumer may use its visibly unconfirmed availability policy |
| Sidecar effects or durability uncertain | `operator-lock-uncertain` / 1 | Preserve the token; retry the identical command |

Malformed argv retains the CLI's existing `usage`/exit 2 behavior. A missing or
invalid success acknowledgment is never proof of protection. Parley's bounded
500 ms call owns child termination/join; timeout can leave a published lock,
and an identical acquisition recovers it. No sidecar expires automatically.

For a move, acquire OLD, copy exact LockText to NEW before the existing registry
rename, and retain ID. Recover TOKEN from the move journal's `turn:` line;
release NEW with that same ID/token. Protection identity is path-independent;
an old-path request cannot delete a replacement registration's lock. Parley
still owns old-sidecar and move-journal cleanup. A private deterministic stage
is bounded to one per document and reclaimed by the next successful operator
transaction under the registry lock, including after a killed publisher.

Producer acceptance uses the real public command on disposable adopted roots:
daemon-absent acquire, lost reply, writer refusal, move/release and unavailable
outcomes. The actual editor freshness/input proof follows in parley.nvim#408.

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
