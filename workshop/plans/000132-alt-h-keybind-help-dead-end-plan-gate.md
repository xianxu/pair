---
gate: plan-quality
issue: 132
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-07-29T22:30:06-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: ParseNvimKeymaps contract breaks on unquoted lhs, and the count guard can't detect it
          detail: |-
            nvim/init.lua:3872, :3877 and :3930 pass an unquoted lhs (open, close,
            tostring(i)), so "second quoted argument is the lhs" captures the desc
            string ('pair: autopair ') as the Key while the parsed-count assertion
            still equals 34 and passes. Name this input class and give the parser one
            property/fuzz strategy line (every Key is a quoted literal or flagged; no
            Key begins with "pair: ") plus a reconciliation that separates
            skipped-by-design from misparsed.
          round: 1
        - id: PQ-2
          severity: Important
          title: Join rule derives false wording for the pane-local chords; dual-meaning keys unhandled
          detail: |-
            Alt+t/w/r exist as keymaps at nvim/init.lua:3653-3658 with desc "right-terminal
            tab helper disabled in draft" — the draft no-op, not shortcut.go:153-158's
            new/close/rename tab — so "catalog Help wins only where the source has none"
            ships the misleading string as user help. Alt+Shift+Enter means append-no-send
            in the draft (init.lua:3632) and toggle-focused-layout in the terminal
            (shortcut.go:176), which one row per key cannot express. State the prose source
            for the role-local chords and how the two-meanings case renders.
          round: 1
        - id: PQ-3
          severity: Important
          title: Drift tests read the gitignored generated bundle, so classification can pass on a stale snapshot
          detail: |-
            cmd/internal/runtimebundle/assets/ is gitignored (.gitignore:34) and regenerated
            only by runtimebundle-generate (Makefile.local:92-95), which reaches `make test`
            only transitively via a prereq that builds bin/pair (Makefile.local:272). The
            plan's own per-task command (go test ./cmd/internal/keyhelp/) therefore asserts
            against whatever snapshot the last build left — add a keymap, run go test, get
            green. Classify against the tree copy (or assert embedded == tree) and keep the
            embedded read for the shipped-fidelity render test.
          round: 1
        - id: PQ-4
          severity: Minor
          title: zellij bind inventory is wrong (20 binds, not 13; Write as well as WriteChars)
          detail: |-
            config.kdl has 20 bind entries across mode blocks and several use
            `Write 27; Write 106;` rather than WriteChars. The derived claim (exactly 2 Run
            binds, Alt h / Alt l) is correct. A lookahead scanner should be tested against a
            Write bind immediately preceding a Run bind — the :157 to :163 shape.
          round: 1
        - id: PQ-5
          severity: Minor
          title: No non-goals; PAIR_CHEATS and ChordName are unnamed restatements of the key model
          detail: |-
            nvim/init.lua:2137 PAIR_CHEATS is a hand-maintained key+label list and the
            display-spelling authority for Alt+⏎/Alt+x/Alt+d; workbenchshortcut.ChordName
            (shortcut.go:302) is a third vocabulary but is a routing name round-tripped by
            ChordFromName (wrapcmd/wrap.go:1502), so reuse would be wrong — say so. Task 9's
            sweep does not grep nvim/. Declare each a derived consumer or an explicit non-goal.
          round: 1
        - id: PQ-6
          severity: Minor
          title: bin/pair-help assumptions do not hold for the new content
          detail: |-
            bin/pair-help:30 measures width with awk `length` (bytes), so glyph keys skew the
            centering the plan calls "already correct"; set -euo pipefail at :7 means a
            non-zero `pair keys` aborts before less, not "the pager will show empty". The
            shim is also verified only by grep plus eyeball, though the bundle carries its
            own copy — a mechanical check of the bundled shim plus `pair keys` output is cheap.
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-07-29T22:36:40-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: 'Three-way KeymapScan, arg-position rule, the no-"pair: "-prefix trap, and a named 3-site allowlist replace the bare count.'
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Per-row wording source plus roleBindings; dual-meaning keys get two rows. Task 6's step text still carries the old rule — raised as Minor below.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Classification reads the tree copies; TestEmbeddedSourcesMatchTree ties the shipped bundle to them; render keeps the embedded read.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: Inventory corrected to 20 bind / 38 unbind / 2 Run and Write <n> named; the :157-to-:163 lookahead shape is still untested but is a close-review detail.
          round: 2
        - id: PQ-5
          disposition: addressed
          note: Explicit Non-goals section covers PAIR_CHEATS, ChordName-as-wire-format, homebrew, and Decide migration; Task 9's sweep now includes nvim/.
          round: 2
        - id: PQ-6
          disposition: addressed
          note: awk byte-width and set -euo pipefail both named, centering moved into Go, pair keys must exit 0 on source error.
          round: 2
      findings:
        - id: PQ-7
          severity: Minor
          title: Three task steps retain pre-fix wording the design sections now override
          detail: |-
            Task 6 Step 2-4 still says "catalog Help wins only where the source has none" —
            the exact rule the Scope note rejects; applied to Alt+t/w/r (nvim desc at
            init.lua:3654) it ships "disabled in draft" as user help, and no planned test
            asserts their wording. Task 5 Step 3 says the helper reads the embedded asset
            "not ../../../", contradicting the CRITICAL split, and Step 2 expects
            mustReadSource while the tests call mustReadTreeSource. Task 2 Step 1 indexes
            ParseNvimKeymaps as a flat slice though it now returns KeymapScan.
          round: 2
      blocked: false
    - "n": 3
      timestamp: "2026-07-29T22:39:06-07:00"
      agent: claude
      dispose:
        - id: PQ-7
          disposition: addressed
          note: Task 6 Step 2-4 now names the per-row source with no prose-wins fallback; Step 1a adds the Alt+t/w/r wording test; Task 5 Step 3 specifies mustReadTreeSource reading ../../../; Task 2 Step 1 uses KeymapScan fields. Only plan.md:644's expected-undefined-symbol list still says mustReadSource — cosmetic, not carried forward.
          round: 3
      blocked: false
