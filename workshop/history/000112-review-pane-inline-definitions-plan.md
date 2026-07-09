# Review Pane Inline Definitions Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add durable inline definitions to pair's review pane, modeled on parley.nvim#161/#166/#167 but routed through pair's existing review agent seam.

**Architecture:** Keep all footnote parsing, insertion, stripping, and diagnostic span calculation in a pure Lua module (`ARCH-PURE`). The review pane is a thin IO shell that writes a request artifact, pokes the existing agent, polls a response artifact, then applies the pure transform. A small Go `pair review definition` helper writes response artifacts so the agent can answer through a deterministic CLI rather than hand-editing files (`ARCH-DRY`, `ARCH-PURPOSE`).

**Tech Stack:** Lua 5.1/Neovim APIs, pair review Lua modules, Go dispatcher/reviewcmd helpers, shell-based headless Neovim tests.

---

## Core Concepts

### Pure Entities

| Name | Lives in | Status |
|------|----------|--------|
| `DefinitionFootnote` | `nvim/review/define.lua` | new |
| `DefinitionFooter` | `nvim/review/define.lua` | new |
| `DefinitionDiagnostic` | `nvim/review/define.lua` | new |
| `ReviewContextWithoutDefinitions` | `nvim/review/define.lua` and `nvim/review/poke_bodies.lua` | modified |

- **DefinitionFootnote** - a selected document span plus a stable markdown footnote id.
  - **Relationships:** 1:N with a review document; one document may hold many definition references.
  - **DRY rationale:** One transform should serve visual definition apply, redefinition, and reopen rehydration.
  - **Future extensions:** If pair later supports non-visual definition requests, the same entity accepts an explicit term/range.

- **DefinitionFooter** - the managed final footer after `---` containing only markdown footnote definitions.
  - **Relationships:** 1:1 with a document when definitions exist; owns N `DefinitionFootnote` lines.
  - **DRY rationale:** The same final-block predicate protects both footer updates and review-context stripping.
  - **Future extensions:** Dedicated coloring or float display can derive from this footer without changing persistence.

- **DefinitionDiagnostic** - derived diagnostic/highlight span for `term[^id]` and its stored definition text.
  - **Relationships:** N:1 with `DefinitionFooter`; diagnostics are derived, never the source of truth.
  - **DRY rationale:** Reopen rendering and just-applied rendering use one span source, matching parley.nvim#167.
  - **Future extensions:** Centered floats can be added by changing display code, not persistence.

- **ReviewContextWithoutDefinitions** - review prompt/document text with only the managed final definition footer removed.
  - **Relationships:** 1:1 transform over the current document content before it is sent to the agent.
  - **DRY rationale:** Footer stripping should not be reimplemented in poke bodies or agent prompt code.
  - **Future extensions:** Other managed document ephemera can compose as additional strip transforms.

### Integration Points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `DefinitionRequestSeam` | `nvim/review/definition_seam.lua` | new | `$PAIR_DATA_DIR` JSON artifacts |
| `ReviewDefinitionCLI` | `cmd/internal/reviewcmd` and `cmd/internal/dispatcher` | new | agent-authored definition response |
| `ReviewPaneDefinitionUI` | `nvim/review.lua` | modified | visual selection, keymap, buffer edit, polling |
| `ReviewDecorationProjection` | `nvim/review/apply.lua` / `nvim/review/projection.lua` | modified | Neovim extmarks/diagnostics |
| `RuntimeBundle` | `cmd/internal/runtimebundle/assets/runtime/files/nvim/review/` | modified | installed pair runtime |

- **DefinitionRequestSeam** - writes `review-definition-request-<tag>.json` and reads/removes `review-definition-result-<tag>.json`.
  - **Injected into:** `ReviewPaneDefinitionUI`.
  - **Future extensions:** Request ids allow multiple outstanding requests later, though v1 keeps one pending request per pane.

- **ReviewDefinitionCLI** - `pair review definition <request-id> <definition...>` writes the result artifact with structured JSON.
  - **Injected into:** the human/agent protocol via poke text; tests use a fake data dir and env.
  - **Future extensions:** `--term` or stdin support can be added if definitions become large.

- **ReviewPaneDefinitionUI** - visual binding extracts the selection, writes request, pokes the agent, polls for result, then applies the pure footnote transform.
  - **Injected into:** live review pane only; no draft-pane behavior changes.
  - **Future extensions:** A mode menu can expose definition if `<M-CR>` conflicts with send behavior.

- **ReviewDecorationProjection** - existing review decoration snapshot/apply path must include definition decorations and exact spans.
  - **Injected into:** review apply/projection already owns extmarks and diagnostics.
  - **Future extensions:** Dedicated namespaces may be introduced if review edits and definitions need independent accept/clear semantics.

- **RuntimeBundle** - generated embedded runtime must include new Lua files.
  - **Injected into:** installed `pair` when running from embedded runtime.
  - **Future extensions:** Any new review Lua module must remain covered by runtime-bundle drift checks.

