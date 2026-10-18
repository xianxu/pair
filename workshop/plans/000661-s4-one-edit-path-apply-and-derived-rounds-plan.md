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
  - `ResolvedEdit{Index, BaseStart, BaseEnd, Explain}` gives the edit's byte range in `base`. `DeriveRound` attaches reasons by that range.
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
  5. Reasons are attached by overlap **in base coordinates**, one rule for every presentation (`attachReasons(spans, resolved)`):
     - each span keeps the byte range of its narrowed old text in the baseline;
     - a span takes the `Explain` of every resolved edit whose `[BaseStart, BaseEnd)` overlaps that range (an empty range touches its point);
     - several distinct reasons join with `"; "`.

     The restamp only rewrites the stamp in place, so baseline and base offsets agree everywhere outside the stamp line, which never forms a span.
  6. Exclusion and no-ops:
     - **Restamp, one baseline for every consumer (PQ-1):** `prepareWrite` computes `baseline := before`, and `baseline = Restamp(before, now)` exactly when the write restamps (every scope except `--frontmatter`, `sync.go:480`). Both `DeriveRound` and `ProposeMarkers` take `baseline`, never raw `before`. `Restamp` replaces the first `Updated …` stamp in place (`file.go:175`), so the stamp line never differs. Tests: a frontmatter write and a propose write on a stamped document each give no stamp span and no stamp marker.
     - **No-op:** if no hunk remains, the result is `ok=false`. The caller then keeps the previous round, as the design says a no-op write must.
  - **DRY rationale:** one source of positions for every verb. `write` spans simply carry no reasons.
- **ProposeMarkers(before, after) → (content string, err):** used when the document's `presentation:` is `propose` (read with the existing `frontmatter(next, "presentation")`).
  - **Size limit:** each hunk may cover at most 3 lines, counting `max(old lines, new lines)`. A larger hunk refuses with `propose-too-large`, and the error names the hunk's line range.
  - **Changed line pairs:**
    - A paired old/new line is narrowed to its changed characters and becomes `🤖~old~{new}`.
    - An old line with no partner becomes `🤖~old~` on that line.
    - A new line with no partner becomes its own line, `🤖{new}`.
  - **Escaping:** marker text escapes `\ < > [ ] { }` the way `marker_codec.esc_quote` does.
  - **Unencodable text:** a changed line whose old text holds `~`, or whose text holds a newline inside a marker, can't be encoded, because the Lua parser doesn't unescape `~` inside a strike. It refuses with `propose-unencodable`. The remedy is the same as for `propose-too-large`: the agent writes that change as a side-doc proposal. The refusal says so, and every edit that can't be a one-line marker takes that one fallback.
  - **Accept/reject asymmetry (try-out cost):**
    - Rejecting an inserted line leaves an empty line.
    - Accepting a deleted whole line leaves an empty line.
    - This is the "no newline encoding" choice the operator made. It goes in the Log, to tune.
  - **Interaction with the round, shadow and hash (PQ-2):**
    - Under propose, the **written** content is `ProposeMarkers(baseline, after)`. Everything downstream follows the written bytes, not the clean proposal:
      - the round is derived from `baseline` to the written content, so the spans are the inserted markers;
      - the shadow is the written content (a section write's shadow body is `SectionBody(written, heading)`, fed to `shadowAfter` in place of `req.Body`);
      - the returned `Hash(target)` is taken over the written content's scope.
    - That keeps the agent's own markers out of the next `diff`, and keeps the next `--if-hash` valid.
    - The substitution runs in `prepareWrite`, right after the content is built and before the content checks, so the 4 MiB and owner checks see the written bytes.
    - Reasons attach by the base-coordinate rule above.
    - **Frontmatter writes are never proposed:** markers inside frontmatter would corrupt it. A frontmatter write is direct under every presentation, and its round is still derived.
    - Test: a propose section write, then `diff`, shows nothing of the agent's own markers, and the returned hash passes the next write's `--if-hash`.
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
  - **Write order (ARCH-ORDER):** a changing write first removes the previous sidecar, *before* the document is replaced. The new round is then written last, after the shadow and state, through `writeAtomic`. If that final write fails, the result is a `Result.Warning` and no round at all. An old round never sits over new text.
    - If the removal itself fails, the write refuses before touching the document.
    - A no-op write (no hunks) removes nothing and keeps the previous round.
  - **ARCH-FUNERAL:** the sidecar is overwritten by each changing write and ends with its document's metadata. It accumulates nothing.
  - **ARCH-SECURE / ARCH-CONSTRAINTS:** the sidecar is Couch-written metadata in the same trust domain as the shadow. `round` and `Register`'s import read it through a size cap (`roundMaxBytes`, 16 MiB, enough for a 4 MiB document's old and new text plus JSON escaping) and validate its shape. Over the cap or malformed, it reports `Code: "round-invalid"` (import skips it). A round that would encode past the cap is not written, with a warning, so it never exceeds the cap. It is never passed through unchecked.
