# Edit-record fixtures (pair#661)

These fixtures are the shared contract for resolving review records `{old, occurrence, new, explain}` against a base text. Both resolvers run them:
- Go: `cmd/internal/sessionsync` `ResolveEdits`, used by `couch --session-sync apply`;
- Lua: `nvim/review/reconstruct.lua` `resolve`, used by the review pane.

Each `*.json` file has `base` and `records`, given in either handoff form: `[...]` or `{"records": [...]}`. It also has one of:
- `result`: the resolved text;
- `invalid`: the bad records, as `{index, reason}`. `index` is 1-based, and the list is sorted by index.

The rules:
- `occurrence` is 1-based and defaults to 1.
- Matches are plain, non-overlapping and counted over bytes.
- Every record resolves against `base`.
- Records sort by offset, with ties broken by input order.
- A record whose span starts before the previous accepted span ends is an `overlap`.

The reasons are exactly `empty old`, `not found` and `overlap`.
