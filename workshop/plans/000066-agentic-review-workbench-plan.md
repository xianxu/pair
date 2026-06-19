# Agentic Review Workbench Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn pair's review into a memory-backed agentic loop where a persistent Claude session proposes content-anchored edit records that an embedded nvim review pane applies undo-ably, styles, and commits via docflow.

**Architecture:** Two workbenches over a shared filesystem — pair's persistent agent (conversational, the *producer*) and an embedded review nvim (document, the *applier/renderer*). The agent emits `{old,occurrence,new,explain}` records to an ephemeral handoff file; nvim file-watches it, applies the records as **in-buffer undo-able** ops, places riding decorations, and commits the round through ariadne's `docflow`. Durable history = git (round commits, with the records embedded in the agent commit body); fine-grained undo = nvim `undofile`; no bespoke sidecar. The contract is locked in issue #66 §Spec + `workshop/pensive/2026-06-18-01-pensive-agentic-review-workbench.md`.

**Tech Stack:** nvim Lua (pure modules under `nvim/`, tested under `nvim -l` per pair's `slug.lua` pattern); `vim.uv` fs_event + buffer API for integration; ariadne `scripts/docflow.sh` (shelled out) for git rounds; zellij actions for the agent poke; headless-nvim shell integration tests (`tests/*.sh`).

---

## Scope note

This plan delivers **Milestone 1 (the spine)** in full bite-sized detail and **outlines M2–M4**. Per the writing-plans scope check, M2–M4 are independent enough to become their own plans once M1 lands and de-risks the novel core (undo-preserving apply + docflow round + reconstruction). M1 is a working, testable vertical slice driven by a *fake* agent — no parley extraction, no real LLM, no window UI yet.

---

## Core concepts

The whole feature's conceptual model (not just M1). The pure core is the record serialization and the records→decorations reconstruction; everything else is a thin IO seam.

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `Record` / `review.record` | `nvim/review/record.lua` | new |
| `review.reconstruct` | `nvim/review/reconstruct.lua` | new |
| `review.markers` | `nvim/review/markers.lua` | new (M2; ported from parley + `review-convention.md`) |
| `review.projection` | `nvim/review/projection.lua` | new (M2; ported from parley) |

- **`review.record`** — a `Record` is the proposal unit. It carries **two anchors**: `occurrence` = the 1-based Nth match of `old` in the round's *base* content (the agent emits this; `review.apply` uses it to locate the edit), and `new_occurrence` = the Nth match of `new` in the *post-apply* content (`apply` computes + adds this; resume-reconstruction uses it to place the decoration). They differ whenever `old` and `new` occur a different number of times — conflating them mislands decorations (the plan-quality judge's finding #1). The agent's handoff record is `{old, occurrence, new, explain}`; the record embedded in the commit body is enriched to `{old, occurrence, new, new_occurrence, explain}`. This module is the single serialization: `encode(records) -> string` and `decode(string) -> records` over a JSON array (via `vim.json`, deterministic under `nvim -l`), plus `embed_in_body(summary, records) -> commit_body` / `extract_from_body(commit_body) -> records` that wrap the JSON in a fenced ` ```review-records ` block. The **same** serialization is written to the handoff file *and* embedded in the agent commit body — one format, three readers (live apply, reconstruction, human in `git log`).
  - **Relationships:** 1:N — a round owns N records. No reference to vim state.
  - **DRY rationale:** Collapses what would otherwise be two formats (handoff payload vs. durable history) into one. The handoff JSON *is* the commit-body block.
  - **Future extensions:** A `severity`/`category` field per record if modes later want richer diagnosis styling — additive to the JSON object, parsers ignore unknown keys.

- **`review.reconstruct`** — pure transform `records, content, which -> { highlights, diagnostics }` (0-based line ranges + `explain`). `which='old'` locates each record by the `occurrence`-th match of `old`; `which='new'` locates by the **`new_occurrence`**-th match of `new` — **never reuse the old-occurrence to find `new`** (finding #1). No vim API — returns plain data the integration layer renders. This is the **resume / from-commit** path: `review.apply` produces *live* decorations from the exact ranges it just edited (no re-find), while `reconstruct` rebuilds them from records `extract_from_body`'d from a frozen commit.
  - **Relationships:** consumes `review.record` output; produces decoration-input data consumed by `review.apply`.
  - **DRY rationale:** One function serves live-render and resume-render — there is no second "reconstruct from commit" path.
  - **Future extensions:** When `review.projection` (M2) lands, reconstruct feeds it the per-content-hash snapshot.

- **`review.markers`** (M2) — the pure 🤖 marker parser, ported from parley's `_parse_marker_sections` against `ariadne/.../review-convention.md`. Human review *requests* live as markers inside the doc; this parses them for the "what's open" summary.

- **`review.projection`** (M2) — ported from parley `projection.lua`: decoration snapshots keyed by content hash, restored on undo/redo. Answers "styling follows undo coherently" in-session without a per-change-event log.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `review.apply` | `nvim/review/apply.lua` | new | nvim buffer API (undo-able edits + extmarks/diagnostics) |
| `review.docflow` | `nvim/review/docflow.lua` | new | `ariadne/scripts/docflow.sh` subprocess |
| `review.handoff` | `nvim/review/handoff.lua` | new | filesystem + `vim.uv` timer poll |
| `review` (orchestrator) | `nvim/review/init.lua` | new | wires the above + `undofile` config |
| `FakeAgent` | `tests/lib/fake-review-agent.sh` | new | process-level fake of the agent (writes a handoff) |
| `review.poke` | `nvim/review/poke.lua` | new (M3) | zellij `write-chars` to the agent pane |
| `review.window` | `nvim/review/window.lua` | new (M3) | `:PairReview` + alt+r zellij pane |

- **`review.apply`** — applies a list of `Record`s to the current buffer as **undo-able** in-buffer ops (`nvim_buf_set_text`), then places extmarks + INFO diagnostics. Critically does **not** reload the file (a reload resets undo). Two correctness rules (findings #1/#2): (a) resolve all `old`@`occurrence` offsets against the **base snapshot up front**, then apply **bottom-to-top** (descending offset) so earlier edits don't drift later ranges; (b) decorate from the **actual ranges it edited** (it knows exactly where each `new` landed — no re-find), and compute each edit's `new_occurrence` to enrich the record before it's embedded in the commit body (the resume path's anchor).
  - **Injected into:** the orchestrator; receives `review.record`/`review.reconstruct` outputs so the offset/range logic stays unit-tested without a buffer.
  - **Future extensions:** swap the minimal extmark render for `review.projection` (M2) without touching the apply path.

- **`review.docflow`** — thin wrapper shelling `docflow start|round|status|ship`. `round --side agent --body <embed_in_body(...)>` is how the agent round's records reach git. Reuses ariadne's proven script; no commit/branch logic reimplemented (ARCH-DRY).
  - **Injected into:** the orchestrator via `DOCFLOW_BIN` (default `docflow`). The suite is hermetic (finding #3): `tests/lib/fake-docflow.sh` makes **real git commits** with docflow's shape (`--author=$AGENT_AUTHOR`, subject `review(<slug>): <side> r<N> — …`, `--body`), so tests assert real author/subject/body without an ariadne checkout. One gated smoke test runs against the real `ariadne/scripts/docflow.sh` (skipped with notice when ariadne / `DOCFLOW_BIN` is absent) to catch drift from the real script.
  - **Future extensions:** `resume` once ariadne #90 lands (session-state summary from the journal).

- **`review.handoff`** — writes (test/fake side) and watches+reads+deletes (nvim side) the ephemeral `review-handoff-<tag>.json` in pair's XDG data dir. The atomic appearance of the file is the "round ready" signal; nvim consumes then unlinks it. **Detection = a `vim.uv` timer poll, NOT `fs_event`** — the repo already learned that macOS FSEvents is flaky/laggy and `nvim/init.lua`'s scrollback watcher uses a timer poll for exactly this reason; follow that established pattern (DRY/consistency). The agent's atomic temp+rename write guarantees the poll never reads a half-written file.
  - **Injected into:** the orchestrator; the poll fires the apply→commit sequence.
  - **Future extensions:** per-round names (`-rN`) if a next round can fire before consume completes (race hardening).

- **`review` orchestrator** — sets `undofile`/`undodir` for the review buffer, registers the handoff watcher, and sequences: human-round commit → (M3: poke) → on handoff: `review.apply` → agent-round commit. The only stateful glue; everything decision-shaped is delegated to the pure modules.

- **`FakeAgent`** — a shell script that, given a doc path + tag, writes a deterministic handoff JSON (one or more records). The process-level fake that lets the loop test run with zero LLM. **Part of the deliverable**, per the constitution's external-service-fake rule.

---

## Milestone 1 — The spine (records → undo-able apply → docflow round → reconstruct), fake-agent driven

Produces: a headless flow where a fake agent's handoff is applied to a buffer undo-ably, styled, and committed as a docflow agent-round, with undo crossing the commit and decorations reconstructable from the commit. Closes via `sdlc milestone-close M1`.

### Task 1: `review.record` — encode/decode records

**Files:**
- Create: `nvim/review/record.lua`
- Test: `nvim/review/record_test.lua`

- [ ] **Step 1: Write the failing test**

```lua
-- nvim/review/record_test.lua — run via `nvim -l nvim/review/record_test.lua`.
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local M = dofile(here .. 'record.lua')

local fails = 0
local function eq(got, want, msg)
  if got ~= want then
    io.stderr:write(string.format('FAIL %s: got %q want %q\n', msg, tostring(got), tostring(want)))
    fails = fails + 1
  end
end

local recs = {
  { old = 'teh', occurrence = 2, new = 'the', new_occurrence = 1, explain = 'typo' },
  { old = 'old API', occurrence = 1, new = 'new API', explain = 'v2 is gone' },
}

-- round-trip (incl. the apply-added new_occurrence field — vim.json keeps it)
local s = M.encode(recs)
local back = M.decode(s)
eq(#back, 2, 'decode count')
eq(back[1].old, 'teh', 'decode old')
eq(back[1].occurrence, 2, 'decode occurrence')
eq(back[1].new_occurrence, 1, 'decode new_occurrence')
eq(back[2].explain, 'v2 is gone', 'decode explain')

-- embed/extract in a commit body that also has prose + a trailer
local body = M.embed_in_body('three edits', recs)
local ex = M.extract_from_body(body)
eq(#ex, 2, 'extract count')
eq(ex[1].new, 'the', 'extract new')
eq(M.extract_from_body('no block here'), nil, 'extract returns nil when absent')

if fails > 0 then os.exit(1) end
print('record_test ok')
```

- [ ] **Step 2: Run test to verify it fails**

Run: `nvim -l nvim/review/record_test.lua`
Expected: FAIL — `record.lua` doesn't exist / `attempt to call field 'encode'`.

- [ ] **Step 3: Write minimal implementation**

```lua
-- nvim/review/record.lua — pure record serialization (issue #66 M1).
-- A Record = { old, occurrence, new, explain }. The SAME JSON is written to
-- the handoff file and embedded in the agent commit body. Uses vim.json
-- (deterministic, available under `nvim -l`); no IO/state otherwise.
local M = {}

local FENCE = '```review-records'

function M.encode(records)
  return vim.json.encode(records)
end

function M.decode(s)
  return vim.json.decode(s)
end

-- Build an agent commit body: prose summary + a fenced records block.
function M.embed_in_body(summary, records)
  return table.concat({ summary, '', FENCE, M.encode(records), '```' }, '\n')
end

-- Pull the records back out of a commit body; nil if no block present.
function M.extract_from_body(body)
  local block = body:match('```review%-records\n(.-)\n```')
  if not block then return nil end
  return M.decode(block)
end

return M
```

- [ ] **Step 4: Run test to verify it passes**

Run: `nvim -l nvim/review/record_test.lua`
Expected: `record_test ok`

- [ ] **Step 5: Wire into the test target + commit**

Add `nvim -l nvim/review/record_test.lua` to the `test-lua` target in `Makefile.local`.

```bash
git add nvim/review/record.lua nvim/review/record_test.lua Makefile.local
git commit -m "#66 M1: review.record — single record serialization (handoff == commit body)"
```

### Task 2: `review.reconstruct` — records → decoration inputs

**Files:**
- Create: `nvim/review/reconstruct.lua`
- Test: `nvim/review/reconstruct_test.lua`

- [ ] **Step 1: Write the failing test**

```lua
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local M = dofile(here .. 'reconstruct.lua')
local fails = 0
local function eq(got, want, msg)
  if got ~= want then
    io.stderr:write(string.format('FAIL %s: got %q want %q\n', msg, tostring(got), tostring(want)))
    fails = fails + 1
  end
end

-- which='new' locates by NEW_OCCURRENCE (Nth match of `new`), not `occurrence`.
local content = 'alpha\nthe value\nbeta\nthe value\n'
local out = M.decorate({ { new = 'the value', new_occurrence = 1, explain = 'first' } }, content, 'new')
eq(out.highlights[1].line, 1, 'new_occurrence=1 → line 1 (0-based)')
eq(out.diagnostics[1].lnum, 1, 'diagnostic lnum')
eq(out.diagnostics[1].message, 'first', 'diagnostic message')

local out2 = M.decorate({ { new = 'the value', new_occurrence = 2, explain = 'second' } }, content, 'new')
eq(out2.highlights[1].line, 3, 'new_occurrence=2 → line 3')

-- ADVERSARIAL (finding #1): `old` and `new` have DIFFERENT occurrence counts.
-- Base had three 'foo'; the edit replaced the 2nd 'foo' with 'bar'. Reusing the
-- old-occurrence (2) to find 'bar' would mis-land; new_occurrence=1 is correct.
local after = 'foo\nbar\nfoo\n'
local rec = { old = 'foo', occurrence = 2, new = 'bar', new_occurrence = 1, explain = 'x' }
eq(M.decorate({ rec }, after, 'new').highlights[1].line, 1, 'bar via new_occurrence=1 → line 1')

-- which='old' locates by `occurrence` against base content.
local base = 'foo\nfoo\nfoo\n'
eq(M.decorate({ rec }, base, 'old').highlights[1].line, 1, "which='old' uses occurrence=2 → line 1")

if fails > 0 then os.exit(1) end
print('reconstruct_test ok')
```

- [ ] **Step 2: Run test to verify it fails**

Run: `nvim -l nvim/review/reconstruct_test.lua`
Expected: FAIL — module/function missing.

- [ ] **Step 3: Write minimal implementation**

```lua
-- nvim/review/reconstruct.lua — pure: records + content → decoration inputs.
-- `which` selects the anchor string: 'new' (post-apply buffer) or 'old'
-- (pre-apply base). Returns 0-based line ranges + explains; no vim API.
local M = {}

-- byte offset of the occurrence-th plain (non-pattern) match of `needle`.
local function nth_offset(haystack, needle, occurrence)
  local from, found = 1, nil
  for _ = 1, occurrence do
    local s, e = haystack:find(needle, from, true)
    if not s then return nil end
    found, from = s, e + 1
  end
  return found
end

local function line_of(content, byte_offset)
  local n = 0
  for i = 1, byte_offset - 1 do
    if content:sub(i, i) == '\n' then n = n + 1 end
  end
  return n -- 0-based
end

function M.decorate(records, content, which)
  which = which or 'new'
  local highlights, diagnostics = {}, {}
  for _, r in ipairs(records) do
    -- which='old' → locate `old` by `occurrence`; which='new' → locate `new`
    -- by `new_occurrence`. Never cross them (finding #1).
    local anchor, occ
    if which == 'old' then anchor, occ = r.old, r.occurrence
    else anchor, occ = r.new, r.new_occurrence end
    local off = anchor and anchor ~= '' and nth_offset(content, anchor, occ or 1)
    if off then
      local lnum = line_of(content, off)
      local last = line_of(content, off + #anchor)
      highlights[#highlights + 1] = { line = lnum, end_line = last }
      diagnostics[#diagnostics + 1] = { lnum = lnum, end_lnum = last, message = r.explain or '' }
    end
  end
  return { highlights = highlights, diagnostics = diagnostics }
end

return M
```

- [ ] **Step 4: Run test to verify it passes**

Run: `nvim -l nvim/review/reconstruct_test.lua`
Expected: `reconstruct_test ok`

- [ ] **Step 5: Wire + commit**

Add the test line to `test-lua`.

```bash
git add nvim/review/reconstruct.lua nvim/review/reconstruct_test.lua Makefile.local
git commit -m "#66 M1: review.reconstruct — records→decorations, occurrence-anchored (live + resume)"
```

### Task 3: `review.docflow` — wrap docflow.sh (with a fake on PATH)

**Files:**
- Create: `nvim/review/docflow.lua`, `tests/lib/fake-docflow.sh`
- Test: `tests/review-docflow-test.sh`

- [ ] **Step 1: Write the failing integration test + the hermetic fake** — create `tests/lib/fake-docflow.sh`: a faithful stand-in that (a) logs its argv to a file (for arg-forwarding assertions) AND (b) makes **real git commits** matching docflow's shape — `start <file>` requires the file to exist, `round --side <s> --body <b>` stages in-scope changes and commits `review(<slug>): <s> r<N> — …` with `--author=$AGENT_AUTHOR` for agent rounds (and no-ops when `git diff --cached --quiet`, like the real one). The test sets `DOCFLOW_BIN=tests/lib/fake-docflow.sh`, drives `review.docflow` headlessly, and asserts the subcommand + `--side`/`--body` are forwarded verbatim. (Model after `tests/queue-send-test.sh`.) Add one **gated smoke test** that points `DOCFLOW_BIN` at `$ARIADNE/scripts/docflow.sh` and is **skipped with a notice** when ariadne isn't checked out — this is the only check that touches the real script, catching drift (finding #3).

- [ ] **Step 2: Run it to verify it fails** — `bash tests/review-docflow-test.sh` → FAIL (module missing).

- [ ] **Step 3: Implement** `review.docflow` — `start(file)`, `round(side, summary, body)`, `status()`, `ship()`, each `vim.system({ docflow_bin, ... })` with `docflow_bin = vim.env.DOCFLOW_BIN or 'docflow'`. The agent round calls `round('agent', summary, record.embed_in_body(summary, records))`.

- [ ] **Step 4: Run it to verify it passes.**

- [ ] **Step 5: Commit** — `#66 M1: review.docflow — shell out to ariadne docflow (reuse, don't reimplement)`.

### Task 4: `review.apply` — undo-able in-buffer apply + decorations

**Files:**
- Create: `nvim/review/apply.lua`
- Test: `tests/review-apply-test.sh` (headless nvim; buffer API can't run under `nvim -l` purely)

- [ ] **Step 1: Write the failing test** — headless nvim opens a scratch buffer with known content, calls `review.apply.apply(buf, records)`, asserts: (a) buffer text reflects each `old`→`new`; (b) one undo (`nvim_buf_call` + `:undo`) reverts ALL of the round's edits to the original (single undo-block); (c) an extmark + diagnostic exist on the changed line (namespace `review`); (d) **drift case** — two records where the earlier-in-file edit changes length; assert both `new`s land on the correct lines (proves bottom-to-top); (e) `apply` returns the records **enriched with `new_occurrence`**. **Also assert the error-free edge cases:** an empty records list is a no-op (no error), and a single-record round applies + single-undo-reverts without throwing `E790`.

- [ ] **Step 2: Run → fail.**

- [ ] **Step 3: Implement** — (1) resolve every record's `old`@`occurrence` to a byte offset against an **up-front base snapshot** of the buffer (reuse `reconstruct`'s exported `nth_offset`); (2) apply **bottom-to-top** (descending offset) via `nvim_buf_set_text` so earlier-in-file edits don't drift later offsets (finding #2); (3) decorate from the **actual ranges just edited** (apply knows where each `new` landed — no re-find), and compute each edit's `new_occurrence` (Nth match of `new` in the final content) to enrich the records for the commit body / resume path. No `:edit!`. **Single-undo-block, E790-safe:** the round's *first* applied edit is a normal change (starts a fresh undo-block); `undojoin` only the subsequent edits — `undojoin` before any change or right after an undo throws `E790: undojoin is not allowed after undo`. Sequence: edit[1]; for i=2..N do `nvim_cmd({cmd='undojoin'})` then edit[i]. Empty list returns early (no edit, no error).

- [ ] **Step 4: Run → pass** (esp. the single-undo-reverts-the-round assertion).

- [ ] **Step 5: Commit** — `#66 M1: review.apply — apply records as one undo-able block + decorations (no reload)`.

### Task 5: `review.handoff` — write/watch/consume

**Files:**
- Create: `nvim/review/handoff.lua`
- Test: `tests/review-handoff-test.sh`

- [ ] **Step 1: Write the failing test** — write a handoff JSON via `review.handoff.write(tag, records)`, assert a watcher registered with `review.handoff.watch(tag, cb)` fires `cb(records)` and the file is unlinked after consume.

- [ ] **Step 2: Run → fail.**

- [ ] **Step 3: Implement** — `path(tag)` = `<XDG_DATA_HOME or ~/.local/share>/pair/review-handoff-<tag>.json`; `write` does atomic temp+rename; `watch` polls via a `vim.uv` timer (NOT `fs_event` — macOS FSEvents is flaky/laggy, per `init.lua`'s scrollback precedent), on appearance reads + `record.decode` + unlinks + calls back.

- [ ] **Step 4: Run → pass.**

- [ ] **Step 5: Commit** — `#66 M1: review.handoff — ephemeral records file is both data and signal`.

### Task 6: `FakeAgent` + orchestrator + the end-to-end loop test

**Files:**
- Create: `tests/lib/fake-review-agent.sh`, `nvim/review/init.lua`
- Test: `tests/review-loop-test.sh`

- [ ] **Step 1: Write the failing end-to-end test** (hermetic: `DOCFLOW_BIN=tests/lib/fake-docflow.sh` from Task 3, which makes real commits) — in a temp git repo with a markdown doc **that exists on disk and is committed** (`docflow start` does `[[ -f "$f" ]] || die`): `review.start(doc)` (docflow start + `undofile` on), simulate a human round by **actually mutating + staging the doc** then `docflow round --side human` (docflow skips a round when `git diff --cached --quiet`, so an unchanged human round silently no-ops and would drift the `r<N>` count), run `fake-review-agent.sh <doc> <tag>` (writes a handoff with 2 records) → assert the orchestrator: applies them (buffer changed), placed decorations, committed an **agent** round (author = agent, subject `review(<slug>): agent r…`, body contains the ` ```review-records ` block), undo crosses the agent commit (text reverts), and `reconstruct` over the agent commit's extracted records reproduces the decorations.

- [ ] **Step 2: Run → fail.**

- [ ] **Step 3: Implement** `fake-review-agent.sh` (emit deterministic records JSON) and `review/init.lua` orchestrator: `start`, `undofile` config, `handoff.watch` → `apply.apply` → `docflow.round('agent', summary, embed)`.

- [ ] **Step 4: Run → pass.**

- [ ] **Step 5: Commit** — `#66 M1: end-to-end loop (fake agent) — handoff→undo-able apply→agent round + reconstruct`.

### Task 7: Milestone close

- [ ] Run the full lua + shell suites: `make test-lua` and the new `tests/review-*-test.sh`.
- [ ] `sdlc milestone-close M1 --issue 66` (auto-dispatches the fresh-context review for `BASE_SHA`..HEAD). Fix Critical/Important before crossing; log the `Review-Verdict:` outcome in #66 `## Log`.
- [ ] Update `atlas/` with the new review surface (the `nvim/review/` module set + the handoff/docflow seam).

---

## Milestone 2 — Port parley's render/projection (outline → own plan)

Replace M1's minimal extmark render with parley's real fidelity by extracting the *consumer half* (per the extraction map in #66): `skill_render` (highlights + diagnostics), `projection` (content-hash decoration snapshots, undo-coherent), `diag_display` (virtual-lines), `mode.lua` + the mode briefs, and the `review.markers` 🤖 parser. Drop everything LLM-invoke. Reuse `review-convention.md`'s marker grammar. Deliverable: undo/redo styling coherence in-session + accumulating agent styling that clears on the next conversation turn.

## Milestone 3 — The review window + agent poke (outline → own plan)

`:PairReview <file>` / alt+r opens a full-screen review nvim pane in the pair session (alt+r both directions; never bare esc). `review.poke` wakes the agent via `zellij --session pair-<tag> action write-chars` + submit. Wire the human-round-commit → poke sequence into the orchestrator. Deliverable: a human in the pane can request review and see the real agent respond.

## Milestone 4 — Real agent protocol (outline → own plan)

The review `SKILL.md` the pair agent follows: do multi-step memory discovery (brain/pensives/datatype skills), then emit `{old,occurrence,new,explain}` records to the handoff — modes set aggressiveness. Decide the no-pair fallback (keep parley's in-process one-shot as a degraded producer of the same contract, or single-path). Deliverable: end-to-end with a real persistent Claude session; cross-session undo + resume (align with ariadne #90).

---

## Open details to resolve in-milestone

- **M1 (resolved in entities + Tasks 1/2/4 — judge finding #1):** occurrence mapping — a Record carries both `occurrence` (Nth `old` in base) and `new_occurrence` (Nth `new` post-apply); `apply` decorates from its own edited ranges and enriches `new_occurrence`; `reconstruct` locates `new` by `new_occurrence`, never by the old index. Tests include the differing-occurrence adversarial case.
- **M1 (resolved in Task 4):** the single-undo-block `undojoin` sequencing (first edit fresh, join 2..N, E790-safe) — specified above and empirically confirmed on nvim 0.11.7 by the plan review. The test pins the empty/single-record edges.
- **M1 (folded into Task 5/6):** macOS FSEvents flakiness → handoff uses a timer poll (matching `init.lua` scrollback), atomic temp+rename write; the e2e human round must actually mutate+stage the doc or docflow no-ops it.
- **M1/M2:** `undofile` robustness — confirm a `docflow` commit (which doesn't rewrite the working file) leaves the undo tree valid; only `ship`'s branch-switch can invalidate it (end-of-review edge).
- **M2:** exact styling-clear trigger — "next conversation turn" vs. explicit end-of-human-turn.
- **M4:** handoff race — keep fixed-name `review-handoff-<tag>.json`, or move to per-round `-rN` names if a next round can fire before consume completes.
- **Estimate caveat (judge finding #4, non-blocking):** `estimate_hours: 30` covers all four milestones but only M1 is detailed; expect growth when M2–M4 get their own plans — or split #66 into an umbrella + sibling issues for per-milestone calibration.

## Revisions

- **2026-06-18 — handoff is a timer poll, not `fs_event`.** Task 5 originally said
  `watch` registers `vim.uv.fs_event`; implementation uses a `vim.uv` timer poll
  instead (macOS FSEvents is flaky/laggy — `init.lua`'s scrollback already polls).
  Reflected in the entity table, the entity description, Task 5 Step 3, and the atlas.
- **2026-06-18 — milestone-review (round 1: I1/I2/I3).** `apply` decorates LIVE from
  the actual edited ranges (no `new`-re-find); `new_occurrence` counts non-overlapping
  to match `reconstruct.nth_offset` (resume re-anchors consistently); the edit loop runs
  inside `nvim_buf_call(buf)` so the undo break + undojoins target `buf` regardless of
  focus; `init` surfaces non-zero docflow exits via `vim.notify` instead of swallowing.
- **2026-06-18 — milestone-review (round 2: unanchorable + overlap).** Decision on the
  "Open detail" of partial/failed anchoring: **surface, don't silently drop.**
  `apply.apply` now returns `(enriched, dropped)`; `dropped` carries a `reason`
  (`empty old` | `not found` | `overlap`), and `on_agent_round` `vim.notify`s when any
  record is dropped — a partial review must never look complete. Overlapping `old`
  ranges are rejected (not corrupted) since bottom-to-top apply assumes non-overlap.
  This is the stable apply→orchestrator return contract M2's renderer and M4's real
  agent will both depend on.
