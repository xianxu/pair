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
    - "n": 2
      timestamp: "2026-09-07T16:32:56-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: not-addressed
          note: Plan text unchanged; run.go:229 is still RegisterTerminalPane, and layoutflow.go:62 / shortcut.go:190 are still the real matchers.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: M1 landed with the restated behavioural-tests acceptance and swept three consumer lists, one the gate had not named.
          round: 2
        - id: PQ-3
          disposition: not-addressed
          note: child.go:174-177 still clears RowDirty into the batch whenever a sink is set, and run.go:660 always sets one.
          round: 2
        - id: PQ-4
          disposition: not-addressed
          note: run.go:1086/1100-1104 still gives the zellij subprocess os.Stdout; wheel path at :457/:462, resize goroutine at :261.
          round: 2
        - id: PQ-5
          disposition: not-addressed
          note: console.go:992-994's hostScan/paintPending reset is still absent from M2.3's restatement of couch's gate rules.
          round: 2
        - id: PQ-6
          disposition: not-addressed
          note: M4 Files still names the GeneratedMirror (manifest.go:476), not zellij/layouts/main-3.kdl.
          round: 2
        - id: PQ-7
          disposition: not-addressed
          note: sanitize/truncate still unexported at couchtty/reserve.go:148,161; no shared home named.
          round: 2
        - id: PQ-8
          disposition: not-addressed
          note: M4.3 still has no Alt+Shift+d step; the six split-rung sites stay unverified.
          round: 2
        - id: PQ-9
          disposition: not-addressed
          note: TestPaintDefersMidSequenceAndIsOwed still splits one hand-chosen index.
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-09-07T16:39:27-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: not-addressed
          note: Finding 9 reads layoutflow.go but asserts "two lines in one file"; workbenchshortcut/shortcut.go:190 is a third title arm the gate had already named, and M3.6's assertion is verbatim unchanged.
          round: 3
        - id: PQ-3
          disposition: addressed
          note: Finding 7 is correct against child.go:176 and run.go:661; the residual "TakeRowDirty" wording in the integration table and M3.4 is shorthand, not a wrong seam.
          round: 3
        - id: PQ-4
          disposition: not-addressed
          note: 'Finding 8 covers stderr but omits the subprocess: run.go:1088/1104 hands zellij both of the pane''s descriptors on the wheel and rename paths.'
          round: 3
        - id: PQ-5
          disposition: addressed
          note: M2's preamble now names the takeover reset as the third rule and maps it to redrawTab; console.go:992-995 confirmed.
          round: 3
        - id: PQ-6
          disposition: addressed
          note: M4 edits the source and regenerates; the "make test runs runtimebundle-generate first" claim checks out at Makefile.local:120.
          round: 3
        - id: PQ-7
          disposition: addressed
          note: rowtext.Sanitize/Fit is named as the shared home with rationale; M3.3's "same helpers couchtty uses" prose is stale against it.
          round: 3
        - id: PQ-8
          disposition: not-addressed
          note: Minor, carried forward — M4.3 still never splits, and the issue's Done-when still requires two halves each drawing a strip.
          round: 3
        - id: PQ-9
          disposition: not-addressed
          note: Minor, carried forward — one hand-chosen split point remains, against the plan's own "any index" note.
          round: 3
      findings:
        - id: PQ-10
          severity: Important
          title: This is the 3rd finding in family consumer-set-not-derived — fix the rule, not the instance
          detail: |-
            Do NOT fix only the two sites named above. The rule: every consumer set
            this plan enumerates must be produced by a command the plan writes down,
            and its acceptance must assert against the consumer's own decision
            function rather than against the plan's list. Measured prevalence in this
            plan, three instances across two families: title consumers (PQ-1, wrong
            twice — from memory, then a one-file read that missed
            workbenchshortcut/shortcut.go:190); tty writers (PQ-4 — Spec named two,
            finding 8 named four, the actual count is five once run.go:1088's
            subprocess-inherited stdout+stderr is included); and M1's own Revisions,
            which records "three consumer sets needed the move, none of them in the
            plan's file list". Concretely: M3.6 asserts on
            workbenchshortcut.RoleForPane and launcher.ClassifyLiveLayout fed the
            degraded title, including the TerminalCommand=="" case that
            zellijpane.paneFrom admits at zellijpane.go:79-84; M2 states which paths
            may reach the pane's fds and either routes RunZellijAction through
            RunZellijActionQuiet or records it as a named exception, since
            TestOnlyOneGoroutineWritesTheHost cannot observe a subprocess write.
          family: consumer-set-not-derived
          round: 3
      blocked: true
    - "n": 4
      timestamp: "2026-09-07T16:42:31-07:00"
      agent: claude
      blocked: true
      protocol_error: no valid findings block
    - "n": 5
      timestamp: "2026-09-07T16:45:52-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: 'Finding 9 derives the set by grep and names all three matcher forms correctly (verified at layoutflow.go:62 and shortcut.go:186); residual — the M3.6 row and ARCH-PURPOSE still print the superseded "run.go:229, #118, #123" list.'
          round: 5
        - id: PQ-4
          disposition: addressed
          note: Finding 8 derives all five writers, routes the subprocess through RunZellijActionQuiet and admits a goroutine-id test cannot see it; residual — M2.3's implement step does not yet name that routing.
          round: 5
        - id: PQ-10
          disposition: addressed
          note: The rule is stated as its own section and applied to both consumer sets, each carrying its command; M4.1 already enumerates by parsing main-3.kdl rather than from the nine line numbers.
          round: 5
        - id: PQ-8
          disposition: not-addressed
          note: M4.3 still never presses Alt+Shift+d, so the six *-split rungs stay unverified; Minor, carried to the close review.
          round: 5
        - id: PQ-9
          disposition: not-addressed
          note: TestPaintDefersMidSequenceAndIsOwed still splits one hand-chosen index rather than every index of a sequence set; Minor, carried to the close review.
          round: 5
      blocked: false
