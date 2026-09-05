# Boundary Review — pair#170 (milestone M1)

| field | value |
|-------|-------|
| issue | 170 — Rescope couch to couch-lite |
| repo | pair |
| issue file | workshop/issues/000170-rescope-couch-to-couch-lite.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 88fe1de011b4c6be58e5a8b20eed89dfa4000f5d..1615695eb1893e2d20c1201f0bf4ccb8369c4505 |
| command | sdlc milestone-close --issue 170 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-02T13:01:40-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M1 delivers what the issue's Spec and the plan's Chunk 1 promise: `SwitchTracker` is a genuinely pure three-line rule with composition-driven tests, `ctrl-space` is switcher-only with the ladder/root-actor concept fully excised (compiler-verified, not grep-verified), `ctrl+backspace` lands in both encodings with the latent `panelkeys.go` modified-flag bug fixed as a side effect, and `leave` becomes a global menu frame instead of five thread-bound exceptions. I ran `go test ./...` (only sandbox pty failures: "operation not permitted") and `BenchmarkMenu100` (open 99 µs, navigation 215 µs against 50 ms/16 ms budgets — envelope comfortably held). Nothing here is a correctness bug I can demonstrate, so nothing blocks. What holds it back from SHIP is that the two *production* seams that actually derive and route the headline behavior — `ExecuteConsoleOperation`'s `arrival` derivation and `Run`'s `HitPrevious` dispatch — have no test at all, plus a plan/code contradiction on `alt+d` that the commit message states backwards, a latent dispatch trap that will bite M2, two stale README paragraphs the plan explicitly named, and a `git diff --check` failure on a step already ticked `[x]`.

## 1. Strengths

- **`switchrule.go` is the right shape.** One value type, three lines of rule, no clock/IO/console reference; `switchrule_test.go` drives sequences rather than asserting one rule at a time, which is the only way this rule is interesting. `Drop` is correctly *not* a `Switch`, with the reason in the comment.
- **`seqKind.intercepts()` / `hit()` (`keys.go:49-73`) is real DRY work, not cosmetic.** It removed a hand-written `{seqSwitch, seqPark}` list from both the reducer and `TestInterceptorRecognisesEverySequenceAtEverySplit`, where the test was quietly asserting the wrong branch. Exactly the consolidation ARCH-DRY asks for.
- **`menuFrameBindsThread` (`menu.go:422-439`) collapses five would-be `leave` exceptions into one honest property (frame scope).** `TestLeaveConfirmationSurvivesAnInventoryRefresh` pins the *async* site that a keystroke-only test cannot reach, and `TestParkConfirmationStillDiesWithItsThread` pins the counterpart so the predicate can't be widened by accident. That pairing is the thing that makes the abstraction safe.
- **Rule 2 (acknowledge on every landing) was found and tested independently of rule 1.** `TestEveryArrivalAcknowledgesTheLandedActor` plus `TestSwitchToAnUnknownActorDoesNotAcknowledge` is the correct guard: a test that only checked `previous` would have passed while the bell stayed lit.
- **The #146 Core-concepts contract was revised at its source** (`workshop/history/plans/000146-…-plan.md`, new `## Revisions` entry) rather than by loosening `core_concepts_contract_test.go`. That's the right call and the reasoning is recorded.
- **`decodeCSIu`'s fix follows the `case 9` (Tab) pattern already in the file** rather than inventing a second shape, and `TestDecodePanelKeysDistinguishesCtrlBackspaceFromBackspace` covers all four encodings of backspace.

## 2. Critical findings

None.

## 3. Important findings

**a) No test covers the production wiring that reaches the switch rule** — `cmd/internal/couchtty/console.go:1339-1344` and `console.go:596-601`.

Every console-level test hand-feeds `arrival` into `switchTo` (`notification_test.go:267-270`, `:217`). The one site that *derives* `arrivalNotification` — `c.menu.InFlight.Operation == "switch" && …AttentionCapture != 0` — is exercised by nothing. Invert that condition and the entire headline behavior ships backwards with a green suite. Likewise `TestConsolePreviousKeyRoutesInBothEncodings` only calls `Interceptor.FeedHit`; it never proves `HitPrevious` reaches `onPreviousHotkey` rather than the `default:` arm at `:600`. The plan's own Task 5 Step 1 demanded this ("raw bytes into `processInput`, not direct method calls — reducer support is not user reachability"); it was honored for the key encodings and skipped for the two seams that matter. Fix: one stdin-driven test that marks attention on c2, presses `\x00`, `\r`, then `\x08`, and asserts the operator lands back on c1 — and its negative twin over an unpaged row.

**b) `Run`'s `default:` arm turns a future `HitDetach` into "open the switcher"** — `console.go:600`.

