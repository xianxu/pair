# Agentic Review Workbench — M2 (render/projection + markers + modes) Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) for the execution approach (superpowers-subagent-driven-development or superpowers-executing-plans). Steps use checkbox (`- [ ]`) syntax.

**Goal:** Port parley's review *consumer-half* infrastructure into pair — make decorations undo/redo-coherent and ride manual edits (projection), parse 🤖 review-request markers, and load the review modes — building on M1's record/apply/reconstruct.

**Architecture:** Three additions on M1's surface. (1) `projection` — per-buffer decoration snapshots keyed by content **hash**; on undo/redo it restores the snapshot for the matching hash, on a novel state it captures the riding decorations. Because an exact hash ⇒ identical content, snapshotting line-based decorations keyed by hash is correct, and composes with M1's record-driven live placement. (2) `markers` — the pure 🤖 grammar parser (review requests). (3) `mode` — the pure mode-brief parser + the stock briefs. The interactive accumulation semantics ("human round adds styling, clears on next conversation turn") are M3's (they need the window + loop); M2 delivers the mechanism.

**Tech Stack:** nvim Lua. Pure modules (`markers`, `mode`) under `nvim/review/`, colocated `*_test.lua`, `make test-lua`. Integration (`projection`, `apply` snapshot/restore, orchestrator wiring) via headless shell tests (`make test-review`). Source of truth for the port: parley `lua/parley/skills/review/{projection,mode}.lua`, `skill_render.lua`, the `parse_markers` parser in `review/init.lua`, and `review-convention.md`.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `review.markers` | `nvim/review/markers.lua` | new |
| `review.mode` | `nvim/review/mode.lua` | new |

- **`review.markers`** — parses 🤖 markers from doc lines into records. A marker is `🤖 [<quoted> | ~strike~] [ alternating []/{} sections ]`. Returns `{ line, col, quoted?, strike?, sections=[{type='user'|'agent', text, byte_start, byte_end}], ready, pending, raw }`. Excludes markers inside fenced/inline code; sections span ≤ a newline budget. Pure (operates on a `lines` array → records); no vim API. Ported from parley `review/init.lua` (`parse_markers`, `parse_marker_sections`, `find_matching_bracket`, fence/inline-code helpers).
  - **DRY rationale:** the single source for "what review requests are open in this doc"; M3's window highlights from it, M4's agent reads it.
  - **Future extensions:** accept/reject application (M3) reads `byte_start/byte_end` to splice; not in M2.

- **`review.mode`** — pure parser for a mode brief's frontmatter (`name`, `order`, `scope`, `deletions`, `frontier`) + body, with `parse(content)`, `directives(mode)` (renders the "## How to apply this round" block), and the IO-seam `load(dir,name)`/`list(dir)`. Ported verbatim from parley `review/mode.lua` (already pure + thin IO). The stock briefs are data: `nvim/review/modes/*.md`.
  - **DRY rationale:** modes are the agent's aggressiveness contract; M4's SKILL.md composes `directives()` into the agent prompt.
  - **Future extensions:** a mode picker UI (M3).

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `review.projection` | `nvim/review/projection.lua` | new | nvim autocmd + per-buffer decoration state |
| `review.apply` (snapshot/restore) | `nvim/review/apply.lua` | modified | nvim extmark/diagnostic read+restore |
| `review` orchestrator | `nvim/review/init.lua` | modified | wires projection record/watch around a round |

