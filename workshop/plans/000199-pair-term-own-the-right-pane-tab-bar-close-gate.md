---
gate: boundary-review
issue: 199
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-07T16:30:11-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Important
          title: The rename-pane consumer set is asserted from memory; the two real title matchers are never named
          detail: |-
            The plan enumerates consumers as "run.go:229, #118, #123", but run.go:229 is
            the RegisterTerminalPane comment, not a title reader. The actual matchers are
            launcher/layoutflow.go:62 (HasPrefix(pane.Title, "[terminal") — the bracket is
            part of the format M3.6 removes) and workbenchshortcut/shortcut.go:186
            (HasPrefix(title, "terminal ")). M3.6's assertion only checks that a rename
            still fires and is not the packed string, so it passes while the format stops
            matching. Derive the set by grep and assert against RoleForPane /
            ClassifyLiveLayout directly.
            (carried from plan-quality PQ-1, deferred to the boundary review)
          family: consumer-set-not-derived
          round: 1
        - id: BR-2
          severity: Important
          title: M1.5's "no couch test edited" is unsatisfiable, and the concept contract binds the moved symbols
          detail: |-
            couchtty/reserve_test.go:15-58 calls Reserve/ChildRows/PaintRow/Release
            directly, so the package will not compile after M1.4 deletes them.
            core_concepts_contract_test.go:45 pins those symbols and resolves their
            declared path from workshop/history/plans/000146-...-plan.md:88
            (cmd/internal/couchtty/reserve.go), so TestCoreConceptsContract fails until
            that row is marked deleted and #199 is registered in conceptPlans. Restate the
            acceptance as "no behavioral couch test edited" and list both files as M1 work.
            (carried from plan-quality PQ-2, deferred to the boundary review)
          family: consumer-set-not-derived
          round: 1
        - id: BR-3
          severity: Important
          title: The strip repaint trigger names ptychild.Child.TakeRowDirty, which is always false in termcmd
          detail: |-
            Child.outputBatchLocked take-and-clears rowDirty into batch.RowDirty whenever a
            sink is set (ptychild/child.go:176), and termcmd always sets one
            (run.go:660). couch reads ch.batch.RowDirty (couchtty/console.go:1135). ptyChunk
            (run.go:600-603) has no RowDirty field and the sink drops it at run.go:661, so
            M3 needs a struct field, not a method call. Failure is silent: the strip
            vanishes on nvim's startup clear and never returns — a Done-when bullet.
            (carried from plan-quality PQ-3, deferred to the boundary review)
          family: wrong-seam-named
          round: 1
        - id: BR-4
          severity: Important
          title: Two writers to the pane's tty sit outside the single-writer envelope, and the M2 test cannot see them
          detail: |-
            OSRuntime.RunZellijAction gives the zellij subprocess os.Stdout (run.go:1088)
            with Stderr = os.Stderr (run.go:1103) — a writer on the wheel path
            (run.go:453-463) and the rename path, outside the writer loop and outside the
            paint gate; RunZellijActionQuiet already exists. The resize goroutine
            (run.go:262) becomes a writer in M3 but is outside M2's stated routing scope. A
            goroutine-id wrapper around stdout sees neither, so
            TestOnlyOneGoroutineWritesTheHost stays green over both.
            (carried from plan-quality PQ-4, deferred to the boundary review)
          family: envelope-claim-unenforced
          round: 1
        - id: BR-5
          severity: Important
          title: M2 restates two of couch's three gate rules; the takeover reset is the missing one
          detail: |-
            couchtty/console.go:981-988 resets hostScan and clears paintPending on a screen
            takeover, because the old child's partial sequence is no longer on screen to
            corrupt. redrawTab is exactly a takeover (HomeAndClear plus a different child's
            replay), so without the reset the gate frames one child's pending bytes against
            another child's stream — the same class of wrong answer as BR-21.
            (carried from plan-quality PQ-5, deferred to the boundary review)
          family: takeover-resets-framing
          round: 1
        - id: BR-6
          severity: Important
          title: M4 modifies the generated mirror of main-3.kdl and config.kdl rather than their source
          detail: |-
            cmd/internal/runtimebundle/assets/runtime/files/zellij/layouts/main-3.kdl and
            .../zellij/config.kdl are Kind GeneratedMirror
            (artifactpath/manifest.go:472-474), regenerated from zellij/layouts/main-3.kdl
            and zellij/config.kdl by make runtimebundle-generate (Makefile.local:138) and
            checked by artifactpath/coverage_test.go:152-184. The files are byte-identical
            today so the nine line numbers hold, but the edit must land on the source and
            the mirror must be regenerated. termcmd/run_test.go:616 is the precedent for
            reading the root layout from a test.
            (carried from plan-quality PQ-6, deferred to the boundary review)
          family: edit-source-not-mirror
          round: 1
        - id: BR-7
          severity: Important
          title: RenderStrip cannot reuse couchtty's sanitize/truncate — they are unexported in another package
          detail: |-
            The plan's ARCH-SECURE and ARCH-DRY sections both rest on "the same helpers
            couchtty uses", but sanitize and truncate are unexported in package couchtty
            (reserve.go) and the Core-concepts table keeps that file otherwise unchanged.
            As written the implementer duplicates them, which is the second escape-stripping
            policy the plan says it is avoiding. Name the shared home (textwidth already
            owns Width; ansi.Strip is the framing dependency).
            (carried from plan-quality PQ-7, deferred to the boundary review)
          family: shared-helper-not-reachable
          round: 1
        - id: BR-8
          severity: Minor
          title: M4's manual acceptance never splits the pane, leaving six of nine borderless sites unverified
          detail: |-
            :138/:139, :165/:166 and :191/:192 are the *-split rungs and only take effect
            after Alt+Shift+d. The Done-when bullet "two right-pane halves each draw their
            own strip" has no step anywhere in the plan, and divider loss in a split is the
            exact failure that caused the 2026-07-27 frameless revert. One keystroke added
            to M4.3.
            (carried from plan-quality PQ-8, deferred to the boundary review)
          family: acceptance-misses-changed-sites
          round: 1
        - id: BR-9
          severity: Minor
          title: The mid-sequence gate test picks one hand-chosen split point over an arbitrary byte stream
          detail: |-
            The plan's own ARCH-ORDER note says a test can split an escape sequence at any
            index; TestPaintDefersMidSequenceAndIsOwed splits one. Parameterize over every
            index of a representative sequence set — the hand-picked case is by construction
            blind to the boundary the author did not think of.
            (carried from plan-quality PQ-9, deferred to the boundary review)
          family: single-interleaving-oracle
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-09-07T16:30:11-07:00"
      agent: claude
      findings:
        - id: BR-10
          severity: Important
          title: Paint's new 1-row behaviour is an undeclared, unpinned change from the moved PaintRow
          detail: |-
            cmd/internal/hostty/reserve.go:102 guards on !usable() (Rows > 1) where
            PaintRow guarded only hostRows == 0, so a 1-row host now draws nothing
            instead of drawing over the child's only line. This is the right change --
            it is the residual half of pair#146's BR-32 -- but the milestone claims a
            MOVE and nothing pins it. Mutation-verified: restoring the old height
            behaviour keeps ./cmd/internal/hostty and ./cmd/internal/couchtty fully
            green. TestDegenerateHeightsNeverProduceAZeroRowChild covers ChildRows and
            Reserve at rows 0 and 1 but never Paint. Add the Paint assertion to that
            loop and record the deviation in the plan's Revisions.
          family: uncovered-negative-assertion
          round: 2
        - id: BR-11
          severity: Important
          title: Four Core-concepts rows claim `new` at cmd/internal/termcmd/strip.go, which does not exist
          detail: |-
            plan lines 114-117 declare TabChip, StripModel, RenderedStrip and
            RenderStrip as `new` at a file that is not in the tree; they are M3 work.
            core_concepts_contract_test.go documents the convention this breaks --
            unshipped rows carry `planned` so the status column doubles as the build
            tracker -- and this milestone is what registered #199 in conceptPlans. The
            contract passes only because the couchtty package filter drops those rows,
            so the guard is silent rather than satisfied. Flip the four rows to
            `planned — M3`.
          family: plan-table-drift
          round: 2
        - id: BR-12
          severity: Important
          title: M1 code landed with the plan-quality gate still blocked and no estimate recorded
          detail: |-
            The plan-gate ledger's only round is `blocked: true` with all nine PQ
            findings open, and the issue's estimate_hours is empty with no Estimate
            block -- so change-code appears not to have cleared. PQ-2 was the only
            M1-scoped finding and its remediation did ship here, but the ledger still
            reports it open, which makes the open set wrong entering M2. PQ-3 in
            particular contradicts the plan's own Integration-points row before M3 is
            written. Re-run the gate so PQ-2 disposes `addressed`, and land the
            estimate. Not a defect in M1's code.
          family: gate-not-cleared-before-code
          round: 2
        - id: BR-13
          severity: Minor
          title: NewReservation has zero production callers; couch constructs the struct directly
          detail: |-
            hostty/reserve.go:49 is the validating constructor, but couchtty/console.go:921
            builds Reservation{} literally. The exported struct makes the door optional
            and the fail-closed methods are what actually protect. Either construct
            through NewReservation at the couch site or drop it and let usable() carry
            the contract alone.
          family: validating-door-bypassed
          round: 2
        - id: BR-14
          severity: Minor
          title: atlas/couch.md's "The reserved row" section still reads as couch-owned mechanism
          detail: |-
            atlas/couch.md:228 describes the reservation with no pointer to
            hostty.Reservation, while architecture.md carries the new note. That
            section is where a reader lands for this subject; add one cross-reference.
          family: atlas-points-at-old-home
          round: 2
        - id: BR-15
          severity: Minor
          title: Paint's doc does not state the caller's obligation to sanitize and clamp its text
          detail: |-
            hostty/reserve.go:97 writes caller text verbatim between SaveCursor and
            RestoreCursor. Faithful to PaintRow, and couchtty.RenderStatusRow sanitizes
            upstream, but the mechanism/policy split just moved that obligation across
            a package boundary and M3 adds a consumer whose input is operator-supplied
            tab names. One doc line at the new door (ARCH-SECURE).
          family: caller-obligation-undocumented
          round: 2
      boundary: M1
      blocked: true
