---
id: 000128
status: working
deps: [pair#127]
github_issue:
created: 2026-07-28
updated: 2026-07-30
estimate_hours: 2.26
started: 2026-07-30T16:48:13-07:00
---

# share escape-sequence framing between termcmd and wrapcmd

## Problem

Two packages in one binary now carry their own escape-sequence framing:

- `wrapcmd`: `otherEscRe` (wrap.go:189) encodes CSI plus OSC terminated by
  BEL-or-ST, and `stripCodexOutputMarkers` (wrap.go:766) is a byte-level marker
  stripper with tail carry (`p.stdoutPending`, wrap.go:310).
- `termcmd`: `queries.go` (#127) frames CSI and OSC to strip capability queries
  out of a tab redraw.

The *policy* tables must stay separate — they are in one case opposed, since
`wrapcmd` strips `\x1b[>7u` so codex stops pushing Kitty flags while `termcmd`
requires `\x1b[>1u` to survive a replay. What should not be duplicated is the
**framing**: "where does this CSI/OSC end".

#127 deliberately scoped the extraction out: the repo is a flat
`cmd/internal/<pkg>` layout with no shared home, so extracting would have created
a package as a side effect of a two-defect bugfix. `otherEscRe` is also consumed
three ways — `ReplaceAll` in `stripTerminalControls` (wrap.go:812) and the
capture-early path (wrap.go:1151), and `FindIndex` **at an offset** in the
colored-run walker (wrap.go:1018) — so a byte-scanner does not drop in; sharing
means rewriting three call sites that feed scrollback capture and agent-output
detection.

## Spec

- One framing helper answering "length of the CSI/OSC/escape at buf[i]", used by
  both packages. Policy tables stay where they are (`ARCH-DRY` applies to the
  framing, not the policy).
- Decide its home: a new `cmd/internal/ansi` package, or hosting it in one of the
  two existing packages if the dependency direction stays acyclic.
- These wrapcmd tests must stay green across the change:
  `stdout_filter_test.go`, `extract_fg_test.go`, `update_agent_output_test.go`.

## Done when

- One framing implementation; `otherEscRe`'s three call sites still behave
  identically, pinned by the tests above.
- `termcmd/queries.go` and `wrapcmd`'s stripper both call it.
- The opposed-policy note in `atlas/architecture.md` still reads true.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. `sdlc estimate-source` reports the calibration
source as stale (#127 tracks recalibration), so the number is provisional but uses
the required method.

Design hours take the ×0.2 spec-quality discount on every primitive **except**
`issue-spec`, which is the plan-authoring cost itself and cannot be discounted by
its own plan (the category error corrected in #132). v2.1 Step 6 halves the design
buffer to +15%; impl is 40% of the v2 table per v3.1 rule 5. `familiarity: 1.0`.

**v2.1 Step 2.5 library-availability check (mandatory for `greenfield-go-module`,
recorded).** Go ANSI parsers exist, and one is already in the tree indirectly.
**Declined, design NOT halved:** the deliverable is not "parse ANSI" but "reproduce
two *specific existing* framings byte-for-byte, including their disagreements" — a
library would replace the very semantics the differential fuzzers must preserve, and
adopting one would change behaviour in an interactive rename path. The design
dialogue was not short-circuited; it was three gate rounds of establishing what the
current code actually does.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.75 impl=0.08
item: greenfield-go-module design=0.20 impl=0.24
item: smaller-go-module design=0.03 impl=0.12
item: smaller-go-module design=0.04 impl=0.14
item: smaller-go-module design=0.04 impl=0.14
item: atlas-docs design=0.03 impl=0.06
item: milestone-review design=0.02 impl=0.20
design-buffer: 0.15
total: 2.26
```

Arithmetic: design 1.11 + impl 0.98 + buffer (1.11 × 0.15 = 0.1665) = **2.2565 → 2.26**.

Item mapping:
- `issue-spec` **0.75 undiscounted** — the durable plan through three plan-quality
  rounds, two of which forced structural rewrites (the Critical infinite-loop finding
  and the SS3 misnaming). Upper-mid of the 0.5–1.5 band: the Spec was inherited from
  #127's close, but the design work was substantial and all of it sits inside the
  measured window.
- `greenfield-go-module` (0.20/0.24) — `cmd/internal/ansi`: four primitives
  (`IsFinalByte`, `TerminatorScan`, `OSCEnd`, `Frame`), two modes, three statuses.
  Design at the band midpoint pre-discount; impl mid-band — the logic is small but
  every branch is a byte-range decision that must match an existing regex exactly.
- `smaller-go-module` (0.03/0.12) — the differential fuzz oracles. Small to write,
  and the thing that converts "behaves identically" from a claim into a check.
- `smaller-go-module` (0.04/0.14) — `wrapcmd`: two `ReplaceAll` sites, the
  colored-run walker, and deleting `otherEscRe` so the compiler finds anything missed.
- `smaller-go-module` (0.04/0.14) — `termcmd`: two adapters, the
  `isTerminalFinalByte` delegation, and the behaviour pins that must pass **before**
  the change (including the SS3 cases).
- `atlas-docs` / `milestone-review` — the opposed-policy note (a Done-when item), one
  close boundary; review impl at top-of-band.

Plausibility note, and the trend I am now obliged to flag rather than restate.
Today's three closes ran est/actual 1.60/0.90 (1.8×), 4.31/1.78 (2.4×) and
3.18/0.37 (**8.6×**). That is three consecutive over-estimates in the same repo under
the same model, and 2.26 sits in exactly the band that produced them. I am still
deriving by the method rather than fudging the total down — an estimate bent to hit
a predicted actual stops being evidence — but I no longer think the estimator is the
only thing at fault: 0.37h for #132 (a new package, two parsers, five drift tests,
six review findings fixed) is not a plausible measurement, and I flagged the
windowing at that close. **If this closes near 0.3–0.6 again, the finding belongs to
`sdlc actual`'s window computation, not to the estimate model**, and #127's
recalibration should not consume these rows until that is settled. Recorded here so
the next reader sees the hypothesis stated in advance of the data rather than after.

## Plan

Durable design: **`workshop/plans/000128-share-escape-framing-plan.md`**. Five tasks,
single review boundary — no `Mx` tags.

- [x] Decide the helper's home (new `cmd/internal/ansi`, or host it in one of the
      two packages if the dependency direction stays acyclic).
- [x] Extract the framing: "length of the CSI / OSC / escape at buf[i]".
- [x] Repoint `termcmd/queries.go` (`csiEnd`, `oscEnd`) at it.
- [x] Repoint `wrapcmd`, handling that `otherEscRe` is consumed three
      structurally different ways — `ReplaceAll` at wrap.go:812 and wrap.go:1151,
      and `FindIndex` **at an offset** in the colored-run walker at wrap.go:1018.
- [x] Keep `stdout_filter_test.go`, `extract_fg_test.go` and
      `update_agent_output_test.go` green — they pin the three call sites.
- [x] Leave the policy tables where they are; re-check the opposed-policy note in
      `atlas/architecture.md` still reads true.

## Log


- **Implemented.** `cmd/internal/ansi` created; `otherEscRe` deleted from production;
  `csiEnd`/`oscEnd`/`isTerminalFinalByte` are one-line delegations. `make test`
  **exit 0**.
- **Proof, not argument.** The retired regex lives in `ansi/oracle_test.go` as a
  differential oracle. `SequenceLen` and `Strip` were fuzzed against it —
  **~20M executions, zero disagreements** — and the seed corpus also runs as a plain
  table on every `go test`, so the check does not depend on anyone starting a fuzz
  session. `termcmd`'s pins were written and passing against the OLD code first, so
  "this changes nothing" is measured. #127's `FuzzStripTerminalQueries`: 8M execs clean.
- **The simplification the gate blessed, taken:** `Frame` has no `Mode`. Only
  `OSCEnd` has two consumers wanting different strictness, so a Lenient CSI arm
  would have been an unused code path. `Mode` lives on `OSCEnd` alone.
- **Two non-obvious facts the implementation had to reproduce**, both found by the
  oracle rather than by reading:
  - `]` is 0x5D, **inside** the two-byte escape class `[0x5C-0x5F]`. So the regex
    never treated an unterminated OSC as incomplete — it matched `\x1b]` as a
    two-byte escape and left the payload as text. Alternative ORDER is therefore
    load-bearing, and `Frame` tries them in the regex's order.
  - A malformed CSI (`\x1b[\x00A`) is `None`, not `Incomplete`. My first cut fell
    through to a catch-all that reported it as truncated, which would have pinned it
    in a caller's pending buffer forever. `'['` has no two-byte fallback (0x5B is
    outside the class), so `frameCSI`'s verdict now stands unmodified.
- Both of my own unit expectations were wrong on first run (`OSCEnd` length 8 not 9;
  the `None`/`Incomplete` case above). The fuzzers were green throughout, because
  both map to 0 through `SequenceLen` — a reminder that a differential oracle proves
  *consumer-visible* equivalence, not that every internal distinction is right.

### 2026-07-30

- **Dep satisfied:** pair#127 is done and archived, so this is unblocked.
- **Home decided: a new leaf `cmd/internal/ansi`.** Hosting in either existing
  package inverts a dependency the wrong way — `wrapcmd` importing `termcmd` (a PTY
  mux with a Runtime seam) or `termcmd` importing `wrapcmd` (codex markers +
  scrollback). `ansi` imports only stdlib. `textwidth` (#132) set the precedent for
  a tiny shared-rule package in this flat layout.
- **The two framings are NOT equivalent today** — checked before planning, and the
  issue does not record this. `otherEscRe` has **four** alternatives (CSI, OSC,
  charset designation `\x1b[()*+]X`, two-byte `\x1b[@-Z\\-_]`); `termcmd` has only
  CSI + OSC. And `termcmd`'s are **looser**: `csiEnd` scans to any final byte
  without validating param/intermediate ranges, and `oscEnd` scans *past* a bare
  ESC where the regex's `[^\x07\x1b]*` stops at one. So the shared helper must be
  the union, and the strict-vs-loose conflict has to be decided rather than
  inherited.
- **Decided: the shared scanner takes the REGEX semantics** (strict ranges, OSC
  stops at a bare ESC). `wrapcmd` is the side with the "behave identically"
  Done-when and the user-visible consequences, and the looser reading is the one
  that *over-strips* — the asymmetric danger this issue's own Log flags. The plan
  proves `termcmd` is unaffected rather than asserting it: its `csiEnd`/`oscEnd` are
  reached only after a `terminalQueryLiterals` prefix match, so inputs are
  well-formed, and Task 4 tests that for every literal in the table.
- **The awkward third call site is the easy one.** `otherEscRe.FindIndex(data[i:])`
  guarded by `loc[0] == 0` (`wrap.go:1018`) is literally "length of the sequence at
  `data[i]`" — i.e. `SequenceLen`'s contract. The issue framed it as the hard part;
  it converts more cleanly than the two `ReplaceAll` sites.
- **Proof strategy:** the retired regex stays alive in `ansi/oracle_test.go` as a
  differential fuzz oracle. "Behaves identically" then rests on a comparison against
  what the code actually used to run, not on re-reading both and reasoning.

### 2026-07-28

- Filed from pair#127's close. #127 added `termcmd/queries.go`, which frames CSI
  and OSC for a second time in this binary; the deferral and its reasoning are
  recorded in #127's Plan and in `atlas/architecture.md`.
- Note for whoever picks this up: the deny-list's two failure directions are NOT
  symmetric. A missed query degrades benignly, but an **over-strip** silently
  removes mouse mode, Kitty encoding, or the cursor shape. #127 added
  `FuzzStripTerminalQueries` as the cheap structural guard; the shared framing
  helper is the natural place to attach a real conformance check against
  emitters if that ever becomes worth building.