content_hash: 1000e6d8917e5da78fde5fe59cc4eb828fc617d2c8db06b4f232d1ba8e0a5f30
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

## Round 2 — 2026-09-07T16:32:56-07:00 (claude) — BLOCKED

### Disposed

- PQ-1 — not-addressed — Plan text unchanged; run.go:229 is still RegisterTerminalPane, and layoutflow.go:62 / shortcut.go:190 are still the real matchers.
- PQ-2 — addressed — M1 landed with the restated behavioural-tests acceptance and swept three consumer lists, one the gate had not named.
- PQ-3 — not-addressed — child.go:174-177 still clears RowDirty into the batch whenever a sink is set, and run.go:660 always sets one.
- PQ-4 — not-addressed — run.go:1086/1100-1104 still gives the zellij subprocess os.Stdout; wheel path at :457/:462, resize goroutine at :261.
- PQ-5 — not-addressed — console.go:992-994's hostScan/paintPending reset is still absent from M2.3's restatement of couch's gate rules.
- PQ-6 — not-addressed — M4 Files still names the GeneratedMirror (manifest.go:476), not zellij/layouts/main-3.kdl.
- PQ-7 — not-addressed — sanitize/truncate still unexported at couchtty/reserve.go:148,161; no shared home named.
- PQ-8 — not-addressed — M4.3 still has no Alt+Shift+d step; the six split-rung sites stay unverified.
- PQ-9 — not-addressed — TestPaintDefersMidSequenceAndIsOwed still splits one hand-chosen index.

## Round 3 — 2026-09-07T16:39:27-07:00 (claude) — BLOCKED

### Disposed

