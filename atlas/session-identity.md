# Session identity and storage

Pair separates identities that used to be partly conflated:

- **Repo scope** — a hidden, stable key derived from the cleaned repo root. It
  owns the scoped data directory and is not shown in user-facing labels.
- **Thread tag** — the immutable repo-local storage key. Direct Pair keeps the
  user-chosen form (`work`, `bugfix`); Couch creates opaque
  `couch-<16 lowercase hex>` tags.
- **Human thread name** — optional mutable metadata on the durable ThreadStore
  record. It is neither a filename nor a zellij socket name.
- **Public session name** — the stable zellij socket binding recorded in
  `session-names.jsonl` for one `{scope, tag}`.
- **Agent** — the resource running under a tag, such as `claude`, `codex`,
  `agy`, or `muse`. A tag can have sessions from more than one agent over time.
- **Native session id** — the agent's own resumable conversation id. Fresh
  launches expose it as recovery state only after Pair establishes a completed
  causal round; an explicit scanner-authorized resume may establish it at the
  launch boundary.

## Native-session forest inventory

`cmd/internal/sessioninventory` is the single model and scanner boundary for
native session storage (#155 M1). Its versioned Claude, Codex, Agy, and Muse
scanners emit facts into a deterministic forest: complete roots, validated
native parent/child edges, and explicit unbound orphans. Missing, conflicting,
malformed, unreadable, or unknown-schema evidence is retained as a stable coded
diagnostic rather than guessed away. Stable IDs, ordering, chronology fallback,
artifact paths, and a forest-only canonical projection are pure functions.

All native I/O crosses one injected runtime: named storage roots, bounded file
reads, read-only SQLite, and process/open-file snapshots. The sibling
`sessioninventorytest` package supplies a persistent stateful fake, while
`make test-native-session-live` checks installed native shapes without printing
paths, IDs, or transcript content. Native parentage establishes topology only;
it is not evidence that a Pair tag owns a root.

M2 adds one round-gated binding lifecycle. Before agent input, the launcher
appends a typed `launch` row containing the Pair-log byte offset and sorted
native-event watermarks. A fresh launch is deliberately provisional: even a
Claude ID minted for invocation is not recovery authority. An explicit resume
may join immediately only when the scanner inventory recognizes its root.

`pair session-watch` scans the complete forest and uses process/open-file facts
only as stable before/after corroboration. A unique exact operator turn followed
by assistant/tool/error progress proposes the root; the watcher appends a
`binding` row only while the launch ordinal is still current, then refreshes the
config cache. Repeated matches remain ambiguous and no timestamp, traversal
order, first/newest file, or native parent edge breaks the tie. After a crash,
the same matcher considers only bytes/events beyond the durable launch
watermarks. A crash before progress therefore preserves nothing; a crash after
progress can reconstruct the binding.

`pair session-inventory [--agent ...] [--scope current|all] [--json]
[--conformance]` exposes the canonical forests, correlations, ambiguities, and
coded diagnostics. A dedicated public DTO keeps schema v1 exact: internal root
coordinates do not leak into evidence, and required position/fingerprint arrays
remain arrays even when empty. Conformance emits only agent/status/count/code
data. Pair's Go store and Neovim history navigation share the versioned
byte-counted log grammar while retaining legacy entries, so authored Markdown
separators round-trip.

The final #155 migration makes inventory queries the only native-session read
authority. Context/token usage, title activity, bounded slug text events, review scoping,
launcher recovery and resume hints, changelog keying, and Neovim age display all
consume an established owner projection. Provisional, ambiguous, and unbound
owners remain explicit absence; only an exact inherited `PAIR_SESSION_ID` can
precede that projection. Compatibility config retains launch arguments but
cannot establish identity. `pair session-inventory --activity --agent <agent>`
is the buffered internal timestamp transport for editor consumers; it emits
nothing until the root is established (ARCH-DRY, ARCH-PURE, ARCH-PURPOSE,
ARCH-MOCK).

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
with username and is different on Linux (`~/.cache/zellij`). Pair therefore
treats zellij's own validator as the acceptance oracle (`ProbeSessionName`); the
numeric budget is calibrated lazily, only after a rejection, purely so the
refusal message can quote real numbers.

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

Pair and Couch deliberately have two independent durable authorities:

- Pair owns `{repo scope, tag}` address claims, scoped artifacts, ledgers, and
  public zellij session bindings. Direct Pair establishes its own claim before
  writing artifacts. A Couch-hosted Pair changes only Couch's pre-reserved claim
  to `established`; the marker is exact registration evidence, not metadata.
- Couch owns `threadstore/manifest.json` and the addressed records under
  `threadstore/records/<scope>/<tag>.json`. ThreadStore alone owns lifecycle,
  admission, mutable human names, descriptions, working paths, and recovery.

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

Operator-authored Pair-log entries use the same publication rule around the
atomic replacement. Each Neovim submission attempt carries a stable opaque
`append_id` in the byte-counted marker. If publication becomes indeterminate,
the retained draft retries with that same ID; the store validates the original
body and completes directory durability without appending a duplicate turn.
After success, even identical later authored text receives a new ID.

The typed joined ledger binding is the source of truth for native recovery.
The older `agent-<tag>` and `config-<tag>-<agent>.json` files remain derived
caches and compatibility surfaces; config disagreement is diagnosed and cannot
override a current ledger generation.

### Codex root identity

A Codex rollout filename supplies only a candidate UUID; it does not prove
which conversation owns the rollout. Pair authorizes an automatic Codex
identity only when the rollout's first JSONL event is a matching
`session_meta`, its `parent_thread_id` is absent or null, and its source is the
observed root source `cli` or `exec`. Subagent, malformed, mismatched, unknown,
oversized, and incomplete first events fail closed. Candidate scans continue
past rejected rollouts so an open subagent cannot hide a later root candidate.

The rule lives in `cmd/internal/sessioninventory` and is shared by launcher,
watcher, context, slug, title, opener, and review queries. Process/open-file
evidence can corroborate a causal-round candidate but cannot select one.
Persisted config IDs are compatibility evidence only: an unavailable binding is
removed from config, its non-resume args are preserved for a fresh launch, and
the operator is warned. Explicit inherited invocation authority remains exact.

Neovim deliberately does not inspect Codex processes or rollouts. Review
target scoping uses the inherited `PAIR_SESSION_ID`, then the established
inventory projection; when neither exists it remains unscoped until the watcher
publishes a validated root binding.

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
