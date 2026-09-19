---
gate: boundary-review
issue: 284
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-19T08:20:19-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: Compaction's Couch refusal strands the validated checkpoint and names an operation that discards it
          detail: |-
            cmd/internal/launcher/compaction.go:105-107 refuses after ContinueCheckpoint
            is read and validated, but prints only the generic "relaunch the thread from
            Couch (Alt+n)" body. Every other failure arm in the same function names the
            retained checkpoint path and the `pair continue --retry <tag>` route (:97,
            :122, :129). Couch's relaunch keeps the conversation and ignores the
            checkpoint, so the operator is pointed at a different operation and never
            told where their continuation went.
          family: refusal-names-the-recovery-route
          round: 1
        - id: BR-2
          severity: Important
          title: The four draft call sites that deliver the notify-on-failure Done-when row have no failing test
          detail: |-
            nvim/lifecycle_command_test.lua tests the module in isolation; nothing
            asserts PairConfirmQuit / PairConfirmDetach / pair_confirm_restart_impl /
            PairConfirmAgentRestart route through _G._pair_lifecycle.run
            (nvim/init.lua:3088, 3098, 3194, 3217). Reverting any one site to
            vim.fn.system leaves the suite green -- the exact class of bug this issue
            exists to kill. Makefile.local:282-341 already drives the real init.lua
            headlessly, so stubbing the helper and asserting each entry point calls it
            is a few lines.
          family: glue-wiring-untested
          round: 1
        - id: BR-3
          severity: Minor
          title: couchRestartGate is called with a hardcoded false for sessionEnvHosted in runCompaction
          detail: |-
            cmd/internal/launcher/compaction.go:105 is correct only because the
            opts.Env.CouchHosted() arm above always returns. Passing
            opts.Env.CouchHosted() costs nothing and keeps the gate correct if that arm
            ever gains a fall-through.
          family: constant-restates-a-caller-invariant
          round: 1
        - id: BR-4
          severity: Minor
          title: The unreadable-record refusal names no remedy
          detail: |-
            cmd/internal/launcher/outerrecord.go:78 fails closed (correct) but tells the
            operator nothing about which artifact is unreadable or that reattaching
            through `pair` rewrites it, so a corrupt record makes a standalone Alt+n
            look permanently broken.
          family: refusal-names-the-recovery-route
          round: 1
        - id: BR-5
          severity: Minor
          title: Garbled comment in keyscmd's hosted-wording branch
          detail: |-
            cmd/internal/keyscmd/keyscmd.go:90 reads "The rule `pair restart` refuses by
            (#284), so the hosted wording is the true one".
          family: comment-accuracy
          round: 1
        - id: BR-6
          severity: Minor
          title: lifecycle_command.lua reads the vim global while taking notify as a dep
          detail: |-
            nvim/lifecycle_command.lua:19 uses vim.log.levels.ERROR directly; the
            sibling nvim/confirm_quit.lua touches no global, so the module can only run
            under `nvim -l`.
          family: module-purity-deps
          round: 1
        - id: BR-7
          severity: Minor
          title: atlas's confirm-modals entry does not mention the new failure-reporting rule
          detail: |-
            atlas/architecture.md:816 still describes the four modals as plain
            shell-outs, with no mention of the shared nvim/lifecycle_command.lua seam or
            the "every confirmed lifecycle command reports its own failure" rule. The
            hosting note (:320-330) and the paste note (:487) were updated correctly.
          family: docs-follow-new-surface
          round: 1
      recipe: milestone-review
      blocked: true
    - "n": 2
      timestamp: "2026-09-19T08:36:06-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: The refusal now names the checkpoint path, but it offers `pair continue --retry <tag>`, which prepareContinuationRetry refuses without a retained v1 restart marker. This arm returns before WriteRestartMarker; a scratch test got "no retained continuation restart intent". Name `pair continue --checkpoint <SourcePath>` once the session is gone, drop "relaunched thread" for the unreadable-record case, and pin the route with a test (3rd in family refusal-names-the-recovery-route; the rule is in the review body).
          round: 2
        - id: BR-2
          disposition: addressed
          note: tests/lifecycle-command-nvim-test.sh drives the real init.lua; reverting the detach or restart call site in a scratch copy gives 2 FAILs each; wired into make test.
          round: 2
        - id: BR-3
          disposition: addressed
          note: compaction.go:108 now passes opts.Env.CouchHosted().
          round: 2
        - id: BR-4
          disposition: addressed
          note: The error names the outer-tty record (or its path from ReadFile) and the remedy (re-attaching with `pair` rewrites it); restart_test's unreadable case asserts the refusal.
          round: 2
        - id: BR-5
          disposition: addressed
          note: keyscmd.go:90-91 now reads coherently.
          round: 2
        - id: BR-6
          disposition: addressed
          note: lifecycle_command.lua takes error_level as a dep and touches no vim global; the Lua test pins it with a sentinel level.
          round: 2
        - id: BR-7
          disposition: addressed
          note: atlas/architecture.md:816 names the shared seam, the report-your-own-failure rule and the wiring test.
          round: 2
      findings:
        - id: BR-8
          severity: Minor
          title: lifecycle_test.go:119-124 still says `pair restart` does not refuse in the adopted case and Alt+n ends the thread
          detail: 'This is the 2nd finding in family comment-accuracy. Rule: when an issue changes a behavior, every in-tree statement citing that issue or describing the old behavior is part of the fix; run `git grep -n ''#284''` and reread each hit for tense. That sweep over cmd/nvim/tests/atlas/README found 27 hits and only this one stale. Rewrite it to say the gate now refuses first and this client-side refusal is the backstop.'
          family: comment-accuracy
          round: 2
        - id: BR-9
          severity: Minor
          title: The Spec says Alt+n joins the focus-after-prefix tests, but TestLifecycleCandidateUsesFocusAfterPrefix still sends only Alt+x
          detail: onRelaunchHotkey reads focus when the chord is handled, so Ctrl+Space followed by Alt+n in one read should relaunch the highlighted pager row, not the thread on screen. Nothing pins that ordering (ARCH-ORDER). Add Alt+n to that test's chord set, or revise the Spec sentence.
          family: design-claim-needs-its-test
          round: 2
      recipe: milestone-review
      blocked: true
    - "n": 3
      timestamp: "2026-09-19T08:55:34-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: compaction.go:109 now names the retained checkpoint and `pair continue --checkpoint <path>`; TestCompactionRefusalNamesARouteThatRuns pins no-marker + route parses + path resolves and validates, and goes red on the old --retry text.
          round: 3
        - id: BR-8
          disposition: addressed
          note: 'lifecycle_test.go:119 now says the client refusal is the BACKSTOP; I re-ran the #284 sweep (24 code hits plus README/atlas) and found no further stale tense.'
          round: 3
        - id: BR-9
          disposition: addressed
          note: TestLifecycleCandidateUsesFocusAfterPrefix is now a chord table carrying Alt+n, asserting the actor arm confirms against the thread the prefix left focused.
          round: 3
      findings:
        - id: BR-10
          severity: Minor
          title: Two comments claim the help and the restart gate can never disagree; they diverge on the unreadable record
          detail: |-
            This is the 3rd finding in family `comment-accuracy`. Do not fix the two
            sites alone. The rule: when two consumers of one predicate have DIFFERENT
            failure policies, a comment may not assert they always agree -- it must
            name the divergence and why. Measured prevalence here: 2 of the 3
            consumers of CouchOwnsRestart carry the overstated claim
            (outerrecord.go:63 "the help cannot promise a reload that the gate
            refuses"; keyscmd.go:90 "appears exactly when Pair's Alt+n cannot
            reload"), while atlas/architecture.md states it correctly. The behavior
            both deny is tested on both sides: keyscmd
            TestUnreadablePresenterRendersPairsPage (fail open, standalone reload row)
            and launcher TestRunRestartRefusesACouchOwnedSessionBeforeMutation's
            "unreadable presenter record" case (fail closed). The Spec chose this
            split deliberately, so the code should say so.
          family: comment-accuracy
          round: 3
        - id: BR-11
          severity: Minor
          title: couchRestartGate skips the presenter read entirely when the tag is unresolved, so that state fails open
          detail: |-
            outerrecord.go:76 guards the read with `!sessionEnvHosted && tag != ""`.
            An unreadable record refuses (fail closed), but an unresolvable tag
            neither decides nor refuses: runRestart proceeds to write the marker,
            the quit intent and the kill. Reached when PAIR_TAG is unset AND
            TagForSessionName misses (a scoped session name absent from the index --
            the `pair-<tag>` legacy prefix always resolves), with Couch presenting
            the client: exactly the thread-ending outcome this issue exists to
            prevent. Narrow, and the comment shows it was considered, but the two
            states are epistemically identical ("cannot tell whether Couch presents
            this session") and get opposite policies. Either refuse there too, with
            the same message, or have the comment say why proceeding is safe rather
            than only that the env half still applies.
          family: undecidable-input-fails-open
          round: 3
        - id: BR-12
          severity: Minor
          title: The lessons rule for this family says "a command that exists today", which does not catch BR-1's failure mode
          detail: |-
            This is the 3rd finding in family `refusal-names-the-recovery-route`
            (BR-4, BR-1, this). Do not fix an instance -- both are already fixed. The
            rule exists at workshop/lessons.md:2330 (from #146) but is scoped to
            EXISTENCE: `pair continue --retry` exists, is a declared verb, and would
            pass its suggested "assert the suggested command is in the verb set"
            test -- yet it could not run, because the arm returns before writing the
            marker --retry consumes. Sharpen that rule to cover preconditions: a
            remedy must be runnable FROM THE STATE THE REFUSAL LEAVES BEHIND, and the
            test must assert the state the remedy needs (here: the retained
            checkpoint resolves and validates), not merely that the verb is spelled
            correctly. compaction_test.go's TestCompactionRefusalNamesARouteThatRuns
            is the model to cite.
          family: refusal-names-the-recovery-route
          round: 3
      recipe: milestone-review
      blocked: false
---

# Gate ledger — pair#284 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-19T08:20:19-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `refusal-names-the-recovery-route` Compaction's Couch refusal strands the validated checkpoint and names an operation that discards it
  cmd/internal/launcher/compaction.go:105-107 refuses after ContinueCheckpoint
  is read and validated, but prints only the generic "relaunch the thread from
  Couch (Alt+n)" body. Every other failure arm in the same function names the
  retained checkpoint path and the `pair continue --retry <tag>` route (:97,
  :122, :129). Couch's relaunch keeps the conversation and ignores the
  checkpoint, so the operator is pointed at a different operation and never
  told where their continuation went.
- **BR-2** [Important] `glue-wiring-untested` The four draft call sites that deliver the notify-on-failure Done-when row have no failing test
  nvim/lifecycle_command_test.lua tests the module in isolation; nothing
  asserts PairConfirmQuit / PairConfirmDetach / pair_confirm_restart_impl /
  PairConfirmAgentRestart route through _G._pair_lifecycle.run
  (nvim/init.lua:3088, 3098, 3194, 3217). Reverting any one site to
  vim.fn.system leaves the suite green -- the exact class of bug this issue
  exists to kill. Makefile.local:282-341 already drives the real init.lua
  headlessly, so stubbing the helper and asserting each entry point calls it
  is a few lines.
- **BR-3** [Minor] `constant-restates-a-caller-invariant` couchRestartGate is called with a hardcoded false for sessionEnvHosted in runCompaction
  cmd/internal/launcher/compaction.go:105 is correct only because the
  opts.Env.CouchHosted() arm above always returns. Passing
  opts.Env.CouchHosted() costs nothing and keeps the gate correct if that arm
  ever gains a fall-through.
- **BR-4** [Minor] `refusal-names-the-recovery-route` The unreadable-record refusal names no remedy
  cmd/internal/launcher/outerrecord.go:78 fails closed (correct) but tells the
  operator nothing about which artifact is unreadable or that reattaching
  through `pair` rewrites it, so a corrupt record makes a standalone Alt+n
  look permanently broken.
- **BR-5** [Minor] `comment-accuracy` Garbled comment in keyscmd's hosted-wording branch
  cmd/internal/keyscmd/keyscmd.go:90 reads "The rule `pair restart` refuses by
  (#284), so the hosted wording is the true one".
- **BR-6** [Minor] `module-purity-deps` lifecycle_command.lua reads the vim global while taking notify as a dep
  nvim/lifecycle_command.lua:19 uses vim.log.levels.ERROR directly; the
  sibling nvim/confirm_quit.lua touches no global, so the module can only run
  under `nvim -l`.
- **BR-7** [Minor] `docs-follow-new-surface` atlas's confirm-modals entry does not mention the new failure-reporting rule
  atlas/architecture.md:816 still describes the four modals as plain
  shell-outs, with no mention of the shared nvim/lifecycle_command.lua seam or
  the "every confirmed lifecycle command reports its own failure" rule. The
  hosting note (:320-330) and the paste note (:487) were updated correctly.

## Round 2 — 2026-09-19T08:36:06-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — The refusal now names the checkpoint path, but it offers `pair continue --retry <tag>`, which prepareContinuationRetry refuses without a retained v1 restart marker. This arm returns before WriteRestartMarker; a scratch test got "no retained continuation restart intent". Name `pair continue --checkpoint <SourcePath>` once the session is gone, drop "relaunched thread" for the unreadable-record case, and pin the route with a test (3rd in family refusal-names-the-recovery-route; the rule is in the review body).
- BR-2 — addressed — tests/lifecycle-command-nvim-test.sh drives the real init.lua; reverting the detach or restart call site in a scratch copy gives 2 FAILs each; wired into make test.
- BR-3 — addressed — compaction.go:108 now passes opts.Env.CouchHosted().
- BR-4 — addressed — The error names the outer-tty record (or its path from ReadFile) and the remedy (re-attaching with `pair` rewrites it); restart_test's unreadable case asserts the refusal.
- BR-5 — addressed — keyscmd.go:90-91 now reads coherently.
- BR-6 — addressed — lifecycle_command.lua takes error_level as a dep and touches no vim global; the Lua test pins it with a sentinel level.
- BR-7 — addressed — atlas/architecture.md:816 names the shared seam, the report-your-own-failure rule and the wiring test.

### Raised

- **BR-8** [Minor] `comment-accuracy` lifecycle_test.go:119-124 still says `pair restart` does not refuse in the adopted case and Alt+n ends the thread
  This is the 2nd finding in family comment-accuracy. Rule: when an issue changes a behavior, every in-tree statement citing that issue or describing the old behavior is part of the fix; run `git grep -n '#284'` and reread each hit for tense. That sweep over cmd/nvim/tests/atlas/README found 27 hits and only this one stale. Rewrite it to say the gate now refuses first and this client-side refusal is the backstop.
- **BR-9** [Minor] `design-claim-needs-its-test` The Spec says Alt+n joins the focus-after-prefix tests, but TestLifecycleCandidateUsesFocusAfterPrefix still sends only Alt+x
  onRelaunchHotkey reads focus when the chord is handled, so Ctrl+Space followed by Alt+n in one read should relaunch the highlighted pager row, not the thread on screen. Nothing pins that ordering (ARCH-ORDER). Add Alt+n to that test's chord set, or revise the Spec sentence.

## Round 3 — 2026-09-19T08:55:34-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — compaction.go:109 now names the retained checkpoint and `pair continue --checkpoint <path>`; TestCompactionRefusalNamesARouteThatRuns pins no-marker + route parses + path resolves and validates, and goes red on the old --retry text.
- BR-8 — addressed — lifecycle_test.go:119 now says the client refusal is the BACKSTOP; I re-ran the #284 sweep (24 code hits plus README/atlas) and found no further stale tense.
- BR-9 — addressed — TestLifecycleCandidateUsesFocusAfterPrefix is now a chord table carrying Alt+n, asserting the actor arm confirms against the thread the prefix left focused.

### Raised

- **BR-10** [Minor] `comment-accuracy` Two comments claim the help and the restart gate can never disagree; they diverge on the unreadable record
  This is the 3rd finding in family `comment-accuracy`. Do not fix the two
  sites alone. The rule: when two consumers of one predicate have DIFFERENT
  failure policies, a comment may not assert they always agree -- it must
  name the divergence and why. Measured prevalence here: 2 of the 3
  consumers of CouchOwnsRestart carry the overstated claim
  (outerrecord.go:63 "the help cannot promise a reload that the gate
  refuses"; keyscmd.go:90 "appears exactly when Pair's Alt+n cannot
  reload"), while atlas/architecture.md states it correctly. The behavior
  both deny is tested on both sides: keyscmd
  TestUnreadablePresenterRendersPairsPage (fail open, standalone reload row)
  and launcher TestRunRestartRefusesACouchOwnedSessionBeforeMutation's
  "unreadable presenter record" case (fail closed). The Spec chose this
  split deliberately, so the code should say so.
- **BR-11** [Minor] `undecidable-input-fails-open` couchRestartGate skips the presenter read entirely when the tag is unresolved, so that state fails open
  outerrecord.go:76 guards the read with `!sessionEnvHosted && tag != ""`.
  An unreadable record refuses (fail closed), but an unresolvable tag
  neither decides nor refuses: runRestart proceeds to write the marker,
  the quit intent and the kill. Reached when PAIR_TAG is unset AND
  TagForSessionName misses (a scoped session name absent from the index --
  the `pair-<tag>` legacy prefix always resolves), with Couch presenting
  the client: exactly the thread-ending outcome this issue exists to
  prevent. Narrow, and the comment shows it was considered, but the two
  states are epistemically identical ("cannot tell whether Couch presents
  this session") and get opposite policies. Either refuse there too, with
  the same message, or have the comment say why proceeding is safe rather
  than only that the env half still applies.
- **BR-12** [Minor] `refusal-names-the-recovery-route` The lessons rule for this family says "a command that exists today", which does not catch BR-1's failure mode
  This is the 3rd finding in family `refusal-names-the-recovery-route`
  (BR-4, BR-1, this). Do not fix an instance -- both are already fixed. The
  rule exists at workshop/lessons.md:2330 (from #146) but is scoped to
  EXISTENCE: `pair continue --retry` exists, is a declared verb, and would
  pass its suggested "assert the suggested command is in the verb set"
  test -- yet it could not run, because the arm returns before writing the
  marker --retry consumes. Sharpen that rule to cover preconditions: a
  remedy must be runnable FROM THE STATE THE REFUSAL LEAVES BEHIND, and the
  test must assert the state the remedy needs (here: the retained
  checkpoint resolves and validates), not merely that the verb is spelled
  correctly. compaction_test.go's TestCompactionRefusalNamesARouteThatRuns
  is the model to cite.

## Open findings

- **BR-10** [Minor] `comment-accuracy` Two comments claim the help and the restart gate can never disagree; they diverge on the unreadable record
- **BR-11** [Minor] `undecidable-input-fails-open` couchRestartGate skips the presenter read entirely when the tag is unresolved, so that state fails open
- **BR-12** [Minor] `refusal-names-the-recovery-route` The lessons rule for this family says "a command that exists today", which does not catch BR-1's failure mode
