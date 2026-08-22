---
gate: plan-quality
issue: 146
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-08-22T12:44:44-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: Intercept's signature can neither express the split point nor satisfy its own split-boundary test
          detail: |-
            `(in []byte) (forward []byte, hits int)` concatenates the bytes before and
            after the hotkey, so `x<ctrl-space>y` cannot route `x` to the old focus and
            `y` to the new; and test 2.3(c) (NUL inside an escape payload) needs framing
            state a stateless per-read function cannot carry, contradicting Decision 10
            and the plan's own split-boundary rule in 1.3(d). `workbenchshortcut.FindChord`
            (shortcut.go:342-352) already returns (before, chord, raw, rest, ok) and
            termcmd's pump already carries partials in `held` (run.go:343, 399, 457) —
            adopt that shape or drop 2.3(c) with a stated reason (ARCH-DRY).
          family: stream-split-contract
          round: 1
        - id: PQ-2
          severity: Important
          title: Console's host-tty and signal boundary is neither named as a seam nor deduplicated from termcmd
          detail: |-
            M1 extracts the child half only; Console re-implements MakeRaw
            (termcmd/run.go:222), SIGWINCH -> GetsizeFull (:244, :975-983), Restore, and
            the `\x1b[r` region reset that termcmd.restoreTerminal already owns
            (:1107-1109) — two sites for one sequence, against the plan's own
            one-constant rule. With no injectable host terminal or signal source in the
            type inventory, Task 2.5's "fakes, no real tty" tests and Task 4.4's signal
            path test cannot be written (ARCH-PURE, ARCH-MOCK, ARCH-DRY). The plan also
            does not say that couchcmd/run.go:171-178 (the StartResult branch that prints
            a line and then blocks on Handle.Wait) is what the console displaces, or who
            constructs and drives the Console.
          family: io-seam-unnamed
          round: 1
        - id: PQ-3
          severity: Important
          title: PanelModel.Filter restates LookupTrees' match rule, and the stated field list is wrong
          detail: |-
            LookupTrees matches name + description only — NamingTable.Lookup scans Name
            and Description (naming.go:44-57) and LookupTrees adds the agent-published
            description via Describe (couch.go:196-220, 276-281). Repo is not matched;
            path resolution lives one layer up in ResolveRef, behind an ActorID exact
            branch (couch.go:317-340). A restated Filter either diverges from the CLI or
            duplicates the rule, and makes false the claim that #148's advisor calls the
            same resolution. Have the panel call LookupTrees, or extract a pure matcher
            both use (ARCH-DRY).
          family: resolution-single-source
          round: 1
        - id: PQ-4
          severity: Important
          title: Root actor is defined two ways, and the smokes verify the weaker one
          detail: |-
            Decision 1 equates "child of the first `couch start`" with the Spec's "the
            session couch launched in", but under it the M2 smoke (`couch start ../pair`
            from brain, Task 2.7) makes pair the root actor and brain — the advisor, what
            the project calls home — is never a couch child, so Task 3.5 verifies the
            project's headline property against a stand-in. The plan also never says what
            becomes of the shell couch was launched from (no key leaves couch, so it is
            blocked until exit). State the definition, the ordering it implies for making
            brain home, and the launching session's fate.
          family: home-actor-contract
          round: 1
        - id: PQ-5
          severity: Minor
          title: The alt-screen resize nudge only fires if the size actually changes
          detail: |-
            TIOCSWINSZ raises SIGWINCH only when the new winsize differs from the stored
            one (both Darwin and Linux), so Decision 5's nudge is a rows-1 -> rows-2 ->
            rows-1 round trip and a visible double reflow of the whole zellij workbench.
            Say that is accepted, or choose a different redraw trigger.
          family: resize-nudge-mechanism
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-08-22T12:51:51-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Interceptor.Feed(in) (before, hit, rest) adopts FindChord's shape; Decision 10 names bracketed paste as the one place framing state is real and Task 2.3 tests the split marker and the pasted-NUL loss.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: New hostty package (Host seam, OSHost/FakeHost, control constants incl. the moved \x1b[r) with termcmd's host half migrated in Task 1.4a; Task 2.6 names couchcmd/run.go:170-178 as what the console displaces and who drives it.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Decision 12 corrects the field list and injects the resolver rather than restating it; Task 3.2's deletion check fails on a re-implemented strings.Contains.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: Decision 1 states root actor = first child, path defaults to ".", the ordering that makes brain home, and the launching shell's fate; Task 3.5's smoke moves to that real configuration.
          round: 2
        - id: PQ-5
          disposition: addressed
          note: Addressed by removal - Decision 5 drops the alt-screen branch entirely, keeps replay as the one mechanism, and records the nudge as a costed fallback.
          round: 2
      blocked: false
