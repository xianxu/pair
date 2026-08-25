# Boundary Review — pair#146 (milestone M3)

| field | value |
|-------|-------|
| issue | 146 — couch: tty switching and attach |
| repo | pair |
| issue file | workshop/issues/000146-couch-tty-switching-and-attach.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 7b800e1960633def33f51b723233ae00faf593df..a14700d88c69b0f1d40a53ae4dc0e683beed7a07 |
| command | sdlc milestone-close --issue 146 --milestone M3 |
| reviewer | codex |
| timestamp | 2026-08-23T22:32:43-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M3’s core focus, replay, resolver, and operation-dispatch foundations are sound, and both the full test suite and race suite pass. The boundary is nevertheless blocked: production bypasses the planned panel model, hotkey routing violates its same-chunk contract, several characters cannot begin a typeahead query, and the Core concepts table materially contradicts the implementation.

## 1. Strengths

- `Focus` and `Up` are genuinely PURE and directly test the full focus ladder, including dead-root behavior (`couchtty/focus.go:52`, `focus_test.go:6`).
- Switching replays the destination child’s query-stripped buffer before repainting the status row (`console.go:276-299`).
- Resolver and operation dispatch are wired through the real console path, avoiding private panel implementations (`couchcmd/run.go:215-257`).
- Background output is retained without painting over the panel (`console.go:478-531`), with meaningful integration coverage.
- `go test ./cmd/... -count=1` and `go test -race ./cmd/... -count=1` both passed.

## 2. Critical findings

### Input after a hotkey can reach the focus being left

At `console.go:573-600`, `pumpStdin` queues the hotkey and immediately processes `rest`. Only the separate Run goroutine changes focus (`console.go:338-343`), so `x<ctrl-space>y` can send `y` to the old child. The pure interceptor test asserts the split but never verifies end-to-end routing.

This is the second finding in family `chunking-invariance`. Establish a rule that terminal behavior is invariant across every legal read split. Serialize interception and focus transition on the Run goroutine, or acknowledge the transition before routing `rest`; test both encodings, multiple hotkeys, and every split point. The panel decoder’s bare-ESC decision (`panelkeys.go:55-62`) belongs in the same sweep because `ESC [ B` changes meaning when split after ESC.

### Production bypasses the panel’s source of truth

`NewPanelModel` correctly consumes `TreeSummary` and retains parked trees, names, and descriptions (`panel.go:75-97`), but production never calls it. `rebuildPanel` constructs a parallel model only from hosted panes (`console.go:652-676`); `pane.desc` is never populated. Consequently parked trees disappear, and successful `name`/`describe` actions leave displayed metadata stale.

This is the third finding in family `dead-field-and-leaked-consumer`. Do not patch individual missing fields. Make every production panel refresh derive from Couch summaries, then join live child IDs for routing. Add integration tests proving a parked tree appears and name/description changes visibly refresh.

### Action keys shadow valid typeahead queries

At `console.go:765-807`, `s`, `x`, `n`, and `d` invoke actions whenever the query is empty; digits jump directly. Therefore no name or description beginning with those characters can be entered as a filter. Existing coverage uses `ari`, avoiding every collision.

Give commands a distinct input namespace—such as a command mode or modified keys—and test queries beginning with every reserved rune. Family: `input-namespace-collision`.

### Core concepts contradict the architecture

The plan places `Console` under “Pure entities” (`plan.md:75-91`), although the source calls it the IO shell and its tests require host/child fakes. It also lists nonexistent `Home` and omits the new `PanelKey`/`DecodePanelKeys` entity.

This is the fifth finding in family `plan-table-drift`. Do not rename isolated cells. Add a validation rule that every table entity resolves and its PURE/INTEGRATION classification agrees with its tests. Classify the thin Console as INTEGRATION and extract its substantial panel/prompt transition policy into a PURE controller.

## 3. Important findings

### Carried M3 smoke work has no recorded evidence

The tracker explicitly carried the composed `kill -9` reattach and real `nvim` in/out checks to M3 (`issue.md:1020-1030`). The M3 completion entry records only the two-actor panel smoke (`issue.md:1272-1279`). The composition previously described as untested remains unaccounted for.

This is the third finding in family `undelivered-plan-step`. Re-audit every M3 and carried checkbox as one enumeration, run the missing checks, and record concrete observations.

### README omits the new interactive surface

