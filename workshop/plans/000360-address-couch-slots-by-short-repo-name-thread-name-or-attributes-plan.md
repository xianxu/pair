# Couch Slot Addressing (prefix, alias, agent) Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Every couch surface that takes a slot reference resolves the
repository part by exact name, alias or unique prefix, refuses ambiguity with
candidates, and `--send-to` can filter by agent.

**Architecture:** One pure resolver (`ResolveRepositoryName`) is shared by the
operation path (`couchcore.resolveSlotInput`) and the message path
(`couchmessage.ResolveRecipient`); each supplies its own candidate set
(enrolled repositories vs live bindings). Aliases live in a new root-store file
read through the store's guarded payload reader, and flow into the inventory
rows so the tab bar and switcher label `alias:N`.

**Tech Stack:** Go; existing couch store (`couchcore/threadstore*.go`), message
broker (`couchmessage`), switcher (`couchtty`), CLI (`couchcmd`).

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `RepositoryName` | `cmd/internal/couchcore/repositoryname.go` | new |
| `RepositoryMatch` | `cmd/internal/couchcore/repositoryname.go` | new |
| `ResolveRepositoryName` | `cmd/internal/couchcore/repositoryname.go` | new |
| `ValidateRepositoryAlias` | `cmd/internal/couchcore/repositoryname.go` | new |
| `FormatRepositoryCandidates` | `cmd/internal/couchcore/repositoryname.go` | new |
| `ErrRepositoryNotFound` | `cmd/internal/couchcore/repositoryname.go` | new |
| `ErrRepositoryAmbiguous` | `cmd/internal/couchcore/repositoryname.go` | new |
| `Route` | `cmd/internal/couchmessage/routing.go` | new |
| `ResolveRecipient` | `cmd/internal/couchmessage/routing.go` | modified |
| `Candidate.Alias` | `cmd/internal/couchmessage/routing.go` | modified |
| `Request.Agent` | `cmd/internal/couchmessage/protocol.go` | modified |
| `ActionableThreadSummary.RepositoryAlias` | `cmd/internal/couchcore/actionableinventory.go` | modified |
| `ActionableThreadSummary.Label` | `cmd/internal/couchcore/actionableinventory.go` | modified |
| `PresentThreads` | `cmd/internal/couchtty/thread_presentation.go` | modified |
| `menuActionsFor` | `cmd/internal/couchtty/menu_switchagent.go` | modified |