`seqDetach`/`HitDetach` exist (`keys.go:45`, `:88`) but no input can produce them, because `alt+d` is deliberately not registered. When M2 adds the one `knownSequences` row, `alt+d` will silently open the switcher until someone remembers to touch `Run` too. Make the dispatch exhaustive (`case HitSwitch:` explicit, `default:` doing nothing) or drop the two enum values until M2 wires them.

**c) Plan and commit message both claim `alt+d` shipped in M1; the code deliberately says otherwise** — `keys.go:122-130` vs plan `## Core concepts` (M1) and `13afe9fe`.

The plan states "`knownSequences` … gains `ctrl+backspace` in both encodings and `alt+d` from the canonical table", and its ARCH-DRY note repeats it. The commit message says "alt+d comes from workbenchshortcut's canonical table … alongside alt+x." The code registers only `ChordAltX`, with a comment explaining why deferring is correct. The *code* made the better call — claiming a chord without wiring it takes pair's own detach key from the child for a whole milestone — but the plan was never revised, so it now documents something that did not ship.

**d) README still describes the deleted root-actor/home concept** — `README.md:315` and `README.md:319`.

`:315` "the first session is \"home\"" and `:319` "then the root actor's `$PAIR_AGENT`" both name a concept this milestone removed; `RootAgent` is in fact just `Getenv("PAIR_AGENT")` from couch's own environment (`couchcmd/run.go:101`). The plan's Task 5 Step 5 named `:319` explicitly in the list of ladder prose to delete, and it was missed while `:336-351` were correctly rewritten. Docs gate: this is user-facing surface describing behavior that no longer exists.

**e) The #170 plan's Core concepts table is not executable, so the milestone's own architectural claims are unpinned** — `cmd/internal/couchtty/core_concepts_contract_test.go:190-200`.

`findConceptPlan` hard-codes #146's plan, so #170's table — which declares `SwitchTracker` new/PURE and `Up` deleted — is asserted nowhere. The milestone *edited* #146's inventory (correctly, to drop `Up`) but did not add `SwitchTracker`, so the package's executable contract now has a hole exactly where the new pure entity is. Cheapest fix: add `{"PURE", "`SwitchTracker`"}` to `conceptInventory` and a row to the #146 table. Better fix (the class, per ARCH-PURPOSE): let `findConceptPlan` collect every plan that declares a `### Pure entities` table, so later milestones' entities are pinned by construction rather than by remembering.

**f) `focus_test.go` was deleted wholesale, dropping an invariant that survives** — deleted file; `focus.go:11-20`.

The plan said *modify* `focus_test.go` (delete `Up`); the file went. That took `TestFocusEquality` with it — the test pinning `FocusPanel() != FocusActor("")`, which `focus.go`'s own comment calls load-bearing ("a bug that produced an empty actor id would silently become 'show the panel'"). Three live comparisons depend on it, including this milestone's new `alt+x` branch at `console.go:1180`. Note `assertDirectTest` still passes only because `menu_perf_test.go:163` happens to contain the token `Focus` — the contract is satisfied by accident, not by coverage. Restore `TestFocusEquality`.

**g) Ticked steps that did not actually complete** — plan Task 6 Steps 1 and 3.

`git diff --check` fails on the range: `cmd/internal/couchtty/keys_test.go:457: new blank line at EOF`. Step 1 also required recording the verification commands in the issue `## Log`, and the issue at HEAD has no M1 log entry. Step 3 (`sdlc milestone-close`) is pre-checked `[x]` before this review ran. Drop the blank line, add the M1 `## Log` entry, and leave Step 3 unticked until the verdict lands.

## 4. Minor findings

- `switchTo` now acknowledges (`console.go:445`) *and* `finishOperation` still acknowledges the dispatch-time capture (`console.go:1269`) — the plan said "move", not "duplicate". Harmless today (`Acknowledge` is sequence-set based and idempotent), but two authorities for one rule; ARCH-DRY.
- Dead `leave` branches left behind by the global-frame change: `menu.go:1074` (`frame.Action != "leave"` is now unreachable) and `menu_render.go:186` (`if leaf == "leave"`, superseded by the early return at `:172`).
- `gofmt -l` flags `keys_test.go` and `notification_test.go` (comment alignment + trailing blank line). Four other files in the repo are already unformatted, so this isn't a regression in standard, but these two are new.
- `Interceptor`'s doc comment (`keys.go:150`) still says "couch has one key"; it now has three hits.
- `TestActiveNonRootExitFallsBackToRootForMenuActions` (`console_panel_regression_test.go:108`) still asserts correctly but its name references the deleted root concept.
- `RootAgent` (`couchcore/couch.go:37`, `couchtty/menu.go:133`) outlives the root-actor concept as a name. Out of M1's scope; worth folding into M4's deletion sweep.
- `switchTo` updates the tracker even when `already == true` (`console.go:439-448`). Only reachable through `Console.Switch`, which is test-only today, so no live impact — but it means `onSwitch` on the current actor would spend the `previous` slot.
- Plan Task 1 Step 1 named two adversarial classes that got no test: "a switch to the actor that is already current" and "a switch whose target equals `previous`". Both are one table row each in `switchrule_test.go`.
- After an active actor exits, `Drop` clears `current`, so the *next* switch copies an empty `current` into `previous` and the operator loses a still-reachable home. Literally Spec-conformant and pinned by `TestSwitchTrackerDrop`, so deliberate — noting it only because it isn't in the Spec's enumerated consequences.
- `onPreviousHotkey`'s "nowhere to return to" notice goes to `c.feed` (the status row), which is not visible while the panel owns the screen — pressing `ctrl+backspace` in the switcher with no previous is silent.

