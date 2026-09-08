# Boundary Review — pair#199 (milestone M1)

| field | value |
|-------|-------|
| issue | 199 — pair term: own the right pane tab bar |
| repo | pair |
| issue file | workshop/issues/000199-pair-term-own-the-right-pane-tab-bar.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | c61f5ca0619d59e3ba795b77d6e12ad18e5517f5..0aa2b6afcb133353349942bb286ef56ff21ed5ee |
| command | sdlc milestone-close --issue 199 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-07T16:30:11-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M1 is a clean, genuinely faithful lift: `Reserve`/`Release`/`PaintRow`/`ChildRows` become `hostty.Reservation` methods, couch is repointed at four call sites, `grep` confirms one implementation of `\x1b[r`, and the move is proved by `couchtty/vtscreen_test.go` — a real VT-emulator end-to-end oracle over the reserved row — passing **unedited**. `go test ./cmd/internal/hostty/ ./cmd/internal/couchtty/ ./cmd/internal/artifactpath/ -count=1` is green (the two failures under the agent sandbox are the known pty restriction; both pass unsandboxed), `go vet` clean, atlas updated. Nothing here blocks correctness. What I'd fix before crossing: one deliberate behaviour change at the degenerate height that is *undeclared and unpinned* (mutation-verified — restoring the old behaviour keeps the whole suite green), the plan's Core-concepts table claiming four entities at a file that doesn't exist, and the plan-quality gate still sitting `blocked: true` with all nine PQ findings open and `estimate_hours:` empty while M1's code has already landed.

## 1. Strengths

- **The move is proved by an unedited oracle, not by assertion.** `cmd/internal/couchtty/vtscreen_test.go:135-171` drives a real VT screen: 40 scrolled lines don't touch row 8, `\x1b[r` from the child triggers re-assertion of `\x1b[1;7r`, teardown leaves a full-height scrollable screen. `git diff --stat` over that file is empty. That is exactly the "a regression there means it was a rewrite" Done-when, made checkable by something the implementor couldn't tune.
- **The deadlock was found, fixed at the right level, and explained.** `couchtty/console.go:914-922` — `bottomReservation(rows uint16)` as a plain function rather than a `c.mu`-reading method. Taking rows as an argument makes `ChildSize` and `applyLayout` obviously safe and leaves the locking discipline where it already lives (ARCH-ORDER: the illegal state is now unrepresentable at the call site, not merely avoided).
- **`Edge`'s asymmetry is named rather than assumed, and fails closed.** `hostty/reserve.go:31-38,58-67` plus `TestAnUnvalidatedTopEdgeFailsClosed`. I mutated `Paint`'s guard back to a bare `Rows == 0` and that test went red immediately — the fail-closed path is a real oracle, not decoration.
- **The hand-maintained-list sweep was complete.** All five consumers of the moved symbols were found and updated: `console.go` (4 sites), `conceptInventory`, `conceptPlans`, `#146`'s Core-concepts row (flipped to `deleted` so the contract now asserts *absence*), and `artifactpath.NonArtifactSources`. That is the `#188` class caught rather than shipped (ARCH-PURPOSE shadow-sweep: pass).
- **`hostty/reserve.go` imports only `fmt`.** Pure by construction; its tests need no fake, no fixture, no IO (ARCH-PURE: pass).

## 2. Critical findings

None.

## 3. Important findings

**I1 — `Paint` silently changed behaviour at `Rows == 1`, and no test pins the change** (`cmd/internal/hostty/reserve.go:102-105`)

`PaintRow` guarded only `hostRows == 0`, so on a 1-row host it drew the status row over the child's only line. `Paint` now guards `!usable()` (`Rows > 1`), so it draws nothing. That is the *right* change — it is precisely what `#146`'s close gate re-raised and never got fixed ("on a 1-row host `PaintRow` still draws the row, so 'simply not drawn' is wrong at `hostRows==1`") and it makes `ChildRows`' doc clause true for the first time. But it is undeclared as a deviation from the "this is a MOVE" claim, and it is unpinned.

