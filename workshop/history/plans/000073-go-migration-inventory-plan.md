# Go Migration Inventory Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Produce the factual artifact/caller/runtime contract that later Go migration issues use before moving behavior.

**Architecture:** This is an inventory-only milestone: no runtime behavior changes, no launcher rewrites, and no new command surfaces. `atlas/architecture.md` already owns the narrative description of Pair and its high-level pieces; #73 should not create a second narrative list. The durable source of truth for the migration contract will live in `atlas/go-migration-inventory.md`, linked from `atlas/index.md` and cross-linked from `atlas/architecture.md`, with architecture.md trimmed to a narrative pointer for inventory-level facts. `ARCH-PURPOSE` shapes the coverage requirement: the issue is not done until every named surface in #73 has a disposition, priority, and caller map. `ARCH-DRY` keeps caller/runtime/disposition facts in one atlas table while architecture.md remains the prose map. `ARCH-PURE` applies as a documentation schema discipline: separate artifact facts from migration judgments so later issues can consume the table without re-inspecting every file.

**Tech Stack:** Markdown atlas docs, shell repository inspection with `rg`/`find`, existing `make build` and targeted smoke tests.

---

## Core Concepts

### Pure Entities

| Name | Lives in | Status |
|------|----------|--------|
| `MigrationInventoryEntry` | `atlas/go-migration-inventory.md` | new |
| `TargetDisposition` | `atlas/go-migration-inventory.md` | new |

- **MigrationInventoryEntry** — one row per installed or runtime-called artifact, capturing current path/type, callers, runtime contract, files/env, target disposition, and priority.
  - **Relationships:** 1:N from migration inventory to artifacts; each artifact may have N callers and N runtime files/env dependencies.
  - **DRY rationale:** Later #74-#79 work should read one contract instead of re-deriving hidden zellij/nvim/script callers in every ticket.
  - **Future extensions:** Can widen with status columns as each artifact is migrated or retired.

- **TargetDisposition** — the classification that says whether an artifact becomes a Go subcommand, remains native, stays as a temporary shim, or is test-only.
  - **Relationships:** 1:1 with each `MigrationInventoryEntry`.
  - **DRY rationale:** Keeps migration decisions single-sourced and prevents #74/#76/#77 from each inventing a different name for the same end state.
  - **Future extensions:** Can split compatibility shim into "keep through #77" vs "keep through #79" if the later packaging work needs finer staging.

### Integration Points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `RepoSurfaceInspection` | `atlas/go-migration-inventory.md` | new | `bin/`, `bin/lib/`, `cmd/`, `nvim/`, `zellij/`, `Makefile`, `Makefile.local`, docs/tests |
| `FunctionalVerification` | `workshop/issues/000073-go-migration-inventory.md` | modified | build/test commands proving no behavior changed |

- **RepoSurfaceInspection** — the manual inspection pass over repository surfaces and call sites.
  - **Injected into:** The inventory rows as cited artifact/caller facts.
  - **Future extensions:** Can become a script later if the inventory starts drifting, but #73 should not add automation unless manual coverage is impossible.

- **FunctionalVerification** — lightweight proof that Pair still builds and the launcher shell remains untouched.
  - **Injected into:** #73 `## Log` and close evidence.
  - **Future extensions:** Later code-bearing issues should use stronger process-level fakes; this issue only needs no-change verification.

## Chunk 1: Inventory Artifact

### Task 1: Create the atlas inventory

**Files:**
- Create: `atlas/go-migration-inventory.md`
- Modify: `atlas/architecture.md`

- [x] **Step 1: Build the artifact list**

Run:

```bash
rg --files bin cmd nvim zellij | sort
sed -n '1,260p' Makefile.local
sed -n '1,120p' Makefile
printf '%s\n' Makefile Makefile.local README.md cmd/pair-scribe/README.md doctor/README.md doctor/SKILL.md
rg -n "PAIR_HOME|PAIR_DATA_DIR|PAIR_TAG|PAIR_AGENT|pair-wrap|pair-slug|pair-context|pair-continuation|pair-changelog|pair-scribe|scrollback-render|session-watch|pair-title|Homebrew|brew|install" README.md atlas docs bin cmd nvim zellij Makefile Makefile.local tests
```

Expected: commands complete and expose all installed/runtime surfaces named by #73, including both `Makefile` and `Makefile.local` plus packaging/install docs. `tests/` is treated as caller and process-fake evidence; test-only seams should be grouped by target artifact/behavior, with each group naming which runtime artifact it exercises, rather than exhaustively inventoried per test file.

- [x] **Step 2: Write the inventory schema and rows**

