# S4 one edit path (apply, derived rounds, review pane on apply) Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Every agent edit to a session-sync document goes through Couch, which keeps the change as a *round* that parley (and the review pane) can show; precise edits arrive as `{old, occurrence, new, explain}` records through one atomic `apply` verb.

**Architecture:** Three pure units sit in `cmd/internal/sessionsync`: the record resolver (records → new content or a list of bad records), the round deriver (before/after → spans, with `explain` attached from resolved records), and the propose converter (base + diff → marker text, or `propose-too-large`). The existing `write` flow (`Sync.write` → `prepareWrite` → effects) gains one input, the content builder, and one output, the round sidecar. `apply` is `write` with a record-built body, and `round` is a read verb. Design: `ops:teams/devinfra/projects/universal-agent-input-interface/design.side/10-s4-edit-path.md`, operator-converged 2026-10-17.

**Tech Stack:** Go (sessionsync, couchcmd), Lua (nvim/review, run with `nvim -l`), shared JSON fixtures.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `EditRecord` / `ParseEdits` | `cmd/internal/sessionsync/edits.go` | new |
| `ResolveEdits` | `cmd/internal/sessionsync/edits.go` | new |
| `LineDiff` (hunks) | `cmd/internal/sessionsync/linediff.go` | new |
| `Round` / `Span` / `DeriveRound` | `cmd/internal/sessionsync/round.go` | new |
| `ProposeMarkers` | `cmd/internal/sessionsync/propose.go` | new |
| `submitsOfDocument` (extracted from `OwedForDocument`) | `cmd/internal/sessionsync/owed.go` | modified |
| `Paths.Round` | `cmd/internal/sessionsync/file.go` | modified |
| Lua `reconstruct.resolve` (pure resolver) | `nvim/review/reconstruct.lua` | new |
| Shared fixtures | `testdata/edit-records/*.json` | new |

- **EditRecord / ParseEdits:** `{old, occurrence, new, explain}`. It accepts both handoff forms, `[...]` and `{"records": [...]}`. `occurrence` defaults to 1 and must be ≥ 1 when present (the same rule as `handoff.lua` `valid_payload`). Unknown fields are ignored, as the Lua decoder does.
  - **DRY rationale:** this is the one Go reading of the review pane's format. The field name stays `explain`, as the design decided, so the format doesn't fork.
- **ResolveEdits(base, records) → (next string, resolved []ResolvedEdit, bad []BadEdit):** a port of `apply.lua` resolution.
  - Every record resolves against `base`. The offset is the Nth plain, non-overlapping byte match (`reconstruct.nth_offset` semantics).
  - Records sort by base offset. An overlap with the previous accepted span is bad.
  - **Difference from Lua:** any bad record means `next` is unused, and the caller refuses the batch (`edits-invalid`). The Lua pane drops bad records instead, and keeps doing so until M2 retires it.
  - `ResolvedEdit{Index, BaseStart, BaseEnd, NewStart, NewEnd, Explain}` gives the edit's range in `next`. That range is how `DeriveRound` attaches reasons.
  - **Future:** the same function serves M2's pane and any other edit producer.
- **LineDiff(a, b) → []Hunk:**
  1. Split into lines, keeping the newlines. A trailing newline is part of the text, as the design requires.
  2. Strip the common prefix and suffix lines.
  3. Run Myers on the middle with an edit-distance cap (below).
  - `Hunk{OldLo, OldHi, NewLo, NewHi}` gives line ranges.
  - **ARCH-CONSTRAINTS:** documents are at most 4 MiB (`documentMaxBytes`). Myers is capped at D ≤ 4096. Past the cap, the whole middle becomes one hunk, so the cost stays bounded on a pathological rewrite.
  - No diff library is in `go.mod`, and `sessionsync.DiffText` shells out to `diff -u` with no hunk structure. Both reasons make a small pure Myers the right size; adding a dependency for about 80 lines isn't worth it.
