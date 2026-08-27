# Boundary Review — pair#149 (milestone M5)

| field | value |
|-------|-------|
| issue | 149 — couch: opaque tags and a human naming layer |
| repo | pair |
| issue file | workshop/issues/000149-couch-opaque-tags-and-a-human-naming-layer.md |
| boundary | milestone M5 |
| milestone | M5 |
| window | 6a714336ae3c8356ecdf2019c1ecc35b60719e81..636ea1071f416110a460485d55245463a61f0f42 |
| command | sdlc milestone-close --issue 149 --milestone M5 |
| reviewer | codex |
| timestamp | 2026-08-26T20:13:00-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M5 has strong migration, validation, and cross-scope foundations, and the complete Go suite passes. It cannot cross the boundary yet: the session-index relocation breaks upgrades with existing live `📁…` sessions, artifact-path enforcement misses real constructors despite claiming exhaustive authority, and the Core concepts table contradicts the implementation.

```findings
findings:
  - id: new
    severity: Critical
    family: durable-index-read-failure-authority
    title: |
      Moving the session-name index strands existing live sessions
    detail: |
      This is the 4th finding in family durable-index-read-failure-authority. ReadSessionNameIndex now reads only the selected scope, while callers convert absence into an empty index; no migration or compatibility read preserves the former global index. Existing live indexed sessions therefore disappear from attach and picker views and are treated as foreign-name collisions. State the class rule for durable index relocation, migrate or compat-read all prior authoritative locations, and fail closed on genuine corruption.
  - id: new
    severity: Critical
    family: artifact-construction-single-authority
    title: |
      The artifact coverage guard does not enforce its claimed constructor closure
    detail: |
      ARCH-DRY and ARCH-PURPOSE fail. productionSource excludes extensionless production scripts, allowing bin/pair-notify to reconstruct outer-tty paths undetected. The manifest also labels launcher/migrate.go and launcher/legacy_live.go resolved consumers although they still construct legacy names, while Neovim derives the image done path instead of consuming its exported binding. Enumerate the entire source class, place intentional legacy constructors behind an explicit artifactpath API, and add negative tests that fail on every constructor outside that authority.
  - id: new
    severity: Critical
    family: core-concept-kind-contract
    title: |
      The M5 Core concepts row names an entity that does not exist at its declared path
    detail: |
      This is the 5th finding in family core-concept-kind-contract. The plan declares pure ArtifactFamily in paths.go, but the implementation defines Family in manifest.go. Do not patch only this row: reapply the inventory rule to every M5 entity and append a plan revision so name, kind, location, and status are mechanically truthful.
  - id: new
    severity: Important
    family: user-facing-policy-docs
    title: |
      Atlas documentation still publishes the retired global artifact layout
    detail: |
      This is the 3rd finding in family user-facing-policy-docs. atlas/architecture.md and the pair-notify row in atlas/go-migration-inventory.md still instruct readers to use global data-dir plus tag formulas. Sweep the documentation class for old artifact formulas and describe exact scoped bindings consistently.
```

1. Strengths

- `MigrateLegacyRecord` is genuinely pure, clones its input, validates before and after enrichment, and leaves revision/occupancy changes to the store layer (`cmd/internal/couchcore/migration.go:13`).
- The migration tests cover corrupt-input preservation, interrupted journal recovery, metadata preservation, and byte-stable reruns.
- `artifactpath.Paths` validates composite components and exports exact bindings from one value (`cmd/internal/artifactpath/paths.go:188`).
- The cross-scope integration uses real shell and Neovim consumers and checks both layouts.
- Standalone registration is ordered before workspace-child launch, with a refusal test proving registration failure prevents launch.

2. Critical findings

- `cmd/internal/launcher/osruntime.go:598` reads only scoped `session-names.jsonl`; `createflow.go:345` silently replaces read failure with an empty index. `legacy_live.go:19` explicitly excludes `📁` sessions, while `session_index.go:353` treats a live unindexed name as foreign. Fix with a tested global-to-scoped migration or explicit compatibility read that preserves corruption authority. The regression fixture must start with only the pre-M5 global index and a detached `📁` session, then prove resume attaches to it rather than minting a suffix.
- `cmd/internal/artifactpath/coverage_test.go:133` recognizes production sources by extension only. Consequently, `bin/pair-notify:60` reconstructs `outer-tty-<tag>` while bypassing the guard. The checked plan also names unchanged constructors such as `launcher/migrate.go:42` as resolved consumers (`manifest.go:97`). Expand the guard across executable/shebang sources and make the current tree fail until every constructor is centralized.
- `workshop/plans/…-plan.md:62` declares `ArtifactFamily` in `paths.go`; the code defines `Family` in `manifest.go:5`. Reconcile the implementation or append a truthful plan revision after sweeping all M5 Core concepts.

