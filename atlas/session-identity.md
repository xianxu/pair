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
ledger-<tag>.jsonl
scrollback-<tag>-<agent>.raw
scrollback-<tag>-<agent>.events.jsonl
pane-<tag>-<agent>.json
```

Pane and helper consumers must treat inherited `PAIR_DATA_DIR` as authoritative.
They should not reconstruct the global XDG path unless `PAIR_DATA_DIR` is absent.

## Public session names

Zellij session names are globally visible, so Pair assigns a readable public
name through `session-names.jsonl` in the global data root. The first
`pair/work` session becomes:

```text
pair-pair-work
```

A second repo with the same display repo name and same tag gets a stable numeric
suffix, for example:

```text
pair-pair-work-2
```

The hidden scope key is stored in the index row, not embedded in zellij names,
picker rows, titles, or pane text.

## Ledger and caches

Each tag has an append-only `ledger-<tag>.jsonl` in its scope dir. Ledger entries
record agent, args, session id, timestamps, repo root/name, and whether a row
came from a legacy import.

The ledger is the source of truth for agent/config inference. The older
`agent-<tag>` and `config-<tag>-<agent>.json` files remain as derived caches and
compatibility surfaces for existing consumers.

## Picker and list scope

Default picker/list views are current-repo scoped:

- live sessions are included only when `session-names.jsonl` maps their public
  name to the current scope key;
- picker rows show readable `repo/tag  agent` annotations;
- unindexed live `pair-*` sessions are treated as legacy candidates, not proof
  that they belong to the current repo.

## Legacy flat data

Flat sidecars under the global root are not silently claimed. If a flat tag is
ambiguous but matches the current repo basename family, Pair shows a manual row:

```text
legacy unscoped <tag>  (manual import)
```

Selecting it copies missing flat sidecars into the current repo scope, including
queued prompt files, preserves the flat source files, avoids overwriting scoped
files, and writes a ledger row with `legacy_import: true`.