Verified by mutation: restoring `if r.Edge != EdgeBottom || r.Rows == 0` — old height behaviour, new edge behaviour — leaves `./cmd/internal/hostty/` and `./cmd/internal/couchtty/` fully green. `TestDegenerateHeightsNeverProduceAZeroRowChild` exercises `ChildRows` and `Reserve` at rows 0 and 1 but never `Paint`.

Fix: add to that loop — `if r.Paint("x") != "" { t.Fatalf("rows=%d drew a row it has no room for", rows) }` — and record the deviation in the plan's `## Revisions` (the lift deliberately fixed the residual half of `#146`'s BR-32, rather than being byte-faithful).

**I2 — the plan's Core-concepts table declares four entities `new` at a file that does not exist** (`workshop/plans/000199-…-plan.md:114-117`)

`TabChip`, `StripModel`, `RenderedStrip`, `RenderStrip` are listed `new` at `cmd/internal/termcmd/strip.go`; `ls cmd/internal/termcmd/` confirms no `strip.go`. The contract test documents the convention this violates in its own comments: *"Rows for milestones that have not shipped carry a `planned` status, which the row loop skips — so the status column doubles as the build tracker."* This milestone is what enrolled `#199` into `conceptPlans` (`core_concepts_contract_test.go:257`), so the table is now a contract input. It passes today only because the package filter drops non-`couchtty` rows — the guard is silent, not satisfied.

Fix: flip those four rows to `planned — M3`. `Edge` and `Reservation` correctly stay `new`.

**I3 — M1's code landed while the plan-quality gate is still blocked and no estimate was recorded** (`workshop/plans/000199-…-plan-gate.md:110`, issue frontmatter line 8)

The ledger's only round is `blocked: true` with all nine PQ findings in `## Open findings`, and `estimate_hours:` is empty with no `## Estimate` block in the issue. Per AGENTS.md §2, `change-code` runs plan-quality first and only then asks for the estimate, so neither appears to have run. PQ-2 was M1-scoped and its remediation *did* ship here (the `#146` row flipped to `deleted`, `#199` registered in `conceptPlans`, both test files listed as M1 work in the plan's Revisions) — but the ledger still reports it open, which makes the remaining open set wrong right when M2 starts. The other eight are live against M2/M3/M4: PQ-3 in particular ("`TakeRowDirty` is always false in `termcmd`; the sink drops it at `run.go:661`") invalidates a row of the plan's own Integration-points table before M3 is written.

Fix: re-run the plan gate so PQ-2 disposes `addressed` and the rest carry forward accurately, and land the estimate. Not a defect in M1's code.

## 4. Minor findings

- `hostty/reserve.go:49` — `NewReservation` has **zero production callers**; couch builds the struct directly (`console.go:921`). The validating door is bypassed by the only consumer; the fail-closed methods are the real (and tested) protection. Either construct through it at the couch site or drop it and let `usable()` carry the whole contract.
- `atlas/couch.md:228` "The reserved row: a reservation, not compositing" still reads as couch-owned mechanism with no pointer to `hostty.Reservation`. That section is where a reader lands for this subject; one cross-reference line.
- `hostty/reserve.go:97-101` — `Paint`'s doc never states the caller's obligation to sanitize and column-clamp `text`. Faithful to `PaintRow`, but the mechanism/policy split just moved that obligation across a package boundary, and M3's consumer feeds it operator-supplied tab names (the plan's own ARCH-SECURE section). One doc line at the new door.
- `hostty/reserve.go:95` — `Release()` ignores its receiver entirely. Correct (teardown is unconditional by design) but reads as if it depends on the reservation; say so in the comment.
- `core_concepts_contract_test.go:45` and `:88` now assert the same three symbols absent from the same file via two rows in two plans. Harmless, explained in the comment, but it is two rows for one fact.

## 5. Test coverage notes