- **Round / Span / DeriveRound(before, after, resolved, presentation, now) → Round:**
  1. Each hunk is narrowed to its changed characters: the common byte prefix and suffix of the hunk's old and new text, trimmed back so a split never lands inside a UTF-8 rune.
  2. `Span{Start, End [2]int; Old, New, Explain string}`. Positions are 0-based [line, byte column] in `after`.
  3. **Byte columns:** nvim extmark columns are bytes. This must be confirmed with parley.nvim:3; see "Contract with parley" below.
  4. A deletion has `Start == End`.
  5. Reasons are attached by overlap. A span takes the `Explain` of every resolved edit whose `[NewStart, NewEnd)` overlaps it (an empty edit range touches its point). Several distinct reasons join with `"; "`.
  6. Exclusion and no-ops:
     - **Restamp:** `DeriveRound` diffs `Restamp(before, now)` against `after`. `Restamp` only replaces the first `Updated …` stamp in place (`file.go:175`), so the stamp line never differs.
     - **No-op:** if no hunk remains, the result is `ok=false`. The caller then keeps the previous round, as the design says a no-op write must.
  - **DRY rationale:** one source of positions for every verb. `write` spans simply carry no reasons.
- **ProposeMarkers(before, after) → (content string, err):** used when the document's `presentation:` is `propose` (read with the existing `frontmatter(next, "presentation")`).
  - **Size limit:** each hunk may cover at most 3 lines, counting `max(old lines, new lines)`. A larger hunk refuses with `propose-too-large`, and the error names the hunk's line range.
  - **Changed line pairs:**
    - A paired old/new line is narrowed to its changed characters and becomes `🤖~old~{new}`.
    - An old line with no partner becomes `🤖~old~` on that line.
    - A new line with no partner becomes its own line, `🤖{new}`.
  - **Escaping:** marker text escapes `\ < > [ ] { }` the way `marker_codec.esc_quote` does.
  - **Unencodable text:** a changed line whose old text holds `~`, or whose text holds a newline inside a marker, can't be encoded, because the Lua parser doesn't unescape `~` inside a strike. It refuses with `propose-unencodable`. **For the TL:** this refusal code is not in the design.
  - **Accept/reject asymmetry (try-out cost):**
    - Rejecting an inserted line leaves an empty line.
    - Accepting a deleted whole line leaves an empty line.
    - This is the "no newline encoding" choice the operator made. It goes in the Log, to tune.
  - **Interaction with the round:** under propose, the written content is `ProposeMarkers(before, after)`, and the round is derived from `before` to that written content. The spans are therefore the inserted markers, with reasons attached through the resolved edits' base ranges mapped by hunk.
- **submitsOfDocument(rows, file, paths, plain, canonical) → []Entry:** the path and spelling matching lifted out of `OwedForDocument` (`owed.go:79-102`). `OwedForDocument` keeps its behaviour by calling it (**ARCH-DRY**). `round` uses it on the raw ledger rows: `cleared` means any submit has `At` after `round.At`, and folded submits count.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `Sync.Apply` + `ApplyRequest` | `cmd/internal/sessionsync/sync.go` | new | flock/lock/hash/owed (existing) |
| `prepareWrite` round + propose output | `cmd/internal/sessionsync/sync.go` | modified | — |
| round sidecar publish/load | `cmd/internal/sessionsync/sync.go`, `registry.go` | new | filesystem (`writeAtomic`) |
| `Sync.Round` | `cmd/internal/sessionsync/sync.go` | new | sidecar + owed-event ledger |
| CLI `apply`, `round` | `cmd/internal/couchcmd/session_sync.go` | modified | argv/JSON |

- **Sync.Apply:** `withDocument` → `guard` → whole-file `target("", false)` hash (exit 6) → `owed` → `folds` → `ResolveEdits(f.content, records)`. A bad batch returns `&Refusal{Exit:1, Code:"edits-invalid", Invalid: bad}` and writes nothing. Otherwise it continues as a whole-file write whose body is `next`. That matches the design's order inside the flock (**ARCH-ORDER**):
  1. lock
  2. hash
  3. owed and gap
  4. resolve
  5. build the content, with the restamp after the records
  6. content checks
  7. diff
  8. replace
  9. shadow
  10. folds
  11. round

  `Refusal` gains an `Invalid []BadEdit` field. The JSON envelope prints it as `Invalid`.