---

# Gate ledger — pair#199 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-07T16:30:11-07:00 (sdlc) — passed

### Raised

- **BR-1** [Important] `consumer-set-not-derived` The rename-pane consumer set is asserted from memory; the two real title matchers are never named
  The plan enumerates consumers as "run.go:229, #118, #123", but run.go:229 is
  the RegisterTerminalPane comment, not a title reader. The actual matchers are
  launcher/layoutflow.go:62 (HasPrefix(pane.Title, "[terminal") — the bracket is
  part of the format M3.6 removes) and workbenchshortcut/shortcut.go:186
  (HasPrefix(title, "terminal ")). M3.6's assertion only checks that a rename
  still fires and is not the packed string, so it passes while the format stops
  matching. Derive the set by grep and assert against RoleForPane /
  ClassifyLiveLayout directly.
  (carried from plan-quality PQ-1, deferred to the boundary review)
- **BR-2** [Important] `consumer-set-not-derived` M1.5's "no couch test edited" is unsatisfiable, and the concept contract binds the moved symbols
  couchtty/reserve_test.go:15-58 calls Reserve/ChildRows/PaintRow/Release
  directly, so the package will not compile after M1.4 deletes them.
  core_concepts_contract_test.go:45 pins those symbols and resolves their
  declared path from workshop/history/plans/000146-...-plan.md:88
  (cmd/internal/couchtty/reserve.go), so TestCoreConceptsContract fails until
  that row is marked deleted and #199 is registered in conceptPlans. Restate the
  acceptance as "no behavioral couch test edited" and list both files as M1 work.
  (carried from plan-quality PQ-2, deferred to the boundary review)