## Chunk 1: Pure Definition Footnotes

### Task 1: Add pure helper tests

**Files:**
- Create: `nvim/review/define.lua`
- Create: `nvim/review/define_test.lua`

- [ ] **Step 1: Write failing tests for slugging, footer apply, redefine, strip, and diagnostics**

Add `nvim/review/define_test.lua` using the existing `nvim -l` style:

```lua
local define = dofile('nvim/review/define.lua')

local function eq(a, b, msg)
  assert(vim.inspect(a) == vim.inspect(b), msg .. '\nleft=' .. vim.inspect(a) .. '\nright=' .. vim.inspect(b))
end

eq(define.footnote_id('Amazon Standard Identification Number'), 'amazon-standard-identification-number', 'slug')

local applied = define.apply_definition_footnote({ 'here is ASIN in context' }, 1, 8, 1, 11, 'ASIN', 'Amazon Standard Identification Number.')
eq(applied.lines, {
  'here is ASIN[^asin] in context',
  '',
  '---',
  '',
  '[^asin]: Amazon Standard Identification Number.',
}, 'apply footnote')
eq(applied.diagnostic_span, { line = 0, col = 8, end_line = 0, end_col = 19 }, 'exact span')

local again = define.apply_definition_footnote(applied.lines, 1, 8, 1, 11, 'ASIN', 'Updated.')
eq(again.lines[1], 'here is ASIN[^asin] in context', 'redefine does not duplicate inline ref')
eq(again.lines[#again.lines], '[^asin]: Updated.', 'redefine updates footer')
```

- [ ] **Step 2: Run the test and verify it fails**

Run: `nvim -l nvim/review/define_test.lua`
Expected: FAIL because `nvim/review/define.lua` does not exist.

- [ ] **Step 3: Implement `nvim/review/define.lua`**

Port the pure pieces from `../parley.nvim/lua/parley/define.lua`, adjusted to pair naming:

- `slice_selection(lines, l1, c1, l2, c2)`
- `footnote_id(term)`
- `format_footnote_line(id, definition)`
- `strip_definition_footnote_footer(text)`
- `apply_definition_footnote(lines, l1, c1, l2, c2, term, definition)`
- `footnote_diagnostics(lines)`

Keep it free of Neovim APIs except test-only `vim.inspect` usage in tests.

- [ ] **Step 4: Run pure tests green**

Run: `nvim -l nvim/review/define_test.lua`
Expected: PASS.

- [ ] **Step 5: Add the test to `Makefile.local`**

Modify `test-lua` so `nvim -l nvim/review/define_test.lua` runs with the other review pure tests.

## Chunk 2: Definition Response CLI

### Task 2: Add `pair review definition`

**Files:**
- Modify: `cmd/internal/reviewcmd/run.go`
- Modify: `cmd/internal/reviewcmd/runcli.go`
- Modify: `cmd/internal/reviewcmd/run_test.go`
- Modify: `cmd/internal/dispatcher/dispatcher.go`
- Modify: `cmd/internal/dispatcher/dispatcher_test.go`

- [ ] **Step 1: Write failing Go tests**

Add tests that call `reviewcmd.RunDefinition` with a fake runtime/data dir and assert:

- missing `$PAIR_DATA_DIR` exits non-zero.
- `request-id` and definition text are required.
- output JSON is written atomically to `review-definition-result-<tag>.json`.
- JSON includes `request_id`, `definition`, optional `term`, and `session` using the same session-priority helper as review targets.

- [ ] **Step 2: Run focused Go tests red**

Run: `go test ./cmd/internal/reviewcmd -run 'TestRunDefinition|TestRunTarget' -count=1`
Expected: FAIL because `RunDefinition` is undefined.

- [ ] **Step 3: Implement CLI and dispatcher registration**

Add dispatcher family `review definition`, wire it to `reviewcmd.RunDefinitionCLI`, and implement:

```text
pair review definition [--term TERM] <request-id> <definition...>
```

Use `encoding/json` and the existing `Runtime.WriteAtomic` seam. The result path is:

```text
$PAIR_DATA_DIR/review-definition-result-<PAIR_TAG|default>.json
```

- [ ] **Step 4: Run Go tests green**

Run: `go test ./cmd/internal/reviewcmd ./cmd/internal/dispatcher -count=1`
Expected: PASS.

## Chunk 3: Review Pane Request and Render

### Task 3: Add request seam and review-pane integration

**Files:**
- Create: `nvim/review/definition_seam.lua`
- Create: `tests/review-definition-test.sh`
- Modify: `nvim/review.lua`
- Modify: `nvim/review/apply.lua`
- Modify: `nvim/review/init.lua`
- Modify: `nvim/review/poke_bodies.lua`
- Modify: `Makefile.local`

- [ ] **Step 1: Write headless integration test**

Create `tests/review-definition-test.sh` with a driver that:

