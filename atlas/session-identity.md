# Session identity and storage

Pair separates identities that used to be partly conflated:

- **Repo scope** — a hidden, stable key derived from the cleaned repo root. It
  owns the scoped data directory and is not shown in user-facing labels.
- **Thread tag** — the immutable repo-local storage key. Direct Pair keeps the
  user-chosen form (`work`, `bugfix`); Couch allocates `C-repo-N` tags; existing
  `couch-<16 lowercase hex>` tags remain valid.
- **Human thread name** — optional mutable metadata on the durable ThreadStore
  record. It is neither a filename nor a zellij socket name.
- **Public session name** — the current zellij socket binding recorded in
  `session-names.jsonl` for one `{scope, tag}`.
- **Agent** — the resource running under a tag, such as `claude`, `codex`,
  `agy`, `muse`, `qoder`, or `grok`. A tag can have sessions from more than one agent
  over time.
- **Native session id** — the agent's own resumable conversation id. Fresh
  launches expose it as recovery state only after Pair establishes a completed
  causal round; an explicit scanner-authorized resume may establish it at the
  launch boundary.

## Couch allocation and terminal bindings

`cmd/internal/couchidentity` owns the host C and independent per-store N/M
allocation. C is a permanent store identity, not a running supervisor ID. The
singleton selection fixes the allocation-authority directory, defaulting to the
real UID account home's `.local/share/pair-host/`; `COUCH_IDENTITY_DIR` or adoption's
`--identity-dir` can name an existing legacy authority. Store counters live in the
selected canonical Couch namespace. Host floors commit before local
counters under ordered locks, so interrupted launches burn numbers. Strict bounded
storage refuses missing/corrupt authority. Local rollback recovers above host
floors; rollback of both authorities is unsupported without proven floors.
The descriptive repository token is capped at 64 ASCII characters and falls
back to `repo` when normalization is empty; C/N carry uniqueness.