- The new `hostty` tests are genuine oracles — I mutated the `Paint` guard and got a red test with a precise message. They run with no IO and no double (ARCH-PURE, ARCH-MOCK: pass).
- The one real gap is I1: `Paint` at `Rows ∈ {0,1}`. That dimension of the guard is currently free.
- Not covered, and probably shouldn't be at this milestone: the `bottomReservation` deadlock class. It surfaced as a four-minute hang, not a failure. A package-level `-timeout` in CI is the cheap catch; a test isn't worth it.
- `Release()` on an unusable reservation is unpinned. Trivial, unconditional by design; noting for completeness only.

## 6. Architectural notes for upcoming work

- **ARCH-DRY: pass.** One `\x1b[r`, verified — every `SetRegion`/`ResetRegion` outside `hostty` is a qualified use of the constant, and the remaining literals are test assertions. `termcmd/run.go:1036` (`restoreTerminal` writing `hostty.ResetRegion` bare) is the one site that should become `res.Release()` when M3 wires the reservation in; today it's correct because `termcmd` doesn't reserve.
- **ARCH-CONSTRAINTS: pass for M1** — pure string building on a path that already concatenated two strings per paint. The plan's envelope claims all target M2/M3.
- **ARCH-ORDER: pass** — `Reservation` is a value carrying no state between events. The plan's transition table stands; note PQ-3 says its `TakeRowDirty` row names a seam that doesn't deliver in `termcmd`, so that row needs correcting before M3 is implemented, not during.
- **Nothing pins the new `hostty` entities.** The concept contract is package-scoped to `couchtty`, so `Edge`/`Reservation` are declared in the plan and asserted by no guard. Not worth a second contract for two symbols, but if `hostty` keeps growing shared mechanism it becomes the same drift `#146` fought.
- **The prose about this mechanism now exists in four hand-maintained places** (hostty package doc, `couchtty/reserve.go` header, `atlas/architecture.md`, `atlas/couch.md`). All four agree today. `atlas/couch.md` is the one that will go stale first.

## 7. Plan revision recommendations

Add one `## Revisions` entry to `workshop/plans/000199-pair-term-own-the-right-pane-tab-bar-plan.md`:

- **The lift was not byte-faithful at `Rows == 1`, deliberately.** `PaintRow(1, text)` drew the row over a 1-row child's only line; `Reservation.Paint` draws nothing, which is what `ChildRows`' doc has always claimed and what `#146`'s close gate flagged as false and never fixed. Recorded as a fix, not drift, and pinned by extending `TestDegenerateHeightsNeverProduceAZeroRowChild` to cover `Paint`.
- **Core-concepts table:** `TabChip` / `StripModel` / `RenderedStrip` / `RenderStrip` move from `new` to `planned — M3`. `cmd/internal/termcmd/strip.go` does not exist and the contract's status column is the build tracker.