README documents only the earlier `ctrl-space` interception (`README.md:280-290`). It does not explain the child→root→panel ladder or the panel’s arrows, Enter, digits, Escape, typeahead, and action keys.

This is the fifth finding in family `docs-lag-the-surface`. Establish the rule that every user-entered key surface must derive or be checked against a README home, rather than patching this one paragraph manually.

## 4. Minor findings

None.

## 5. Test coverage notes

The current suite thoroughly covers isolated focus, panel-model, decoder, replay, resolver, and action-dispatch behavior. Missing composition tests are exactly where the findings live:

- Same-read hotkey plus trailing input routed through a running Console.
- Every legal input chunking around hotkeys and escape sequences.
- Production panel refresh from Couch state, including parked and renamed trees.
- Typeahead beginning with every command or jump key.
- The carried real-stack reattach and nvim smoke.

## 6. Architectural notes

- **ARCH-DRY — flag:** production duplicates the `NewPanelModel` construction rule.
- **ARCH-PURE — flag:** Console is mislabeled PURE and contains panel/prompt policy that should be extracted from its IO duties.
- **ARCH-PURPOSE — flag:** parked-tree visibility, unrestricted typeahead, and correct post-hotkey routing are committed switcher behavior, not optional follow-ups.
- **ARCH-MOCK — pass:** M3 adds no new external dependency; the PTY path continues through `TerminalHandle`, the stateful fake, and live conformance coverage.

## 7. Plan revision recommendations

Append revisions recording:

- The production PanelModel wiring correction and the invariant that every refresh derives from `TreeSummary`.
- The input-arbitration correction and split-point test enumeration.
- The collision-free command/typeahead key design.
- Core concepts corrections: Console → INTEGRATION, add `PanelKey`/`DecodePanelKeys`, remove or implement `Home`.
- Concrete evidence for the carried reattach and nvim smoke, or an explicit scope decision if they move again.

```findings
findings:
  - id: new
    severity: Critical
    family: chunking-invariance
    title: |
      Input after a hotkey can be routed to the focus being left
    detail: |
      console.go:573-600 queues the hotkey but processes rest before the Run goroutine acknowledges the focus change. This is the 2nd finding in family chunking-invariance; enumerate both input framers and test every legal read split rather than fixing one byte grouping.
  - id: new
    severity: Critical
    family: dead-field-and-leaked-consumer
    title: |
      Production bypasses NewPanelModel and loses parked or updated tree metadata
    detail: |
      panel.go:75-97 implements the planned TreeSummary model, but console.go:652-676 rebuilds rows only from hosted panes and pane.desc is never populated. This is the 3rd finding in family dead-field-and-leaked-consumer; make all production refreshes consume the shared summary source and join routing IDs afterward.
  - id: new
    severity: Critical
    family: input-namespace-collision
    title: |
      Panel action keys make valid typeahead prefixes unreachable
    detail: |
      console.go:765-807 consumes s, x, n, d and digits as commands when the query is empty, so names and descriptions beginning with those characters cannot be searched. Separate command input from filter text and test every reserved prefix.
  - id: new
    severity: Critical
    family: plan-table-drift
    title: |
      The Core concepts table misclassifies and misstates M3 entities
    detail: |
      The plan's Pure entities table classifies Console as PURE despite its IO dependencies, lists nonexistent Home, and omits DecodePanelKeys. This is the 5th finding in family plan-table-drift; enforce entity existence and kind classification across the complete table, then append a Revisions entry.
  - id: new
    severity: Important
    family: undelivered-plan-step
    title: |
      M3 completion evidence omits the smoke work explicitly carried into this milestone
    detail: |
      issue lines 1020-1030 carry the composed kill -9 reattach and real nvim in-and-out checks to M3, while lines 1272-1279 record only the two-actor panel smoke. This is the 3rd finding in family undelivered-plan-step; enumerate every M3 and carried checkbox and supply evidence for each.
  - id: new
    severity: Important
    family: docs-lag-the-surface
    title: |
      README does not document the M3 focus ladder or panel controls
    detail: |
      README.md:280-290 documents only ctrl-space interception, not child-to-root-to-panel navigation or the keys a user types in the panel. This is the 5th finding in family docs-lag-the-surface; establish an enforced documentation home for every user-entered key surface.
```

---

## Re-review — 2026-08-24T16:02:55-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 146 — couch: tty switching and attach |
| repo | pair |
| issue file | workshop/issues/000146-couch-tty-switching-and-attach.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 7b800e1960633def33f51b723233ae00faf593df..fd363ff306323326da6fc28625d70cb39ec39c12 |
| command | sdlc milestone-close --issue 146 --milestone M3 |
| reviewer | codex |
| timestamp | 2026-08-24T16:02:55-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