- **prepareWrite:** after the content checks it computes `round` (and, under propose, replaces `next` with the marker content *before* the content checks, so the 4 MiB and owner checks see the bytes that will be written). `preparedWrite` gains `round []byte` (nil = keep previous).
- **Round sidecar:**
  - **Paths:**
    - Legacy layout: `Paths.Round = .BASE.round`.
    - Registered layout: `state/<ID>/round.json`, from `registryTxn.metadata`.
  - **Lifecycle:**
    - Rename keeps the `<ID>` directory, so the sidecar carries over unchanged.
    - Forget's `exportLegacy` copies it to `.BASE.round`.
    - `Register` imports `.BASE.round`.
    - `cleanup` removes it with the directory.
  - **Write order (ARCH-ORDER):** the round is written last, after the shadow and state, through `writeAtomic`. A failure there becomes a `Result.Warning`, not an error. The document write has already committed, and `result_hash` lets a reader tell that a sidecar is stale.
  - **ARCH-FUNERAL:** the sidecar is overwritten by each changing write and ends with its document's metadata. It accumulates nothing.
  - **ARCH-SECURE:** the sidecar is Couch-written metadata in the same trust domain as the shadow. `round` validates the sidecar's JSON shape on read, and a malformed sidecar reports `Code: "round-invalid"`. It is never passed through unchecked.
- **Sync.Round(file) → (Round, cleared bool, err):** `withDocument` in read mode, so no write lock is needed. It loads the sidecar and reads `s.Ledger.Read(f.owner)` → `submitsOfDocument`. A missing sidecar returns a null `Round` with `cleared=false`. It refuses `not-recording` like `status` does.
- **CLI:**
  - Verbs: `apply EDITS.json FILE --if-hash H [--fold ID] [--json]` and `round FILE --json`.
  - The reply contract that parley.nvim:3 planned against: `{Code, Error, Round, Cleared}`. `apply --json` adds `Hash` and `Invalid`.
  - Wiring: verbs and flags go in the `parseSessionSyncCLI` map (`session_sync.go:51`), and help lines in `run.go:1436`.
  - Neither verb needs a cost entry beyond `--session-sync`'s existing `costLight`.

**Test surface:**
- **Pure units:** colocated `_test.go` files with no IO: `edits_test.go`, `linediff_test.go` (including a property test that applying the hunks reconstructs `b`), `round_test.go` and `propose_test.go`.
- **Integration:** the `newRig` harness in `sync_test.go` (real temp files, real ledger), plus `couchcmd/session_sync_test.go` `runSS` for the CLI. There is no new fake: the seams are the filesystem and the ledger, which the rig already makes real.

### Contract with parley (to settle before landing)

The TL relayed parley.nvim:3's planned contract: `round FILE --json` → `{Code, Error, Round, Cleared}`, where `Round` is the sidecar and `result_hash` is the whole-file hash. M1 builds exactly that. One detail the contract doesn't pin is whether **columns are bytes or characters**. I'll send parley.nvim:3 byte columns (the nvim extmark unit) before writing `round.go`, and change it if they planned on characters.

---

## Chunk 1: M1, Couch

- [ ] M1 — `apply`, the derived round on every write, the round sidecar, `round --json`, the propose converter, and the shared fixtures with the Lua resolver passing them.

### Task 1: shared fixtures + Go resolver

**Files:**
- Create: `testdata/edit-records/README.md`, `testdata/edit-records/*.json`
- Create: `cmd/internal/sessionsync/edits.go`, `edits_test.go`

Fixture shape: `{"name", "base", "records": [...], "result": "…"}` or, for a refusal, `{"name", "base", "records", "invalid": [{"index": 1, "reason": "not found"}]}`. Reasons are exactly `empty old`, `not found`, `overlap`, the strings `apply.lua` uses today. The fixtures cover:
- `single`, `nth-occurrence`, `adjacent-not-overlapping`, `overlap`, `not-found`, `empty-old`
- `occurrence-past-last`, `multibyte` (🤖 in old/new), `trailing-newline`, `insert-by-repeating-context`
- `sort-by-offset` (records given out of order), `both-forms` (`{"records": [...]}`)

- [ ] **Step 1:** Write the fixtures and `TestResolveEditsFixtures`, which walks `testdata/edit-records` and checks `result` or `invalid`. Run `env -u COUCH_THREAD_SCOPE -u COUCH_THREAD_TAG go test ./cmd/internal/sessionsync -run TestResolveEdits`. Expected: FAIL (undefined).
- [ ] **Step 2:** Implement `ParseEdits` and `ResolveEdits`. Run again. Expected: PASS.
- [ ] **Step 3:** Commit `#661 M1: edit records resolver and shared fixtures`.

### Task 2: Lua resolver on the same fixtures

**Files:**
- Modify: `nvim/review/reconstruct.lua` (add the pure `M.resolve(base, records) → result, bad`)
- Modify: `nvim/review/apply.lua` (use `reconstruct.resolve` for its up-front resolution, keeping its drop behaviour until M2)
- Create: `nvim/review/resolve_fixtures_test.lua`
- Modify: `Makefile.local` (`test-lua` list)