```findings
findings:
  - id: new
    severity: Important
    family: uncovered-negative-assertion
    title: |
      Paint's new 1-row behaviour is an undeclared, unpinned change from the moved PaintRow
    detail: |
      cmd/internal/hostty/reserve.go:102 guards on !usable() (Rows > 1) where
      PaintRow guarded only hostRows == 0, so a 1-row host now draws nothing
      instead of drawing over the child's only line. This is the right change --
      it is the residual half of pair#146's BR-32 -- but the milestone claims a
      MOVE and nothing pins it. Mutation-verified: restoring the old height
      behaviour keeps ./cmd/internal/hostty and ./cmd/internal/couchtty fully
      green. TestDegenerateHeightsNeverProduceAZeroRowChild covers ChildRows and
      Reserve at rows 0 and 1 but never Paint. Add the Paint assertion to that
      loop and record the deviation in the plan's Revisions.
  - id: new
    severity: Important
    family: plan-table-drift
    title: |
      Four Core-concepts rows claim `new` at cmd/internal/termcmd/strip.go, which does not exist
    detail: |
      plan lines 114-117 declare TabChip, StripModel, RenderedStrip and
      RenderStrip as `new` at a file that is not in the tree; they are M3 work.
      core_concepts_contract_test.go documents the convention this breaks --
      unshipped rows carry `planned` so the status column doubles as the build
      tracker -- and this milestone is what registered #199 in conceptPlans. The
      contract passes only because the couchtty package filter drops those rows,
      so the guard is silent rather than satisfied. Flip the four rows to
      `planned — M3`.
  - id: new
    severity: Important
    family: gate-not-cleared-before-code
    title: |
      M1 code landed with the plan-quality gate still blocked and no estimate recorded
    detail: |
      The plan-gate ledger's only round is `blocked: true` with all nine PQ
      findings open, and the issue's estimate_hours is empty with no Estimate
      block -- so change-code appears not to have cleared. PQ-2 was the only
      M1-scoped finding and its remediation did ship here, but the ledger still
      reports it open, which makes the open set wrong entering M2. PQ-3 in
      particular contradicts the plan's own Integration-points row before M3 is
      written. Re-run the gate so PQ-2 disposes `addressed`, and land the
      estimate. Not a defect in M1's code.
  - id: new
    severity: Minor
    family: validating-door-bypassed
    title: |
      NewReservation has zero production callers; couch constructs the struct directly
    detail: |
      hostty/reserve.go:49 is the validating constructor, but couchtty/console.go:921
      builds Reservation{} literally. The exported struct makes the door optional
      and the fail-closed methods are what actually protect. Either construct
      through NewReservation at the couch site or drop it and let usable() carry
      the contract alone.
  - id: new
    severity: Minor
    family: atlas-points-at-old-home
    title: |
      atlas/couch.md's "The reserved row" section still reads as couch-owned mechanism
    detail: |
      atlas/couch.md:228 describes the reservation with no pointer to
      hostty.Reservation, while architecture.md carries the new note. That
      section is where a reader lands for this subject; add one cross-reference.
  - id: new
    severity: Minor
    family: caller-obligation-undocumented
    title: |
      Paint's doc does not state the caller's obligation to sanitize and clamp its text
    detail: |
      hostty/reserve.go:97 writes caller text verbatim between SaveCursor and
      RestoreCursor. Faithful to PaintRow, and couchtty.RenderStatusRow sanitizes
      upstream, but the mechanism/policy split just moved that obligation across
      a package boundary and M3 adds a consumer whose input is operator-supplied
      tab names. One doc line at the new door (ARCH-SECURE).
```

---

## Re-review — 2026-09-07T17:03:57-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 199 — pair term: own the right pane tab bar |
| repo | pair |
| issue file | workshop/issues/000199-pair-term-own-the-right-pane-tab-bar.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | c61f5ca0619d59e3ba795b77d6e12ad18e5517f5..3e02db1d8ba5fab2a2faf94dd6ea950cb59d0ba9 |
| command | sdlc milestone-close --issue 199 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-07T17:03:57-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Verification complete. Here is the review.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M1 is a faithful, well-pinned lift: `couchtty.ChildRows/Reserve/Release/PaintRow` become `hostty.Reservation` methods, couch is repointed at four call sites through a deliberately lock-free `bottomReservation(rows)`, and the move is proved by oracles the implementor could not tune — `couchtty/vtscreen_test.go` and `console_live_test.go` are byte-untouched (`git diff --stat` empty) and the full `go test ./cmd/... -count=1` suite is green unsandboxed, `go vet` and `go build ./...` clean. The two sandboxed failures are the known pty restriction, not this diff. I re-ran the prior round's mutation check: reverting `Paint`'s guard to the old `r.Rows == 0` turns `TestNothingIsPaintedOnARowThatWasNeverReserved` and `TestAnUnvalidatedTopEdgeFailsClosed` red, so BR-10's claimed fix is real. Nothing in M1's code blocks the boundary. What holds it back from SHIP is document state that M2–M4 will read: the plan corrected three gate findings in one place each and left seven restatements of the superseded facts standing elsewhere in the same file, and the DECSTBM measurement the whole design rests on cites a probe directory that does not exist anywhere in the tree or in git history.

## 1. Strengths