- **RepositoryName** — `{Key, Dir, Alias string}`. `Key` is the caller's
  identity for the repository (primary root for operations, family name for
  messaging); `Dir` is the directory basename; `Alias` may be empty.
  - **Relationships:** N candidates per resolution; 0..1 alias per repository.
  - **DRY rationale:** today two exact-match rules exist
    (`enrolledPrimary`, `ResolveRecipient`'s `f != family`); both widen the
    same way, so one rule.
  - **Future extensions:** case-insensitive matching would widen here only.
- **ResolveRepositoryName(raw, repos) (RepositoryName, RepositoryMatch, error)**
  — precedence: exact `Dir` or `Alias` (all exact hits must share one `Key`,
  else `ErrRepositoryAmbiguous`), then unique prefix of `Dir` or `Alias`
  (deduplicated by `Key`; several keys → `ErrRepositoryAmbiguous`), else
  `ErrRepositoryNotFound`. Errors carry `FormatRepositoryCandidates` output.
  Rows sharing a `Dir` with different `Key`s (two `parley` checkouts in
  different fleet roots) are ambiguous at exact match, never silently picked
  (#353 BR-5).
- **FormatRepositoryCandidates(repos)** — sorted `dir (alias)` list, at most
  12 entries then `and N more`, capped at 1024 bytes, so a miss stays inside
  `MaxReceiptDetailBytes`-scale bounds (lessons: derive transport limits from
  worst-case domain bounds).
- **ValidateRepositoryAlias(alias, selfKey, repos)** — `workspaceRepoName`
  token; not equal to any other repository's `Dir` or `Alias`; equal to its own
  `Dir` is refused (pointless). Empty means clear and is valid.
- **Route** — `{Target, Agent string}`; replaces the bare target string in
  `ResolveRecipient` and `Broker.Send`. `Agent` empty means any agent.
- **ResolveRecipient(route, candidates, aliases, now)** — canonicalizes the
  route's repository through `ResolveRepositoryName` over the candidates'
  families (+ `aliases` map, family → alias), then runs today's exact/family
  selection, applying the agent filter **after** the family identity check.
  An exact `repo:N` whose binding's agent differs from `route.Agent` is
  refused. A miss wraps `ErrInvalidTarget` with the live slots listed
  (`repo:N agent`), bounded the same way.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `ThreadStore.RepositoryAliases` / `SetRepositoryAlias` | `cmd/internal/couchcore/repositoryalias_store.go` | new | root store file `repository-aliases.json` |
| `Couch.repositoryNames` | `cmd/internal/couchcore/slotcontext.go` | new | manifest `SlotRepositories` + alias file |
| `Couch.resolveSlotInput` / `enrolledPrimary` | `cmd/internal/couchcore/slotcontext.go` | modified | slot catalog discovery |
| `Couch.SetRepositoryAlias` + `alias` operation | `cmd/internal/couchcore/ops.go`, `operationdispatch.go` | new | store write |
| `Broker` alias hook | `cmd/internal/couchmessage/broker.go` | modified | alias source injected by `couchcmd/message_service.go` |
| `--send-to … --agent` | `cmd/internal/couchcmd/cli.go`, `messages.go` | modified | argv |

- **repository-aliases.json** — `{"schema_version":1,"aliases":{"<primary
  root>":"blog"}}` in the **root** store (`familyRootStore()`), read with
  `readOptionalPayload` + `strictThreadStoreJSON`, written with
  `writeStoreAtomicLocked` under `withLock`. A separate file, not a manifest
  field: the manifest decodes with `DisallowUnknownFields`, so a new field
  would make the whole store unreadable to any older couch binary (slots build
  their own). ARCH-FUNERAL: created by `SetRepositoryAlias`; one entry per
  enrolled repository, so bounded by enrollment; clearing deletes the entry;
  entries whose root is no longer enrolled are ignored on read and dropped on
  the next write.
  - **Injected into:** `Couch.repositoryNames` (operations, inventory) and
    the broker via `messageService` (read fresh per send, outside the broker
    lock).
- **Broker alias hook** — `NewBroker` gains `aliases func(context.Context)
  (map[string]string, error)`; nil means none. `messageService` builds it from
  a read-only root store: family = `filepath.Base(primary root)`; a family
  that maps to two different aliases is left out (ambiguous family is already
  refused by repository identity).
- **Fakes:** the existing `SlotCatalogFake` and store test fixtures; the
  broker tests' existing fake endpoints. No new external seam.

## Decisions

- Thread names are not addresses (operator, 2026-09-30); rename retires in #363.
- Prefix fallback for operations applies only when `FleetRoot/<raw>` does not
  exist (`errors.Is(err, fs.ErrNotExist)` from an `Lstat`); an existing but
  broken sibling keeps today's error instead of silently routing elsewhere.
  An exact enrolled name or alias is checked first.
- `--agent` requires a repository target; no cross-repository selection.
- Matching is case-sensitive, like directory names today.
- The "alias" switcher action is added on live `:0` rows now; removing rename
  and describe is #363's.

## Tasks

### Task 1: Pure repository-name resolver

**Files:** Create `cmd/internal/couchcore/repositoryname.go`,
`cmd/internal/couchcore/repositoryname_test.go`.

- [ ] Write table tests: exact dir; exact alias; exact dir beats another
      repo's prefix (`pair` vs `pair-tools`); unique prefix (`parley` →
      `parley.nvim`); prefix of an alias; ambiguous prefix lists both;
      exact dir shared by two keys is ambiguous; alias of A equal to dir of B
      is ambiguous; no match lists candidates; candidate list truncates at 12
      with `and N more` and stays ≤1024 bytes for 128 long names; empty or
      `:`-containing raw is `ErrRepositoryNotFound`.
- [ ] Tests for `ValidateRepositoryAlias`: valid, empty (clear), collides
      with other dir, collides with other alias, equals own dir, invalid token.
- [ ] Run `go test ./cmd/internal/couchcore -run 'RepositoryName|RepositoryAlias'` → FAIL.
- [ ] Implement.
- [ ] Re-run → PASS. Commit `#360: shared repository-name resolver`.

### Task 2: Alias store

**Files:** Create `cmd/internal/couchcore/repositoryalias_store.go`,
`repositoryalias_store_test.go`.

- [ ] Tests on a temp-dir store: absent file → empty map; set → read back;
      clear → entry gone; set on a slot-local store view writes the root
      store; unknown JSON field refused; entry for an un-enrolled root ignored
      on read and dropped on next write; collision refused through
      `ValidateRepositoryAlias` and leaves the file byte-identical
      (`cp`+`cmp` style assertion).
- [ ] Run → FAIL. Implement `RepositoryAliases() (map[string]string, error)`
      and `SetRepositoryAlias(primaryRoot, alias string) error` (validates
      against enrolled names inside the same lock as the write: lesson
      "revalidate authority before mutation").
- [ ] Run → PASS. Commit.

### Task 3: Operation-side resolution

**Files:** Modify `cmd/internal/couchcore/slotcontext.go`; tests in
`workspaceref_test.go` / a new `slotcontext_resolution_test.go`.

- [ ] Tests through `WorkspaceReferencePath` with `SlotCatalogFake`:
      `parley:1` → `parley.nvim` slot 1; `blog:1` with alias; an existing
      sibling `parley` directory wins over the prefix; a missing slot lists
      existing numbers (`slot pair:7 does not exist; existing: pair:1,
      pair:2`); a miss lists enrolled repositories; ambiguous prefix refused.
      Mutation check: revert the fallback, the prefix test fails.
- [ ] Implement `repositoryNames()` (manifest + aliases) and route both
      `resolveSlotInput`'s `ref.Repo != ""` branch and `enrolledPrimary`
      through `ResolveRepositoryName` per the Decisions order.
- [ ] Verify the switcher filter (`couchtty/menu.go` `ParseWorkspaceReference`
      use) and `couchcmd/run.go` go through `WorkspaceReferencePath`; add one
      CLI-level test resolving `parley:1`.
- [ ] Run `go test ./cmd/internal/couchcore ./cmd/internal/couchcmd` → PASS. Commit.

### Task 4: Message-side resolution and `--agent`

**Files:** Modify `cmd/internal/couchmessage/{routing,protocol,broker}.go`,
`cmd/internal/couchcmd/{cli,messages,message_service}.go`; tests in
`routing_test.go`, `broker_test.go`, `protocol_test.go`, `cli_test.go`.

- [ ] Pure tests on `ResolveRecipient`: `parley:1` → binding
      `parley.nvim:1`; `parley` family; alias `blog` family and `blog:1`;
      ambiguous prefix → `ErrAmbiguous` listing both; unknown → wraps
      `ErrInvalidTarget` listing live slots; `--agent codex` with two free
      slots (claude, codex) picks codex; no codex → `ErrNoRecipient`; exact
      slot with mismatched agent refused; two families with the same name and
      different repositories stay `ErrAmbiguous` even when the agent filter
      would leave one (#353 BR-5).
- [ ] Broker test (actual `Broker.Send` with fake endpoints): a send to
      `parley:1` is delivered to the `parley.nvim:1` endpoint (assert the
      endpoint received the body), and `--agent` selection delivers to the
      codex endpoint, not the claude one.
- [ ] Protocol: `Request.Agent` allowed only on `send`, must be
      `validFamily`; `ValidateRequest` accepts prefixes/aliases unchanged
      (they are already family-shaped). Test rejection on `actors`/`status`.
- [ ] CLI: `couch --send-to T [--agent A] --message M` parse tests (agent
      before or after? fixed order: `--send-to T --agent A --message M`);
      `--agent` empty or flag-shaped refused.
- [ ] `couch --actors` prints and JSON-encodes `Alias`.
- [ ] Wire `messageService` alias hook. Run
      `go test ./cmd/internal/couchmessage ./cmd/internal/couchcmd` → PASS. Commit.

### Task 5: Alias operation and labels

**Files:** Modify `cmd/internal/couchcore/{ops,operationdispatch,actionableinventory}.go`,
`cmd/internal/couchtty/{thread_presentation,menu,menu_switchagent}.go`.

- [ ] Declare `alias` (ExecuteDirectStore, EffectMetadata, RowAction):
      args `ref` (repository name, alias, path or `repo:N`; required on CLI),
      `alias` (optional; omitted reads), `clear` (FlagOnly boolean). Dispatch
      resolves `ref` to a primary root via `WorkspaceReferencePath` +
      `SlotIdentity.PrimaryRoot`, then `SetRepositoryAlias`.
- [ ] Inventory: fill `RepositoryAlias` on rows whose repository is enrolled
      (slot rows via `Target.Slot.PrimaryRoot`, `:0` rows via scope →
      primary root). `Label()` uses the alias for slot rows.
- [ ] `PresentThreads`: group name = alias when the group's rows carry one.
      Test: with alias `blog`, slot labels are `blog:1` and the `:0` label
      `blog`; the tab bar model (`statusModelLocked` output) shows the same.
- [ ] Switcher: `menuActionsFor` offers `alias` on live, non-slot rows with a
      repository root (same predicate as `add-slot`), as a text-input frame
      like `name`. Update the offered-implies-declared/permitted sweeps.
- [ ] Run `go test ./cmd/internal/couchcore ./cmd/internal/couchtty` → PASS. Commit.

### Task 6: Docs and full verification

**Files:** `cmd/internal/couchcmd/skills/couch/SKILL.md`, `couchcmd/run.go`
usage, `README.md`, `atlas/couch.md`.

- [ ] Skill table: short forms (`parley:1`, aliases) and
      `--send-to pair --agent codex`; note misses return candidates, so no
      inventory read is needed before sending.
- [ ] `couch --help`: `--send-to repo[:N] [--agent NAME] --message TEXT`,
      `couch alias`.
- [ ] README + atlas: resolution order, alias file and its lifecycle.
- [ ] `TMPDIR=<scratchpad> make test` fully green (memory: full suite before
      close). Live smoke (operator): in couch, `couch --send-to parley:1
      --message …` from another slot, and alias `xianxu.dev` → `blog` shows
      `blog:1` on the tab bar.

## Revisions

### 2026-10-01 — implementation reconciliation

- **No CLI form for `alias`.** Couch's public argv exposes only `--list`,
  `--show`, `--archived` and the message forms; switcher operations (rename,
  describe) are not argv-reachable, so `alias` is a switcher action on live
  `:0` rows like them. Task 5's `ref` argument is the primary root the switcher
  supplies; `clear` is set by an empty entry. The planned CLI rendering of
  `RepositoryAliasResult` was removed as unreachable.
- **Alias source on `messageAuthority`.** The broker is built in
  `newMessageService`, which has no couch handle, so the alias hook is a
  read-only probe on `messageAuthority` (`couchcmd/message_service.go`), set via
  `Broker.SetAliases` instead of a `NewBroker` parameter.
- **Message misses keep their codes.** An unknown repository returns the same
  `not-dispatched` / `unavailable` codes as a repository with no live slot,
  with the live slots appended, rather than `invalid-target`: from live
  bindings alone a typo and an offline repository are indistinguishable.
- **Alias validation is stricter than a directory name** (no whitespace), so
  every alias is also a valid message family.
- **Switcher filter** matches `repo:N` by directory or alias prefix without
  the uniqueness rule, since a filter lists every candidate.
- Entity mapping as delivered: `RepositoryAliasResult`, `Couch.RepositoryAlias`
  (`couchcore/repositoryalias.go`), `ApplyRepositoryAliases`
  (`couchcore/actionableinventory.go`), `menuAliasOffered`
  (`couchtty/menu_switchagent.go`), `messageFamilyAliases`
  (`couchcmd/message_service.go`), `Couch.repositoryPrimary` /
  `repositoryNames` (`couchcore/slotcontext.go`; `enrolledPrimary` removed).

### 2026-10-01 — close review round 1 (REWORK)

- BR-1: message-side name resolution covers the enrolled families plus the
  live ones (`Broker.SetFamilies`, `messageFamilies`), so an offline exact
  repository refuses instead of prefix-routing to another; a prefix shared by
  an offline and a live repository is ambiguous.
- Routing errors when the enrolled set cannot be read name the store, the
  alias file and the reset.
- Aliases label only the slot rows and the `:0` row (scope match and starting
  at the primary root); subdirectory threads keep their labels.
- The alias shadow check uses the repository's fleet root, the anchor
  `repositoryPrimary` resolves against.
- One bounded-list renderer (`FormatBoundedList`) serves repository and
  live-slot candidates.
