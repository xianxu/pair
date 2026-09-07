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

---

## Re-review — 2026-09-06T18:20:54-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 198 — couch: let the operator choose layout3 |
| repo | pair |
| issue file | workshop/issues/000198-couch-let-the-operator-choose-layout3.md |
| boundary | whole-issue close |
| milestone | — |
| window | 13e0f0f52e080495a0b88184d71bc5345614e826..5028f03852bf79cd9b0d7498f346c218438e8ff5 |
| command | sdlc close --issue 198 |
| reviewer | claude |
| timestamp | 2026-09-06T18:20:54-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The feature is delivered and correct: layout is a couch-global typed value that reaches argv only at a cold boundary, the witness rides the existing `StartRegistered` CAS, the guard is a pure function over rows startup already read, and the #179 cold/warm split is preserved (`warmresume_test.go` byte-identical) *and* strengthened. Round 2's `hostLayoutFor`, `type Layout = launcher.LayoutMode`, `*Layout`, README content and comment sweep are all genuinely in the tree, and I mutation-checked four of them red. What blocks SHIP is narrower and self-referential: **two of the nine tests round 2 added to close findings pass with their fix reverted** — `TestRefusalNeverNamesACommandThatWouldNotRun` stays green after I delete `hostLayoutFor`'s agreement loop (because the *other* fix, a total `Flag()`, makes `LayoutUnknown` render as `--layout2`), and `readme_test.go`'s new `"--layout3"`/`"--layout2"` entries stay green after I delete README.md:268-269 (because pair's own flag docs at README.md:13,18,441 already contain those substrings). Both fixes are right; neither is pinned, which is exactly the failure this window's own `lessons.md` entry was written about. Two Minors follow, both repeats of families already in play.

## 1. Strengths

- **`hostLayoutFor` (`cmd/internal/couchcore/layout.go:116-136`) is the right shape for BR-3** — it answers "is there one layout that can host every conflicting thread" rather than patching the string, and `KnownLayout` keeps `LayoutUnknown` out of any suggested command. The production behavior is correct; only its test is weak.
- **The alias consolidation is the better of the two options BR-5 offered.** `type Layout = launcher.LayoutMode` with `Flag()` living in `launcher` beside the parser, and `extractLayoutRequest` now switching on `Layout2.Flag()`/`Layout3.Flag()` (`launcher/args.go:156-158`), means the emitter and the parser cannot drift by construction. `TestCouchLayoutFlagsAreWhatPairParses` is not tautological despite the alias — I reverted `launchArgsAcceptLayout` and it went red on the real contract (`resume` accepting a layout flag at all).
- **`TestWarmReattachSendsNoLayoutEvenInLayout3` now covers the destructive mode from both angles.** I made the `StartRegistered` arm record the layout unconditionally; it failed with `warm reattach overwrote the witness: "layout3"`. BR-2 is properly closed.
- **`TestLayoutWitnessPersistsUnderItsOnDiskKey`** — I renamed the `json:"layout"` tag to `layout_mode` and it went red. BR-8 closed at the layer that matters.
- **`TestGuardAddsNoSessionEnumeration`** counts a declared IO budget instead of asserting prose, and its comment explains why it must measure the *refusing* run. This remains the model for ARCH-CONSTRAINTS.
- **The atlas section (`atlas/couch.md:901-932`)** states the cold-boundary rule, the blocking set, the parked-never-blocks remedy and the pre-#198 normalization, and records *why* the pin reversed rather than deleting it.

## 2. Critical findings

None. `gofmt -l` and `go vet ./cmd/...` are clean; the whole layout suite passes (33 tests). Remaining `./cmd/...` failures are the known `fork/exec … operation not permitted` / pty class in this environment, including `TestLayoutFlagReachesTheCouch` (fails at `pty.Open()`, `layout_cli_test.go:98`) — not this diff, but it means the CLI→domain end-to-end is unverified here and should be confirmed on the operator's machine.

## 3. Important findings

**Two guards added to close round-2 findings pass with their fix reverted.** Prevalence measured across all nine tests `70dd36c7` added: 2 of 9 vacuous.