content_hash: 453abe138ad06545e5e997fdb6f5f05d78f7f9c7dffe5aed0330cae6fbe7bd8f
---

# Gate ledger — pair#146 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-22T12:44:44-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `stream-split-contract` Intercept's signature can neither express the split point nor satisfy its own split-boundary test
  `(in []byte) (forward []byte, hits int)` concatenates the bytes before and
  after the hotkey, so `x<ctrl-space>y` cannot route `x` to the old focus and
  `y` to the new; and test 2.3(c) (NUL inside an escape payload) needs framing
  state a stateless per-read function cannot carry, contradicting Decision 10
  and the plan's own split-boundary rule in 1.3(d). `workbenchshortcut.FindChord`
  (shortcut.go:342-352) already returns (before, chord, raw, rest, ok) and
  termcmd's pump already carries partials in `held` (run.go:343, 399, 457) —
  adopt that shape or drop 2.3(c) with a stated reason (ARCH-DRY).
- **PQ-2** [Important] `io-seam-unnamed` Console's host-tty and signal boundary is neither named as a seam nor deduplicated from termcmd
  M1 extracts the child half only; Console re-implements MakeRaw
  (termcmd/run.go:222), SIGWINCH -> GetsizeFull (:244, :975-983), Restore, and
  the `\x1b[r` region reset that termcmd.restoreTerminal already owns
  (:1107-1109) — two sites for one sequence, against the plan's own
  one-constant rule. With no injectable host terminal or signal source in the
  type inventory, Task 2.5's "fakes, no real tty" tests and Task 4.4's signal
  path test cannot be written (ARCH-PURE, ARCH-MOCK, ARCH-DRY). The plan also
  does not say that couchcmd/run.go:171-178 (the StartResult branch that prints
  a line and then blocks on Handle.Wait) is what the console displaces, or who
  constructs and drives the Console.
- **PQ-3** [Important] `resolution-single-source` PanelModel.Filter restates LookupTrees' match rule, and the stated field list is wrong
  LookupTrees matches name + description only — NamingTable.Lookup scans Name
  and Description (naming.go:44-57) and LookupTrees adds the agent-published
  description via Describe (couch.go:196-220, 276-281). Repo is not matched;
  path resolution lives one layer up in ResolveRef, behind an ActorID exact
  branch (couch.go:317-340). A restated Filter either diverges from the CLI or
  duplicates the rule, and makes false the claim that #148's advisor calls the
  same resolution. Have the panel call LookupTrees, or extract a pure matcher
  both use (ARCH-DRY).
- **PQ-4** [Important] `home-actor-contract` Root actor is defined two ways, and the smokes verify the weaker one
  Decision 1 equates "child of the first `couch start`" with the Spec's "the
  session couch launched in", but under it the M2 smoke (`couch start ../pair`
  from brain, Task 2.7) makes pair the root actor and brain — the advisor, what
  the project calls home — is never a couch child, so Task 3.5 verifies the
  project's headline property against a stand-in. The plan also never says what
  becomes of the shell couch was launched from (no key leaves couch, so it is
  blocked until exit). State the definition, the ordering it implies for making
  brain home, and the launching session's fate.
- **PQ-5** [Minor] `resize-nudge-mechanism` The alt-screen resize nudge only fires if the size actually changes
  TIOCSWINSZ raises SIGWINCH only when the new winsize differs from the stored
  one (both Darwin and Linux), so Decision 5's nudge is a rows-1 -> rows-2 ->
  rows-1 round trip and a visible double reflow of the whole zellij workbench.
  Say that is accepted, or choose a different redraw trigger.

## Round 2 — 2026-08-22T12:51:51-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Interceptor.Feed(in) (before, hit, rest) adopts FindChord's shape; Decision 10 names bracketed paste as the one place framing state is real and Task 2.3 tests the split marker and the pasted-NUL loss.
- PQ-2 — addressed — New hostty package (Host seam, OSHost/FakeHost, control constants incl. the moved \x1b[r) with termcmd's host half migrated in Task 1.4a; Task 2.6 names couchcmd/run.go:170-178 as what the console displaces and who drives it.
- PQ-3 — addressed — Decision 12 corrects the field list and injects the resolver rather than restating it; Task 3.2's deletion check fails on a re-implemented strings.Contains.
- PQ-4 — addressed — Decision 1 states root actor = first child, path defaults to ".", the ordering that makes brain home, and the launching shell's fate; Task 3.5's smoke moves to that real configuration.
- PQ-5 — addressed — Addressed by removal - Decision 5 drops the alt-screen branch entirely, keeps replay as the one mechanism, and records the nudge as a costed fallback.

## Open findings

(none — every finding has been disposed)