Four prior findings are addressed and pinned, but two Critical findings remain open. BR-42 fixed same-read ordering but not the required chunking-invariance class: a Kitty hotkey split immediately after `ESC` is still forwarded to the child and missed. BR-45 corrected the table entries, but the promised mechanical audit does not exist, and the plan still says `Console` holds no policy despite substantial panel policy in that type.

## 1. Strengths

- Production now derives panel rows from injected `Couch.Summarize(nil)` data and joins routing state separately ([console.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/console.go:696)).
- Commands use a collision-free `:` namespace, while ordinary letters and digits remain typeahead ([console.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/console.go:819)).
- The README and atlas document the focus ladder, controls, summary source, and one-shot repo-default handoff.
- The cold-create policy reuses `SkipConfigPicker`, consumes its environment signal at process entry, and has stateful lifecycle tests.
- `go test ./... -count=1` and `go test -race ./cmd/... -count=1` passed at the reviewed head.

## 2. Critical findings

### BR-42 remains open — Kitty hotkeys are not invariant across legal read splits

[keys.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/keys.go:110) explicitly accepts forwarding a real sequence when the read boundary follows its initial `ESC`. Thus `"\x1b"` followed by `"[32;5u"` does not trigger couch’s hotkey and leaks the sequence to the child. The existing split test starts later in the sequence, at `"\x1b[32;"`, so it cannot catch this case.

This is the third occurrence in family `chunking-invariance`. Fix the rule, not another split point: both input framers need an explicit ambiguity mechanism and enumeration of every boundary for every recognized sequence.

### BR-45 remains open — the claimed Core-concepts enforcement is absent

The corrected table now places `Console` under Integration and adds `DecodePanelKeys`, but no test or audit verifies table declarations or PURE/INTEGRATION classifications. The plan nevertheless claims such an audit exists at lines 658–659. It also says `Console` “holds no policy” at line 163, while panel transitions, prompts, command interpretation, and action selection remain in [console.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/console.go:779).

This is the sixth occurrence in family `plan-table-drift` and an `ARCH-PURE` violation. Implement the promised complete audit and either extract the policy into a PURE controller or revise the plan’s architectural claim.

## 3. Important findings

None.

## 4. Minor findings

- `git diff --check` fails on extensive trailing whitespace in the captured M3 review transcript.

## 5. Test coverage notes

BR-43 is pinned by production wiring, parked-summary, target-join, and refresh tests. BR-44 is pinned across all formerly reserved prefixes. BR-47 is pinned from the shared `PanelControls` inventory.

The suite lacks the exact BR-42 case: Kitty `ctrl-space` split after byte one. It also lacks the Core-concepts audit asserted by the plan.

## 6. Architectural notes

- `ARCH-DRY` — pass: summaries, resolver behavior, operations, and panel-control documentation have shared sources.
- `ARCH-PURE` — flag: substantial panel policy remains inside the IO-heavy `Console`.
- `ARCH-PURPOSE` — flag: missing a legal hotkey split violates the core “ctrl-space always reaches couch” purpose.
- `ARCH-MOCK` — pass: PTY and launcher behavior use stateful doubles behind production seams, with live conformance coverage.

## 7. Plan revision recommendations

- Revise the input-serialization entry to acknowledge that the child-side interceptor still mishandles the `ESC` split; specify one ambiguity rule and exhaustive split enumeration for both framers.
- Replace the unsupported “table audit checks every row” statement with an implemented audit.
- Reconcile the “Console holds no policy” claim with the implementation, either through extraction or an explicit architectural revision.