- `cmd/internal/couchcore/layout_test.go:170` — I replaced `hostLayoutFor`'s body with `return conflicts[0].Layout, true` and the whole test stayed green, including the "blocking set that disagrees with itself" case. The reason is that its two assertions are *`ParseLayout` accepts every `--layout*` field* and *no `--unknown`* — and since BR-6 made `Flag()` total, `LayoutUnknown.Flag()` is now `--layout2`, so the degraded message is well-formed but **wrong**: it would tell an operator blocked by a mixed layout2/layout3 set to "park them first: couch --layout2", which is itself a startup that refuses. `TestRefusalKeepsTheConcreteRemedyWhenOneHostExists` pins the positive direction only. Fix: assert the negative — in the no-single-host cases require the generic wording (`whichever couch can host it`) and that the only `couch --layout*` in the message is `requested.Flag()`.
- `cmd/internal/couchcmd/readme_test.go:144-145` — I deleted README.md:268-269 (the two `couch --layout2|3` usage lines) and `TestREADMEDocumentsTheOperatorFacingSurface` stayed green: `--layout3` and `--layout2` already appear at README.md:13, 18-19 and 441 documenting **pair's** flags. Only `"refuses to start"` (:146) is load-bearing. Fix: use `"couch --layout2"` / `"couch --layout3"`, matching the command-prefixed convention its sibling `TestREADMEDocumentsOnlyThePublicProjection` already uses for `"couch --list"`.

The rule behind both: **a test written to close a review finding is not done until the fix is reverted and the test goes red** — and a whole-document substring guard must anchor on a string unique to the surface it guards, or a neighbouring section satisfies it for free. Applying the rule is the deliverable, not the two edits: revert-check every test this round adds before the next close attempt.

## 4. Minor findings

- **2nd in family `refusal-must-name-a-runnable-remedy`** — a blocking set consisting only of `LayoutUnknown` witnesses leaves *no* runnable remedy: `park` is `PresentationTUI, RowAction` (`ops.go:189`), so it needs a running couch, and every `couch --layoutN` refuses. The fallback at `layout.go:160` says "park every thread listed above -- from whichever couch can host it", but in that shape there is none. Don't fix the string; the covering rule is *a refusal must terminate in an action reachable with the tools the operator has, and when no in-tool action exists it must name the out-of-tool one* (kill the zellij session, or the record path to repair). Reachable only via a hand-edited or future-version witness, which is why it is Minor.
- **2nd in family `untagged-empty-sentinel`** — `ActionableThreadSummary.Layout`'s doc says the value is "already normalized … an unreadable one as `LayoutUnknown`", but the `input.Unreadable` branch (`actionableinventory.go:208-212`) constructs rows without touching `Layout`, so they carry a raw `Layout("")` — a fourth meaning of empty. Harmless today (`ThreadUnusable` never holds a session), but `#199`/`#200` read this struct. Covering rule: *a field documented as normalized must be normalized at every construction site of its struct*; prevalence here is 2 sites in `ProjectActionableThreads`, 1 unnormalized.
- `cmd/internal/couchcmd/cli.go:48` hand-writes `strings.HasPrefix(arg, "--layout")` as the filter and re-implements `launcher.extractLayoutRequest`'s loop (same `--` sentinel, same one-layout rule). The *value* is derived via `ParseLayout`, so this is not the BR-5 duplication returning — but a layout flag not spelled `--layoutN` would silently not be stripped.
- `layoutConflictRefusal` returns a nil `error` for an empty conflict slice; the one call site guards on `len > 0`, so the nil-error branch is unreachable — a `panic`/no-op contract would read more honestly than a nil error a caller could accidentally return.

## 5. Test coverage notes

