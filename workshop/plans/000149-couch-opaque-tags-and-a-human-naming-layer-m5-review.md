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