- **Sync.Round(file) → (Round, cleared bool, err):** goes through `withDocument`, which takes the registry flock and the document flock like every verb, so it reads a consistent sidecar and ledger. It loads the sidecar and reads `s.Ledger.Read(f.owner)` → `submitsOfDocument`. A missing sidecar returns a null `Round` with `cleared=false`. It refuses `not-recording` like `status` does.
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
- A round-write failure gives a `Result.Warning`, the document is still written, and **no** sidecar remains. To force the failure, the write seam makes the new round's write fail.
- A failure to remove the previous sidecar refuses with nothing written.

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
- [ ] **Step 4:** Land M1 (`sdlc pr`, then `sdlc merge --yes`), then report the merge hash to the hub so parley#383 and ariadne#437 can start.

## Chunk 2: M2, the review pane on apply

- [ ] M2 — review pane edits arrive through `apply`; nvim never writes the reviewed file.

M2 is outlined here and refined in a `## Revisions` entry at its start, once M1's surface is real. The open facts are where the review agent's instructions live and how an arbitrary reviewed file is registered.

### Task 8: register-on-first-apply for reviewed files

_Seam change, stated: this gives `apply` a registration side effect. On an unregistered file it creates a registry entry, which gives it an owner. The M2 revision decides whose ownership that is (the reviewing slot) and confirms that `write` keeps its refuse-unregistered behaviour._

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

## Decisions (TL, 2026-10-18)

1. **Byte columns** in span positions, to be confirmed with parley.nvim:3 before `round.go`.
2. **`propose-unencodable`:** a new refusal code for a changed line with `~` in its old text. The design doesn't have it, and the Lua grammar can't express it.
3. **Reasons on a shared span:** a span touched by several records joins their `explain` with `"; "`.
4. **Round-write failure is a warning, not an error:** the document write has already committed, and `result_hash` exposes staleness.
5. **Landing:** M1 and M2 land together by default. M1 lands alone if the TL wants parley #383 and the ariadne skill unblocked sooner.

## Revisions

- **2026-10-18, TL approval.** All five decisions were accepted, with two refinements and a landing order:
  - **(2) `propose-unencodable`** takes the same side-doc proposal remedy as `propose-too-large`: one fallback for every edit that can't be a one-line marker.
  - **(4) No stale round may survive a failed round write:** the previous sidecar is removed before the document is replaced. Task 5's tests were changed to match.
  - **(5) M1 lands alone first,** to unblock parley#383 and ariadne#437. M2 follows on its own branch.
    - Under "one issue, one branch", that means M1 lands through `sdlc merge` at the milestone, and M2 continues on #661 from main.
    - Task 7 step 4 becomes "land M1".
- **2026-10-18, plan-quality round 1.** PQ-1: one restamp-aware baseline feeds both `DeriveRound` and `ProposeMarkers`; frontmatter writes don't restamp, so they are diffed raw. PQ-2: under propose, the shadow, the returned hash and the round all follow the written marker bytes, and frontmatter is never proposed. Minors: `Round` takes the flocks; the sidecar has a 16 MiB read and write cap; reasons attach by one base-coordinate rule (`attachReasons`); M2's registration side effect is stated. The prose test lists stay: they double as each task's RED checklist.
- **2026-10-18, implementation and M1 review round 1.** The code departs from Core concepts as follows:
  - **Myers cap:** D ≤ 1024 (`maxDiffEdits`), not 4096. The trace grows as D², and 4096 would cost about 128 MB at the cap.
  - **Submit matching:** `submitsOfDocument → []Entry` became `submitNames` (the per-row matching shared with `OwedForDocument`) plus `SubmittedSince` (the `cleared` query).
  - **No restamp shift:** the `Updated` stamp is fixed width, so `DeriveRound(baseline, after, resolved)` takes no `before`.
  - **`at`** keeps full precision.
  - **Propose proposes the body only** (`ProposeDocument`). Under propose, a body write must leave the frontmatter byte-identical or it refuses with `propose-unencodable`, so no verb can mark it or flip `presentation:` (BR-2, and round 2). `write --frontmatter` stays direct.
  - **Unpaired blank lines** are not marked. An inserted blank line is written as is, and a removed one is kept (BR-3; a bare 🤖 never settles). The Lua fixture test now requires every 🤖 Couch writes to parse.
  - **`ParseEdits`** refuses `occurrence < 1`.