content_hash: 68df7a269e49a5c0fced64d6291f4610b0ca59d3709531a5b44525511a934a66
---

# Gate ledger — pair#132 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-07-29T22:30:06-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] ParseNvimKeymaps contract breaks on unquoted lhs, and the count guard can't detect it
  nvim/init.lua:3872, :3877 and :3930 pass an unquoted lhs (open, close,
  tostring(i)), so "second quoted argument is the lhs" captures the desc
  string ('pair: autopair ') as the Key while the parsed-count assertion
  still equals 34 and passes. Name this input class and give the parser one
  property/fuzz strategy line (every Key is a quoted literal or flagged; no
  Key begins with "pair: ") plus a reconciliation that separates
  skipped-by-design from misparsed.
- **PQ-2** [Important] Join rule derives false wording for the pane-local chords; dual-meaning keys unhandled
  Alt+t/w/r exist as keymaps at nvim/init.lua:3653-3658 with desc "right-terminal
  tab helper disabled in draft" — the draft no-op, not shortcut.go:153-158's
  new/close/rename tab — so "catalog Help wins only where the source has none"
  ships the misleading string as user help. Alt+Shift+Enter means append-no-send
  in the draft (init.lua:3632) and toggle-focused-layout in the terminal
  (shortcut.go:176), which one row per key cannot express. State the prose source
  for the role-local chords and how the two-meanings case renders.
- **PQ-3** [Important] Drift tests read the gitignored generated bundle, so classification can pass on a stale snapshot
  cmd/internal/runtimebundle/assets/ is gitignored (.gitignore:34) and regenerated
  only by runtimebundle-generate (Makefile.local:92-95), which reaches `make test`
  only transitively via a prereq that builds bin/pair (Makefile.local:272). The
  plan's own per-task command (go test ./cmd/internal/keyhelp/) therefore asserts
  against whatever snapshot the last build left — add a keymap, run go test, get
  green. Classify against the tree copy (or assert embedded == tree) and keep the
  embedded read for the shipped-fidelity render test.