- **`review.projection`** — per-buffer state `{ records = {[hash]=snapshot}, order=[], watching=bool, autocmd }`, `MAX_RECORDS` FIFO cap. `decide(records,h) → 'restore'|'capture'` (pure). `project(buf)`: if the apply-guard is set or buf invalid, return; else hash the buffer and either `apply.apply_snapshot(buf, records[h])` (undo/redo → re-render) or capture `apply.snapshot(buf)` (novel forward state → riding decorations). `record(buf)` / `record_empty_for(buf, base)` / `ensure_watch(buf)` (TextChanged+InsertLeave, *not* TextChangedI) / `set_applying(buf, v)`. Hash = `vim.fn.sha256(joined lines)`. Ported from parley `projection.lua`.
  - **Injected into:** the orchestrator calls record/record_empty_for/ensure_watch around each round; the watcher calls `apply.snapshot`/`apply.apply_snapshot`. Depends on `apply`, not vice-versa (no cycle).
  - **Future extensions:** per-round-keyed accumulation (M3's "human round adds without clearing agent's").

- **`review.apply` snapshot/restore** — add `M.snapshot(buf)` → `{ hl = [{line,end_line}], diags = [{lnum,end_lnum,message}] }` and `M.apply_snapshot(buf, snap)`. **Critical fidelity fix (M2 plan review):** M1's `place()` uses *ranged* extmarks (`end_row`/`hl_eol`), so `snapshot` MUST capture each extmark's `end_row` — read via `nvim_buf_get_extmarks(buf, HL, 0, -1, {details=true})` and keep `{line=m[2], end_line=details.end_row or m[2]}`. A row-only list (parley's `m[2]`, which worked only because parley used per-line `nvim_buf_add_highlight`) would lose the multi-line span and mis-restore. `apply_snapshot` restores the **two layers independently** (parley-faithful): clear both ns via a shared `clear(buf)` helper (also used by `place`, so "cleared" means the same everywhere), then place ranged extmarks from `snap.hl` and `vim.diagnostic.set(DIAG, ...)` from `snap.diags`. Not zipped — after riding, the extmark (ridden) and its diagnostic (stale) decouple, so they must be restored as separate arrays, not paired.
  - **Injected into:** projection.

- **`review` orchestrator wiring** — `on_agent_round` becomes: capture `base` (pre-apply content) → `projection.set_applying(buf,true)` → `apply.apply` → save + docflow round → `projection.record_empty_for(buf, base)` (undo-to-before-round clears style) → `projection.record(buf)` (snapshot the placed decorations) → `projection.ensure_watch(buf)` → `set_applying(buf,false)`.

---

## Milestone 2 — projection + markers + modes

### Task 1: `review.markers` — the pure 🤖 parser

**Files:** Create `nvim/review/markers.lua`, Test `nvim/review/markers_test.lua`

- [ ] **Step 1: failing test** — assert on a `lines` fixture: a `🤖[fix this]` → one marker, `sections[1].type=='user'`, `ready==true`; a `🤖<quoted>{agent note}` → `quoted.text=='quoted'`, `sections[1].type=='agent'`, `pending==true`; a `🤖~old~` → `strike.text=='old'`; a 🤖 inside a ``` fence ``` is NOT parsed; alternating `🤖[a]{b}[c]` → 3 sections in order. **Cover the subtle invariants too:** `🤖[a]{b}` (last section `{}`) → `pending==true, ready==false` (last-section determines both); `🤖~old~[reply]` → `ready==false` (a strike is never "ready", even with a trailing `[]`). Model the harness on `nvim/slug_test.lua` (`dofile` + `eq` + `os.exit(1)`).
- [ ] **Step 2: run → fail** (`nvim -l nvim/review/markers_test.lua`).
- [ ] **Step 3: implement** — port `find_matching_bracket`, `parse_marker_sections`, fence-range + inline-code exclusion, and `parse_markers(lines)` from parley `review/init.lua:23-323` (verbatim logic; drop the LLM/quickfix bits). Expose `M.parse_markers(lines)` and `M.parse_marker_sections(text,pos,len,opts)` (the highlighter seam). Pure — no vim API except none (all string ops).
- [ ] **Step 4: run → pass** (`markers_test ok`).
- [ ] **Step 5: wire into `test-lua` + commit** — `#66 M2: review.markers — pure 🤖 review-request parser`.

### Task 2: `review.mode` — pure mode parser + stock briefs

**Files:** Create `nvim/review/mode.lua`, `nvim/review/modes/{developmental,line-editing,copy-editing,proofreading,fact-check,free-form}.md`, Test `nvim/review/mode_test.lua`

- [ ] **Step 1: failing test** — `parse` a frontmatter string → `{name,scope,deletions,frontier,body}` with defaults applied + invalid-flag rejection; `directives(mode)` contains the right scope/frontier/deletions lines per the flags; `list(dir)` over the real `modes/` returns the briefs sorted by `order` then name, filtering files whose frontmatter `name` ≠ basename.
- [ ] **Step 2: run → fail.**
- [ ] **Step 3: implement** — port `mode.lua` verbatim from parley `review/mode.lua` (`parse`, `directives`, `load`, `list`, `VALID`/`DEFAULT`); copy the 6 stock briefs from parley `review/modes/*.md` unchanged.
- [ ] **Step 4: run → pass.**
- [ ] **Step 5: wire + commit** — `#66 M2: review.mode — pure mode-brief parser + stock modes (ported from parley)`.

### Task 3: `apply.snapshot` / `apply.apply_snapshot` — decoration read + restore

**Files:** Modify `nvim/review/apply.lua`, Test `tests/review-apply-test.sh` (extend)

- [ ] **Step 1: failing test** (headless, append to the apply driver) — apply records including a **multi-line** edit (`new` spanning ≥2 lines); `local snap = apply.snapshot(buf)`; assert `snap.hl[1].end_line > snap.hl[1].line` for the multi-line one (the span is captured, not just the start row) and `snap.diags[1].message` is the explain. Then clear decorations, `apply.apply_snapshot(buf, snap)`, and assert the restored extmark has the same `{line,end_line}` range (query with `details=true`) and the diagnostics are back.
- [ ] **Step 2: run → fail.**
- [ ] **Step 3: implement** — extract a private `clear(buf)` from `place` (clears HL ns; the diagnostic set replaces). `M.snapshot(buf)`: `nvim_buf_get_extmarks(buf, HL, 0, -1, {details=true})` → `hl={ {line=m[2], end_line=(m[4] and m[4].end_row) or m[2]} }`; `vim.diagnostic.get(buf, {namespace=DIAG})` → `diags={ {lnum,end_lnum,message} }`. `M.apply_snapshot(buf, snap)`: `clear(buf)`, then place a ranged extmark per `snap.hl` entry (`end_row`,`hl_eol`,`DiffChange`) and `vim.diagnostic.set(DIAG, buf, <snap.diags mapped to INFO>)` — two independent layers.
- [ ] **Step 4: run → pass.**
- [ ] **Step 5: commit** — `#66 M2: apply.snapshot/apply_snapshot — decoration read+restore for projection`.

### Task 4: `review.projection` + orchestrator integration

**Files:** Create `nvim/review/projection.lua`, Modify `nvim/review/init.lua`, Test `tests/review-projection-test.sh`

- [ ] **Step 1: failing end-to-end test** (headless): open a buffer, apply a round (use a **multi-line** edit so the span is exercised), assert decorations present. **(a) undo** → cleared (empty pre-round snapshot); **redo** → restored, and the restored extmark's `end_row` matches (query `details=true`). **(b) manual edit** elsewhere → decorations still present; **undo it** → restored to the round snapshot. **(c) round-2 idempotence** (the second mis-restore risk): apply a *second* round, then undo back to the round-1 state → **round-1's decorations are still present, not cleared** (proves `record_empty_for`'s `if records[h]==nil` guard: round-2's pre-content == round-1's output, already recorded WITH its decorations). Drive via `nvim --headless -u NONE` + a result file.
- [ ] **Step 2: run → fail.**
- [ ] **Step 3: implement** — port `projection.lua` (state, `decide`, `project`, `record`, `record_empty_for`, `ensure_watch`, `set_applying`, `reset`, `put` with `MAX_RECORDS`), calling `apply.snapshot`/`apply.apply_snapshot`. **Preserve two guards verbatim:** `record_empty_for` only records the empty snapshot `if records[hash]==nil` (else round-2 undo wrongly clears round-1's styling); `project` early-returns when `set_applying` is set. Port `reset(buf)` (deletes the autocmd) — the headless test needs it to avoid double-attached watchers across cases. Then wire `on_agent_round`: capture `base` before apply, `set_applying(true)`, apply, `record_empty_for(buf, base)`, `record(buf)`, `ensure_watch(buf)` (lazy — AFTER the snapshots exist, so it never fires before they're recorded), `set_applying(false)`.
- [ ] **Step 4: run → pass** (undo/redo coherence + riding).
- [ ] **Step 5: wire `tests/review-projection-test.sh` into `test-review` + commit** — `#66 M2: review.projection — undo/redo-coherent, riding decorations (no clear-on-apply)`.

### Task 5: Milestone close

- [ ] Full suite: `make test-lua` + `make test-review`.
- [ ] Update `atlas/review-workbench.md` (projection/markers/modes now present; M2 done).
- [ ] **Use ariadne's sdlc directly** (pair's binary is stale): from pair's dir, `<ariadne-built sdlc> milestone-close --issue 66 --milestone M2 --verified '<evidence>'` — omit `--actual` to get the measured active-time-v3 value, then pass it. Fix Critical/Important from the auto-dispatched review before crossing; log the verdict.

---

## Open details to resolve in-milestone

- **Diagnostics don't auto-ride like extmarks.** Extmarks shift with edits; `vim.diagnostic` is line-set and won't shift mid-edit until re-snapshot (same in parley). Acceptable for M2 (exact-hash restore is correct; mid-edit diag lines are best-effort). Note it; M3 can re-place on round.
- **`set_applying` coverage** — round 2+ apply edits fire the (now-attached) watcher; the guard must wrap the whole apply. Test a second round to confirm no mid-apply capture corrupts the snapshot.
- **Marker rendering deferred to M3** — M2 ships the parser only; highlighting 🤖 sections in the buffer (the `ParleyReview*` hl groups) is window UI (M3).
- **Modes are data-only in M2** — parsed + `directives()` available, but not yet composed into an agent prompt (that's M4's SKILL.md) nor surfaced in a picker (M3).