Singleton adoption keeps the store and its allocation authority in place and
leaves existing C/N/M, tags and terminal bindings unchanged. Durable assignment
can name machine + slot without a Couch process ID; retained conversation names
still carry their original C. Copying or moving a store is not adoption: the
allocator treats a different canonical store path as a new enrollment. Preserve
both the local snapshot and its non-regressed host authority during recovery.
Explicit `COUCH_ISOLATED_ROOT` scopes independent authority and storage beneath
one test/diagnostic directory; changing HOME or XDG alone cannot move production
singleton ownership. See [Couch adoption](couch.md#singleton-adoption-and-isolation-366).

`couchcore.launchTrackedThread` persists a pending `SessionBinding` before its
blocked helper runs. Registration, including established-registration recovery
with a dead helper, promotes the binding before clearing Start. Unknown evidence
keeps capacity occupied. Warm attach keeps M; cold creation gets a new M.
`--couch-session-v1` and the consumed structured intent carry exact scope/tag,
name, start nonce and create/attach disposition independently of native resume.
Older launchers reject the leading flag; they cannot silently choose a name.

`launcher.SessionOwnerProbe` reads actual live pane commands between exact
server-generation observations. Complete scoped artifact paths prove ownership;
name/index presence alone does not. Foreign proof means absence for the selected
conversation, unknown refuses, and mutations revalidate the server generation.
Passive inventory avoids per-row pane queries. Compatibility index publication
replaces one exact scope/tag association, retaining unrelated addresses.

Durable current/pending bindings select the exact terminal for registration,
recovery, park, detach, and failed-launch cleanup; the compatibility index is
only a fallback for legacy records without a binding. Cold creation requires
absence of the previous terminal, including attached sessions. Warm attachment
carries the original server proof through retention and title preparation and
revalidates it immediately before the attach handoff.

The owner parser fixtures run in ordinary tests; live Zellij ownership and socket
budget conformance runs before releases supporting a changed Zellij version.

## Native-session forest inventory

`cmd/internal/sessioninventory` is the single model and scanner boundary for
native session storage (#155 M1). Its versioned Claude, Codex, Agy, Muse,
Qoder, and Grok scanners emit facts into a deterministic forest: complete roots, validated
native parent/child edges, and explicit unbound orphans. Missing, conflicting,
malformed, unreadable, or unknown-schema evidence is retained as a stable coded
diagnostic rather than guessed away. Stable IDs, ordering, chronology fallback,
artifact paths, and a forest-only canonical projection are pure functions.

All native I/O crosses one injected runtime: named storage roots, chunked file
reads, read-only SQLite, and process/open-file snapshots. The sibling
`sessioninventorytest` package supplies a persistent stateful fake, while
`make test-native-session-live` checks installed native shapes without printing
paths, IDs, or transcript content. Native parentage establishes topology only;
it is not evidence that a Pair tag owns a root.

#346 separates durable identity from optional transcript parsing. `QueryResumeTarget`
projects the current owner ledger: a confirmed UUID wins, otherwise a v3
existing-conversation resume request remains usable under probation. An unconfirmed
fresh Pair-chosen ID needs a matching root filename before reuse; if it has not
materialized, Alt+n starts fresh with a new UUID. This check uses metadata, not
transcript-body parsing. A failed or partial listing without a matching filename
keeps materialization unknown and refuses restart before destructive effects.
The pure history projection cannot inspect filenames and conservatively omits
unconfirmed chosen IDs; launch consumers use the runtime projection.
Legacy v1/v2 bindings
remain usable without their old device/inode proof. Missing files, cache loss,
and parser failures cannot erase this resume target. Conflicting confirmations
remain ambiguous; a fresh launch never inherits an earlier launch's target.

The selected scope owns `session-inventory-catalog.json`, a rebuildable cache of
scanner facts, filesystem fingerprints, parser offsets, and scanner state.
These validate parsed contents and suffix provenance, not durable UUID identity.

Launch preparation records the requested UUID and origin (`resume` or
`chosen-id`), Pair input offset, and metadata-only native-file boundaries. It
never preconfirms a resume by reading the old transcript. If a complete baseline
cannot be read, startup proceeds; the watcher acquires a complete observation
epoch later and considers only subsequent input and native bytes. Silence and
ambiguity retain probation.

`pair wrap` spawns `pair session-watch` for its launch ordinal after starting the
agent. The watcher lives with that agent across client detach. Fresh and resumed
launches share correlation: new files and appended activity in existing files
are candidates, but events beginning before the corresponding byte boundary
cannot confirm this launch. Current exact Pair sends followed by native
assistant/tool/error progress must uniquely satisfy the existing matching
thresholds. Repeated matches remain ambiguous; newest-file order is no tie-breaker.
A Pair-chosen UUID also permits a newly created matching root filename to serve
as acknowledgment when its absence at launch was known.

The watcher commits confirmation only while that launch is current. Requested
A and observed D are separate ledger facts, preserving both the effective
identity and debugging history. Same-root confirmation is idempotent; competing
confirmed roots are refused. Optional Codex lifecycle observation continues
after confirmation. An unbound v1 launch has no reliable boundary for historic
repair.

For a stopped unbound launch with a complete recorded baseline, run
`PAIR_DATA_DIR=<scoped-dir> pair session-repair <agent> <tag> --scope-key <scope>`
to preview exact prompt/progress correlation. Add `--apply` to publish the
current-launch confirmation. Preview is read-only; apply never rewrites saved
launch arguments or needs a live agent PID. Ambiguous or insufficient evidence
is reported without guessing.

For Codex, the same detached watcher remains attached after a proof-bearing
binding and incrementally follows only that authorized rollout generation. It
projects captured `task_started`, `task_complete`, and `turn_aborted` envelopes
with their native `turn_id`, timestamp, and absolute transcript-record offset
into `lifecycle-<tag>.jsonl`. A locked newline commit and stable
`(launch ordinal, artifact generation, transcript offset)` identity make retry
safe. `pair-wrap` opens the journal at its prior EOF before spawning Codex,
accepts only its current launch, and feeds committed records into the shared
notification reducer; partial lines wait, while replacement, truncation,
malformed records, or stale launches fail closed (`ARCH-IDENTITY`, `ARCH-DRY`).

`pair session-inventory [--agent ...] [--scope current|all] [--json]
[--conformance]` exposes the canonical forests, correlations, ambiguities, and
coded diagnostics. A dedicated public DTO keeps schema v1 exact: internal root
coordinates do not leak into evidence, and required position/fingerprint arrays
remain arrays even when empty. Conformance emits only agent/status/count/code
data. Pair's Go store and Neovim history navigation share the versioned
byte-counted log grammar while retaining legacy entries, so authored Markdown
separators round-trip.

`QueryResumeTarget` supplies Couch/launcher admission, review scoping, and
changelog identity without native-body reads. `QuerySession` separately supplies
parsed contents for optional context/token usage, title activity, and native
text consumers. Its parser/proof diagnostics do not change the durable target. That ledger
read uses the same chunked JSONL framer as transcripts, without arbitrary
record/file-size cutoffs (#297). Native transcripts, ledger rows, Pair logs and
configs, and SQLite result bodies have no matching writer ceiling; reader caps
made valid growing evidence disappear (#237, #297). Reads remain chunked at
64 KiB, with schema/identity/path checks intact. This is not a constant-memory
promise: full Codex/Claude/Muse/Qoder/Grok scans retain a complete record; Agy and
incremental validation retain observed records, and ledger/log consumers retain
their parsed input. The selected-
scope catalog is the shared persistent advancement owner: an accepted suffix is
published monotonically through `CatalogStore`, and later unchanged queries
reuse that parser cursor without rereading body bytes. Catalog loss falls back
to the durable ledger proof. On a filesystem with no generation token, a proof
artifact that is the same file and not smaller, but whose metadata moved (for
example growth, or a resuming agent bumping ctime without writing, #328), is
re-read from byte zero. The content, not the metadata, decides whether the
root still validates. A failed parsed proof leaves contents unavailable and records a
`binding_stale` diagnostic; the resume target remains usable. Neovim's review fallback uses the bounded `--owner`
projection rather than the diagnostic whole-inventory rendering. The Go review
target writer uses the durable target, including probation; an exact inherited
`PAIR_SESSION_ID` retains its existing precedence. Compatibility config retains launch arguments but
cannot establish identity. Alt+X reads local sidecars and paints its confirmation
without starting inventory/activity work; age/idle enrichment is omitted from
the modal. `make test-session-inventory-conformance` runs the one-second installed
metadata budget and all four provider comparisons. Run it for #156 verification,
before any scanner/provider-contract version change, and in the monthly operator
maintenance pass (ARCH-DRY, ARCH-PURE, ARCH-PURPOSE, ARCH-MOCK).

## Data layout

The global Pair data root is still `${XDG_DATA_HOME:-~/.local/share}/pair`.
Repo-scoped launch state lives under:

```text
<global>/repos/<scope-key>/
```

Tag sidecars keep their exact durable tag inside that scope:

```text
draft-<tag>.md
log-<tag>.md
queue-<tag>/
agent-<tag>
config-<tag>-<agent>.json
agent-default-<agent>.json
agent-ready-<tag>-<agent>.json
ledger-<tag>.jsonl
lifecycle-<tag>.jsonl
session-inventory-catalog.json
scrollback-<tag>-<agent>.raw
scrollback-<tag>-<agent>.events.jsonl
pane-<tag>-<agent>.json
```

Those names are descriptive storage vocabulary, not construction instructions.
Current code obtains them only from the `artifactpath` methods and exact
environment bindings below.

`cmd/internal/artifactpath` is the constructor authority for the complete
family list (including review, continuation, PID, parked, image, layout, and
diagnostic sidecars not repeated above). The launcher resolves the composite
address once, validates every result remains below the selected scope, and
exports exact `PAIR_*_PATH` bindings. Shell, Neovim, and KDL consumers use those
bindings directly; they do not combine `PAIR_DATA_DIR` and `PAIR_TAG`.

## Public session names

Zellij session names are globally visible, so Pair assigns a readable public
name through `session-names.jsonl` in the selected repository scope. The format
is:

```text
📁{repo}[-{residual tag tokens}]
```

Reads merge the former global index before the selected-scope index so live
pre-M5 sessions remain visible during upgrade; new rows are written only to the
selected scope. Missing files mean no bindings, while malformed or unreadable
present files fail closed before attachment, rename, restart, or orphan cleanup.

The first `pair/work` session becomes `📁pair-work`; a `pair` tag in the `pair`
repo becomes just `📁pair`, because a tag token already carried by the repo is
dropped. `parley.nvim` with tag `parley_nvim` becomes `📁parley-nvim`.

Three rules produce it:

1. **Repo** is the first alphanumeric token of the normalized display name
   (`parley.nvim` → `parley`).
2. **Residual** is the tag's tokens with a leading token matching the repo
   dropped — exactly one token, not the whole run, so tags `pair-x` and
   `pair-pair-x` stay distinct names.
3. **`📁` is the ownership prefix**, 4 bytes and needing no separator, where the
   previous `pair-` cost 5.

A second repo with the same display repo name and same tag gets a stable numeric
suffix, for example `📁pair-work-2`. The hidden scope key is stored in the index
row, not embedded in zellij names, picker rows, titles, or pane text.

### Why the prefix is load-bearing

The prefix is Pair's ownership marker in zellij's **global** namespace. It is
what keeps `delete-session --force` off a stranger's abandoned session — the
global list routinely contains foreign names. Discovery accepts **both** `📁` and
the legacy `pair-` (`isPairSessionName`); only `📁` is ever emitted.

### The budget is discovered, not assumed

A session name is a **socket filename**. On the machine this was measured on,
macOS allows **24 bytes** — and that number is the socket path's, so it varies
with username and is different on Linux (`~/.cache/zellij`). zellij's own
validator stays the oracle (`ProbeSessionName`), but since `#215` assignment does
not ask it once per candidate. **Acceptance is monotone in length** — a session
name is a socket filename, so if zellij takes an n-byte name it takes every
shorter one — and `sessionNameAcceptor` exploits exactly that, keeping a bracket:

    accepted   <= longestOK    every name this long or shorter fits
    shortestBad <= rejected    every name this long or longer does not

Only a length strictly between the two costs a subprocess, and each such probe
closes the gap. The ladder reuses lengths heavily across suffixes, so the bracket
converges in a handful of probes and the rest are arithmetic: naming went from 52
subprocesses to 3-4, flat as the index grows, where it had been O(threads this
repo ever had) inside couch's registration deadline.

The bracket learns from an **acceptance** as readily as from a refusal, which is
what makes it O(1) on every machine. An earlier cut went arithmetic only after a
rejection, so on a host whose socket directory is short enough that the longest
candidate fits — Linux `~/.cache/zellij` against macOS's temp path — nothing was
ever refused and it paid one probe per suffix, quietly restoring O(threads) on
the machines with the most headroom.

Every answer therefore derives from a probe of THIS machine; no constant is ever
used to decide acceptance. Where a numeric limit is wanted for a refusal
*message*, `measureAcceptedLimit` narrows one through the same acceptor and
returns `measured=false` unless it observed both an acceptance and a refusal — a
search that saturates at either bound has found a search bound, not a boundary,
and callers must say something else rather than quote it.

Three units are in play and each answers a different question — mixing them was
the original bug:

| unit | question |
|---|---|
| bytes | will zellij accept this socket name? |
| runes | where may a string be cut without splitting a character? |
| columns | how wide is this in `pair list`? (`📁` is 1 rune, 2 columns) |

### Overflow: refuse, then drop whole tokens

An overlong name is **refused at the create prompt**, quoting the real limit,
rather than silently shortened — silent truncation is what produced
`pair-parley_nv-parley_nv`, which its owner could not explain.

Where shortening still happens (non-interactive paths, which have no prompt to
refuse at), the ladder drops residual tag tokens **whole**, from the right,
before truncating anything; only once no residual is left does the repo token
shrink, by rune, to a 4-byte floor. The ladder resolves **length only** —
collisions go to the numeric suffix, because a shorter name is some other tag's
natural name.

### The name is not invertible

Rules 1 and 2 discard information, so a tag cannot be recovered from a `📁` name
by string surgery. `session-names.jsonl` is the only inverse
(`TagForSessionName`), and a `📁` name absent from it yields **no** tag rather
than a plausible-looking wrong one. The legacy `pair-<tag>` form *is* invertible
and keeps its `TrimPrefix` fallback — that is a different scheme, not a shortcut.

### Migration from `pair-`

A ledger row pins a name only once it is already `📁`-prefixed; a legacy row
falls through and re-mints. The superseded zellij record is reclaimed at the
create flow's commit point, and **only when already `EXITED`** — an attached
session is someone's live terminal and a detached one is resumable work, so
migration is never what destroys either.

Pair cannot rename a live zellij session underneath itself, so a running session
migrates by being quit and relaunched.

## Independent Pair and Couch authorities

Couch verified park preserves the exact Pair address; it is not a new native
identity state. Cold resume uses the current durable target, including a requested
UUID under probation, through the existing `{repo scope, tag}` marker. It requires
the same native ID before launch for a resume. An unmaterialized Pair-chosen ID
(the agent never took a turn) is no conversation: resume and relaunch refuse it
at once with `resume-binding-unbound`, naming reboot, and a parked slot whose
conversation is lost (`binding-lost`) is not offered resume at all (pair#367,
reversing #346 M2's fresh restart, which waited for a ready nonce Pair never
learned). Every fresh launch must hand Pair its registration nonce through the
orientation's attempt (`freshNonceReachesPair`). It never allocates or adopts a marker, chooses
a newest transcript, or consults current path/root/repository launch defaults.
The ledger owns native identity; the native forest supplies parsed observations.
Couch stores the last successfully registered launch profile.

Pair and Couch deliberately have two independent durable authorities:

- Pair owns `{repo scope, tag}` address claims, scoped artifacts, ledgers, and
  public zellij session bindings. Direct Pair establishes its own claim before
  writing artifacts. A Couch-hosted Pair changes only Couch's pre-reserved claim
  to `established`; the marker is exact registration evidence, not metadata.
- Couch owns `threadstore/manifest.json` and the addressed records under
  `threadstore/records/<scope>/<tag>.json`. ThreadStore alone owns lifecycle,
  mutable human names, descriptions, working paths, and recovery.

The composed boundary preserves both owners: Pair establishes its marker before
the zellij handoff without touching Couch files; Couch observes that evidence
and then performs the creating→live transition for the exact helper identity.
Malformed, mismatched, invalid, or unreadable markers are unknown evidence and
fail closed; missing and reserved markers are absent evidence.

Standalone Pair does not open or upsert Couch's ThreadStore. `pair resume`
addresses an exact Pair tag (with Pair's own ledger permitted to invert a public
`📁...` session name), and Pair's picker uses Pair-owned live bindings and tag
history. Couch's mutable names and paths remain Couch-only resolution inputs:
they neither decorate the Pair picker nor become Pair resume addresses
(ARCH-DRY, ARCH-PURPOSE, ARCH-PURE).

### Session names are also filename components

`quit-<session>` and `restart-<session>` markers embed the name, so `📁` now
appears in filenames under `~/.cache/pair`. `artifactpath.ResolvePairCache`
owns their construction. Its session-name contract accepts Unicode basenames
while rejecting empty names, traversal, and NUL; strict ASCII validation for
thread tags remains unchanged.

## Ledger and caches

Each tag has an append-only `ledger-<tag>.jsonl` in its scope dir. Current typed
rows are a `launch`/`binding` union: physical line ordinal is the generation
key, and a binding is current only when it joins the newest exact
`{scope,tag,agent}` launch. The shared locked store owns append/fsync; malformed
lines consume their ordinal instead of being silently reused. Historical
launcher rows remain readable during migration.

Authority publication has one result vocabulary across the typed and
compatibility ledgers: an incomplete unterminated row is non-authoritative; a
complete row whose file/directory durability is uncertain is indeterminate and
is reconciled by exact physical ordinal plus encoded bytes; a cleanup failure
after durability is committed and does not roll lifecycle state back. Launcher
and watcher consumers preserve those outcomes rather than treating every error
as a missing row.

Operator-authored Pair-log entries use the same publication rule around two
atomic replacements. Each Neovim submission attempt carries a stable opaque
`append_id` in the byte-counted marker. The first replacement records
`state=prepared` before dispatch; the parser retains it for audit but excludes
it from correlation facts. A normal dispatch is followed by an exact-ID
`state=submitted` replacement, and only then can the entry match a native user
turn. An unchanged retry reuses its ID, while edited, cleared, indeterminate,
and compose-without-submit preparations remain permanently ineligible rather
than claiming an input occurred. After success, even identical later authored
text receives a new ID.

The submitted transition is also gated by the production Zellij action result,
not merely by calling the send function. Focus, write, and submit failure leave
the attempt prepared and the authored draft intact. Refocus failure after a
successful submit is a UI warning rather than a delivery rollback. If the
submitted-marker replacement then fails, Neovim retains that dispatched append
ID and performs commit-only recovery before accepting another authored send;
the original body is never dispatched twice.

Before dispatch, the editor retains a finer delivery phase. `written` means the
exact agent-facing body is already staged and retry may execute only the pending
submit/compose action; changing the authored body is blocked until that state
resolves. `indeterminate` means a failed write might have partially affected the
composer and automatic retry is refused. `composed` is a completed transfer but
never submitted evidence. These phases prevent replayed Zellij effects from
changing the native input relative to its Pair-log body.

The typed joined ledger binding is the source of truth for native recovery.
The older `agent-<tag>` and `config-<tag>-<agent>.json` files remain derived
caches and compatibility surfaces; config disagreement is diagnosed and cannot
override a current ledger generation.

### Codex root identity

Codex scanning treats source metadata as open-world: `vscode`, unfamiliar valid
values, and absent source fields do not by themselves reject a root. Understood parent/subagent evidence still prevents
child transcripts from being selected as roots. Parsing checks apply to fresh
correlation and optional contents, never to the continued usability of a saved
UUID. Scanner schema changes invalidate obsolete cached rejections.

The scanner lives in `cmd/internal/sessioninventory`. Process/open-file evidence
can corroborate a causal-round candidate but cannot select one. Compatibility
config IDs cannot override the current ledger; config retains launch options.
Neovim does not inspect Codex processes or rollouts itself.

`agent-default-<agent>.json` is different from `config-<tag>-<agent>.json`: it
has only `{agent,args}` and belongs to the repo/agent, not to a work tag or
native conversation. Fresh `pair <agent>` creates use it as the lowest-priority
argument source after explicit `-- <args>` and tag-specific config. It is written
only after the launched `pair wrap` child publishes a matching
`agent-ready-<tag>-<agent>.json` record for the launch nonce.

## Picker and list scope

Default picker/list views are current-repo scoped:

- live sessions are included only when the current scope's
  `session-names.jsonl` maps their public name to the current scope key;
- picker rows use Pair's repo/tag history and live session bindings; Couch human
  names and paths never decorate them, and selection always retains the tag;
- `pair <agent>` marks different-agent live rows unavailable and switches a
  different-agent historical tag to the requested driver, seeding from a
  matching continuation doc when present or an auto-continuation draft over
  Pair's tag files and parked scrollback when not;
- unindexed live `pair-*` sessions are treated as legacy candidates, not proof
  that they belong to the current repo;
- a legacy `pair-*` session and a new `📁` one coexist in one snapshot, both
  discoverable with the right tag.

## Legacy flat data

Flat sidecars under the global root are not silently claimed. If a flat tag is
ambiguous but matches the current repo basename family, Pair shows a manual row:

```text
legacy unscoped <tag>  (manual import)
```

Selecting it copies missing flat sidecars into the current repo scope, including
queued prompt files, preserves the flat source files, avoids overwriting scoped
files, and writes a ledger row with `legacy_import: true`.