Add `atlas/go-migration-inventory.md` as the authoritative contract table, deriving the high-level piece vocabulary from `atlas/architecture.md` but adding columns that architecture.md intentionally lacks:

- scope and non-goals;
- target disposition vocabulary;
- priority vocabulary;
- artifact inventory covering `bin/*`, `bin/lib/*`, `cmd/*`, `nvim/*`, `zellij/*`, `Makefile.local`, packaging/install docs, and test-only seams;
- hidden caller summary for zellij layout/config and nvim shell-outs;
- recommended sequence adjustment notes, if any.

Then update `atlas/architecture.md` so its `## Pieces` section points to `atlas/go-migration-inventory.md` for exhaustive artifact/caller/runtime/disposition facts. Keep architecture.md as the narrative map and avoid duplicating the full table there.

- [x] **Step 3: Check coverage against the repository**

Run:

```bash
rg --files bin cmd nvim zellij | sort > /tmp/pair-artifacts.txt
{
  rg --files bin cmd nvim zellij
  printf '%s\n' Makefile Makefile.local README.md cmd/pair-scribe/README.md doctor/README.md doctor/SKILL.md
} | sort -u > /tmp/pair-inventory-required.txt
rg --no-filename -o "`(bin/[^`]+|cmd/[^`]+|nvim/[^`]+|zellij/[^`]+|Makefile|Makefile.local|README.md|cmd/pair-scribe/README.md|doctor/README.md|doctor/SKILL.md)`" atlas/go-migration-inventory.md \
  | sed 's/^`//; s/`$//' \
  | sort -u > /tmp/pair-inventory-documented.txt
comm -23 /tmp/pair-inventory-required.txt /tmp/pair-inventory-documented.txt
```

Expected: `comm` prints nothing for explicitly inventoried paths, or every printed path is explicitly covered by a grouped row with a stated grouping rule in the inventory. This command does not prove completeness for grouped `cmd/`, `nvim/`, or test-only surfaces; the inventory must name the grouping rule and representative files for those surfaces. Separately inspect `tests/` and record grouped test-only seams such as process fakes, shell integration tests, and headless nvim drivers, each tied to the runtime artifact/behavior it exercises.

### Task 2: Link the inventory from atlas

**Files:**
- Modify: `atlas/index.md`
- Modify: `atlas/architecture.md`

- [x] **Step 1: Add the inventory link**

Add `atlas/go-migration-inventory.md` to the atlas index in the existing style.

Update `atlas/architecture.md` to cross-link the inventory from `## Pieces` and `## Packaging migration target (#72)`.

- [x] **Step 2: Verify discoverability**

Run:

```bash
rg -n "go-migration-inventory|Go migration inventory" atlas/index.md atlas/go-migration-inventory.md
```

Expected: both the index and inventory are found.

## Chunk 2: Issue Closeout

### Task 3: Update #73

**Files:**
- Modify: `workshop/issues/000073-go-migration-inventory.md`

- [x] **Step 1: Record the estimate**

Set `estimate_hours: 2.56` and add a `## Estimate` block derived from estimate-logic-v2 Method A:

```estimate
model: estimate-logic-v2
familiarity: 1.0
item: atlas-docs        design=0.1 impl=0.2
item: pensive           design=0.5 impl=0.2
item: cross-cutting-refactor design=0.5 impl=0.3
item: milestone-review  design=0.1 impl=0.3
design-buffer: 0.3
total: 2.56
```

Expected: `sdlc change-code --issue 73 --dry-run` accepts the estimate reconciliation.

- [x] **Step 2: Mark the plan checklist as complete**

Check off #73 plan items as the inventory and atlas link land, and add a log entry summarizing:

- inventory path;
- no behavior changes;
- verification commands.

### Task 4: Verify and close

**Files:**
- No runtime files expected.

- [x] **Step 1: Enter implementation gate**

Run:

```bash
sdlc change-code --issue 73
```

Expected: gate passes and creates/uses the issue branch.

- [x] **Step 2: Run verification**

Run:

```bash
make build
make test
```

Expected: all pass. If there is a known pre-existing failure, record the exact failing target and rerun the focused no-change verification (`make build`, `bash tests/pair-continue-test.sh`, `bash tests/pair-session-watch-test.sh`) before close.

- [x] **Step 3: Inspect diff for behavior changes**

Run:

```bash
git diff --stat
git diff -- bin cmd nvim zellij Makefile.local
```

Expected: no runtime changes under `bin`, `cmd`, `nvim`, `zellij`, or `Makefile.local`.

- [ ] **Step 4: Close #73**

Run:

```bash
sdlc close --issue 73 --verified "make test; git diff confirms no runtime changes"
```

Expected: close gate runs the mandatory fresh-context review and archives #73 if approved.