```findings
dispose:
  - id: BR-42
    disposition: not-addressed
    note: |
      Same-read suffix ordering is pinned, but keys.go:110-119 still forwards a Kitty hotkey split immediately after ESC; the required every-split enumeration is absent.
  - id: BR-43
    disposition: addressed
    note: |
      Production injects Couch.Summarize(nil), rebuildPanel starts from NewPanelModel, and wiring, parked-row, refresh, and pure target-join tests pin the path.
  - id: BR-44
    disposition: addressed
    note: |
      Commands now require the colon namespace, and tests pin typeahead beginning with s, x, n, d, and digits plus namespaced direct jumps.
  - id: BR-45
    disposition: not-addressed
    note: |
      The named table cells changed, but no claimed entity/kind audit exists and the plan still says Console holds no policy despite console.go:779-945.
  - id: BR-46
    disposition: addressed
    note: |
      The tracker records the composed kill -9 reattach evidence and a Spec/Plan revision explains why the literal layout3-only nvim step does not apply to the shipped layout2 configuration.
  - id: BR-47
    disposition: addressed
    note: |
      README documents the full focus ladder and namespaced controls, with coverage derived from couchtty.PanelControls.
findings:
  - id: new
    severity: Minor
    family: generated-artifact-hygiene
    title: |
      The captured M3 review transcript fails git diff --check
    detail: |
      workshop/plans/000146-couch-tty-switching-and-attach-m3-review.md contains extensive added trailing whitespace and space-before-tab lines; clean the generated artifact or its writer.
```

---

## Re-review — 2026-08-24T16:21:31-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 146 — couch: tty switching and attach |
| repo | pair |
| issue file | workshop/issues/000146-couch-tty-switching-and-attach.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | d18bce347ba407e54f60b8b40ca8778eb950c57f..d18bce347ba407e54f60b8b40ca8778eb950c57f |
| command | sdlc milestone-close --issue 146 --milestone M3 |
| reviewer | codex |
| timestamp | 2026-08-24T16:21:31-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The M3 behavior and regression suites are broadly strong: BR-42’s routing fix is reachable through the production console and tests enumerate the relevant stream splits; the full Go suite, focused race suite, vet, and diff hygiene checks pass. The boundary remains blocked by BR-45: the Core-concepts table is manually corrected, but the promised class-level enforcement has no executable test, and `Console`’s source comment still contradicts the corrected classification. The supplied review window itself is empty (`Base == Head`), so this verdict assesses the current tree.

## 1. Strengths

- BR-42 is pinned at both levels: exhaustive interceptor split coverage in `keys_test.go:215` and the composed production-path `ESC`/Kitty regression in `console_test.go:155`.
- Panel sequences are exercised at every listed split in `panelkeys_test.go:37`.
- Hotkey acknowledgement serializes focus changes before routing suffix bytes (`console.go:653-674`).
- README and atlas changes cover the M3 focus ladder, panel controls, and console architecture.
- `git diff --check` passes for both the full M3 range and the supplied review window, resolving BR-48.

## 2. Critical findings

### BR-45 remains open — Core-concepts enforcement is not test-pinned

`workshop/plans/000146-couch-tty-switching-and-attach-plan.md:75-148`, `cmd/internal/couchtty/console.go:41-46`

The table is now factually improved, but no test parses it to verify declared symbol existence, deletion, location, classification, or direct PURE coverage. Therefore reverting a table row cannot make any test fail, which fails the explicit prior-fix acceptance rule.

The stale `Console` comment also still says it “holds no policy,” while the corrected plan and implementation acknowledge that it owns transient panel and UI-transition policy.

Fix sketch: add a table-contract test or equivalent mechanically derived check covering every due-through-M3 row, then correct the stale `Console` comment. Demonstrate that misplacing `Console`, restoring nonexistent `Home`, or changing a declared path makes the test fail.

## 3. Important findings

None.

## 4. Minor findings

- `console.go:613-631` introduces an uncancellable stdin reader goroutine, while `Run` still neither closes the host nor joins its background workers. This is the third finding in family `signal-goroutine-outlives-close`; fix the lifecycle rule, not only this reader: one owner must stop, close, and join every console background worker.

## 5. Test coverage notes

Verified green:

- `go test ./... -count=1`
- Focused `go test -race` across `couchtty`, `couchcmd`, `couchcore`, `ptychild`, and `hostty`
- Focused `go vet`
- `git diff --check` for the full M3 range and supplied window

BR-42’s regression is reachable through `Console.pumpStdin`, not merely a pure decoder test. BR-45 remains unpinned because no executable Core-concepts audit exists.

## 6. Architectural notes

- **ARCH-DRY — pass.** Input framing has one interceptor inventory, and panel operations/resolution continue through shared injected sources.
- **ARCH-PURE — flag.** The table now honestly classifies `Console` as INTEGRATION, but its source comment still claims policy-free glue. If M4 adds further transitions, extract the panel interaction reducer rather than expanding `onPanelKey`.
- **ARCH-PURPOSE — flag.** BR-42 delivers the chunking-invariance class. BR-45 delivers corrected prose but not the promised enforcement, leaving the recurring `plan-table-drift` class able to return.
- **ARCH-MOCK — pass.** Production and tests share the injected Host/Child/Runner boundaries, with stateful fake and real-side conformance coverage.

