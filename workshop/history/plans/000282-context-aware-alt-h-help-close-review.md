# Boundary Review — pair#282 (whole-issue close)

| field | value |
|-------|-------|
| issue | 282 — Alt+h help knows whether it runs under couch, and shows couch's keys there |
| repo | pair |
| issue file | workshop/issues/000282-context-aware-alt-h-help.md |
| boundary | whole-issue close |
| milestone | — |
| window | 3618034175b07f7e9ad4305e2c6986beb09b51b0..86f28eedd3ab54a2dfc93de86456d0fd865067c9 |
| command | sdlc close --issue 282 |
| reviewer | claude |
| timestamp | 2026-09-18T12:34:52-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The boundary delivers what the issue's (thrice-revised) Spec and Done-when rows claim: Couch's chords become one declared table (`couchkeys`), couchtty's framing/routing, `couch --help` and Pair's Alt+h page all derive from it, the presenter fact is recorded per-attach by the client that knows it, and standalone `pair keys` is untouched. I verified behavior rather than prose: I rendered both pages and `couch --help` from the real tables, walked the framing/routing tables, and ran two mutation checks (`RecordOuterTTY(tag, false)` in `lifecycle.go:82` → `TestRunLaunchAttachRecordsCouchPresenter` fails; `actorReserved` forced `true` → `TestCouchChordContract` fails on all four switcher chords), so the new tests bite. Every targeted test for the diff passes; the full `go test ./...` here shows 57 packages ok and 15 failing packages whose every failure line is `operation not permitted` (pty/exec/mktemp restrictions in this review environment), matching the known-environmental pattern — I could not independently reproduce a clean full-suite run, which is the only reason this isn't "verified end to end by me". Nothing found rises above Minor.

**1. Strengths**

- `couchkeys` is the right shape for the layering question the Spec raised: pure data, Pair never imports Couch's console, and switcher rows *derive* their bytes from `workbenchshortcut.ChordEncodings` instead of restating them (`couchkeys.go:95`), with `TestSwitcherChordsCarryPairsEncodings` pinning it. I confirmed no switcher encoding is single-byte, so moving the lifecycle chords into the shared `len(encoding) > 1` loop in `knownSequences` is byte-for-byte framing-neutral.
- `actorReserved` reading `Scope` (`couchtty/keys.go:140`) collapses the old "routing table vs help table" drift class into one field, and `TestEachHitHasOneScope` + `TestEveryCouchActionDispatches` close the "declared but undispatched / dispatched but undeclared" holes that previously shipped alt+n consumed-and-dropped.
- The presenter/hosting split is genuinely two facts from two authorities, and `TestPresenterAndHostingAreIndependent` walks all four quadrants including the adopted-session case that motivated the third revision. `TestCouchClientRefusesRestartMarker` pins the previously-untested second gate (`createflow.go:162`) that makes the Alt+n hosted wording true.
- Docs moved in the same window and fixed pre-existing lies, not just new surface: README 378-381 (“Alt+d is intercepted by Couch”) and `atlas/couch.md:737` (“other panes retain Pair’s existing in-process reload”) had been wrong since #245/#249. `TestREADMEDocumentsEveryCouchChord` makes the README home a derived requirement.
- The three plan-quality advisories are visibly satisfied in the diff: `FuzzDecodeOuterRecord` exists and asserts `Couch==true` only for the exact shape, the Alt+Shift+C carve-out is stated at `workbenchshortcut/shortcut_test.go:748`, and the README anchors landed correctly despite the earlier-edit drift.

**2. Critical findings** — none.

**3. Important findings** — none.

**4. Minor findings**