- Revert-verified red (fix genuinely pinned): `TestWarmReattachSendsNoLayoutEvenInLayout3`, `TestLayoutWitnessPersistsUnderItsOnDiskKey`, `TestCouchLayoutFlagsAreWhatPairParses`. Direct-assertion, non-vacuous by inspection: `TestFlagIsTotalOverUnsetAndUnknownLayouts`, `TestKnownLayoutRejectsWhatCouchCannotLaunch`, `TestRefusalDoesNotDoubleTheProgramPrefix`, `TestRefusalKeepsTheConcreteRemedyWhenOneHostExists`.
- Untestable-here: the CLI→`Couch.Layout` end-to-end (`TestLayoutFlagReachesTheCouch`) needs a real pty. It is the only thing standing between "ParseCLI produced Layout3" and "the domain got it"; confirm it green on the operator's machine.
- The `extractLayoutFlag` shape matrix is good (flag before/after path, `--` dash-path, contradictory pair, all five non-launch forms). Not covered: a repeated identical flag (`--layout3 --layout3`), which is silently accepted — correct, but unpinned.
- The process-death-between-launch-and-CAS ordering is dispositioned as "ignore" in the plan and still has no test; acceptable, and the fake can seed it if `#199`/`#200` make it matter.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — pass.** The layout vocabulary now has one owner (`launcher`), one formatter, one parser, and the couch side aliases it. The only residue is the `"--layout"` prefix filter noted above.
- **ARCH-PURE — pass.** `ParseLayout`, `NormalizeLayout`, `KnownLayout`, `ResolveLayoutConflicts`, `holdsSession`, `hostLayoutFor`, `layoutConflictRefusal` are all pure and unit-tested with hand-built values and no fake; the guard consumes rows the caller already read.
- **ARCH-PURPOSE — pass on the sweep, flag on the class-vs-instance axis.** `grep -rn "pinned to layout2\|operator decision 2026-08-22" cmd/ atlas/` is empty; atlas, README, usage, `couch.go`'s rationale and the test premises all moved in this window. `workshop/projects/couch.md:233,881` still say `--layout2`, and that is *correct* — both are dated historical narrative the project datatype says to append to, not overwrite. Where it flags: round 2 answered BR-4 by extending an enumeration without checking the extension could fail, which is the instance rather than the class.
- **ARCH-MOCK — pass.** `pair` stays behind the `Runner` seam; `DetachedQueries()` extends the existing `FakeThreadArtifactCollisionChecker` rather than adding a parallel double; the cross-binary claim is now an in-module round trip through `launcher.ParseArgs` instead of a comment.
- **ARCH-CONSTRAINTS — pass.** Budget declared, basis stated, placement chosen *because* of the budget, budget counted in a test.
- **ARCH-SECURE — pass.** Both raw-string boundaries normalize; `ValidateThreadRecord` deliberately does not reject an unknown layout, so a hand-edited record surfaces as `LayoutUnknown` rather than disappearing into `ThreadUnusable`. The forward-only compatibility is now stated correctly at `thread.go:74-79`. The soft spot is that the visible failure still dead-ends (Minor 1).
- **ARCH-ORDER — pass on the enumeration.** One residual for `#199`/`#200`: `Couch.Layout`'s comment says "IMMUTABLE for the process lifetime", but it is an exported field assigned after `New` at `run.go:264`. The invariant is comment-enforced. The plan's reason (not changing the `Runtime` interface) is sound; if `#199`/`#200` add a second writer, promote it to a constructor parameter then.
- The destructive path is confirmed to be `ActionAttach`-only (`launcher/createflow.go:244-250`), so a cold resume of a parked thread into the other layout cannot reach `ConfirmLayoutChange`/`DeleteSession`. That is worth stating in the atlas alongside the cold/warm rule — right now the guarantee lives only in `createflow.go`.

## 7. Plan revision recommendations

The Core-concepts table matches the code on every row, and the existing `## Revisions` already records the alias, the `*Layout`, `hostLayoutFor` and the strictjson correction. Two small additions:

