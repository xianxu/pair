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

---

## Re-review — 2026-09-07T17:21:53-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 199 — pair term: own the right pane tab bar |
| repo | pair |
| issue file | workshop/issues/000199-pair-term-own-the-right-pane-tab-bar.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | c61f5ca0619d59e3ba795b77d6e12ad18e5517f5..76397937614321b2f3a5cc44266cead61301b3a8 |
| command | sdlc milestone-close --issue 199 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-07T17:21:53-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The Go code in this window is genuinely good: the lift to `hostty.Reservation` is faithful, the one deliberate behaviour change is now declared *and* pinned by a test that fails without it, the deadlock fix is at the right level, the `\x1b[r`-lives-in-one-package invariant holds when I run M1.6's corrected grep (0 lines), and `go test ./cmd/...` is clean apart from the known sandbox pty refusals. What blocks the boundary is not the code — it is that BR-17 was raised with a *runnable acceptance command* and the commit named for sweeping it leaves that command returning violations, including the exact `scratchpad/199-probe/` string BR-16 quoted, still standing at line 149 of the plan in the same commit that landed the probe at `cmd/probes/zellijscrollregion`. BR-4 landed the same way: six writers derived in the prose, but M2.3/M2.3b — the steps an implementer executes — still carry only the stdout routing. And the probe that BR-16 asked for, now the sole apparatus behind this design's load-bearing fact, ships a data race on the buffer its verdict reads and an unguarded slice that can panic before the verdict prints. M2/M3 read the plan and re-run the probe; both are wrong in ways their own gate findings already named.

## 1. Strengths

- **The deliberate deviation is declared and mutation-proof.** `hostty/reserve.go:107-116` documents the `Rows == 1` change from `PaintRow`, and `TestNothingIsPaintedOnARowThatWasNeverReserved` (`reserve_test.go:108-118`) reddens the instant the guard is reverted to `r.Rows == 0` — last round's I1 is a real fix with a real failing test behind it, not a plausible-looking diff.
- **The move is proved by an oracle the implementor could not tune.** `git diff --stat c61f5ca0 76397937 -- cmd/internal/couchtty/{console_live,vtscreen}_test.go` is empty, and those are the end-to-end reserved-row tests. That is the Done-when's "a regression there means it was a rewrite", made checkable.
- **M1.6's invariant now actually checks itself.** I ran the corrected form: zero production `SetRegion`/DECSTBM sites outside `hostty`. BR-18 addressed, and the plan records *why* the filter is load-bearing rather than just adding it.
- **The consumer-set sweep for the moved symbols is complete** — `console.go` ×4, `conceptInventory`, `conceptPlans`, `#146`'s row flipped `deleted` (I confirmed the `deleted` branch is live: `\bReserve\b`/`\bPaintRow\b` are genuinely absent from `couchtty/reserve.go`), `artifactpath.NonArtifactSources`, `.gitignore`. `TestEveryMainPackageIsIgnoredAtTheRepoRoot` passes with the new probe in the tree.
- **`hostty/reserve.go` imports only `fmt`** and its tests need no fake, no fixture, no IO. ARCH-PURE: pass.

## 2. Critical findings

None.

## 3. Important findings

**BR-17 re-raised — the sweep swept the sentences the finding quoted, not the facts.** BR-17's own stated acceptance was `grep -n 'TakeRowDirty\|run\.go:229\|same helpers' workshop/plans/000199-*-plan.md` returning only the correction passage. It returns `:568` — M3.3, still "sanitizing and truncating via the same helpers `couchtty` uses", the exact prose PQ-7 established is unimplementable. Measured prevalence, all in `workshop/plans/000199-…-plan.md`, seven live sites:

1. `:149` — finding 5 still ends "Probe kept at `scratchpad/199-probe/`", in the commit that landed the probe at `cmd/probes/zellijscrollregion`.
2. `:59-64` — finding 8's writer table has **five** rows while `:67` and `:83` say "six writers"; the sixth (subprocess stderr) exists only in prose.
3. `:380` — ARCH-SECURE still says `RenderStrip` sanitizes "exactly as `couchtty.RenderStatusRow` does (`sanitize`, `truncate`)".
4. `:568` — M3.3, as above.
5. `:571` — M3.6's assertion is still the weak "a rename still fires and is not the packed string" that PQ-1 said passes while the format stops matching; `:114` claims "M3.6 asserts exactly this" about `RoleForPane`/`ClassifyLiveLayout` and the `TerminalCommand == ""` case.
6. `:505-523` — M2.3's four pieces never mention giving `runZellij` a `stderr io.Writer`, although `:650` claims they do.
7. `:210` + `:216` — the Core-concepts table keeps `couchtty/reserve.go` `unchanged` while the `rowtext` entry says both renderers depend on the new package.