## 5. Test coverage notes

The pure layer is well covered and the tests pin real logic: removing the `if !t.currentViaNotification` guard reddens `switchrule_test.go`; removing the `switchTo` acknowledgement reddens `TestEveryArrivalAcknowledgesTheLandedActor`; removing the `menuFrameBindsThread` early-continue in `reconcileMenuFrames` reddens `TestLeaveConfirmationSurvivesAnInventoryRefresh`. No mocks reassert the implementation. The gap is entirely at the console↔input boundary (finding 3a): the model is proven, the wiring is not. Suite status: `go test ./...` clean except pre-existing sandbox pty failures ("operation not permitted") across `couch`, `couchcmd`, `couchcore`, `couchtty`, `hostty`, `keyscmd`, `ptychild`, `termcmd` — none related to this diff. Envelope: `BenchmarkMenu100` open 99 µs / filter 87 µs / navigation 215 µs / render 202 µs, all far inside the committed 50 ms and 16 ms budgets; the plan asked for this measurement and it isn't recorded anywhere, so paste it into the issue `## Log`.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — pass, with one flag.** `seqKind.intercepts()/hit()` and `menuFrameBindsThread` are model consolidations, not cosmetic ones. Flagged: the duplicated notification acknowledgement (Minor #1) and the two dead `leave` branches.
- **ARCH-PURE — pass.** `SwitchTracker`, `arrival`, `menuFrameBindsThread` and `decodeCSIu` are all deterministic values/functions; `switchrule_test.go` imports only `testing` and `couchcore`. IO stays in `Console`. The M1 table's PURE claims hold.
- **ARCH-PURPOSE — flag.** M1's purpose is "a notification hop never costs the operator their place." The rule is delivered and proven; the derivation that decides *whether an arrival is a notification hop* is delivered and unproven (3a). The plan wrote the enumeration ("drive through the production input path") and the round swept only part of it — the instance, not the class. Same shape in 3e: the executable contract was repaired for the row that broke and not extended to the rows this milestone added.
- **ARCH-MOCK — N/A/pass.** No new external binary or service; existing `FakeHost`/`FakeChild`/`FakeRunner` seams reused, production and test flow share the same boundary.
- **ARCH-CONSTRAINTS — pass, unrecorded.** Keystroke path held: one extra byte comparison per input byte, one extra `knownSequences` row (exact-string, no timing window), and `reconcileRootSelection` is O(inventory) once per `ctrl-space` rather than per keystroke. Measured above; record it.
- **Forward:** M2 wires `alt+d`, so fix 3b before then or the first detach press opens the switcher. M2 also makes `leave` detach rather than park, which puts real weight on the global-frame change — the all-detached-couch case is already covered by `TestLeaveConfirmationNeedsNoLiveThread`, which is good ground to build on.

## 7. Plan revision recommendations

The plan needs a `## Revisions` entry dated 2026-09-02 recording:

1. **`alt+d` deferred from M1 to M2.** M1's `## Core concepts` prose ("`knownSequences` … gains … `alt+d` from the canonical table") and the M1 `ARCH-DRY` bullet both describe surface that did not ship. State the reason the code gives: intercepting the chord without the console branch takes pair's own detach key from the child for a milestone and returns nothing. Note that `seqDetach`/`HitDetach` were landed early as placeholders.
2. **`panelkeys.go:98` left unchanged**, already argued in Task 4 Step 3 — worth promoting into Revisions so the Spec's "two existing sites" claim doesn't read as unfinished at close.
3. **Task 6 Step 3 un-ticked** until this gate records a verdict, and Step 1's `git diff --check` re-run after the EOF blank line is removed.
4. **Core concepts table scope.** Either add `SwitchTracker` to `core_concepts_contract_test.go`'s `conceptInventory` (and the #146 table), or record the decision that #170's table is documentation only — currently the plan implies a contract that nothing enforces.