- **Add `KnownLayout` and `hostLayoutFor` to the pure-entities table** (`cmd/internal/couchcore/layout.go`, new). They arrived with the round-2 fixes and the Revisions mention `hostLayoutFor` in prose only, so the table is no longer the greppable enumeration it claims to be.
- **Task 3's body still reads "an older binary ignores it"** (plan line 533). The convention is to append rather than overwrite, and Revisions §4 does correct it — but a one-line pointer at the body site (`→ wrong; see Revisions §4`) stops a reader taking the paragraph at face value.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Corrected in plan Revisions section 4 and at thread.go:74-79; forward-only compatibility now stated, with TestPre198RecordDecodesThroughTheProductionPath pinning the old-record direction.
  - id: BR-2
    disposition: addressed
    note: |
      Verified by reverting: recording the layout unconditionally in the StartRegistered arm turns TestWarmReattachSendsNoLayoutEvenInLayout3 red on the witness assertion.
  - id: BR-3
    disposition: not-addressed
    note: |
      Code fix is correct and present, but no test fails without it -- deleting hostLayoutFor's agreement loop leaves TestRefusalNeverNamesACommandThatWouldNotRun green, because a total Flag() renders LayoutUnknown as --layout2.
  - id: BR-4
    disposition: not-addressed
    note: |
      README content added, but the guard's new want strings are satisfied by pair's own flag docs at README.md:13,18,441 -- deleting README.md:268-269 leaves TestREADMEDocumentsTheOperatorFacingSurface green.
  - id: BR-5
    disposition: addressed
    note: |
      Type aliased to launcher.LayoutMode, Flag() moved beside the parser, and the conformance test goes red when launchArgsAcceptLayout stops admitting resume.
  - id: BR-6
    disposition: addressed
    note: |
      StartEvent.Layout is a *Layout and Flag() is total; TestFlagIsTotalOverUnsetAndUnknownLayouts plus the warm-reattach witness assertion pin both halves.
  - id: BR-7
    disposition: addressed
    note: |
      couch.go:424, launch_existing.go:35,46 and detach.go:27 all now name the layout flag generically rather than --layout2.
  - id: BR-8
    disposition: addressed
    note: |
      TestLayoutWitnessPersistsUnderItsOnDiskKey goes red when the json tag is renamed to layout_mode.
  - id: BR-9
    disposition: addressed
    note: |
      Message now leads with "cannot start in ..."; TestRefusalDoesNotDoubleTheProgramPrefix pins it.
  - id: BR-10
    disposition: addressed
    note: |
      Plan Revisions records the test-file grouping decision and the PairTag/ThreadTag fixture naming.
findings:
  - id: new
    severity: Important
    family: guard-passes-without-the-fix
    title: |
      Two of the nine tests added to close round 2's findings pass with their fix reverted
    detail: |
      Measured prevalence: 2 of 9 tests added by 70dd36c7. Deleting hostLayoutFor's
      agreement loop (layout.go:116-136) leaves layout_test.go:170 green, because
      BR-6's total Flag() renders LayoutUnknown as --layout2, so the message is
      well-formed but tells an operator blocked by a mixed set to run a couch that
      would itself refuse. Deleting README.md:268-269 leaves readme_test.go:144-145
      green, because pair's own flag docs at README.md:13,18,441 already contain
      those substrings. The rule, not the two edits, is the deliverable: a test
      written to close a finding is not done until the fix is reverted and it goes
      red, and a whole-document substring guard must anchor on a string unique to
      the surface it guards -- README's sibling test already uses the
      command-prefixed form ("couch --list"). Concretely: assert the NEGATIVE
      direction of the refusal (no concrete `couch --layoutN` host remedy when no
      single host exists, only the requested layout), and switch the README want
      strings to "couch --layout2"/"couch --layout3".
  - id: new
    severity: Minor
    family: refusal-must-name-a-runnable-remedy
    title: |
      A blocking set that is entirely LayoutUnknown leaves the operator with no reachable remedy at all
    detail: |
      This is the 2nd finding in family refusal-must-name-a-runnable-remedy. Do not
      fix this instance -- the covering rule is that a refusal must terminate in an
      action reachable with the tools the operator has, and when no in-tool action
      exists it must name the out-of-tool one. `park` is PresentationTUI/RowAction
      (ops.go:189), so it needs a running couch, and with a lone LayoutUnknown
      session-holder every `couch --layoutN` refuses. The fallback at layout.go:160
      says "from whichever couch can host it" when there is none; it should name the
      zellij session to kill or the record to repair. Reachable only via a
      hand-edited or newer-version witness, hence Minor.
  - id: new
    severity: Minor
    family: untagged-empty-sentinel
    title: |
      ActionableThreadSummary.Layout is documented as normalized but the Unreadable branch never normalizes
    detail: |
      This is the 2nd finding in family untagged-empty-sentinel. Do not fix this
      instance -- the covering rule is that a field documented as normalized must be
      normalized at every construction site of its struct. actionableinventory.go:208-212
      builds the input.Unreadable rows without touching Layout, so they carry a raw
      Layout("") while the field comment (actionableinventory.go:121-125) promises
      Layout2 or LayoutUnknown. Measured prevalence: 2 construction sites in
      ProjectActionableThreads, 1 unnormalized. Harmless today because ThreadUnusable
      never holds a session, but pair#199/#200 consume this struct.
