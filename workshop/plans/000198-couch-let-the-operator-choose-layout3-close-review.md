# Boundary Review — pair#198 (whole-issue close)

| field | value |
|-------|-------|
| issue | 198 — couch: let the operator choose layout3 |
| repo | pair |
| issue file | workshop/issues/000198-couch-let-the-operator-choose-layout3.md |
| boundary | whole-issue close |
| milestone | — |
| window | 13e0f0f52e080495a0b88184d71bc5345614e826..aebf40dd858736947abc8205b6e8591402cb95e7 |
| command | sdlc close --issue 198 |
| reviewer | claude |
| timestamp | 2026-09-06T17:59:46-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The feature is delivered and the hard parts are right: layout is a typed value with a single argv formatter, the #179 cold/warm split is preserved *and* strengthened (`warmresume_test.go` is byte-identical, and `run_test.go` was tightened from `--layout2` to any `--layout`), the witness rides the existing `StartRegistered` CAS so there is no second write to reconcile, the projection normalizes the untrusted persisted field at exactly one point, and the ARCH-CONSTRAINTS "no extra enumeration" budget is pinned by an actual call counter rather than prose. I ran the layout suite: 24/24 pass, `gofmt`/`go vet` clean. The remaining pty failures in `couchcore`/`couchcmd` are the known environmental `ptychild: operation not permitted` class, not this diff. What keeps it from SHIP is a cluster of Importants: the refusal message can print an unrunnable remedy (`couch --unknown`) on exactly the paths the design justified as "visible failure"; README — which has an existing guard test enumerating operator-facing flags — was not updated for the new `--layout2|--layout3` surface; and `couchcore.Layout` re-declares a layout vocabulary `launcher` already owns, with no test pinning that couch's emitted flag is what pair actually parses.

## 1. Strengths

- **The cold/warm split is pinned twice, deliberately.** `layout_launch_test.go:58-96` explains *why* it needs its own test alongside `warmresume_test.go` (that one runs a default couch, so it would still pass if the warm branch started echoing `c.Layout`), and it asserts both the argv *and* that the witness is not overwritten. That is the one destructive failure mode, covered from the angle that actually moved.
- **`TestGuardAddsNoSessionEnumeration` (`layout_guard_test.go:110-122`) pins an IO budget, not a result** — and the comment explains why it measures the *refusing* run specifically (an admitted run legitimately re-asks). A declared budget nothing counts is a rule that cannot fail; this one can.
- **PQ-1's fix is genuinely reachable and genuinely pinned.** `actionableinventory.go:225` normalizes, and `TestProjectionNormalizesAbsentLayoutAndDoesNotBlockDefaultStartup` goes red if it carries `record.Layout` through raw. This is the finding→fix→failing-test chain the claimed-fixes rule asks for.
- **The six-state disposition is a table in code and a table in tests** (`layout.go:63-75`, `layout_test.go:63-78`), including the `Resumable()` anti-test. Adding a seventh `ActionableThreadState` fails the suite instead of silently defaulting.
- **`atlas/couch.md` was swept properly** — all four pinned sites plus a new section stating the cold-boundary rule, the blocking set, and the pre-#198 normalization, with the reversed pin's reasoning recorded rather than deleted.

## 2. Critical findings

None.

## 3. Important findings

**`cmd/internal/couchcore/layout.go:112-127` — the refusal can print a command that does not exist.** `other` starts at `conflicts[0].Layout` and degrades to `LayoutUnknown` when the conflicts disagree; it is then interpolated into `park them first:  couch --%s`, yielding `couch --unknown`. Two reachable paths: a single conflict whose witness is `LayoutUnknown` (a hand-edited or newer-version record — the case ARCH-SECURE's `LayoutUnknown` exists to surface *visibly*), and a mixed blocking set (reachable via the process-death-between-launch-and-CAS event the plan's ARCH-ORDER section documents and disposes as "ignore"). In the mixed case there is no single correct layout to suggest at all, so the message's premise fails, not just its formatting. Fix sketch: compute `other` as an `(Layout, bool)`; when it is not a single known layout, drop the `couch --<other>` line and say "park every thread listed above from whichever couch can host it". Add a test asserting the refusal never contains `--unknown` and never names a flag `ParseLayout` would reject.