- **2026-10-18, M2 start (TL decisions on five questions).** Chunk 2 is superseded by "M2 as built" below. The decisions:
  1. **Same engine, pair-level verb:** `pair review apply`, with no owner, ledger or registry. Pair must work without Couch. The rule becomes: only the shared engine writes, never nvim.
  2. **The operator lock replaces the gate.**
  3. **The xx-fix workbench protocol folds into ariadne#437** (ariadne:1). Its delta is written below.
  4. **docflow commits the round, and resume reads it back from the commit.** The landed file retires.
  5. **Re-estimate:** in the issue's `## Estimate`.

### M2 as built (supersedes Chunk 2 Tasks 8–11)

**Engine (ARCH-DRY).** The pure content step that `Sync.write` runs today is lifted into one exported function, which both `couch --session-sync apply` and `pair review apply` call:
- **Function:** `sessionsync.BuildEdit(base string, edits []EditRecord, now time.Time, restamp bool) (Edit, error)`, where `Edit{Next string; Round Round; Changed bool}`.
- **Steps:** resolve (all or nothing), the restamp-aware baseline, propose under `presentation: propose`, then derive the round.
- **Refusals:** the same as `apply` (`edits-invalid`, `propose-too-large`, `propose-unencodable`).
- `Sync.write` keeps its owner, ledger and shadow steps around it. Its tests must stay green unchanged.

