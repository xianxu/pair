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
- **Native session id** — the agent's own resumable conversation id, captured by
  the launcher or `pair session-watch`.

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

Pane and helper consumers must treat inherited `PAIR_DATA_DIR` as authoritative.
They should not reconstruct the global XDG path unless `PAIR_DATA_DIR` is absent.

## Public session names

Zellij session names are globally visible, so Pair assigns a readable public
name through `session-names.jsonl` in the global data root. The format is:

```text
📁{repo}[-{residual tag tokens}]
```

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

## Durable thread index

Couch's namespace contains `threadstore/manifest.json` plus addressed records
under `threadstore/records/<scope>/<tag>.json`. Couch and Launcher both decode
the lower-layer `threadrecord.Record` acceptance contract; Couch maps it to its
rich lifecycle type, while Launcher maps it to a portable read-only projection
without importing couchcore or writing recovery state. One strict decoder
rejects duplicate keys, unknown fields, and trailing values for both. A
cross-reader mutation table covers every required top-level, address,
incarnation, start-claim, policy-shape, generation, and path/address invariant.
Launcher uses the same scoped exact-tag/name/path matcher that Couch adapts to
its richer records. Missing/corrupt/incomplete stores fail closed and Couch
retains journal-recovery ownership.

Human thread names lead standalone resume and picker views, but resolution
returns the immutable tag. Duplicate names are ambiguous; duplicate picker
labels expose tag disambiguators. Existing direct Pair artifacts win before
fuzzy name/path matching, preserving old `pair resume <tag>` behavior. Legacy
`pair rename` deliberately does not resolve human thread names: it moves tag
files and must never mutate an opaque thread identity.

### Session names are also filename components

`quit-<session>` and `restart-<session>` markers embed the name, so `📁` now
appears in paths under the data root. It is safe on APFS/ext4 and carries no
shell or glob metacharacters — but that is a property the scheme depends on.

## Ledger and caches

Each tag has an append-only `ledger-<tag>.jsonl` in its scope dir. Ledger entries
record agent, args, session id, timestamps, repo root/name, and whether a row
came from a legacy import.

The ledger is the source of truth for agent/config inference. The older
`agent-<tag>` and `config-<tag>-<agent>.json` files remain as derived caches and
compatibility surfaces for existing consumers.

### Codex root identity

A Codex rollout filename supplies only a candidate UUID; it does not prove
which conversation owns the rollout. Pair authorizes an automatic Codex
identity only when the rollout's first JSONL event is a matching
`session_meta`, its `parent_thread_id` is absent or null, and its source is the
observed root source `cli` or `exec`. Subagent, malformed, mismatched, unknown,
oversized, and incomplete first events fail closed. Candidate scans continue
past rejected rollouts so an open subagent cannot hide a later root candidate.

The rule lives in `cmd/internal/transcript` and is shared by launcher live
capture, session watching, context usage, slugging, and review targeting.
Process-tree and birth-time discovery locate candidates only; neither grants
identity by itself. Persisted Codex IDs are revalidated at automatic config
picker and `Alt+n` restart boundaries. An invalid binding is removed from the
config, its non-resume args are preserved for a fresh launch, and the operator
is warned. Explicitly typed `codex resume <id>` remains user authority.

Neovim deliberately does not inspect Codex processes or rollouts. Review
target scoping uses the inherited `PAIR_SESSION_ID`, then Pair's config cache;
when neither exists it remains unscoped until the Go watcher publishes a
validated root identity.

`agent-default-<agent>.json` is different from `config-<tag>-<agent>.json`: it
has only `{agent,args}` and belongs to the repo/agent, not to a work tag or
native conversation. Fresh `pair <agent>` creates use it as the lowest-priority
argument source after explicit `-- <args>` and tag-specific config. It is written
only after the launched `pair wrap` child publishes a matching
`agent-ready-<tag>-<agent>.json` record for the launch nonce.

## Picker and list scope

Default picker/list views are current-repo scoped:

- live sessions are included only when `session-names.jsonl` maps their public
  name to the current scope key;
- picker rows lead with `repo/human-name  agent` when ThreadIndex has a name,
  otherwise `repo/tag`; selection always retains the tag;
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