- [ ] **Step 1:** Write `resolve_fixtures_test.lua`. It reads every `testdata/edit-records/*.json` with `vim.json.decode` and asserts `resolve` gives the same result or bad list. Run `nvim -l nvim/review/resolve_fixtures_test.lua`. Expected: FAIL (no `resolve`).
- [ ] **Step 2:** Extract `resolve` from `apply.lua`'s resolution loop into `reconstruct.lua`, and have `apply.lua` call it. Run the new test, then `make test-lua` and `make test-review`. Expected: PASS, with apply.lua's behaviour unchanged.
- [ ] **Step 3:** Commit `#661 M1: Lua resolver shares the edit-record fixtures`.

### Task 3: line diff + round derivation

**Files:**
- Create: `cmd/internal/sessionsync/linediff.go`, `linediff_test.go`, `round.go`, `round_test.go`

Tests:
- `LineDiff`: identity, pure insert, pure delete, replace, trailing-newline change, and the D-cap fallback (a forced cap of 2 on a big change gives one hunk). A property test applies the hunks and gets `b` back.
- `DeriveRound`:
  - character narrowing within a line;
  - a multi-line hunk;
  - deletion (`Start == End`);
  - rune-safe narrowing (🤖 adjacent to a change);
  - the restamp excluded (only the `Updated` line differs gives `ok=false`);
  - reasons attached by overlap, joined with `; `;
  - no reason on spans with no overlapping edit.

- [ ] **Step 1:** Write the tests. Expected: FAIL.
- [ ] **Step 2:** Implement. Expected: PASS.
- [ ] **Step 3:** Commit `#661 M1: line diff and round derivation`.

### Task 4: propose converter

**Files:**
- Create: `cmd/internal/sessionsync/propose.go`, `propose_test.go`

Tests:
- A one-line replacement gives `🤖~old~{new}`, narrowed to the changed characters.
- A three-line hunk gives three markers.
- A four-line hunk gives `propose-too-large`, naming lines.
- A pure deletion gives `🤖~line~`, and a pure insertion gives its own `🤖{line}` line.
- Escaping works for `\<>[]{}`.
- `~` in old text gives `propose-unencodable`.
- Round-trip: the output parsed by the Lua `markers.lua` grammar. This is checked in Task 2's Lua test via a `propose` fixture subset (`testdata/edit-records/propose/*.json`, `{base, after, markers}`), so both sides agree on the marker text.

- [ ] **Step 1:** Write the tests, plus the propose fixtures and the Lua markers round-trip assertions. Expected: FAIL.
- [ ] **Step 2:** Implement. Expected: PASS.
- [ ] **Step 3:** Commit `#661 M1: propose converter`.

### Task 5: write and apply produce rounds; sidecar in both layouts