**`README.md:267-270` — README update missing for the `--layout2|--layout3` flag.** `usage()` gained the flag (`run.go:672,681`) and `TestUsageMentionsTheLayoutFlag` pins that, but the README couch usage block still reads `couch [<repo>]` with no layout line. The repo already owns the enumeration: `cmd/internal/couchcmd/readme_test.go:129` (`TestREADMEDocumentsTheOperatorFacingSurface`) exists precisely for "flags that change what couch does with the operator's terminal", and a flag that adds a third pane is the paradigm case. It was not extended, so the guard passed vacuously. Fix: add the flag to the README block and add `"--layout3"` to that test's `want` list, so the next such flag cannot slip past either (ARCH-PURPOSE — the class is "operator-facing flag", and the enumeration already exists).

**`cmd/internal/couchcore/layout.go:11-54` — ARCH-DRY: the layout vocabulary now exists twice, in packages that already depend on one another.** `launcher/layout.go:8-11` defines `LayoutMode` with the identical `layout2`/`layout3` constants, `ParseLayoutMode`, and `ResolveLayout`'s "default is Layout2" rule; `launcher/args.go:156-158` owns the set of accepted flag spellings. `couchcore` imports `launcher` in ~8 non-test files already, so this is not an import-cycle constraint. The consequence is not hypothetical: couch's entire feature rests on `pair` parsing the flag it emits, and the only in-tree evidence for that is a prose comment at `couch.go:450-458` saying it was measured by hand. Fix sketch (cheapest first): add a conformance test in `couchcore` asserting `launcher.ParseArgs([]string{"resume", "tag", Layout3.Flag()})` yields `LayoutRequest{Mode: launcher.Layout3, Explicit: true}` — that turns the measured claim into a check. Better: `type Layout = launcher.LayoutMode` and derive `Flag()` from it, so a third layout is one enum value in one place (ARCH-MOCK's live-conformance clause, satisfied in-module for free).

**`cmd/internal/couchcore/layout.go:54`, `starttransaction.go:36-39`, `launch_existing.go:124-131` — the empty `Layout` carries three different meanings, and one of them formats as `--`.** `ThreadRecord.Layout == ""` means "pre-#198, therefore layout2"; `StartEvent.Layout == ""` means "warm, do not record"; `Couch.Layout == ""` means "constructed without `New`", and `Flag()` on it emits a bare `--` into argv. The plan asserts the third is unreachable because `New` defaults it, but `Couch` is built by struct literal in ~10 test files (`park_test.go:273`, `actionableinventory_test.go:197`, `archive_test.go:137`, …), so the invariant is comment-enforced on an exported field, not type-enforced. `#199`/`#200` will consume this surface. Fix sketch: make `Flag()` total — `if l == Layout3 { return "--layout3" }; return "--layout2"` — or return `(string, bool)`; and give `StartEvent` a `*Layout` (or a `RecordLayout bool`) so "warm chose no layout" is a distinct value rather than a third overload of `""` (ARCH-ORDER: the legal states should not be an untagged empty string).

## 4. Minor findings

- `cmd/internal/couchcore/couch.go:424` — the comment block's opening line still reads "`pair resume <tag> --layout2` rather than a bare `pair`", directly contradicting the new rationale two paragraphs below it.
- `cmd/internal/couchcore/detach.go:27` — "Reattaching is a fresh `pair resume <tag> --layout2`" describes the *warm* path as sending a layout flag, which is the inverse of the #179 invariant. Pre-existing, but this diff is the sweep that should have caught it.
- `cmd/internal/couchcore/launch_existing.go:35,46` — "`--layout2` is dropped" now means "the layout flag is dropped"; the literal reads as a surviving pin.
- The refusal renders as `couch: couch cannot start in layout3: …` — `renderError` (`run.go:665`) already prefixes `couch: `. `startupResumeRefusal` avoids this by leading with `%w`.
- `layoutConflictRefusal` indexes `conflicts[0]` with no length check; safe at its one call site, but it is a package-level function.

## 5. Test coverage notes

- **The gap that shipped the Important above:** every refusal-message test uses a single known-layout conflict (`layout_guard_test.go:32-58`). There is no test of the message for a `LayoutUnknown` conflict or a mixed conflict set, which is exactly why `couch --unknown` is reachable. `TestProjectionMarksUnreadableLayoutUnknown` covers the *classification* but stops at `ResolveLayoutConflicts`, never rendering.
- **Persistence is tested at the Go-struct level, not the JSON level.** The plan's Task 3 test unmarshalled a literal `{"schema_version":2,…}`; what landed (`layout_projection_test.go:74-95`) round-trips through `toPersistedThreadRecord`/`fromPersistedThreadRecord`. Nothing pins the on-disk key `layout`, so renaming the tag in `threadrecord/record.go:81` would pass the suite and silently drop every witness. One `json.Unmarshal` of a raw pre-#198 record closes it.
- **No cross-binary conformance test** — see the ARCH-DRY finding.
- The process-death-between-launch-and-CAS ordering is dispositioned in the plan ("ignore, self-correcting, fails conservative") but has no test. Acceptable given the disposition, worth noting since the fake could seed it.
- I could not run the pty-dependent tests (`TestLayoutFlagReachesTheCouch`, the three `TestInteractiveLaunch*`) — `ptychild: operation not permitted` in this environment. Those should be confirmed green on the operator's machine before close.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — flag** (finding 3). Within `couchcore` the consolidation is clean: one `Flag()`, one normalization point. Across the module it is a second vocabulary.
- **ARCH-PURE — pass.** `ParseLayout`, `NormalizeLayout`, `ResolveLayoutConflicts`, `holdsSession`, and `layoutConflictRefusal` are all pure and unit-tested with hand-built summaries; the guard consumes rows the caller already read rather than reaching for IO.
- **ARCH-PURPOSE — flag** (README, above). The shadow-sweep otherwise holds: the atlas sites, the `couch.go` rationale, and the test premises were all corrected in this window rather than deferred, and `grep -rn "pinned to layout2\|operator decision 2026-08-22" cmd/ atlas/ README.md` returns nothing.
- **ARCH-MOCK — pass with a gap.** `pair` sits behind the `Runner` seam with a fake recording argv, and the new `DetachedQueries()` counter extends the existing `FakeThreadArtifactCollisionChecker` rather than adding a parallel double. The gap is the missing conformance check on the flag the fake never validates.
- **ARCH-CONSTRAINTS — pass.** Budget declared (zero added enumeration), basis stated (measured), placement chosen *because* of the budget, and the budget is counted in a test. This is the model for how the envelope should be handled.
- **ARCH-SECURE — pass.** Both raw-string boundaries normalize; `LayoutUnknown` refuses rather than fabricating a value the guard would trust; no schema bump, so an older binary still loads the store. The one soft spot is that the visible failure it produces is currently unrunnable advice (finding 1).
- **ARCH-ORDER — pass on the enumeration, flag on representation.** Immutability of `Couch.Layout` genuinely collapses the ordering problem, and the events the caller cannot block are named and dispositioned individually rather than swept. The representation of "no layout chosen" is the weak point (finding 4).
- **For `#199`/`#200`:** they consume this surface. Landing the `Flag()` totality fix and the `launcher.LayoutMode` consolidation *before* the tab-strip work is cheaper than after two more consumers exist.

## 7. Plan revision recommendations

The plan still matches the code on every Core-concepts row — each entity exists at its stated path, PURE entities test without IO, INTEGRATION points show the expected modification. Two small drifts worth a `## Revisions` entry so the plan stops describing files that do not exist:

- **Test file locations.** Task 3 names `cmd/internal/couchcore/thread_test.go`; Task 5 names `startup_test.go`. The tests landed in `layout_projection_test.go` and `layout_guard_test.go` (a better grouping — record the decision, not just the move).
- **Task 3's old-record test changed shape.** The plan's `TestThreadRecordWithoutLayoutRoundTrips` unmarshalled raw JSON; what shipped round-trips Go structs. Either record the substitution or, better, add the JSON-level test and leave the plan as written (see §5).
- Cosmetic: the plan's fixtures use `PairTag`; the code uses `ThreadTag`.

```findings
findings:
  - id: new
    severity: Important
    family: refusal-must-name-a-runnable-remedy
    title: |
      layoutConflictRefusal prints `couch --unknown` when conflicts disagree or carry LayoutUnknown
    detail: |
      cmd/internal/couchcore/layout.go:112-127 degrades `other` to LayoutUnknown and
      interpolates it into "park them first: couch --%s". Reachable via a hand-edited or
      newer-version witness (the case LayoutUnknown exists to surface visibly) and via a
      mixed blocking set after the process-death-mid-CAS event the plan documents. In the
      mixed case there is no single correct layout to suggest, so the message's premise
      fails, not just its wording. No test renders the refusal for either case.
  - id: new
    severity: Important
    family: operator-surface-undocumented
    title: |
      README update missing for the new `--layout2|--layout3` flag
    detail: |
      usage() gained the flag (run.go:672,681) and a test pins that, but README.md:267-270
      still shows `couch [<repo>]` with no layout line. The repo already owns the
      enumeration -- readme_test.go:129 TestREADMEDocumentsTheOperatorFacingSurface covers
      exactly "flags that change what couch does with the operator's terminal" -- and it was
      not extended, so the guard passed vacuously. Fix both the README block and that test's
      want list (ARCH-PURPOSE: the class, not the instance).
  - id: new
    severity: Important
    family: vocabulary-duplicated-across-packages
    title: |
      couchcore.Layout re-declares launcher.LayoutMode; nothing pins that pair parses the emitted flag
    detail: |
      launcher/layout.go:8-11 already owns layout2/layout3, ParseLayoutMode, and the
      "default is Layout2" rule; launcher/args.go:156-158 owns the accepted flag spellings.
      couchcore imports launcher in ~8 non-test files, so there is no cycle constraint.
      The cross-binary contract couch now depends on is evidenced only by a prose comment at
      couch.go:450-458 saying it was measured by hand. Minimum: a conformance test asserting
      launcher.ParseArgs("resume","tag",Layout3.Flag()) yields {Mode: launcher.Layout3,
      Explicit: true}. Better: alias the type so a third layout is one edit in one place
      (ARCH-DRY, ARCH-MOCK).
  - id: new
    severity: Important
    family: untagged-empty-sentinel
    title: |
      Empty Layout means three different things, and one of them formats as a bare `--`
    detail: |
      ThreadRecord.Layout=="" means pre-#198/layout2; StartEvent.Layout=="" means
      warm/do-not-record (starttransaction.go:36-39, launch_existing.go:124-131);
      Couch.Layout=="" means constructed without New, and Flag() (layout.go:54) emits a bare
      "--" into argv. The plan calls the third unreachable, but Couch is built by struct
      literal in ~10 test files, so the invariant is comment-enforced on an exported field.
      #199/#200 will consume this surface. Make Flag() total, and give StartEvent a *Layout
      or an explicit RecordLayout bool (ARCH-ORDER).
  - id: new
    severity: Minor
    family: stale-layout-pin-comment
    title: |
      Three comments still name `--layout2` as the flag couch sends or drops
    detail: |
      couch.go:424 opens the rationale block with "`pair resume <tag> --layout2` rather than
      a bare `pair`", contradicting the new rule below it. launch_existing.go:35,46 say
      "`--layout2` is dropped" where they now mean the layout flag. detach.go:27 describes a
      warm reattach as "a fresh `pair resume <tag> --layout2`", which is the inverse of the
      #179 invariant (pre-existing, but this was the sweep).
  - id: new
    severity: Minor
    family: on-disk-format-untested
    title: |
      No JSON-level test pins the persisted `layout` key
    detail: |
      layout_projection_test.go:74-95 round-trips Go structs through
      toPersistedThreadRecord/fromPersistedThreadRecord. The plan's Task 3 test unmarshalled
      a raw `{"schema_version":2,...}` record instead. As written, renaming the json tag in
      threadrecord/record.go:81 would keep the suite green while silently dropping every
      witness on disk.
  - id: new
    severity: Minor
    family: doubled-error-prefix
    title: |
      The refusal renders as "couch: couch cannot start in layout3"
    detail: |
      renderError (run.go:665) already prefixes "couch: ". startupResumeRefusal avoids the
      stutter by leading with %w; the new message leads with the literal word "couch".
  - id: new
    severity: Minor
    family: plan-names-files-that-do-not-exist
    title: |
      Plan cites test files the work did not create
    detail: |
      Task 3 names thread_test.go and Task 5 names startup_test.go; the tests landed in
      layout_projection_test.go and layout_guard_test.go (a better grouping). Task 2's
      fixtures use PairTag where the code uses ThreadTag. Worth a "## Revisions" entry so
      the plan stops describing paths the tree does not have.
```