3. Important findings

- `atlas/architecture.md:543` and `:1012`, plus `atlas/go-migration-inventory.md:135`, still document global tag-derived paths. Update the whole obsolete-formula class. README needs no M5 change because no new user-run command or flag was introduced.

4. Minor findings

None.

5. Test coverage notes

`git diff --check` and `go test ./... -count=1` pass. Focused artifactpath, couchcore, launcher, and runtimebundle tests also pass. Those green results confirm the new logic’s happy paths, but also demonstrate that the two principal regressions are not currently pinned:

- no test starts from a pre-M5 global session index;
- no negative coverage test recognizes extensionless constructors such as `bin/pair-notify`.

There were no prior-round findings to dispose.

6. Architectural notes for upcoming work

- **ARCH-DRY: flag.** Artifact construction still has parallel authorities outside `artifactpath`.
- **ARCH-PURE: pass.** Pure migration and path derivation are separated cleanly from filesystem/process effects.
- **ARCH-PURPOSE: flag.** The single-authority goal is not enforced, and backward-compatible session identity is not preserved.
- **ARCH-MOCK: pass.** No new external dependency class is introduced; integration behavior uses existing seams, temporary stateful stores, and live shell/Neovim checks.

7. Plan revision recommendations

Append `## Revisions` entries covering:

- the compatibility/migration policy for relocating `session-names.jsonl`;
- the corrected Core concepts entity name and location after a full M5 table sweep;
- the actual constructor-classification rule, including extensionless sources and explicit legacy-path authority.

---

## Re-review — 2026-08-26T20:40:37-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 149 — couch: opaque tags and a human naming layer |
| repo | pair |
| issue file | workshop/issues/000149-couch-opaque-tags-and-a-human-naming-layer.md |
| boundary | milestone M5 |
| milestone | M5 |
| window | 6a714336ae3c8356ecdf2019c1ecc35b60719e81..8b8d521b4035fc36fa4322fa57a9cf1d4db711db |
| command | sdlc milestone-close --issue 149 --milestone M5 |
| reviewer | codex |
| timestamp | 2026-08-26T20:40:37-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M5 establishes a strong pure artifact-path layer and robust journaled migration, but it cannot cross the boundary. Two prior Critical classes remain incomplete: legacy-index compatibility bypasses Couch claim/quiescence, and selected companion paths are still derived outside `artifactpath`. Standalone registration also creates live incarnations without any terminal lifecycle, potentially reserving repository capacity forever.

```findings
dispose:
  - id: BR-31
    disposition: not-addressed
    note: |
      OSRuntime now merges legacy-global and scoped indexes, but ClaimNewThreadAddress and QuiesceThreadSession still read only the scoped index; their tests seed only that location. DecodeSessionNameIndex also validates JSON syntax but not required binding fields.
  - id: BR-32
    disposition: not-addressed
    note: |
      The named extensionless and image-done sites are pinned, but the constructor class remains open: opener derives scrollback events and changelog companions from other paths, while Neovim derives the changelog ready marker. The coverage guard cannot see these suffix derivations.
  - id: BR-33
    disposition: addressed
    note: |
      The complete M5 Core concepts inventory now names existing entities at their actual files, and the pure entities have direct IO-free tests.
  - id: BR-34
    disposition: not-addressed
    note: |
      The originally cited rows changed, but atlas/architecture.md still presents tag-derived PAIR_DATA_DIR/XDG formulas and calls the retired hardcoded file-family enumeration canonical.
findings:
  - id: new
    severity: Critical
    family: incarnation-quiescence-before-capacity-release
    title: |
      Standalone Pair incarnations have no terminal transition and occupy capacity forever
    detail: |
      This is the 3rd finding in family incarnation-quiescence-before-capacity-release. RegisterStandalonePair records the short-lived launcher as live, UpsertStandalonePair accumulates later launchers, and neither detach nor full-quit cleanup marks or removes the incarnation; admission intentionally counts every stored incarnation. State the class rule across create, attach, detach, restart, full quit, and external death, then test the production lifecycle through a stateful session fake so capacity is released only after whole-incarnation quiescence.
```

1. Strengths

- `MigrateLegacyRecord` is genuinely pure and preserves occupancy; the store migration uses the existing recoverable journal and pins corruption, interruption, and byte-stable reruns.
- `artifactpath.Paths` validates composite components and exports exact bindings from one value.
- The cross-scope integration exercises real shell and Neovim consumers plus both layouts.
- The global-only detached-session test genuinely pins the new `OSRuntime` compatibility merge.
- Full Go tests, focused race tests, diff checks, and runtime-bundle determinism pass.