1. Opens a buffer containing `here is ASIN in context`.
2. Calls an exposed test helper (for example `_G.PairReviewTest.define_selection`) with the ASIN span.
3. Asserts request JSON was written.
4. Writes a fake result JSON as the CLI would.
5. Drives the poll/apply function.
6. Asserts buffer text contains `ASIN[^asin]`, footer contains `[^asin]: ...`, diagnostics/highlights span only `ASIN[^asin]`, and redefine updates instead of duplicating.

- [ ] **Step 2: Run integration test red**

Run: `bash tests/review-definition-test.sh`
Expected: FAIL because no definition seam/UI exists.

- [ ] **Step 3: Implement `definition_seam.lua`**

Add pure-ish path helpers plus JSON read/write wrappers:

- `request_path(data_dir, tag)`
- `result_path(data_dir, tag)`
- `write_request(data_dir, tag, request)`
- `read_result(data_dir, tag)`
- `clear_result(data_dir, tag)`

Follow existing seam style in `nvim/review/seam.lua`.

- [ ] **Step 4: Extend review apply/render surface**

Add functions in `nvim/review/apply.lua` that place definition decorations using the existing `HL` and `DIAG` namespaces:

- `place_definition(buf, diagnostic)` for just-applied definitions.
- `rehydrate_definitions(buf)` or equivalent called from `review.start`.

Use exact spans from `define.footnote_diagnostics`; do not whole-line highlight.

- [ ] **Step 5: Wire visual binding and polling in `nvim/review.lua`**

Bind visual definition to an available key. Preferred v1 binding: `<M-d>` to avoid overloading `<M-CR>` review send. The command should:

1. Read visual marks.
2. Use `define.slice_selection`.
3. Guard empty/whitespace selections with `vim.notify`.
4. Write request JSON containing request id, file, term, selected range, and stripped document context.
5. Poke the agent with a clear instruction:
   `Definition requested for <term>. Reply by running: pair review definition <id> <definition>`
6. Start/poke a timer that polls for the result.
7. On result, apply the footnote transform, write the buffer, re-render definitions, and clear the result artifact.

- [ ] **Step 6: Run review definition integration green**

Run: `bash tests/review-definition-test.sh`
Expected: PASS.

## Chunk 4: Agent Context Stripping

### Task 4: Keep definition footer out of review prompts

**Files:**
- Modify: `nvim/review/poke_bodies.lua`
- Modify: `nvim/review/poke_bodies_test.lua`
- Modify if needed: `nvim/review/init.lua`

- [ ] **Step 1: Add failing poke-body tests**

Add a test proving a document body ending with a managed definition footer is stripped from any review-context prose sent to the agent, while a normal trailing `---` block with prose is preserved.

- [ ] **Step 2: Run poke body tests red**

Run: `nvim -l nvim/review/poke_bodies_test.lua`
Expected: FAIL until stripping is wired.

- [ ] **Step 3: Wire `define.strip_definition_footnote_footer` through review agent context**

Use the same helper from Chunk 1. If existing poke bodies do not include full content, wire stripping at the request/document field that definition requests send.

- [ ] **Step 4: Run tests green**

Run: `nvim -l nvim/review/poke_bodies_test.lua`
Expected: PASS.

## Chunk 5: Docs, Runtime Bundle, and Verification

### Task 5: Update docs and generated runtime

**Files:**
- Modify: `atlas/review-workbench.md`
- Modify: `atlas/index.md` only if a new linked page is introduced.
- Modify: `cmd/internal/runtimebundle/assets/runtime/files/nvim/review/define.lua`
- Modify: `cmd/internal/runtimebundle/assets/runtime/files/nvim/review/definition_seam.lua`
- Modify generated runtime manifest files if `make runtimebundle-generate` updates them.
- Modify: `workshop/issues/000112-review-pane-inline-definitions.md`
- Modify: `workshop/plans/000112-review-pane-inline-definitions-plan.md`

- [ ] **Step 1: Update atlas**

Document the review definition flow, keybinding, request/result seam, durable footnote footer, and exact-span decoration behavior in `atlas/review-workbench.md`.

- [ ] **Step 2: Regenerate embedded runtime**

Run: `make runtimebundle-generate`
Expected: generated runtime assets include new review Lua modules.

- [ ] **Step 3: Run focused verification**

Run:

```bash
nvim -l nvim/review/define_test.lua
nvim -l nvim/review/poke_bodies_test.lua
bash tests/review-definition-test.sh
go test ./cmd/internal/reviewcmd ./cmd/internal/dispatcher -count=1
```

Expected: all pass.

- [ ] **Step 4: Run broader verification**

Run:

```bash
make test-review
go test ./...
git diff --check
```

Expected: all pass.

- [ ] **Step 5: Close**

Use `sdlc close --issue 112 --verified '<evidence>'`. If close writes a huge review sidecar, compact it to verdict/window/findings/verification before committing close metadata.