**Files:**
- Modify: `cmd/internal/sessionsync/sync.go` (`ApplyRequest`, `Sync.Apply`, and `prepareWrite`'s round, propose and round publish; `Refusal.Invalid`)
- Modify: `cmd/internal/sessionsync/file.go` (`Paths.Round`), `registry.go` (`metadata`, `exportLegacy`, `Register` import, `loadMetadata`/`validateMetadata`)
- Modify: `cmd/internal/sessionsync/sync_test.go` (+ `registry_test.go`)

Rig tests:
- `write --section` leaves a round whose spans match the change, has no explain, and has `result_hash == Hash(file)`.
- A no-op write keeps the previous round.
- `apply` with two good records leaves a round with both reasons.
- `apply` with one bad record refuses with `edits-invalid` and `Invalid` listing it, and the file, shadow, state and round are all byte-identical to before.
- `apply` on a stale hash gives exit 6, and under the operator lock gives exit 3, with nothing written in either case.
- `presentation: propose` with a small change writes markers, and a large change refuses with `propose-too-large` and nothing written.
- The registered layout writes `round.json`.
- After `rename`, `round.json` is still there.
- After `forget`, `.BASE.round` is there, and after `register` it is back.
- A round-write failure (the round path made a directory) gives a `Result.Warning`, and the document is still written.

- [ ] **Step 1:** Write the tests. Expected: FAIL.
- [ ] **Step 2:** Implement. Expected: PASS. Also run the whole `./cmd/internal/sessionsync` package to show that existing write behaviour is unchanged.
- [ ] **Step 3:** Commit `#661 M1: apply and derived rounds in the write path`.

### Task 6: `round` read verb + `cleared`; CLI

**Files:**
- Modify: `cmd/internal/sessionsync/owed.go` (extract `submitsOfDocument`), `sync.go` (`Sync.Round`)
- Modify: `cmd/internal/couchcmd/session_sync.go`, `run.go` (help), `session_sync_test.go`

Tests:
- `cleared` is false before a submit and true after `r.deliver(id, "submitted: <file>")` stamped after `at`.
- A submit before `at` doesn't clear.
- A folded submit still clears.
- A missing sidecar gives a null `Round`.
- A malformed sidecar gives `round-invalid`.
- `OwedForDocument`'s existing tests still pass after the extraction.
- CLI:
  - `apply` parse forms;
  - the `apply --json` refusal carries `Code: edits-invalid` and `Invalid`;
  - the `round --json` envelope has exactly `{Code, Error, Round, Cleared}` (plus omitted-empty fields);
  - exit codes 1, 3 and 6 through `runSS`.

- [ ] **Step 1:** Write the tests. Expected: FAIL.
- [ ] **Step 2:** Implement. Expected: PASS.
- [ ] **Step 3:** Commit `#661 M1: round verb with cleared, CLI apply/round`.

### Task 7: atlas, live check, milestone close

- [ ] **Step 1:** Atlas: add the edit path, the round sidecar and the verbs to the session-sync atlas page (`atlas/` file found by `grep -l session-sync atlas/`).
- [ ] **Step 2:** Live check against a real document, from the slot's built binary:
  - `read --all`, `apply` with a good record, and `round --json` shows the span with its reason;
  - a submit of the file makes `round --json` report `cleared: true`;
  - a bad record changes nothing (compare `shasum` before and after).
  - Record it in the Log.
- [ ] **Step 3:** Run the recorded suite (`sdlc local-check run --name suite`), then `sdlc milestone-close --issue 661 --milestone M1`.
- [ ] **Step 4:** Tell parley.nvim:3 and the ariadne skill owner (via the hub) that M1 has landed on the branch and give the contract. Both issues depend on M1 landing on main. Land M1 first only if the TL wants the dependents unblocked before M2; otherwise both milestones land together.

## Chunk 2: M2, the review pane on apply

- [ ] M2 — review pane edits arrive through `apply`; nvim never writes the reviewed file.

M2 is outlined here and refined in a `## Revisions` entry at its start, once M1's surface is real. The open facts are where the review agent's instructions live and how an arbitrary reviewed file is registered.

### Task 8: register-on-first-apply for reviewed files

- [ ] A reviewed file that isn't registered is registered on its first `apply`, through S1's registry (`Sync.Register`). Test: an `apply` on a fresh, unregistered markdown file registers it and writes a round.

### Task 9: the review agent writes EDITS.json and runs apply

- [ ] Locate the review agent's handoff instructions (the producer of `$PAIR_REVIEW_HANDOFF_PATH` payloads) and switch them to `apply` on the file under review, with exit 6 handled by re-reading and rebuilding.
  - The landed summary moves into the agent's reply.
  - Test: `tests/review-apply-test.sh`, rewritten to drive `couch --session-sync apply` and assert the file changed through Couch.

### Task 10: the pane highlights the round; retire nvim apply and reconcile

- [ ] On reload, the pane runs `round FILE --json` and decorates the spans with `review_diag`.
- [ ] Retire `apply.lua`'s buffer apply, the dropped list, `reconcile.lua` and the handoff watcher's apply path, with their tests. Keep `reconstruct.resolve` and the shared fixtures.
- [ ] Rule test: a review round never calls `nvim_buf_set_text` on the reviewed buffer. Assert it with a grep-based Lua test over `nvim/review/`, or a buffer-write spy in `review-apply-test.sh`.

### Task 11: close

- [ ] Atlas (`atlas/review-workbench.md`), recorded suite, then `sdlc close --issue 661`.

---

## Decisions to confirm with the TL

1. **Byte columns** in span positions, to be confirmed with parley.nvim:3 before `round.go`.
2. **`propose-unencodable`:** a new refusal code for a changed line with `~` in its old text. The design doesn't have it, and the Lua grammar can't express it.
3. **Reasons on a shared span:** a span touched by several records joins their `explain` with `"; "`.
4. **Round-write failure is a warning, not an error:** the document write has already committed, and `result_hash` exposes staleness.
5. **Landing:** M1 and M2 land together by default. M1 lands alone if the TL wants parley #383 and the ariadne skill unblocked sooner.