2. Critical findings

- **BR-31 — not addressed:** [thread_claim.go:75](/Users/xianxu/workspace/pair/cmd/internal/launcher/thread_claim.go:75) and [thread_claim.go:246](/Users/xianxu/workspace/pair/cmd/internal/launcher/thread_claim.go:246) bypass the overlap reader and open only scoped `session-names.jsonl`. A legacy-only binding therefore neither blocks address reuse nor gets quiesced. Consolidate all readers behind one strict legacy-plus-scoped authority and add global-only claim/quiescence tests.

- **BR-32 — not addressed (ARCH-DRY, ARCH-PURPOSE):** [opener/run.go:167](/Users/xianxu/workspace/pair/cmd/internal/opener/run.go:167) derives events and several changelog companions; [init.lua:2992](/Users/xianxu/workspace/pair/nvim/init.lua:2992) independently reconstructs the ready marker. The negative guard at [coverage_test.go:143](/Users/xianxu/workspace/pair/cmd/internal/artifactpath/coverage_test.go:143) enumerates only the two previously named sites. Move the complete companion family into `artifactpath` and make the guard fail for every external suffix derivation.

- **New — standalone lifecycle:** [standalone.go:50](/Users/xianxu/workspace/pair/cmd/internal/couchcore/standalone.go:50) records every launcher as live, [standalone.go:71](/Users/xianxu/workspace/pair/cmd/internal/couchcore/standalone.go:71) accumulates later incarnations, and cleanup has no ThreadStore transition. Because [admission.go:112](/Users/xianxu/workspace/pair/cmd/internal/couchcore/admission.go:112) conservatively counts all incarnations, a completed direct Pair session can permanently consume capacity.

3. Important findings

- **BR-34 — not addressed:** [architecture.md:338](/Users/xianxu/workspace/pair/atlas/architecture.md:338), [architecture.md:500](/Users/xianxu/workspace/pair/atlas/architecture.md:500), and [architecture.md:672](/Users/xianxu/workspace/pair/atlas/architecture.md:672) retain derived formulas; [architecture.md:981](/Users/xianxu/workspace/pair/atlas/architecture.md:981) still declares the retired hardcoded enumeration canonical. Replace these with exact bindings and `artifactpath` authority. No README update is required because M5 adds no user-run command or flag.

4. Minor findings

None.

5. Test coverage notes

Verified:

- `git diff --check`
- `go test ./... -count=1`
- Focused `-race` tests for `artifactpath`, `launcher`, and `couchcore`
- `make runtimebundle-drift-check`

Missing load-bearing regressions:

- Legacy-global-only address claim and quiescence.
- Negative coverage for every companion-path derivation.
- Stateful standalone create → detach/re-attach → full-quit/external-death lifecycle and capacity release.

6. Architectural notes for upcoming work

- **ARCH-DRY: flag.** Durable-index reading and companion construction still have parallel authorities.
- **ARCH-PURE: pass.** Migration and path derivation are cleanly separated from IO.
- **ARCH-PURPOSE: flag.** The promised complete constructor and compatibility closures remain partial.
- **ARCH-MOCK: flag.** Standalone lifecycle lacks a stateful fake covering the production zellij/session boundary.

7. Plan revision recommendations

Append a `## Revisions` entry recording:

- One strict overlap reader/validator for every session-index consumer.
- The exhaustive companion-path inventory and enforcement strategy.
- The standalone incarnation lifecycle across create, detach, attach, restart, quit, and external death.
- The remaining atlas formula sweep.

---

## Re-review — 2026-08-26T21:10:37-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 149 — couch: opaque tags and a human naming layer |
| repo | pair |
| issue file | workshop/issues/000149-couch-opaque-tags-and-a-human-naming-layer.md |
| boundary | milestone M5 |
| milestone | M5 |
| window | 6a714336ae3c8356ecdf2019c1ecc35b60719e81..bbdb92a123e479d6657c6fdc449391b1bee8f458 |
| command | sdlc milestone-close --issue 149 --milestone M5 |
| reviewer | codex |
| timestamp | 2026-08-26T21:10:37-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M5’s migration, scoped-path model, and legacy-index overlap reader are well implemented and fully green under the repository suite. The boundary remains blocked because the artifact coverage test still cannot enforce its claimed single-constructor authority. Documentation also retains contradictory pre-M5 path and changelog behavior. BR-35 is withdrawn because verified incarnation quiescence is explicitly assigned to #152; conservative occupancy is the documented safety contract.