- **The move is proved by an oracle, not an assertion.** `cmd/internal/couchtty/vtscreen_test.go` drives a real VT emulator over the reserved row and is unedited; `reserve_test.go:11-13` leaves a forwarding note where the mechanism tests went rather than a hole. That is the Done-when's "a regression there means it was a rewrite," made checkable.
- **BR-10's fix is reachable and mutation-verified.** `cmd/internal/hostty/reserve_test.go:91-103` plus the declared deviation at `cmd/internal/hostty/reserve.go:103-108`. I reverted the guard in a scratch copy; both tests went red immediately. This is the one place the round could have shipped a green-but-empty fix, and it did not.
- **The deadlock was fixed at the right level.** `cmd/internal/couchtty/console.go:914-922` — `bottomReservation(rows uint16)` as a plain function, not a `c.mu`-reading method. Taking rows as an argument makes `ChildSize` and `applyLayout` obviously safe and leaves the locking discipline where it already lives (ARCH-ORDER: illegal state made unrepresentable at the call site).
- **ARCH-PURE by construction.** `cmd/internal/hostty/reserve.go` imports only `fmt`; all four methods are string producers, and every test runs with no fake, no fixture, no IO.
- **The hand-maintained-list sweep is complete.** All five consumers of the moved symbols were found: `console.go` (4 sites), `conceptInventory`, `conceptPlans`, `#146`'s Core-concepts row flipped to `deleted` so the contract now asserts *absence*, and `artifactpath.NonArtifactSources`. That is the `#188` class caught rather than shipped (ARCH-PURPOSE shadow-sweep: pass).

## 2. Critical findings

None.

## 3. Important findings

**N1 — the probe behind the design's load-bearing measurement does not exist** (`workshop/plans/000199-…-plan.md:150`)

Finding 5 is the plan's own "load-bearing fact… this plan's M1/M3 would have been built on sand," and it closes with *"Probe kept at `scratchpad/199-probe/`."* There is no `scratchpad/` in the tree, nothing matching `*199*probe*` on disk, no `scratchpad/` path ever added in `git log --all --diff-filter=A`, and no `.gitignore` entry that would hide one. Meanwhile `atlas/architecture.md:490-493` now states the result as settled fact for every future reader. ARCH-MOCK at-review: behavior we depend on (zellij's emulator honoring a pane process's DECSTBM) has a one-time manual reading and no reproducible apparatus or conformance check. Fix: land the probe as a tracked package — `cmd/probes/couchstartrecovery` is the repo's own precedent — or correct the plan and atlas to say the method is recorded but the apparatus was not retained.

**N2 — a correction landed at one site and its restatements were not swept** — *This is the 2nd finding in family `plan-table-drift`. Per the escalation, I am not asking for these seven sites to be edited one by one; the deliverable is the rule plus the sweep.*

Three plan-gate findings were disposed `addressed` this round on the strength of one corrected passage each, while the plan continues to assert the superseded fact elsewhere. Measured prevalence: 3 corrections, 7 stale sites, all in `workshop/plans/000199-…-plan.md`.

| corrected in | still says the old thing at |
|---|---|
| finding 7 (`:39-49`) — the seam is `batch.RowDirty` in the Sink callback, not `Child.TakeRowDirty()` | `:271` Integration-points table, **Wraps: `ptychild.Child.TakeRowDirty`**; `:561` M3.4 "repaint on … `TakeRowDirty`"; `:562` M3.5 "on `TakeRowDirty`, re-`Reserve`" |
| finding 9 (`:100-130`) — `run.go:229` is `RegisterTerminalPane`, not a title reader (confirmed at `cmd/internal/termcmd/run.go:226-231`) | `:304` "`#118`'s tab-strip titles and `#123`'s registry read it (`run.go:229`)"; `:391` ARCH-PURPOSE prints the superseded "`run.go:229`, `#118`, `#123`" **and calls it an enumeration "written out rather than left as sweep"** — the exact shape the plan's own PQ-10 rule forbids |
| `rowtext` entry (`:214-222`) — couchtty's `sanitize`/`truncate` are unexported, so "reuse them" was never implementable | `:372-373` ARCH-SECURE "sanitizes and truncates exactly as `couchtty.RenderStatusRow` does (`sanitize`, `truncate`)"; `:560` M3.3 "via the same helpers `couchtty` uses" |