## 7. Plan revision recommendations

No additional scope revision is needed. After adding enforcement, append a revision stating exactly which table invariants the executable check derives and pins; do not describe another manual audit as enforcement.

```findings
dispose:
  - id: BR-42
    disposition: addressed
    note: |
      The exact ESC-split Kitty hotkey is exercised through Console, while interceptor and panel-key tests enumerate the recognized sequences at every listed byte split.
  - id: BR-43
    disposition: addressed
    note: |
      Prior disposition remains valid; production consumes the summary-derived panel and the wiring is exercised.
  - id: BR-44
    disposition: addressed
    note: |
      Prior disposition remains valid; printable prefixes remain typeahead and panel commands use the colon namespace.
  - id: BR-45
    disposition: not-addressed
    note: |
      The rows were manually corrected, but no executable table-contract audit exists, so reverting a path, symbol, deletion, or classification cannot fail a test; Console's source comment also still claims it holds no policy.
  - id: BR-46
    disposition: addressed
    note: |
      Prior disposition remains valid under the recorded layout2 Spec and Plan revision.
  - id: BR-47
    disposition: addressed
    note: |
      Prior disposition remains valid; README coverage derives from the shared panel-control inventory.
  - id: BR-48
    disposition: addressed
    note: |
      git diff --check passes for both the complete M3 range and the supplied review window.
findings:
  - id: new
    severity: Minor
    family: signal-goroutine-outlives-close
    title: |
      The timed stdin framer adds another background worker that Console cannot cancel or join
    detail: |
      This is the 3rd finding in family signal-goroutine-outlives-close. console.go:613-631 starts a reader blocked on an arbitrary io.Reader, while Run does not own an ordered stop, host close, and worker join. State the class rule: a console lifecycle has one teardown owner that cancels or closes every input/signal seam and joins every worker before returning.
```

---

## Re-review — 2026-08-24T16:38:07-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 146 — couch: tty switching and attach |
| repo | pair |
| issue file | workshop/issues/000146-couch-tty-switching-and-attach.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 6a873efb64fa8776d9d028fc1f9c8b986cc5ad2a..6a873efb64fa8776d9d028fc1f9c8b986cc5ad2a |
| command | sdlc milestone-close --issue 146 --milestone M3 |
| reviewer | codex |
| timestamp | 2026-08-24T16:38:07-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The current implementation and focused/full test suites are healthy, and BR-49’s newly added reader goroutine disappeared with the revised design. However, BR-45 remains blocking: the new contract test validates rows that exist but cannot detect a completely omitted entity row—the exact class that omitted `DecodePanelKeys` previously. Removing the entire `PanelKey`/`DecodePanelKeys` row in a scratch copy still produced a green contract test. The supplied review window is empty (`Base == Head`), so dispositions were verified against HEAD and its parent diff.

### 1. Strengths

- `cmd/internal/couchtty/console.go:349-447` now centralizes input framing, timers, focus transitions, and routing in the existing event loop.
- `cmd/internal/couchtty/console.go:662-680` has one blocking stdin reader instead of the nested reader introduced previously.
- Moving `couchtty.Console` into the PURE table in a scratch copy correctly failed because `console.go` imports `io` and `os`.
- The corrected Console comment at `console.go:42-46` accurately describes its integration responsibility.
- README and atlas changes cover the M3 user-facing controls and architecture.

### 2. Critical findings

BR-45 remains open — `cmd/internal/couchtty/core_concepts_contract_test.go:28-69`.

`TestCoreConceptsContract` only validates rows parsed from the plan. Deleting an entire row removes it from the test’s input and therefore passes. In a scratch copy, deleting the complete `PanelKey`/`DecodePanelKeys` row from `workshop/plans/000146-couch-tty-switching-and-attach-plan.md:85` left the test green.

This is the recurring `plan-table-drift` class, not merely another bad row. Make the audit bidirectional: every declared table entry must resolve correctly, and every in-scope/annotated core concept must appear in the table. Add a deletion test that removes a whole row and goes red. Source-adjacent concept annotations or one typed inventory from which the table is derived would avoid creating another hand-maintained inventory.

### 3. Important findings

None.

### 4. Minor findings

None. BR-49 is withdrawn because the design change removed the additional worker it identified. The pre-existing console teardown work remains explicitly scheduled for M4.