```

---

## Re-review — 2026-09-06T18:34:09-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 198 — couch: let the operator choose layout3 |
| repo | pair |
| issue file | workshop/issues/000198-couch-let-the-operator-choose-layout3.md |
| boundary | whole-issue close |
| milestone | — |
| window | 13e0f0f52e080495a0b88184d71bc5345614e826..fba6b350ab543badc15dfc38e5f0f026cf824192 |
| command | sdlc close --issue 198 |
| reviewer | claude |
| timestamp | 2026-09-06T18:34:09-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Round 2's three Importants are all genuinely closed, and I verified each by reverting the fix in a scratch tree at HEAD rather than reading the commit message: deleting the couch block from `README.md` turns `TestREADMEDocumentsTheOperatorFacingSurface` red on all three new want-strings, and collapsing `layoutRemedy`'s two early-return branches turns all three refusal tests red with the exact defect BR-3/BR-12 named printed in the failure output. That is the property BR-11 was about, and it now holds. The feature itself is sound: the destructive `pair` path (`createflow.go:250` → `ConfirmLayoutChange` → `DeleteSession`) is unreachable from couch by two independent mechanisms — a warm reattach sends no flag at all, so `LayoutResolution.Conflict` cannot be set (it is only set in the `Explicit` arm), and a cold resume carries `ResumeRequired`, which `createflow.go:238` refuses before the `ActionAttach` conflict branch is reached. 67 layout-related tests pass; `gofmt` and `go vet` are clean; the remaining `couchcore`/`couchcmd`/`cmd/couch` failures are the known `ptychild: operation not permitted` environmental class, which unfortunately includes this diff's own `TestLayoutFlagReachesTheCouch`. What keeps this off SHIP is three Minors: BR-13 was not disposed at all, and two new ones (refusal text asserted at only one cardinality; the plan still describes a `hostLayoutFor` the tree no longer has). None blocks the gate.

## 1. Strengths

- **The mutation discipline was actually applied this round, and it worked.** `layout_test.go:167-172` documents *why* the assertion is negative — "these assert the NEGATIVE direction … which is the half that goes red when the remedy logic is removed" — and it does. The three `lessons.md` entries on the three ways a mutation check lies (cached results, a non-compiling mutation, mutating the wrong thing) are the class-level deliverable BR-11 asked for, not the two edits.
- **`readme_test.go:141-150` fixed the guard, not just the document.** The comment states the rule ("COMMAND-PREFIXED … a bare `--layout3` is satisfied by those and this guard would pass with the couch block deleted") and the assertion follows the sibling convention. This is the finding answered at the class.
- **`layoutRemedy` (`layout.go:112-155`) answers three cases separately with a stated reason for not collapsing them.** The unreadable branch leaves the tool entirely (`couch --show <tag>` + `zellij kill-session`) because `park` is a TUI row action that needs a running couch — the reachability analysis BR-12 asked for, encoded in the code's own doc comment.
- **`TestGuardAddsNoSessionEnumeration` (`layout_guard_test.go:110-122`) counts an IO budget rather than asserting a result**, and explains why it must measure the *refusing* run. A declared envelope that nothing counts is a rule that cannot fail.
- **`TestCouchLayoutFlagsAreWhatPairParses` (`layout_test.go:126-138`) turned a prose "measured, not reasoned" claim into a round trip through `launcher.ParseArgs`.** The whole feature rests on that contract and it was previously evidenced only by a comment.

## 2. Critical findings

None.

## 3. Important findings

None.

## 4. Minor findings

- `cmd/internal/couchcore/layout.go:172-176` — the refusal reads "1 thread already **hold** a session"; `noun` is pluralized but the verb is not, and the unreadable remedy says "**this thread's** layout cannot be read" while naming only `conflicts[0]`'s tag when the list above may show several. Every refusal test happens to use a cardinality that hides both.
- `workshop/plans/000198-…-plan.md:780` describes `hostLayoutFor`, which `d6a32638` deleted; the Core-concepts pure-entities table (`:56-62`) still omits `KnownLayout`, `layoutRemedy`, `layoutConflictRefusal`, and `holdsSession`.
- `holdsSession` lives in `layout.go:84` while its siblings `Live()`/`Detached()`/`Resumable()` live in `actionableinventory.go:128-139`. Justified by the doc comment; noting only for discoverability.
- `README.md:274-276` says couch "refuses to start when a thread is already running or detached" — `ThreadBusy` also blocks (`layout.go:84-86`).

## 5. Test coverage notes

- **Mutation-verified, in a scratch tree at `fba6b350`:** deleting `README.md:268-269` + the paragraph → 3 assertions red; replacing `layoutRemedy`'s unreadable and `!single` branches with a fallthrough → `TestRefusalOffersNoHostRemedyWhenNoCouchCanHostThemAll/{unreadable_mixed_in,blocking_set_that_disagrees_with_itself}`, `TestRefusalNamesAnOutOfToolRemedyWhenNoCouchCanStart`, and `TestRefusalExplainsThePerThreadRouteWhenLayoutsDisagree` all red, printing exactly the `park them first: couch --layout2` advice BR-3 described.
- `TestLayoutFlagReachesTheCouch` — the only end-to-end pin that the CLI flag reaches `Couch.Layout` — **could not be run here** (`pty.Open`: operation not permitted). It is new in this diff. The issue's Log records `go test ./cmd/...` green on the implementor's machine and the operator ran the full manual cycle; worth one confirming local run before merge since it is the seam every `ParseCLI` test above it depends on.
- `TestLayoutWitnessPersistsUnderItsOnDiskKey` and `TestPre198RecordDecodesThroughTheProductionPath` close BR-1/BR-8 at the byte level, including the negative direction (a pre-#198 record emits no `layout` key at all, which is the contract strictjson enforces).
- Still uncovered, and correctly dispositioned as "ignore, self-correcting" in the plan: process death between launch and the witness CAS. Its *consequence* (a mixed blocking set) is now rendered by a test even though the event that produces it is not simulated.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — pass.** BR-5's alias landed: `type Layout = launcher.LayoutMode`, `Flag()` beside the parser, and `extractLayoutRequest` now parses against `Flag()` (`args.go:156-158`). `grep '"--layout2"\|"--layout3"'` over non-test `cmd/` returns exactly two hits, both inside `Flag()`.
- **ARCH-PURE — pass.** `ParseLayout`, `NormalizeLayout`, `KnownLayout`, `ResolveLayoutConflicts`, `holdsSession`, `layoutRemedy`, `layoutConflictRefusal` are pure and unit-tested with hand-built values and no fake; the guard consumes rows `StartInteractive` already read.
- **ARCH-PURPOSE — pass, with one residual.** Shadow sweep is clean: `pinned to layout2` / `operator decision 2026-08-22` appear nowhere in `cmd/`, `atlas/`, or `README.md`; the atlas gained a "Layout is couch-wide and never mixed" section recording the reversed pin's reasoning rather than deleting it. The residual is BR-13's class rule (below), which was never stated.
- **ARCH-MOCK — pass.** `pair` stays behind the `Runner` seam; the detached-query counter extends `FakeThreadArtifactCollisionChecker` rather than adding a parallel double; the in-module conformance test replaces the hand measurement. No fake models pair's durable workbench-layout record, but I traced the only branch that would need one (`createflow.go:250`) and it is unreachable from couch — noting, not flagging.
- **ARCH-CONSTRAINTS — pass.** Budget declared (zero added enumeration), placement chosen because of it, budget counted in a test. This remains the model for the envelope.
- **ARCH-SECURE — pass.** Both raw-string boundaries normalize with distinct failure modes (projection → `LayoutUnknown`, CLI → error/exit 2); `LayoutUnknown` refuses visibly rather than fabricating a value the guard trusts; forward-only compatibility documented at `thread.go:74-83` and pinned.
- **ARCH-ORDER — pass.** `Couch.Layout` immutable after construction collapses the ordering space; the witness rides the existing `StartRegistered` CAS so there is no second write and no revision to reconcile; `StartEvent.Layout *Layout` gives "no layout chosen" its own value. `Layout("")` on the Unreadable projection row is the one untagged-empty state left — see BR-13.
- **For #199/#200:** they consume `ActionableThreadSummary`. Landing BR-13's rule (normalize at every construction site) before those two exist is cheaper than after.

## 7. Plan revision recommendations

One entry, covering the class rather than the instance (see the second finding):

- **Re-verify the plan against the tree at the round's final commit.** Revisions §3 ("The refusal needed a `hostLayoutFor`") describes `70dd36c7`'s intermediate shape; `d6a32638` replaced `hostLayoutFor` with `layoutRemedy` and the message no longer "drops the concrete remedy" — it names three. Rewrite §3 to the as-built three-case remedy, and add `KnownLayout`, `layoutRemedy`, `layoutConflictRefusal`, `holdsSession` to the pure-entities table so it is again the greppable enumeration it claims to be. The checkable form: `grep '^func ' cmd/internal/couchcore/layout.go` must be a subset of the table, and every backticked identifier in Revisions must resolve in the tree.

```findings
dispose:
  - id: BR-3
    disposition: addressed
    note: |
      Mutation-verified: collapsing layoutRemedy's two early returns reprints the exact `park them first: couch --layout2` advice and turns three tests red.
  - id: BR-4
    disposition: addressed
    note: |
      Mutation-verified: deleting the README couch block turns all three new want-strings red; they are command-prefixed, so pair's own flag docs no longer satisfy them.
  - id: BR-11
    disposition: addressed
    note: |
      Both previously-vacuous tests now go red under revert, and the rule landed as three lessons.md entries on how a mutation check lies.
  - id: BR-12
    disposition: addressed
    note: |
      layoutRemedy's unreadable branch names the out-of-tool remedy (couch --show + zellij kill-session); TestRefusalNamesAnOutOfToolRemedyWhenNoCouchCanStart goes red without it.
  - id: BR-13
    disposition: not-addressed
    note: |
      No code change, no rule statement, no plan entry -- actionableinventory.go:209-211 still builds the Unreadable rows with a raw Layout("").
