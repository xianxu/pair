---
gate: plan-quality
issue: 199
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-06T21:56:34-07:00"
      agent: claude
      findings:
        - id: PQ-1
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
          family: consumer-set-not-derived
          round: 1
        - id: PQ-2
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
          family: consumer-set-not-derived
          round: 1
        - id: PQ-3
          severity: Important
          title: The strip repaint trigger names ptychild.Child.TakeRowDirty, which is always false in termcmd
          detail: |-
            Child.outputBatchLocked take-and-clears rowDirty into batch.RowDirty whenever a
            sink is set (ptychild/child.go:176), and termcmd always sets one
            (run.go:660). couch reads ch.batch.RowDirty (couchtty/console.go:1135). ptyChunk
            (run.go:600-603) has no RowDirty field and the sink drops it at run.go:661, so
            M3 needs a struct field, not a method call. Failure is silent: the strip
            vanishes on nvim's startup clear and never returns — a Done-when bullet.
          family: wrong-seam-named
          round: 1
        - id: PQ-4
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
          family: envelope-claim-unenforced
          round: 1
        - id: PQ-5
          severity: Important
          title: M2 restates two of couch's three gate rules; the takeover reset is the missing one
          detail: |-
            couchtty/console.go:981-988 resets hostScan and clears paintPending on a screen
            takeover, because the old child's partial sequence is no longer on screen to
            corrupt. redrawTab is exactly a takeover (HomeAndClear plus a different child's
            replay), so without the reset the gate frames one child's pending bytes against
            another child's stream — the same class of wrong answer as BR-21.
          family: takeover-resets-framing
          round: 1
        - id: PQ-6
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
          family: edit-source-not-mirror
          round: 1
        - id: PQ-7
          severity: Important
          title: RenderStrip cannot reuse couchtty's sanitize/truncate — they are unexported in another package
          detail: |-
            The plan's ARCH-SECURE and ARCH-DRY sections both rest on "the same helpers
            couchtty uses", but sanitize and truncate are unexported in package couchtty
            (reserve.go) and the Core-concepts table keeps that file otherwise unchanged.
            As written the implementer duplicates them, which is the second escape-stripping
            policy the plan says it is avoiding. Name the shared home (textwidth already
            owns Width; ansi.Strip is the framing dependency).
          family: shared-helper-not-reachable
          round: 1
        - id: PQ-8
          severity: Minor
          title: M4's manual acceptance never splits the pane, leaving six of nine borderless sites unverified
          detail: |-
            :138/:139, :165/:166 and :191/:192 are the *-split rungs and only take effect
            after Alt+Shift+d. The Done-when bullet "two right-pane halves each draw their
            own strip" has no step anywhere in the plan, and divider loss in a split is the
            exact failure that caused the 2026-07-27 frameless revert. One keystroke added
            to M4.3.
          family: acceptance-misses-changed-sites
          round: 1
        - id: PQ-9
          severity: Minor
          title: The mid-sequence gate test picks one hand-chosen split point over an arbitrary byte stream
          detail: |-
            The plan's own ARCH-ORDER note says a test can split an escape sequence at any
            index; TestPaintDefersMidSequenceAndIsOwed splits one. Parameterize over every
            index of a representative sequence set — the hand-picked case is by construction
            blind to the boundary the author did not think of.
          family: single-interleaving-oracle
          round: 1
      blocked: true
---

# Gate ledger — pair#199 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-06T21:56:34-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `consumer-set-not-derived` The rename-pane consumer set is asserted from memory; the two real title matchers are never named
  The plan enumerates consumers as "run.go:229, #118, #123", but run.go:229 is
  the RegisterTerminalPane comment, not a title reader. The actual matchers are
  launcher/layoutflow.go:62 (HasPrefix(pane.Title, "[terminal") — the bracket is
  part of the format M3.6 removes) and workbenchshortcut/shortcut.go:186
  (HasPrefix(title, "terminal ")). M3.6's assertion only checks that a rename
  still fires and is not the packed string, so it passes while the format stops
  matching. Derive the set by grep and assert against RoleForPane /
  ClassifyLiveLayout directly.
- **PQ-2** [Important] `consumer-set-not-derived` M1.5's "no couch test edited" is unsatisfiable, and the concept contract binds the moved symbols
  couchtty/reserve_test.go:15-58 calls Reserve/ChildRows/PaintRow/Release
  directly, so the package will not compile after M1.4 deletes them.
  core_concepts_contract_test.go:45 pins those symbols and resolves their
  declared path from workshop/history/plans/000146-...-plan.md:88
  (cmd/internal/couchtty/reserve.go), so TestCoreConceptsContract fails until
  that row is marked deleted and #199 is registered in conceptPlans. Restate the
  acceptance as "no behavioral couch test edited" and list both files as M1 work.
