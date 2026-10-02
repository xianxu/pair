---
id: 000382
status: open
deps: []
github_issue:
created: 2026-10-01
updated: 2026-10-01
estimate_hours:
card_mirror: 'bb09cec513303b3f11d188e8acfb35414f9f0f8e' # card fields mirrored from issue-cards; edit via sdlc
---

# pair session-repair resolves data dir from caller env, not --scope-key

## Problem

`pair session-repair <agent> <tag> --scope-key <scope>` ignores `--scope-key`
when locating artifacts. `RunRepairCLI` (`cmd/internal/sessionwatch/recover.go`)
takes `DataDir` from `PAIR_DATA_DIR`, falling back to `adapt.DataDir()`, then
calls `artifactpath.ResolveScoped(DataDir, tag)`. The scope key only goes into
the ledger owner check.

Seen on 2026-10-01 while repairing brain from inside a pair session for a
different repo. `PAIR_DATA_DIR` pointed at
`~/.local/share/pair/repos/68f32487e79d85bd` (pair), so the tool failed with
`open .../repos/68f32487e79d85bd/ledger-couch-6b111ea230c149dc.jsonl: no such
file`, even though `--scope-key 2e51fcf9799b1d8f` named brain. It only worked
with `PAIR_DATA_DIR=~/.local/share/pair/repos/2e51fcf9799b1d8f` set by hand.
Outside any session the fallback is the data root, not a scoped dir, so it
likely fails there too.

## Spec

- Derive the scoped data dir from `--scope-key`, the same way pair derives
  `repos/<scope>` for a session, rather than from the caller's env. Refuse when
  an explicit `PAIR_DATA_DIR` disagrees with the scope, rather than silently
  using it.

## Done when

- A test: run with `PAIR_DATA_DIR` set to another scope's dir and `--scope-key`
  naming the target. It reads the target scope's ledger, or refuses with the
  mismatch named.
- A test: run with no `PAIR_DATA_DIR` resolves `repos/<scope-key>` under the
  data root.

## Plan

- [ ]

## Log

### 2026-10-01