Fix the rule, not the sites: before disposing a correction, run the superseded token across the artifact and require the grep to return only the passage that documents the correction — and treat "the narrative says X, the numbered step says Y" as the drift, because the step is what gets executed.

**BR-4 re-raised — the envelope claim is still carried by prose, not by a step.** `runZellij` (`cmd/internal/termcmd/run.go:1099-1105`) hardwires `cmd.Stderr = os.Stderr` for both methods; I confirmed it. Finding 8 now names this correctly, but M2.3's four implement pieces route only stdout through `Quiet`, and M2.3b asserts only "no `termcmd` site calls `RunZellijAction`" — an assertion a `Quiet`-only refactor passes while the noisiest failure path stays open. Add the fifth piece to M2.3 (`runZellij(args, stdout, stderr io.Writer)`, captured and logged) and extend M2.3b to assert the fd handed to the subprocess, not just the method name.

**N1 — the probe races on the buffer its verdict reads** (`cmd/probes/zellijscrollregion/main.go:143-153, 177-197`). `var seen strings.Builder` is written by the reader goroutine (`:149`) and read by main at `:177`, `:180-181` and `:197` with no synchronization. `strings.Builder` is explicitly not safe for concurrent use; `String()` hands out the backing slice while `Write` may be growing it. `go run -race` would flag this, and a torn read produces exactly the failure mode the probe's own doc comment says it was fixed for three times: a confident wrong verdict. Guard with a `sync.Mutex` (or have the goroutine own a `bytes.Buffer` and hand the final string over a channel on EOF), and give the goroutine a cancellation path so its extent is bounded by `main`.

**N2 — an unused diagnostic can panic before the verdict prints** (`main.go:200`). `frame[len(frame)-3000:]` has no length guard; a pty that yields fewer than 3000 bytes panics. `tailOf` (`:86-91`) is the safe helper this file already has, forty lines up (ARCH-DRY). Two further problems in the same line: the value is only printed, never used in the verdict, and it searches for `"scroll line 0 "` with a trailing space against a stream where `probe.sh` emits `scroll line 0\n` — so whether it can ever be true depends on zellij's row padding. It reported `false` in the recorded run, and the plan (`:143`) quotes that `false` as part of finding 5's evidence. Either make it a real check or delete it; do not leave an unverified line in the output block the atlas cites.

**N3 — the atlas describes one probe home and the code now has two** (`atlas/index.md:17-22`, family `atlas-points-at-old-home`, 2nd in family). **This is the 2nd finding in family `atlas-points-at-old-home`.** Earlier rounds have not fixed instances — BR-14 is still open — so do NOT fix these two sites one by one; state the rule and sweep it. The rule: an atlas statement about *where a class of thing lives and what runs it* must be re-derived whenever the code adds or moves a home, and the derivation belongs in the atlas rather than the answer. Measured prevalence, three sites: `atlas/index.md:17-22` says probes live in `probes/` and "`make test-smoke` runs **every** directory under `probes/`, so a new probe is covered by existing" — the new probe is under `cmd/probes/`, which that sentence does not reach, and `grep -n "smoke\|probes" Makefile` returns nothing at all, so the target the sentence rests on does not exist; and `atlas/couch.md:228` (BR-14) still presents the reserved row as couch-owned mechanism with no pointer to `hostty.Reservation`.

## 4. Minor findings

- `main.go:120,142,156` — every `os.Exit` path (`:178`, `:190`, `:209`) skips the deferred `zellij delete-session` and `os.Remove(layout)`, so an inconclusive run leaks a live zellij session with a `sleep 30` pane plus a temp KDL. Wrap the body in a `run() int` and `os.Exit(run())`.
- `main.go:186-192` — `dump-screen`'s output is discarded (`_ = out`) but its failure aborts the probe with `PROBE-ERROR`; a call that contributes nothing to the verdict should not be able to kill a good measurement.
- `plan.md:596-601` — M4.4 asks for the `atlas/architecture.md` row-primitive entry that M1 already landed in `0aa2b6af`; only the `config.kdl` half remains.
- Line references in finding 8 drift by 1-2 (`run.go:1086` → 1085/1087, `:1091` → 1090, `:1100-1105` → 1099-1105). The five `RunZellijAction` call sites (191, 457, 463, 961, 963) are exactly right.