`:271` is the load-bearing one: it is the Integration-points row an M3 implementer reads to learn which seam to wrap, and it still names the method finding 7 proved reads false forever.

**The rule:** when a gate finding corrects a fact, the unit of repair is the fact, not the sentence the finding quoted. Grep the superseded token across the artifact and replace every occurrence in the same round. The checkable form, one line per correction, runnable before the next disposition: `grep -n 'TakeRowDirty\|run\.go:229\|same helpers' workshop/plans/000199-*-plan.md` must return only the passage that documents the correction itself.

**BR-4 remains half-open** (see dispositions): the plan's answer routes every `RunZellijAction` through `RunZellijActionQuiet`, but `runZellij` hardwires `cmd.Stderr = os.Stderr` at `cmd/internal/termcmd/run.go:1104` for *both* methods. So "M2's envelope covers all five" is still false for the subprocess's second descriptor — a failing `zellij action scroll-up` writes into the pane per wheel tick, outside the writer loop and outside the paint gate, and after M4 there is no frame to absorb it.

## 4. Minor findings

- `workshop/plans/000199-…-plan.md:452` — M1.6 is ticked claiming its grep "returns nothing"; run as written it returns 27 lines. The invariant does hold (every hit is a `_test.go` fixture; no production file outside `hostty` emits `SetRegion` or a DECSTBM sequence), but the stated command contradicts the tick. Corrected form: append `| grep -v _test.go`.
- `cmd/internal/hostty/reserve.go:49` — `NewReservation` validates only the edge; `rows` is unchecked, so the "validating door" admits `rows: 0`. Noted under BR-13 rather than separately.

## 5. Test coverage notes

- The new `hostty` tests cover the whole legal state space of `Reservation`: normal (24), both degenerate heights (0, 1), the refused edge through the constructor, and the refused edge *bypassing* the constructor. `TestAnUnvalidatedTopEdgeFailsClosed` is the valuable one — it pins the fail-closed contract that actually protects, given `NewReservation` has no production caller.
- Mutation check performed and reverted; working tree verified clean afterward.
- Gap: `Release()` is unconditional, so a `Reservation{Rows: 1}` that never reserved still emits `ResetRegion`. Faithful to the old `Release()` and harmless (idempotent), but it is the one method with no `usable()` agreement and no test asserting the asymmetry is intentional.
- Gap: `Edge` and `Reservation` are declared `new` in the plan's Core-concepts table but `conceptPackage` is `cmd/internal/couchtty/` only, so M1's actual deliverables are dropped by the package filter and unpinned by the contract. That is the known `#188` shape, not a regression — but it means the contract test's green tells you nothing about the rows this milestone added.

## 6. Architectural notes

Walking the markers explicitly against the diff:

- **ARCH-DRY — pass.** This milestone *is* the DRY consolidation. One implementation of `\x1b[r`; `bottomReservation` gives couch's four call sites a single constructor; `reserve.go` composes from `hostty`'s constants and spells no escape twice.
- **ARCH-PURE — pass.** Pure mechanism in `hostty`, policy (`RenderStatusRow`) left in `couchtty`, IO (`writeOwn`, `io.WriteString`) untouched at the boundary.
- **ARCH-PURPOSE — pass on M1's own scope; flagged for the document.** The consumer shadow-sweep is complete and every consumer derives from the one source. The instance-vs-class lens is where N2 lands: three findings were answered at the site each named while enumerable siblings of the same class stayed in the tree.
- **ARCH-MOCK — flagged (N1).** No new external dependency, and the tty stays behind `hostty.FakeHost` so production and test share the boundary. But the one external behavior the design newly depends on — zellij's emulator honoring a pane's DECSTBM — has no retained apparatus and no conformance cadence.
- **ARCH-CONSTRAINTS — pass.** M1 adds no runtime cost; `Paint` is five string concatenations on a non-hot path. The keystroke-path budget is M2/M3's to enforce.
- **ARCH-SECURE — flagged, Minor (BR-15).** `Paint` writes caller text verbatim between `SaveCursor` and `RestoreCursor` and clamps nothing. Faithful to `PaintRow`, and `RenderStatusRow` sanitizes upstream — but the mechanism/policy split just moved that obligation across a package boundary, and M3's consumer feeds it operator-supplied tab names.
- **ARCH-ORDER — pass.** `Reservation` carries no state between events; `usable()` collapses the `(Edge, Rows)` product into one legal predicate and every method fails closed off it, so there is no boolean constellation to enumerate. `Edge` is an `int`, so `Edge(5)` is representable — and `NewReservation` rejects it by inequality rather than by an allow-list, which is the right default.

