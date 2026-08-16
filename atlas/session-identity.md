# Session identity and storage

Pair separates four identities that used to be partly conflated:

- **Repo scope** — a hidden, stable key derived from the cleaned repo root. It
  owns the scoped data directory and is not shown in user-facing labels.
- **Display tag** — the repo-local work item name the user types, such as
  `work` or `bugfix`. Two repos can both have `work`.
- **Agent** — the resource running under a tag, such as `claude`, `codex`, or
  `agy`. A tag can have sessions from more than one agent over time.
- **Native session id** — the agent's own resumable conversation id, captured by
  the launcher or `pair-session-watch`.

## Data layout

The global Pair data root is still `${XDG_DATA_HOME:-~/.local/share}/pair`.
Repo-scoped launch state lives under:

```text
<global>/repos/<scope-key>/
```

Tag sidecars keep their readable local names inside that scope:

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
- picker rows show readable `repo/tag  agent` annotations;
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