- **PQ-3** [Important] `wrong-seam-named` The strip repaint trigger names ptychild.Child.TakeRowDirty, which is always false in termcmd
  Child.outputBatchLocked take-and-clears rowDirty into batch.RowDirty whenever a
  sink is set (ptychild/child.go:176), and termcmd always sets one
  (run.go:660). couch reads ch.batch.RowDirty (couchtty/console.go:1135). ptyChunk
  (run.go:600-603) has no RowDirty field and the sink drops it at run.go:661, so
  M3 needs a struct field, not a method call. Failure is silent: the strip
  vanishes on nvim's startup clear and never returns — a Done-when bullet.
- **PQ-4** [Important] `envelope-claim-unenforced` Two writers to the pane's tty sit outside the single-writer envelope, and the M2 test cannot see them
  OSRuntime.RunZellijAction gives the zellij subprocess os.Stdout (run.go:1088)
  with Stderr = os.Stderr (run.go:1103) — a writer on the wheel path
  (run.go:453-463) and the rename path, outside the writer loop and outside the
  paint gate; RunZellijActionQuiet already exists. The resize goroutine
  (run.go:262) becomes a writer in M3 but is outside M2's stated routing scope. A
  goroutine-id wrapper around stdout sees neither, so
  TestOnlyOneGoroutineWritesTheHost stays green over both.
- **PQ-5** [Important] `takeover-resets-framing` M2 restates two of couch's three gate rules; the takeover reset is the missing one
  couchtty/console.go:981-988 resets hostScan and clears paintPending on a screen
  takeover, because the old child's partial sequence is no longer on screen to
  corrupt. redrawTab is exactly a takeover (HomeAndClear plus a different child's
  replay), so without the reset the gate frames one child's pending bytes against
  another child's stream — the same class of wrong answer as BR-21.
- **PQ-6** [Important] `edit-source-not-mirror` M4 modifies the generated mirror of main-3.kdl and config.kdl rather than their source
  cmd/internal/runtimebundle/assets/runtime/files/zellij/layouts/main-3.kdl and
  .../zellij/config.kdl are Kind GeneratedMirror
  (artifactpath/manifest.go:472-474), regenerated from zellij/layouts/main-3.kdl
  and zellij/config.kdl by make runtimebundle-generate (Makefile.local:138) and
  checked by artifactpath/coverage_test.go:152-184. The files are byte-identical
  today so the nine line numbers hold, but the edit must land on the source and
  the mirror must be regenerated. termcmd/run_test.go:616 is the precedent for
  reading the root layout from a test.
- **PQ-7** [Important] `shared-helper-not-reachable` RenderStrip cannot reuse couchtty's sanitize/truncate — they are unexported in another package
  The plan's ARCH-SECURE and ARCH-DRY sections both rest on "the same helpers
  couchtty uses", but sanitize and truncate are unexported in package couchtty
  (reserve.go) and the Core-concepts table keeps that file otherwise unchanged.
  As written the implementer duplicates them, which is the second escape-stripping
  policy the plan says it is avoiding. Name the shared home (textwidth already
  owns Width; ansi.Strip is the framing dependency).
- **PQ-8** [Minor] `acceptance-misses-changed-sites` M4's manual acceptance never splits the pane, leaving six of nine borderless sites unverified
  :138/:139, :165/:166 and :191/:192 are the *-split rungs and only take effect
  after Alt+Shift+d. The Done-when bullet "two right-pane halves each draw their
  own strip" has no step anywhere in the plan, and divider loss in a split is the
  exact failure that caused the 2026-07-27 frameless revert. One keystroke added
  to M4.3.
- **PQ-9** [Minor] `single-interleaving-oracle` The mid-sequence gate test picks one hand-chosen split point over an arbitrary byte stream
  The plan's own ARCH-ORDER note says a test can split an escape sequence at any
  index; TestPaintDefersMidSequenceAndIsOwed splits one. Parameterize over every
  index of a representative sequence set — the hand-picked case is by construction
  blind to the boundary the author did not think of.

## Open findings

- **PQ-1** [Important] `consumer-set-not-derived` The rename-pane consumer set is asserted from memory; the two real title matchers are never named
- **PQ-2** [Important] `consumer-set-not-derived` M1.5's "no couch test edited" is unsatisfiable, and the concept contract binds the moved symbols
- **PQ-3** [Important] `wrong-seam-named` The strip repaint trigger names ptychild.Child.TakeRowDirty, which is always false in termcmd
- **PQ-4** [Important] `envelope-claim-unenforced` Two writers to the pane's tty sit outside the single-writer envelope, and the M2 test cannot see them
- **PQ-5** [Important] `takeover-resets-framing` M2 restates two of couch's three gate rules; the takeover reset is the missing one
- **PQ-6** [Important] `edit-source-not-mirror` M4 modifies the generated mirror of main-3.kdl and config.kdl rather than their source
- **PQ-7** [Important] `shared-helper-not-reachable` RenderStrip cannot reuse couchtty's sanitize/truncate — they are unexported in another package
- **PQ-8** [Minor] `acceptance-misses-changed-sites` M4's manual acceptance never splits the pane, leaving six of nine borderless sites unverified
- **PQ-9** [Minor] `single-interleaving-oracle` The mid-sequence gate test picks one hand-chosen split point over an arbitrary byte stream