```findings
dispose:
  - id: BR-31
    disposition: addressed
    note: |
      One strict legacy-global-then-scoped reader now serves launch, address claim, and quiescence; global-only and malformed-row regressions fail if those callers bypass it.
  - id: BR-32
    disposition: not-addressed
    note: |
      The named consumers are fixed, but the guard still omits production command packages and permits any classified source to declare itself a Constructor, so it does not enforce artifactpath as the sole constructor.
  - id: BR-34
    disposition: not-addressed
    note: |
      The principal data-layout section was corrected, but architecture.md still publishes derived scrollback paths and obsolete session-keyed changelog-ready behavior that contradict the exact bindings.
  - id: BR-35
    disposition: withdrawn
    note: |
      The approved Spec and Plan explicitly assign verified whole-incarnation quiescence and capacity release to pair#152; retaining unknown occupancy in pair#149 is the required fail-closed behavior.
```

1. Strengths

- `MigrateLegacyRecord` is genuinely pure and preserves occupancy, while `ThreadStore.MigrateLegacyRecords` uses the existing journal authority for atomic, idempotent enrichment ([migration.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/migration.go:13)).
- The session-index relocation now has one strict overlap reader, with scoped rows overriding legacy rows and malformed present state failing closed ([session_index.go](/Users/xianxu/workspace/pair/cmd/internal/launcher/session_index.go:202)).
- Legacy-only claim and quiescence regressions directly exercise production callers ([thread_claim_test.go](/Users/xianxu/workspace/pair/cmd/internal/launcher/thread_claim_test.go:145)).
- Complete scrollback and changelog companion sets now derive inside `artifactpath`, and the cross-scope integration exercises Go, Bash, Neovim, and both layouts.
- The M5 Core concepts table now names existing entities at their actual paths; its PURE entities have direct IO-free tests.

2. Critical findings

- **BR-32 — not addressed (ARCH-DRY, ARCH-PURPOSE):** The coverage walks `cmd/internal` rather than all production `cmd` packages, excluding `cmd/couch`, `cmd/pair-go`, and `cmd/pair-launch-helper` ([coverage_test.go](/Users/xianxu/workspace/pair/cmd/internal/artifactpath/coverage_test.go:85)). More fundamentally, it accepts `Constructor` for any classified file without asserting that constructor authority is confined to `cmd/internal/artifactpath` ([coverage_test.go](/Users/xianxu/workspace/pair/cmd/internal/artifactpath/coverage_test.go:53)). A new external constructor can therefore pass by living in an omitted command package or declaring itself a constructor. This is the second finding in family `artifact-construction-single-authority`; do not add another named-site exception. Define the complete production-source set, reject `Constructor` outside `artifactpath`, and add mutation tests proving constructors in both `cmd/pair-go` and an already-scanned source make the suite fail.

3. Important findings

- **BR-34 — not addressed:** [architecture.md](/Users/xianxu/workspace/pair/atlas/architecture.md:615) still says scrollback inputs are derived as `$PAIR_DATA_DIR/scrollback-<tag>-<agent>…`. More seriously, its changelog section says Neovim re-resolves the session ID and polls a session-keyed `.ready` marker ([architecture.md](/Users/xianxu/workspace/pair/atlas/architecture.md:861)), while production now consumes the stable exact `$PAIR_CHANGELOG_READY_PATH`. This is the fourth finding in family `user-facing-policy-docs`; sweep the whole path-policy class rather than only these lines. README needs no M5 update because no user-run command, flag, or keybinding was introduced.

4. Minor findings

None.

5. Test coverage notes

Verified successfully:

- `git diff --check`
- `go vet ./...`
- `go test ./... -count=1`
- `make test`

BR-31’s regressions are load-bearing. BR-32’s named-site tests are useful, but no test currently proves the declared constructor authority is closed over every production source or rejects an externally classified constructor.

6. Architectural notes for upcoming work

- **ARCH-DRY: flag.** `artifactpath` contains the intended authority, but its enforcement admits parallel constructors.
- **ARCH-PURE: pass.** Path derivation and record migration are deterministic and separated cleanly from IO.
- **ARCH-PURPOSE: flag.** The milestone promises mechanically enforced sole construction; the current guard delivers only partial detection.
- **ARCH-MOCK: pass.** No new uncovered external dependency was introduced; relevant integrations use real Bash/Neovim consumers or existing stateful seams. Deferring lifecycle release to #152 is consistent with this principle.

7. Plan revision recommendations

Append a `## Revisions` entry recording:

- the corrected production-source closure, including top-level `cmd/*` packages;
- the invariant that only `artifactpath` may carry `Constructor`;
- mutation tests that demonstrate both omitted-root and misclassified-constructor failures;
- the remaining atlas sweep, including stable changelog-ready binding semantics.