**`pair review apply EDITS.json FILE --if-hash H [--json]`** (`cmd/internal/reviewcmd`, a new subverb):
- **Order:** an exclusive flock (in pair's data dir, `review-apply-<sha8(abspath)>.flock`, so nothing lands in the repository), then the operator lock, then the whole-file hash, then `BuildEdit` (with `restamp=false`: reviewed documents aren't session files), then an atomic replace, then the round artifact written last.
- **Operator lock (S1's):** `FILE.lock` with `holder: operator` refuses with exit 3. A lock older than 30 minutes adds ping advice, as `couch` does. It reuses `sessionsync.ParseHolder`.
- **Exit codes:** 0 written; 1 `edits-invalid` / `propose-*` / `refused`; 3 `operator-lock`; 6 `stale` (with the current hash).
- **Round artifact:** `$PAIR_REVIEW_ROUND_PATH`, which is `review-round-<tag>.json` in pair's data dir. It replaces both `review-handoff-<tag>.json` and `review-landed-<tag>.json`.
  - Contents: `{file, round, body}`. `round` is the sidecar shape from M1.
  - `body` is the commit body: a fenced `review-records` block (the existing `record.embed_in_body` format) whose records come from the round's spans, as `{old, new, explain, new_occurrence}`. `new_occurrence` is the Nth plain match of `new` in the result at the span's position, so resume (`reconstruct_on_open`) works unchanged.
  - Like M1's round sidecar, the artifact is removed before the document moves and written after it, so it is never stale.
  - It is registered in `artifactpath`, with retention like the handoff it replaces.
- **`--json` reply:** `{Code, Error, Hash, Invalid, Round, Body}`. Without `--json` the verb prints `hash:` and the body path.

**The pane (nvim):**
- **Round watch:** `handoff.watch`'s 100 ms poll moves to the round artifact. A new round arrives with an unseen `result_hash` and a `file` equal to the pane's file.
  - If the buffer is unmodified, the pane reloads it from disk (`edit!`) and decorates the spans with `apply.place` in the `review_diag` namespace. The message is the span's `explain`, or "changed by the agent". The span is set to `no_highlight` when it is a deletion or its new text is a marker proposal.
  - A modified buffer can't occur, because the lock is held (below). If one does, the pane warns and doesn't reload.
- **Lock:** the pane writes `FILE.lock` (`holder: operator`, `at:`) on the first buffer change of the operator's turn (TextChanged/TextChangedI). It removes the lock on send (`finish_human_turn`, after the save) and at VimLeave.
  - While the lock is held, the agent's apply refuses with exit 3 and waits for the operator's send.
  - The S1 30-minute ping rule applies.
- **Retired by pair#664** (ordering (a), TL 2026-10-18), not in M2; M2 keeps the legacy handoff working beside the new path:
  - `apply.apply` (the buffer apply; `render`, `snapshot`, `apply_snapshot`, `place` and `clear*` stay);
  - `reconcile.lua`;
  - `gate.lua` with `on_defer`, `pending_records`, `has_pending` and the winbar pending state;
  - `set_base` / v0;
  - `handoff.lua`'s handoff and landed halves, with `did_land` and `restore_controller`'s pending-handoff and pending-landed checks;
  - the `agent_applied` poke and the dropped/conflict notifications.

  `record.lua`, `reconstruct.lua` (with `resolve` and the shared fixtures) and `projection.lua` stay.
- **What nvim still writes:** for an applied round, nothing; the pane only reloads and decorates. The legacy handoff keeps its buffer apply until pair#664, which adds the rule test.
- **Readiness and docflow:** docflow stages only the recorded in-scope documents (ariadne `docflow.sh:161`), so an untracked `FILE.lock` is never committed. Readiness's clean check must ignore an untracked `FILE.lock` for the reviewed file. A test pins it.

**Tests:**
- The legacy tests (`review-reconcile`, `review-handoff`, and the Lua `reconcile`, `gate`, `handoff_acceptance` and `handoff_context`) stay green in M2. pair#664 retires them.
- Rewrite `review-loop-test.sh` and `tests/lib/fake-review-agent.sh`. The fake agent runs `pair review apply`, then commits the artifact's `body` with docflow. The test asserts:
  - the buffer reloads with the decorations;
  - the commit body is verbatim;
  - one round per apply;
  - the lock is held after an operator edit, so the fake agent's apply exits 3 and nothing changes;
  - after the send, apply succeeds.
- Update `review-window-test.sh` (the pending and winbar parts go), `review-producer-context-test.sh` and the restore tests (they switch from the handoff and landed env vars to `PAIR_REVIEW_ROUND_PATH`), and `poke_bodies_test`.
- Go: `reviewcmd` tests for `pair review apply` (exit codes 0/1/3/6, nothing written on a refusal, the artifact's body round-trips through `record.extract_from_body`), and `sessionsync` tests for `BuildEdit`.

**Protocol delta for ariadne#437** (xx-fix `construct/local/fix/SKILL.md`, "Pair review workbench", lines 201–293). Exact replacements:
1. **Rule 1 (propose as records into the handoff) becomes: apply as records through pair.**
   - Write `{old, occurrence, new, explain}` records to a temp `EDITS.json` (`[...]` or `{"records": [...]}`) and run `pair review apply EDITS.json FILE --if-hash H --json`. H is the whole-file hash: `shasum -a 256` hex truncated to 16, the same function as `couch --session-sync read --all`, and also returned as `Hash` by the previous apply.
   - Never edit the file in place, and never hand-write `🤖~…~{…}` markers. Under `presentation: propose`, pair writes them.
   - **Exits:**
     - 0: applied.
     - 1 `edits-invalid`: nothing was written; `Invalid` lists `{index, reason}`. Fix those records and apply again.
     - 1 `propose-too-large` / `propose-unencodable`: write that change as a side-doc proposal linked in place by `→ [proposal](…)`.
     - 3 `operator-lock`: the operator is editing. Wait for their "finished my edits" poke, then re-read.
     - 6 `stale`: re-read the file, rebuild the records against it, apply again.
2. **The activation-context envelope and the context-mismatch section are deleted.** The hash and the operator lock replace them.
3. **Rule 2 (commit after the applied poke) becomes: commit right after a successful apply.**
   - Run `docflow round --side agent -m <summary> --body <Body>`, with `Body` taken verbatim from the apply reply (the round as records). Never regenerate it.
   - There is no "applied N edit(s)" poke any more.
   - The human side is unchanged (`docflow round --side human` on the "finished my edits" poke).
4. **The reconcile-marker section is deleted.** An exit 6 replaces reconciliation.
5. **`$PAIR_REVIEW_HANDOFF_PATH` and `$PAIR_REVIEW_LANDED_PATH` are gone.** The agent needs no path; pair finds the round artifact from the session.

In pair, `poke_bodies.review_opened` and `with_context` change to name `pair review apply`, `agent_applied` is deleted, and `workshop/targets/review-protocol.md` seams 2 and 2b become "seam 2: `pair review apply` + the round artifact".

**Order, the one decision left for the TL (ARCH-ORDER).** The legacy handoff path is the only way today's agents (xx-fix before ariadne#437) apply, and it needs nvim's buffer apply. Retiring it before #437 lands breaks every live review in between. Two shapes:
- **(a) Recommended: two steps.**
  1. M2 lands the engine, `pair review apply`, the round watch, the lock and the new tests. The pane accepts an applied round, and still accepts a legacy handoff (through the existing apply and reconcile code) until #437 lands.
  2. A follow-up issue, filed at the M2 close with `deps: ariadne#437`, retires the handoff, `apply.apply`, reconcile, gate and landed, with their tests, and adds the "nvim never writes" rule test. M2's Done-when "nvim never writes the reviewed file" is then met by that follow-up, which the issue records.
- **(b) One step:** M2 waits for ariadne#437 and retires everything at once. Nothing lands until #437 does.

  **TL decision: (a).** The follow-up is pair#664 (`deps: ariadne#437`).