- **BR-3** [Important] `wrong-seam-named` The strip repaint trigger names ptychild.Child.TakeRowDirty, which is always false in termcmd
  Child.outputBatchLocked take-and-clears rowDirty into batch.RowDirty whenever a
  sink is set (ptychild/child.go:176), and termcmd always sets one
  (run.go:660). couch reads ch.batch.RowDirty (couchtty/console.go:1135). ptyChunk
  (run.go:600-603) has no RowDirty field and the sink drops it at run.go:661, so
  M3 needs a struct field, not a method call. Failure is silent: the strip
  vanishes on nvim's startup clear and never returns — a Done-when bullet.
  (carried from plan-quality PQ-3, deferred to the boundary review)
- **BR-4** [Important] `envelope-claim-unenforced` Two writers to the pane's tty sit outside the single-writer envelope, and the M2 test cannot see them
  OSRuntime.RunZellijAction gives the zellij subprocess os.Stdout (run.go:1088)
  with Stderr = os.Stderr (run.go:1103) — a writer on the wheel path
  (run.go:453-463) and the rename path, outside the writer loop and outside the
  paint gate; RunZellijActionQuiet already exists. The resize goroutine
  (run.go:262) becomes a writer in M3 but is outside M2's stated routing scope. A
  goroutine-id wrapper around stdout sees neither, so
  TestOnlyOneGoroutineWritesTheHost stays green over both.
  (carried from plan-quality PQ-4, deferred to the boundary review)