- `workbenchshortcut/shortcut.go:167` — the hosted Alt+n row renders 124 columns wide (standalone page widest: 104). `keyhelp.Center` derives its pad from the widest line, so on a ≤124-col terminal the *whole* Couch/hosted page silently stops being centred and that row wraps under `less`. Shortening to e.g. "no reload under Couch; may end the thread — relaunch from the switcher" restores the budget.
- `couchtty/menu.go:22` — `menuControls` is still a hand-maintained restatement of the same six chords now owned by `couchkeys` (and its Alt+d Action text no longer matches `couchkeys`' wording). It is consumed only by `TestREADMEDocumentsEveryPanelControl` (its comment claiming "the panel renderer... consume the same key inventory" is stale — nothing renders it). Derive its chord rows from `couchkeys.Bindings()` or drop them.
- `couchkeys/couchkeys.go:85` — the "Couch switcher" section lists only the four lifecycle chords, but `Ctrl+Space` has a *second* meaning in the switcher (`console.go:1314` → `KeyCtrlSpace` = start a thread) that neither `couch --help` nor Alt+h documents, while the section title reads as the switcher's key list. A switcher-scope row for Ctrl+Space would follow the page's own `(key, context)` rule.
- `keyscmd/keyscmd.go:48` — `Deps` has three required seams and no nil handling; a future caller constructing `Deps{Sources: …}` panics, and a panic in `pair keys` kills the floating pane before `less` opens under `bin/pair-help`'s `set -euo pipefail` — the exact dead-help-key failure this package's always-exit-0 contract exists to prevent. Defaulting nil `Getenv`/`CouchPresents` in `RunWith` is three lines.
- `launcher/outerrecord.go:54` — `PresentedByCouch` matches on tag alone, while sibling predicates use scope+tag (`createflow.go:511`) or tag+valid-scope (`lifecycle.go:105`). A `pair resume <same-tag>` run from inside a Couch thread's right terminal inherits `COUCH_THREAD_TAG` and records `presenter=couch`. Harmless today; worth either tightening or a one-line note on why tag alone is the rule.

**5. Test coverage notes**

Coverage matches the bug classes this diff could ship: codec round-trip + legacy + malformed + fuzz for the cross-version parser; fake-runtime assertions on both the create and attach write sites; a real-filesystem test for the reader and for the non-tty removal branch (the event the plan named as most likely mishandled); framing/routing contract tests driven from the declaration. The cross-repo invariant this feature rests on — Couch sets `COUCH_THREAD_TAG` on *every* attach including warm reattach — is independently pinned by the existing `couchcore/warmresume_test.go:124`, so a regression there fails loudly rather than silently blanking Couch's section. Hermetic-ness is handled deliberately (`dispatcher_test.go:358`, `keyscmd` `deps()`), which matters in a repo routinely tested inside a Couch thread. Two gaps, both acceptable: `Layer`'s claimed-chord drop has no production consumer today (`Claimed()` is empty), so it is proven only by the synthetic Alt+l probe plus the label-level guard `TestNoPairRowSharesAnEveryPaneCouchKey`; and the tty-write branch of `OSRuntime.RecordOuterTTY` is exercised only by the operator smoke (the path resolution it shares with the reader is pinned by the removal-branch test).

**6. Architectural notes**

ARCH-DRY pass — four restatements collapse into one table; the only residue is `menuControls` (Minor above). ARCH-PURE pass — `couchkeys`, `Layer`, `sections`, the codec and `dispatchFor` are pure and tested without IO (`DefaultSources` is the embedded bundle, not the filesystem); IO stays in `RecordOuterTTY`/`ReadOuterPresenter`/`Run`. ARCH-PURPOSE pass — shadow-sweep of the consumers (framing, routing, `couch --help`, Pair's page, README guard, atlas) shows each deriving from `couchkeys`; no part of the stated purpose is deferred. ARCH-MOCK pass — `fakeRuntime` extended behind the same `Runtime` seam, reader tested against a real temp dir, live conformance via the operator smoke. ARCH-CONSTRAINTS pass on the declared envelope (one env read + one small file read on a one-shot UI path, ~6 ms measured), with the width regression noted above as the only envelope-adjacent cost. ARCH-SECURE pass — the record is parsed strictly at the boundary, a legacy shape is a defined value rather than a guess, malformed degrades visibly to Pair's page plus stderr, and no credential is involved. ARCH-ORDER pass — the record carries one boolean with last-attach-wins semantics; the interesting interleavings (two clients, Couch crash, failed tty probe, quit cleanup) are enumerated in the plan and the failed-probe case is the one with a test. ARCH-FUNERAL pass — no new artifact family; one ~16-byte line on a record whose writers, cleanup and GC are unchanged. Forward-looking: `keyscmd` now names Couch directly (`couchkeys.HelpSections`/`Claimed`), so a second host would need a registry rather than an import; `Layer` is already host-agnostic and is the right place for that seam when it arrives.

**7. Plan revision recommendations**

- The durable plan's step checkboxes are all still `- [ ]` even though every task landed; a short `## Revisions` entry (or a tick pass) recording "Tasks 1-8 implemented in `dbc489ba`..`9d0e257c`" would keep the plan an accurate record rather than a to-do list that outlived its work.
- Optional: note in the plan's consumer enumeration that `couchtty.menuControls` remains a hand-maintained restatement and why it was left (README-guard-only, not rendered), so the waiver is explicit rather than implied by "its Keys already cover the chords".

```findings
findings:
  - id: new
    severity: Minor
    family: help-row-width-budget
    title: |
      Hosted Alt+n row is 124 cols wide and silently disables centering for the whole page
    detail: |
      Measured: standalone page widest line = 104 cols, hosted/Couch page = 124
      ("Alt+n does not reload under Couch and may end the thread; relaunch from the
      Couch switcher (outside the agent pane)"). keyhelp.Center pads from the widest
      line, so on a <=124-col terminal centering vanishes for every section and the
      row wraps under less. Shorten the HostedHelp wording.
  - id: new
    severity: Minor
    family: couch-chord-single-source
    title: |
      couchtty.menuControls still restates Couch's chord inventory by hand
    detail: |
      couchkeys is now the source for Couch's chords, but menu.go:22 keeps a parallel
      list of the same six chords with its own Action wording (Alt+d reads "detach this
      thread · all + leave couch here" against couchkeys' "detach every live thread and
      leave Couch"). Only TestREADMEDocumentsEveryPanelControl consumes it, and its
      comment claiming the panel renderer consumes it too is stale. Derive the chord
      rows from couchkeys.Bindings() or drop them.
  - id: new
    severity: Minor
    family: scope-section-key-coverage
    title: |
      The "Couch switcher" section omits Ctrl+Space's in-switcher meaning
    detail: |
      console.go:1314 gives Ctrl+Space a second meaning when the panel has focus
      (KeyCtrlSpace = start a thread), but couchkeys declares Ctrl+Space only at
      ScopeEveryPane, so neither `couch --help` nor Alt+h documents it while the
      section title reads as the switcher's key list. A switcher-scope row would
      follow the page's own (key, context) rule.
  - id: new
    severity: Minor
    family: seam-struct-requires-nil-defaults
    title: |
      keyscmd.Deps panics on a nil seam, defeating the always-exit-0 contract
    detail: |
      RunWith calls deps.CouchPresents() and passes deps.Getenv unconditionally. A
      caller constructing Deps{Sources: ...} panics; a non-zero exit from `pair keys`
      kills the floating pane before less opens under bin/pair-help's set -euo
      pipefail -- the dead-help-key failure #132 fixed. Default nil Getenv to
      os.Getenv and nil CouchPresents to a false func.
  - id: new
    severity: Minor
    family: couch-ownership-predicate-variants
    title: |
      PresentedByCouch matches on tag alone where sibling predicates also match scope
    detail: |
      createflow.go:511 uses scope+tag and lifecycle.go:105 uses tag+valid-scope, but
      outerrecord.go:54 uses tag only. A `pair resume <same-tag>` run from inside a
      Couch thread's right terminal inherits COUCH_THREAD_TAG and records
      presenter=couch. Harmless today; tighten or record why tag alone is the rule.
```