## 5. Test coverage notes

`./cmd/internal/hostty/` reservation tests: 8 tests, all IO-free, all green. The fail-closed path (`TestAnUnvalidatedTopEdgeFailsClosed`) and the one-row deviation (`TestNothingIsPaintedOnARowThatWasNeverReserved`) are both real oracles — reverting either guard reddens them. `TestCoreConceptsContract` enumerates both new `#199` rows and its `deleted` branch is reachable. Full `go test ./cmd/...`: every failure is "operation not permitted" (sandbox pty), plus `pairhelp_shim_test.go:51` and `agent_restart_test.go:84`, both process-spawn dependent and untouched by this diff.

Gaps: the new probe has no test and nothing runs it; `Edge`/`Reservation` are declared PURE in the plan's Core-concepts table but `conceptRowsForPackage` filters to `couchtty` paths, so no contract pins them — the same silent-guard shape last round's I2 found, now applying to the rows this milestone shipped. M3's `termcmd` rows will land in the same hole (`#188`).

## 6. Architectural notes for upcoming work

- **ARCH-DRY:** pass on the lift (M1.6 verified). Flagged on the probe (N2, `tailOf`).
- **ARCH-PURE:** pass. `reserve.go` is `fmt`-only; the `Reservation` value replaces four functions that each re-decided the edge.
- **ARCH-PURPOSE:** the shadow-sweep over the moved symbols is complete. Flagged on the plan (BR-17) — the class was named and the instances were fixed one at a time again.
- **ARCH-MOCK:** improved. The apparatus is tracked and self-locating (BR-16 addressed), but nothing re-runs it and there is no conformance cadence; N1/N2 mean the apparatus itself can now be the source of a wrong reading. Before M3 leans on finding 5 again, run it under `-race`.
- **ARCH-CONSTRAINTS:** pass for M1 (pure string composition, no runtime path changed). The probe's fixed 8-second window fails safe (`PROBE-INCONCLUSIVE`, exit 2) — good.
- **ARCH-SECURE:** flagged, BR-15 still open. `Paint` (`hostty/reserve.go:107-124`) writes caller text verbatim between `SaveCursor` and `RestoreCursor` and its doc names no sanitize/clamp obligation. That obligation just crossed a package boundary away from `couchtty.RenderStatusRow`, and M3's consumer takes operator-supplied tab names. One doc line at the new door.
- **ARCH-ORDER:** pass on the lift — `bottomReservation(rows uint16)` (`console.go:914-922`) makes the deadlocking call-site state unrepresentable rather than merely avoided, which is the right shape. Flagged on the probe: unsynchronized shared state (N1), a reader goroutine with no cancellation, and cleanup that `os.Exit` bypasses.

## 7. Plan revision recommendations

1. **A `## Revisions` entry stating the BR-17 rule as an executable check**, not another site-list: name the superseded tokens (`scratchpad/199-probe`, `same helpers`, `run.go:229`, `TakeRowDirty`, "five writers"), record the grep, and require it to return only correction passages before any round disposes a fact-correcting finding. Then sweep the seven sites above.
2. **Correct finding 5's closing line** (`:149`) to `cmd/probes/zellijscrollregion`, and either drop `earliest scroll line still on screen` from the quoted output or make it a checked assertion.
3. **Extend M2.3 with a fifth piece** — `runZellij` takes `stderr io.Writer`; captured output is logged, never written to the pane fd — and extend M2.3b to assert the descriptor handed to the subprocess. Update finding 8's table to six rows so the table and the prose agree.
4. **Rewrite M3.6's assertion** to the one finding 9 already claims it makes: `RoleForPane` and `ClassifyLiveLayout` fed the degraded title, including the `TerminalCommand == ""` case at `zellijpane.go:79-84`.
5. **M4.4:** drop the `atlas/architecture.md` half (landed in M1); keep the `config.kdl` scroll-indicator correction.
6. **Add an `atlas/` step for the probe home** — one sentence reconciling `probes/` and `cmd/probes/`, and either restore `make test-smoke` or delete the claim that it covers new probes automatically.