- **BR-5** [Important] `takeover-resets-framing` M2 restates two of couch's three gate rules; the takeover reset is the missing one
  couchtty/console.go:981-988 resets hostScan and clears paintPending on a screen
  takeover, because the old child's partial sequence is no longer on screen to
  corrupt. redrawTab is exactly a takeover (HomeAndClear plus a different child's
  replay), so without the reset the gate frames one child's pending bytes against
  another child's stream — the same class of wrong answer as BR-21.
  (carried from plan-quality PQ-5, deferred to the boundary review)
- **BR-6** [Important] `edit-source-not-mirror` M4 modifies the generated mirror of main-3.kdl and config.kdl rather than their source
  cmd/internal/runtimebundle/assets/runtime/files/zellij/layouts/main-3.kdl and
  .../zellij/config.kdl are Kind GeneratedMirror
  (artifactpath/manifest.go:472-474), regenerated from zellij/layouts/main-3.kdl
  and zellij/config.kdl by make runtimebundle-generate (Makefile.local:138) and
  checked by artifactpath/coverage_test.go:152-184. The files are byte-identical
  today so the nine line numbers hold, but the edit must land on the source and
  the mirror must be regenerated. termcmd/run_test.go:616 is the precedent for
  reading the root layout from a test.
  (carried from plan-quality PQ-6, deferred to the boundary review)
- **BR-7** [Important] `shared-helper-not-reachable` RenderStrip cannot reuse couchtty's sanitize/truncate — they are unexported in another package
  The plan's ARCH-SECURE and ARCH-DRY sections both rest on "the same helpers
  couchtty uses", but sanitize and truncate are unexported in package couchtty
  (reserve.go) and the Core-concepts table keeps that file otherwise unchanged.
  As written the implementer duplicates them, which is the second escape-stripping
  policy the plan says it is avoiding. Name the shared home (textwidth already
  owns Width; ansi.Strip is the framing dependency).
  (carried from plan-quality PQ-7, deferred to the boundary review)
- **BR-8** [Minor] `acceptance-misses-changed-sites` M4's manual acceptance never splits the pane, leaving six of nine borderless sites unverified
  :138/:139, :165/:166 and :191/:192 are the *-split rungs and only take effect
  after Alt+Shift+d. The Done-when bullet "two right-pane halves each draw their
  own strip" has no step anywhere in the plan, and divider loss in a split is the
  exact failure that caused the 2026-07-27 frameless revert. One keystroke added
  to M4.3.
  (carried from plan-quality PQ-8, deferred to the boundary review)
- **BR-9** [Minor] `single-interleaving-oracle` The mid-sequence gate test picks one hand-chosen split point over an arbitrary byte stream
  The plan's own ARCH-ORDER note says a test can split an escape sequence at any
  index; TestPaintDefersMidSequenceAndIsOwed splits one. Parameterize over every
  index of a representative sequence set — the hand-picked case is by construction
  blind to the boundary the author did not think of.
  (carried from plan-quality PQ-9, deferred to the boundary review)

## Round 2 — 2026-09-07T16:30:11-07:00 (claude) — BLOCKED

### Raised

- **BR-10** [Important] `uncovered-negative-assertion` Paint's new 1-row behaviour is an undeclared, unpinned change from the moved PaintRow
  cmd/internal/hostty/reserve.go:102 guards on !usable() (Rows > 1) where
  PaintRow guarded only hostRows == 0, so a 1-row host now draws nothing
  instead of drawing over the child's only line. This is the right change --
  it is the residual half of pair#146's BR-32 -- but the milestone claims a
  MOVE and nothing pins it. Mutation-verified: restoring the old height
  behaviour keeps ./cmd/internal/hostty and ./cmd/internal/couchtty fully
  green. TestDegenerateHeightsNeverProduceAZeroRowChild covers ChildRows and
  Reserve at rows 0 and 1 but never Paint. Add the Paint assertion to that
  loop and record the deviation in the plan's Revisions.