## 7. Plan revision recommendations

Two `## Revisions` entries the plan needs before M2 opens:

1. **The `Paint` one-row deviation.** BR-10 asked for the test *and* a Revisions entry; only the test landed (declared at the door in `reserve.go:103-108`, but nowhere in the plan). Add: *"M1 is a move with one deliberate deviation — `couchtty.PaintRow` guarded `hostRows == 0` and drew on a 1-row terminal; `Reservation.Paint` guards `!usable()` and draws nothing, so `Reserve` and `Paint` now agree. This is the residual half of `#146`'s BR-32, pinned by `TestNothingIsPaintedOnARowThatWasNeverReserved`."*
2. **The N2 sweep**, recording the rule and the seven sites so the next round can check it rather than re-derive it — and correcting `:391`'s ARCH-PURPOSE enumeration, which currently contradicts the plan's own PQ-10 rule two hundred lines above it.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Finding 9 records the grep and all three matcher forms; I re-ran it and it reproduces layoutflow.go:56,59,62 and shortcut.go:184-189. Residual restatements at :304 and :391 folded into the new plan-table-drift finding.
  - id: BR-2
    disposition: addressed
    note: |
      M1.5 restated in Revisions to behavioural tests; console_live_test.go and vtscreen_test.go diff empty; conceptPlans, conceptInventory and #146's row all landed and the contract is green.
  - id: BR-3
    disposition: addressed
    note: |
      Finding 7 names batch.RowDirty in the Sink callback and checks out against child.go:176 and run.go:661. Residual at the Integration-points table (:271) and M3.4/M3.5 folded into the new plan-table-drift finding.
  - id: BR-4
    disposition: not-addressed
    note: |
      The stdout half is derived and routed via RunZellijActionQuiet, but runZellij hardwires cmd.Stderr = os.Stderr at run.go:1104 for BOTH methods, so the subprocess's second descriptor is still unrouted and "M2's envelope covers all five" is still false.
  - id: BR-5
    disposition: addressed
    note: |
      M2's preamble names the takeover reset as couch's third rule and maps it to redrawTab; M2.3 step 1 resets hostScan plus the deferred slot. Verified against console.go:992-995.
  - id: BR-6
    disposition: addressed
    note: |
      M4's Files block now names zellij/layouts/main-3.kdl and zellij/config.kdl as the SOURCE, explains the GeneratedMirror, and adds the regenerate step.
  - id: BR-7
    disposition: addressed
    note: |
      rowtext.Sanitize/Fit is named as the shared home with the "unexported in another package" rationale. Residual at ARCH-SECURE (:372-373) and M3.3 (:560) folded into the new plan-table-drift finding.
  - id: BR-8
    disposition: not-addressed
    note: |
      M4.3 still has no Alt+Shift+d step; the six *-split rungs stay unverified. Minor, carried.
  - id: BR-9
    disposition: not-addressed
    note: |
      TestPaintDefersMidSequenceAndIsOwed still splits at one hand-picked index inside "\x1b[3". Minor, carried.
  - id: BR-10
    disposition: addressed
    note: |
      Mutation-verified: reverting Paint's guard to r.Rows == 0 turns reserve_test.go:66 and :101 red. The plan-Revisions half of the ask is still missing and is listed as a plan revision recommendation.
  - id: BR-11
    disposition: addressed
    note: |
      All four strip.go rows flipped to "planned — M3" in 692aa11e; the rowtext row carries the same status.
  - id: BR-12
    disposition: addressed
    note: |
      Plan-gate rounds 5 and 6 are blocked:false, PQ-2 disposed addressed in round 2, and estimate_hours 7.52 plus an itemized Estimate block landed. Only PQ-8/PQ-9 remain open, consistent with BR-8/BR-9.
  - id: BR-13
    disposition: not-addressed
    note: |
      NewReservation still has zero production callers; console.go:922 still builds the struct literal. It also validates only the edge, not rows. Minor, carried.
  - id: BR-14
    disposition: not-addressed
    note: |
      atlas/couch.md:228 "The reserved row" still describes the mechanism with no pointer to hostty.Reservation. Minor, carried.
  - id: BR-15
    disposition: not-addressed
    note: |
      hostty/reserve.go:97-108 documents save/restore and the one-row deviation but still states no caller obligation to sanitize or clamp. Minor, carried.