- **PQ-4** [Minor] zellij bind inventory is wrong (20 binds, not 13; Write as well as WriteChars)
  config.kdl has 20 bind entries across mode blocks and several use
  `Write 27; Write 106;` rather than WriteChars. The derived claim (exactly 2 Run
  binds, Alt h / Alt l) is correct. A lookahead scanner should be tested against a
  Write bind immediately preceding a Run bind — the :157 to :163 shape.
- **PQ-5** [Minor] No non-goals; PAIR_CHEATS and ChordName are unnamed restatements of the key model
  nvim/init.lua:2137 PAIR_CHEATS is a hand-maintained key+label list and the
  display-spelling authority for Alt+⏎/Alt+x/Alt+d; workbenchshortcut.ChordName
  (shortcut.go:302) is a third vocabulary but is a routing name round-tripped by
  ChordFromName (wrapcmd/wrap.go:1502), so reuse would be wrong — say so. Task 9's
  sweep does not grep nvim/. Declare each a derived consumer or an explicit non-goal.
- **PQ-6** [Minor] bin/pair-help assumptions do not hold for the new content
  bin/pair-help:30 measures width with awk `length` (bytes), so glyph keys skew the
  centering the plan calls "already correct"; set -euo pipefail at :7 means a
  non-zero `pair keys` aborts before less, not "the pager will show empty". The
  shim is also verified only by grep plus eyeball, though the bundle carries its
  own copy — a mechanical check of the bundled shim plus `pair keys` output is cheap.

## Round 2 — 2026-07-29T22:36:40-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Three-way KeymapScan, arg-position rule, the no-"pair: "-prefix trap, and a named 3-site allowlist replace the bare count.
- PQ-2 — addressed — Per-row wording source plus roleBindings; dual-meaning keys get two rows. Task 6's step text still carries the old rule — raised as Minor below.
- PQ-3 — addressed — Classification reads the tree copies; TestEmbeddedSourcesMatchTree ties the shipped bundle to them; render keeps the embedded read.
- PQ-4 — addressed — Inventory corrected to 20 bind / 38 unbind / 2 Run and Write <n> named; the :157-to-:163 lookahead shape is still untested but is a close-review detail.
- PQ-5 — addressed — Explicit Non-goals section covers PAIR_CHEATS, ChordName-as-wire-format, homebrew, and Decide migration; Task 9's sweep now includes nvim/.
- PQ-6 — addressed — awk byte-width and set -euo pipefail both named, centering moved into Go, pair keys must exit 0 on source error.

### Raised

- **PQ-7** [Minor] Three task steps retain pre-fix wording the design sections now override
  Task 6 Step 2-4 still says "catalog Help wins only where the source has none" —
  the exact rule the Scope note rejects; applied to Alt+t/w/r (nvim desc at
  init.lua:3654) it ships "disabled in draft" as user help, and no planned test
  asserts their wording. Task 5 Step 3 says the helper reads the embedded asset
  "not ../../../", contradicting the CRITICAL split, and Step 2 expects
  mustReadSource while the tests call mustReadTreeSource. Task 2 Step 1 indexes
  ParseNvimKeymaps as a flat slice though it now returns KeymapScan.

## Round 3 — 2026-07-29T22:39:06-07:00 (claude) — passed

### Disposed

- PQ-7 — addressed — Task 6 Step 2-4 now names the per-row source with no prose-wins fallback; Step 1a adds the Alt+t/w/r wording test; Task 5 Step 3 specifies mustReadTreeSource reading ../../../; Task 2 Step 1 uses KeymapScan fields. Only plan.md:644's expected-undefined-symbol list still says mustReadSource — cosmetic, not carried forward.

## Open findings

(none — every finding has been disposed)