- PQ-1 — not-addressed — Finding 9 reads layoutflow.go but asserts "two lines in one file"; workbenchshortcut/shortcut.go:190 is a third title arm the gate had already named, and M3.6's assertion is verbatim unchanged.
- PQ-3 — addressed — Finding 7 is correct against child.go:176 and run.go:661; the residual "TakeRowDirty" wording in the integration table and M3.4 is shorthand, not a wrong seam.
- PQ-4 — not-addressed — Finding 8 covers stderr but omits the subprocess: run.go:1088/1104 hands zellij both of the pane's descriptors on the wheel and rename paths.
- PQ-5 — addressed — M2's preamble now names the takeover reset as the third rule and maps it to redrawTab; console.go:992-995 confirmed.
- PQ-6 — addressed — M4 edits the source and regenerates; the "make test runs runtimebundle-generate first" claim checks out at Makefile.local:120.
- PQ-7 — addressed — rowtext.Sanitize/Fit is named as the shared home with rationale; M3.3's "same helpers couchtty uses" prose is stale against it.
- PQ-8 — not-addressed — Minor, carried forward — M4.3 still never splits, and the issue's Done-when still requires two halves each drawing a strip.
- PQ-9 — not-addressed — Minor, carried forward — one hand-chosen split point remains, against the plan's own "any index" note.

### Raised

- **PQ-10** [Important] `consumer-set-not-derived` This is the 3rd finding in family consumer-set-not-derived — fix the rule, not the instance
  Do NOT fix only the two sites named above. The rule: every consumer set
  this plan enumerates must be produced by a command the plan writes down,
  and its acceptance must assert against the consumer's own decision
  function rather than against the plan's list. Measured prevalence in this
  plan, three instances across two families: title consumers (PQ-1, wrong
  twice — from memory, then a one-file read that missed
  workbenchshortcut/shortcut.go:190); tty writers (PQ-4 — Spec named two,
  finding 8 named four, the actual count is five once run.go:1088's
  subprocess-inherited stdout+stderr is included); and M1's own Revisions,
  which records "three consumer sets needed the move, none of them in the
  plan's file list". Concretely: M3.6 asserts on
  workbenchshortcut.RoleForPane and launcher.ClassifyLiveLayout fed the
  degraded title, including the TerminalCommand=="" case that
  zellijpane.paneFrom admits at zellijpane.go:79-84; M2 states which paths
  may reach the pane's fds and either routes RunZellijAction through
  RunZellijActionQuiet or records it as a named exception, since
  TestOnlyOneGoroutineWritesTheHost cannot observe a subprocess write.

## Round 4 — 2026-09-07T16:42:31-07:00 (claude) — BLOCKED

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 5 — 2026-09-07T16:45:52-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Finding 9 derives the set by grep and names all three matcher forms correctly (verified at layoutflow.go:62 and shortcut.go:186); residual — the M3.6 row and ARCH-PURPOSE still print the superseded "run.go:229, #118, #123" list.
- PQ-4 — addressed — Finding 8 derives all five writers, routes the subprocess through RunZellijActionQuiet and admits a goroutine-id test cannot see it; residual — M2.3's implement step does not yet name that routing.
- PQ-10 — addressed — The rule is stated as its own section and applied to both consumer sets, each carrying its command; M4.1 already enumerates by parsing main-3.kdl rather than from the nine line numbers.
- PQ-8 — not-addressed — M4.3 still never presses Alt+Shift+d, so the six *-split rungs stay unverified; Minor, carried to the close review.
- PQ-9 — not-addressed — TestPaintDefersMidSequenceAndIsOwed still splits one hand-chosen index rather than every index of a sequence set; Minor, carried to the close review.

## Open findings

- **PQ-8** [Minor] `acceptance-misses-changed-sites` M4's manual acceptance never splits the pane, leaving six of nine borderless sites unverified
- **PQ-9** [Minor] `single-interleaving-oracle` The mid-sequence gate test picks one hand-chosen split point over an arbitrary byte stream