findings:
  - id: new
    severity: Important
    family: unreproducible-measurement
    title: |
      The probe behind finding 5 -- the design's load-bearing measurement -- does not exist anywhere in the repo
    detail: |
      The plan closes finding 5 with "Probe kept at scratchpad/199-probe/", but there
      is no scratchpad/ on disk, nothing matching *199*probe*, no such path ever added
      in git log --all --diff-filter=A, and no .gitignore entry hiding one. The plan
      calls this "the load-bearing fact" without which M1/M3 "would have been built on
      sand", and atlas/architecture.md:490-493 now states the result as settled fact.
      ARCH-MOCK at-review: behavior we depend on has a one-time manual reading and no
      retained apparatus or conformance check. Land the probe as a tracked package
      (cmd/probes/couchstartrecovery is the repo's own precedent) or correct the plan
      and atlas to say the method is recorded but the apparatus was not kept.
  - id: new
    severity: Important
    family: plan-table-drift
    title: |
      Three gate corrections landed at one site each and left seven restatements of the superseded facts in the same file
    detail: |
      This is the 2nd finding in family plan-table-drift. Do NOT fix these seven sites
      one by one -- the deliverable is the rule and the sweep. Measured prevalence, all
      in workshop/plans/000199-...-plan.md: (1) finding 7 corrects the repaint seam to
      batch.RowDirty, but :271's Integration-points row still names
      ptychild.Child.TakeRowDirty as the wrapped seam and M3.4/M3.5 (:561,:562) still
      say "repaint on TakeRowDirty" -- :271 is what an M3 implementer reads to learn
      which seam to wrap; (2) finding 9 establishes run.go:229 is RegisterTerminalPane
      and not a title reader (confirmed at termcmd/run.go:226-231), but :304 still says
      the #118/#123 consumers "read it (run.go:229)" and :391's ARCH-PURPOSE still
      prints the superseded list AND calls it an enumeration "written out rather than
      left as sweep", which is exactly the shape the plan's own PQ-10 rule forbids;
      (3) the rowtext entry establishes couchtty's sanitize/truncate are unreachable,
      but ARCH-SECURE (:372-373) and M3.3 (:560) still say "exactly as
      couchtty.RenderStatusRow does" / "the same helpers couchtty uses".
      The rule: when a gate finding corrects a fact, the unit of repair is the fact,
      not the sentence the finding quoted -- grep the superseded token across the
      artifact and replace every occurrence in the same round. Checkable form, runnable
      before the next disposition:
      grep -n 'TakeRowDirty\|run\.go:229\|same helpers' workshop/plans/000199-*-plan.md
      must return only the passage documenting the correction itself.
  - id: new
    severity: Minor
    family: acceptance-command-does-not-hold
    title: |
      M1.6 is ticked claiming its grep returns nothing; run as written it returns 27 lines
    detail: |
      plan :452. Every hit is a _test.go fixture and no production file outside hostty
      emits SetRegion or a DECSTBM sequence, so the invariant M1.6 defends does hold --
      but the stated command contradicts the tick, and the next reader to re-run it
      gets output. Corrected form: append "| grep -v _test.go".
```