```findings
dispose:
  - id: BR-4
    disposition: not-addressed
    note: |
      Finding 8 now derives six writers, but M2.3's four pieces and M2.3b still cover stdout only; run.go:1103 still hardwires cmd.Stderr = os.Stderr for both methods.
  - id: BR-8
    disposition: not-addressed
    note: |
      M4.3 (plan :601-605) is verbatim unchanged; still no Alt+Shift+d step, and the issue Done-when still requires two halves each drawing a strip.
  - id: BR-9
    disposition: not-addressed
    note: |
      M2.1's TestPaintDefersMidSequenceAndIsOwed still splits one hand-picked index inside "\x1b[3".
  - id: BR-13
    disposition: not-addressed
    note: |
      console.go:922 still builds Reservation{} literally; the round instead added rows validation to NewReservation, so the door and the production path now carry different contracts for the same type.
  - id: BR-14
    disposition: not-addressed
    note: |
      atlas/couch.md:228 is unchanged; only atlas/architecture.md was touched in this window.
  - id: BR-15
    disposition: not-addressed
    note: |
      hostty/reserve.go:107-124 Paint still documents save/restore and the one-row deviation but names no caller obligation to sanitize or clamp.
  - id: BR-16
    disposition: addressed
    note: |
      Probe landed at cmd/probes/zellijscrollregion, self-locating, with artifactpath and .gitignore updated; see N1/N2 for defects in the landed apparatus.
  - id: BR-17
    disposition: not-addressed
    note: |
      The finding's own acceptance grep still returns M3.3 (:568); seven superseded facts remain, including the scratchpad path at :149 in the commit that landed the probe.
  - id: BR-18
    disposition: addressed
    note: |
      M1.6 now carries "| grep -v _test.go | grep -v hostty/"; I ran it and it returns nothing.
findings:
  - id: new
    severity: Important
    family: shared-state-unsynchronized
    title: |
      The probe's reader goroutine and its verdict share a strings.Builder with no synchronization
    detail: |
      cmd/probes/zellijscrollregion/main.go:143-153 writes `seen` from a reader
      goroutine while main reads it at :177, :180-181 and :197.
      strings.Builder is not safe for concurrent use; String() exposes the
      backing slice while Write may be growing it, so `go run -race` flags this
      and a torn read yields a confident wrong verdict -- the exact failure class
      the probe's own doc comment says it hit three times. The goroutine also has
      no cancellation path, so its extent is not bounded by main. Guard with a
      mutex, or have the goroutine own the buffer and hand the final string over
      a channel at EOF.
  - id: new
    severity: Important
    family: external-input-assumed-wellformed
    title: |
      An unused diagnostic slices the pty stream without a bounds check and can panic before the verdict prints
    detail: |
      cmd/probes/zellijscrollregion/main.go:200 evaluates
      frame[len(frame)-3000:] with no length guard, so a pty yielding fewer than
      3000 bytes panics on the line before the landmarks are checked. tailOf
      (:86-91) is the safe helper this same file already defines (ARCH-DRY). The
      value is never used in the verdict, and it matches "scroll line 0 " with a
      trailing space against a stream where probe.sh emits "scroll line 0\n" --
      yet its `false` output is quoted as evidence in the plan (:143) behind an
      atlas claim. Delete it or make it a checked assertion.
  - id: new
    severity: Important
    family: atlas-points-at-old-home
    title: |
      This is the 2nd finding in family atlas-points-at-old-home -- the atlas names one probe home and the code now has two
    detail: |
      Do NOT fix these sites one at a time; BR-14 is still open from the last
      round, which is the family reporting that the enumeration was never
      written. The rule: an atlas statement about where a class of thing lives
      and what runs it must be re-derived whenever the code adds or moves a home,
      and the atlas records the derivation rather than the answer. Measured
      prevalence, three sites: atlas/index.md:17-22 says probes live in probes/
      and "make test-smoke runs every directory under probes/, so a new probe is
      covered by existing" -- the new probe is under cmd/probes/, which that
      sentence does not reach, and grep -n "smoke\|probes" Makefile returns
      nothing, so the target it rests on does not exist; atlas/couch.md:228
      (BR-14) still presents the reserved row as couch-owned mechanism with no
      pointer to hostty.Reservation.
  - id: new
    severity: Minor
    family: exit-path-drops-cleanup
    title: |
      Every os.Exit path in the probe skips its deferred session and temp-file cleanup
    detail: |
      cmd/probes/zellijscrollregion/main.go registers cleanup at :120, :142 and
      :156, then exits via os.Exit at :178, :190 and :209 -- so an inconclusive
      or errored run leaks a live zellij session with a `sleep 30` pane and a
      temp KDL. Wrap the body in `run() int` and call os.Exit(run()). Same file,
      :186-192: dump-screen's output is discarded (`_ = out`) yet its failure
      aborts the probe, letting an irrelevant call kill a good measurement.
```