### 5. Test coverage notes

Passed:

- Focused contract, interceptor, and console regressions
- `go test -race ./cmd/internal/couchtty -count=1`
- `go test ./... -count=1`
- `go vet ./...`
- Full M3-range and supplied-window `git diff --check`

Mutation results:

- Misclassifying Console as PURE: red.
- Removing the entire `PanelKey`/`DecodePanelKeys` row: incorrectly green.

Live external smokes were not rerun during this read-only review.

### 6. Architectural notes for upcoming work

- **ARCH-DRY — pass.** Production input framing has one event-loop owner.
- **ARCH-PURE — pass.** Console is correctly classified as integration; reusable policies remain separately tested.
- **ARCH-PURPOSE — flag.** BR-45’s fix validates existing instances but does not enforce table completeness, leaving the original omission class possible.
- **ARCH-MOCK — pass.** No new external dependency bypasses the existing runner/host seams; fake and live-conformance surfaces remain intact.
- M4 should complete the broader lifecycle rule behind BR-49: one teardown owner closes/cancels every input and signal seam and joins every worker before `Run` returns.

### 7. Plan revision recommendations

After fixing BR-45, append a `## Revisions` entry correcting the current claim that the contract enforces the complete inventory. Record the new bidirectional rule and the whole-row deletion mutation that proves omissions fail.

```findings
dispose:
  - id: BR-45
    disposition: not-addressed
    note: |
      The contract validates present rows but cannot detect an omitted row; deleting the complete PanelKey and DecodePanelKeys row still passes, so the recurring plan-table-drift class is not pinned.
  - id: BR-49
    disposition: withdrawn
    note: |
      The revised event-loop design removes the additional nested stdin worker this finding identified; full ownership and joining of the remaining lifecycle workers stays explicitly scheduled for M4.
```

---

## Re-review — 2026-08-24T16:46:21-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 146 — couch: tty switching and attach |
| repo | pair |
| issue file | workshop/issues/000146-couch-tty-switching-and-attach.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | ca480ceaf156c573c30704737b98665e3a53f054..ca480ceaf156c573c30704737b98665e3a53f054 |
| command | sdlc milestone-close --issue 146 --milestone M3 |
| reviewer | codex |
| timestamp | 2026-08-24T16:46:21-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The literal Base–Head window is empty, but the current Head’s BR-45 correction was independently verified. The Core concepts table accurately classifies `Console`, excludes nonexistent `Home`, includes `PanelKey`/`DecodePanelKeys`, and is protected by a bidirectional inventory contract. All three original failure modes produced red tests under isolated mutation; the full Go suite passes.

## Strengths

- The explicit inventory at `core_concepts_contract_test.go:26` prevents omitted, added, duplicated, or reclassified rows.
- The table correctly places `Console` under Integration and describes its IO/policy role honestly at `workshop/plans/000146-couch-tty-switching-and-attach-plan.md:143`.
- The revision records both the reason and enforcement change at `workshop/plans/000146-couch-tty-switching-and-attach-plan.md:832`.
- Pure entities have direct non-IO logic tests; integration points use shared seams and the stateful `FakeRunner`.

## Critical findings

None.

## Important findings

None.

## Minor findings

None.

## Test coverage notes

- `go test ./... -count=1`: passed.
- `git diff --check`: passed; worktree remained clean.
- Scratch mutations confirmed:
  - deleting `PanelKey`/`DecodePanelKeys` fails with `missing PURE row`;
  - adding `Home` fails as unexpected, absent, and untested;
  - classifying `Console` as PURE fails with kind mismatch and `io`/`os` dependency errors.

## Architectural notes for upcoming work

- `ARCH-DRY`: Pass—one contract helper owns inventory comparison; no duplicated production behavior was introduced.
- `ARCH-PURE`: Pass—pure rows point to deterministic core logic; `Console` and terminal dependencies are correctly classified as integration.
- `ARCH-PURPOSE`: Pass—the fix addresses the recurring table-drift class, including whole-row omission, rather than only restoring one named row.
- `ARCH-MOCK`: Pass—the existing stateful `FakeRunner` and live conformance seam remain represented; this correction introduces no new external dependency.

## Plan revision recommendations

None.

```findings
dispose:
  - id: BR-45
    disposition: addressed
    note: |
      The corrected table and bidirectional inventory contract cover all three original drift modes, and isolated mutations made each mode fail.
```