findings:
  - id: new
    severity: Minor
    family: count-varying-text-unasserted
    title: |
      The refusal's count-varying text is only ever rendered at one cardinality
    detail: |
      layout.go:172-176 pluralizes the noun but not the verb, so a single conflict reads
      "1 thread already hold a session in another layout" -- I saw this rendered verbatim
      while mutation-checking. layout.go:130-135 has the mirrored gap: the unreadable
      remedy says "this thread's layout cannot be read" and names only conflicts[0]'s tag,
      while the list printed above it can show several unreadable rows. The rule is that
      operator text whose wording depends on a count must be asserted at each cardinality
      it can render, not only at the one the fixture happens to use; every refusal test
      currently picks a cardinality that hides both. Not a reachability defect -- the
      remedy still terminates, iterating once per unreadable thread -- which is why this
      is not another instance of refusal-must-name-a-runnable-remedy.
  - id: new
    severity: Minor
    family: plan-names-files-that-do-not-exist
    title: |
      Plan Revisions still describes hostLayoutFor, which this window deleted, and the entity table omits four functions
    detail: |
      This is the 2nd finding in family plan-names-files-that-do-not-exist. Do NOT fix this
      instance alone -- the covering rule is that the plan's Core-concepts table and
      Revisions must be re-verified against the tree at the FINAL commit of the round they
      describe, not written from the mid-round state. Measured prevalence: 2 rounds. Round 3
      named thread_test.go and startup_test.go, which never existed; round 4's Revisions
      section 3 (plan.md:780) describes `hostLayoutFor` and "the message now drops the
      concrete remedy", both of which d6a32638 replaced with layoutRemedy's three-case
      answer, and the pure-entities table (plan.md:56-62) omits KnownLayout, layoutRemedy,
      layoutConflictRefusal and holdsSession. The enumeration the rule implies is checkable:
      `grep '^func ' cmd/internal/couchcore/layout.go` must be a subset of the table, and
      every backticked identifier in Revisions must resolve in the tree.
```