- **BR-11** [Important] `plan-table-drift` Four Core-concepts rows claim `new` at cmd/internal/termcmd/strip.go, which does not exist
  plan lines 114-117 declare TabChip, StripModel, RenderedStrip and
  RenderStrip as `new` at a file that is not in the tree; they are M3 work.
  core_concepts_contract_test.go documents the convention this breaks --
  unshipped rows carry `planned` so the status column doubles as the build
  tracker -- and this milestone is what registered #199 in conceptPlans. The
  contract passes only because the couchtty package filter drops those rows,
  so the guard is silent rather than satisfied. Flip the four rows to
  `planned — M3`.
- **BR-12** [Important] `gate-not-cleared-before-code` M1 code landed with the plan-quality gate still blocked and no estimate recorded
  The plan-gate ledger's only round is `blocked: true` with all nine PQ
  findings open, and the issue's estimate_hours is empty with no Estimate
  block -- so change-code appears not to have cleared. PQ-2 was the only
  M1-scoped finding and its remediation did ship here, but the ledger still
  reports it open, which makes the open set wrong entering M2. PQ-3 in
  particular contradicts the plan's own Integration-points row before M3 is
  written. Re-run the gate so PQ-2 disposes `addressed`, and land the
  estimate. Not a defect in M1's code.
- **BR-13** [Minor] `validating-door-bypassed` NewReservation has zero production callers; couch constructs the struct directly
  hostty/reserve.go:49 is the validating constructor, but couchtty/console.go:921
  builds Reservation{} literally. The exported struct makes the door optional
  and the fail-closed methods are what actually protect. Either construct
  through NewReservation at the couch site or drop it and let usable() carry
  the contract alone.
- **BR-14** [Minor] `atlas-points-at-old-home` atlas/couch.md's "The reserved row" section still reads as couch-owned mechanism
  atlas/couch.md:228 describes the reservation with no pointer to
  hostty.Reservation, while architecture.md carries the new note. That
  section is where a reader lands for this subject; add one cross-reference.
- **BR-15** [Minor] `caller-obligation-undocumented` Paint's doc does not state the caller's obligation to sanitize and clamp its text
  hostty/reserve.go:97 writes caller text verbatim between SaveCursor and
  RestoreCursor. Faithful to PaintRow, and couchtty.RenderStatusRow sanitizes
  upstream, but the mechanism/policy split just moved that obligation across
  a package boundary and M3 adds a consumer whose input is operator-supplied
  tab names. One doc line at the new door (ARCH-SECURE).

## Open findings

- **BR-1** [Important] `consumer-set-not-derived` The rename-pane consumer set is asserted from memory; the two real title matchers are never named
- **BR-2** [Important] `consumer-set-not-derived` M1.5's "no couch test edited" is unsatisfiable, and the concept contract binds the moved symbols
- **BR-3** [Important] `wrong-seam-named` The strip repaint trigger names ptychild.Child.TakeRowDirty, which is always false in termcmd
- **BR-4** [Important] `envelope-claim-unenforced` Two writers to the pane's tty sit outside the single-writer envelope, and the M2 test cannot see them
- **BR-5** [Important] `takeover-resets-framing` M2 restates two of couch's three gate rules; the takeover reset is the missing one
- **BR-6** [Important] `edit-source-not-mirror` M4 modifies the generated mirror of main-3.kdl and config.kdl rather than their source
- **BR-7** [Important] `shared-helper-not-reachable` RenderStrip cannot reuse couchtty's sanitize/truncate — they are unexported in another package
- **BR-8** [Minor] `acceptance-misses-changed-sites` M4's manual acceptance never splits the pane, leaving six of nine borderless sites unverified
- **BR-9** [Minor] `single-interleaving-oracle` The mid-sequence gate test picks one hand-chosen split point over an arbitrary byte stream
- **BR-10** [Important] `uncovered-negative-assertion` Paint's new 1-row behaviour is an undeclared, unpinned change from the moved PaintRow
- **BR-11** [Important] `plan-table-drift` Four Core-concepts rows claim `new` at cmd/internal/termcmd/strip.go, which does not exist
- **BR-12** [Important] `gate-not-cleared-before-code` M1 code landed with the plan-quality gate still blocked and no estimate recorded
- **BR-13** [Minor] `validating-door-bypassed` NewReservation has zero production callers; couch constructs the struct directly
- **BR-14** [Minor] `atlas-points-at-old-home` atlas/couch.md's "The reserved row" section still reads as couch-owned mechanism
- **BR-15** [Minor] `caller-obligation-undocumented` Paint's doc does not state the caller's obligation to sanitize and clamp its text
