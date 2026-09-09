---
gate: boundary-review
issue: 213
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-09T08:13:37-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: termcmd's wheel predicate still compares the RAW button, so every modified wheel tick misses scroll and leaks SGR bytes to the child
          detail: |-
            cmd/internal/termcmd/run.go:459,465 use `event.Button == mouseinput.WheelUp/WheelDown` — the exact
            raw-button comparison this diff's new mouseinput doc comment warns against, and the one pre-existing
            consumer that does not derive from the newly-added BaseButton. Verified against the real pumpStdin path
            with appMouse false: "\x1b[<68;8;5M" (shift+wheel), "\x1b[<72;8;5M" (alt+wheel) and "\x1b[<80;8;5M"
            (ctrl+wheel) all fall into `default:` and produce mux="write:<raw>" with rt="", while plain 64 produces
            rt="scroll-up". So a modified wheel neither scrolls the zellij viewport nor is withheld — it writes SGR
            bytes into a child that never enabled mouse reporting. Fix: `base := mouseinput.BaseButton(event.Button)`
            and switch on it, plus three rows in the run_test.go:169 table. ARCH-DRY / ARCH-PURPOSE: the diff fixed
            the site the plan gate named and left the enumerable sibling of the same class in the tree.
          family: format-knowledge-ownership
          round: 1
        - id: BR-2
          severity: Important
          title: Plan item "Manual" is checked while the Log says the operator gesture check has not been run
          detail: |-
            workshop/issues/000213-….md marks `- [x] Manual. Ctrl+scroll in a live couch session scrolls and pane
            sizes hold` as delivered, but the Log's Evidence paragraph in the same file states "The gesture itself
            still wants one operator check after a couch restart, since a running couch is on the old binary."
            That manual step IS Done-when #1 and is the only claim automation cannot make. Either run it and record
            the observation in ## Log, or uncheck the box and state the outstanding check plus its reason in the
            close --verified evidence.
          family: unverified-claim-marked-done
          round: 1
        - id: BR-3
          severity: Minor
          title: README's couch paragraph still says scroll inside an attached session is unaffected
          detail: |-
            README.md:406 reads "couch enables click reporting for itself and withholds every report from a child
            that never asked for one, so mouse selection and scroll inside an attached Pair session are unaffected."
            couch now REWRITES one report class. One clause noting that ctrl+scroll scrolls rather than resizing
            keeps the user-facing sentence true; atlas/couch.md already documents it fully.
          family: docs-behavior-drift
          round: 1
        - id: BR-4
          severity: Minor
          title: Nothing automated fires when the zellij upgrade makes this filter redundant
          detail: |-
            The filter is correctly documented as deletable and names mouse_scroll_resize, but no test, version pin
            or conformance check ties it to zellij 0.44.3 — after an upgrade it keeps stripping silently and nothing
            goes red. Pre-existing pattern (the repo documents 0.44.3 in prose at a dozen sites with no pin), so
            noted rather than blocked. ARCH-MOCK: behavior we depend on from an external binary has no live
            conformance check.
          family: deletable-workaround-untriggered
          round: 1
        - id: BR-5
          severity: Minor
          title: The raw-button-comparison class produced two instances this round and no lessons.md rule
          detail: |-
            AGENTS.md section 4 asks for a lessons.md rule preventing the mistakes a review found. PQ-1 caught the
            class in the plan, and Important #1 above is a second live instance in termcmd. One line — "an SGR
            button field carries modifier bits; compare mouseinput.BaseButton(b), never the raw value" — is the
            rule shape that would have caught both.
          family: lessons-not-captured
          round: 1
        - id: BR-6
          severity: Minor
          title: WithButton's `sep < 0` branch is unreachable once Parse has succeeded
          detail: |-
            cmd/internal/mouseinput/mouseinput.go:75-78. A report that Parse accepts always contains a ';', so the
            guard can never return false there. Harmless, but it reads as a live failure mode to the next reader.
          family: unreachable-guard
          round: 1
      blocked: true
---

# Gate ledger — pair#213 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-09T08:13:37-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `format-knowledge-ownership` termcmd's wheel predicate still compares the RAW button, so every modified wheel tick misses scroll and leaks SGR bytes to the child
  cmd/internal/termcmd/run.go:459,465 use `event.Button == mouseinput.WheelUp/WheelDown` — the exact
  raw-button comparison this diff's new mouseinput doc comment warns against, and the one pre-existing
  consumer that does not derive from the newly-added BaseButton. Verified against the real pumpStdin path
  with appMouse false: "\x1b[<68;8;5M" (shift+wheel), "\x1b[<72;8;5M" (alt+wheel) and "\x1b[<80;8;5M"
  (ctrl+wheel) all fall into `default:` and produce mux="write:<raw>" with rt="", while plain 64 produces
  rt="scroll-up". So a modified wheel neither scrolls the zellij viewport nor is withheld — it writes SGR
  bytes into a child that never enabled mouse reporting. Fix: `base := mouseinput.BaseButton(event.Button)`
  and switch on it, plus three rows in the run_test.go:169 table. ARCH-DRY / ARCH-PURPOSE: the diff fixed
  the site the plan gate named and left the enumerable sibling of the same class in the tree.
- **BR-2** [Important] `unverified-claim-marked-done` Plan item "Manual" is checked while the Log says the operator gesture check has not been run
  workshop/issues/000213-….md marks `- [x] Manual. Ctrl+scroll in a live couch session scrolls and pane
  sizes hold` as delivered, but the Log's Evidence paragraph in the same file states "The gesture itself
  still wants one operator check after a couch restart, since a running couch is on the old binary."
  That manual step IS Done-when #1 and is the only claim automation cannot make. Either run it and record
  the observation in ## Log, or uncheck the box and state the outstanding check plus its reason in the
  close --verified evidence.
- **BR-3** [Minor] `docs-behavior-drift` README's couch paragraph still says scroll inside an attached session is unaffected
  README.md:406 reads "couch enables click reporting for itself and withholds every report from a child
  that never asked for one, so mouse selection and scroll inside an attached Pair session are unaffected."
  couch now REWRITES one report class. One clause noting that ctrl+scroll scrolls rather than resizing
  keeps the user-facing sentence true; atlas/couch.md already documents it fully.
- **BR-4** [Minor] `deletable-workaround-untriggered` Nothing automated fires when the zellij upgrade makes this filter redundant
  The filter is correctly documented as deletable and names mouse_scroll_resize, but no test, version pin
  or conformance check ties it to zellij 0.44.3 — after an upgrade it keeps stripping silently and nothing
  goes red. Pre-existing pattern (the repo documents 0.44.3 in prose at a dozen sites with no pin), so
  noted rather than blocked. ARCH-MOCK: behavior we depend on from an external binary has no live
  conformance check.
- **BR-5** [Minor] `lessons-not-captured` The raw-button-comparison class produced two instances this round and no lessons.md rule
  AGENTS.md section 4 asks for a lessons.md rule preventing the mistakes a review found. PQ-1 caught the
  class in the plan, and Important #1 above is a second live instance in termcmd. One line — "an SGR
  button field carries modifier bits; compare mouseinput.BaseButton(b), never the raw value" — is the
  rule shape that would have caught both.
- **BR-6** [Minor] `unreachable-guard` WithButton's `sep < 0` branch is unreachable once Parse has succeeded
  cmd/internal/mouseinput/mouseinput.go:75-78. A report that Parse accepts always contains a ';', so the
  guard can never return false there. Harmless, but it reads as a live failure mode to the next reader.

## Open findings

- **BR-1** [Important] `format-knowledge-ownership` termcmd's wheel predicate still compares the RAW button, so every modified wheel tick misses scroll and leaks SGR bytes to the child
- **BR-2** [Important] `unverified-claim-marked-done` Plan item "Manual" is checked while the Log says the operator gesture check has not been run
- **BR-3** [Minor] `docs-behavior-drift` README's couch paragraph still says scroll inside an attached session is unaffected
- **BR-4** [Minor] `deletable-workaround-untriggered` Nothing automated fires when the zellij upgrade makes this filter redundant
- **BR-5** [Minor] `lessons-not-captured` The raw-button-comparison class produced two instances this round and no lessons.md rule
- **BR-6** [Minor] `unreachable-guard` WithButton's `sep < 0` branch is unreachable once Parse has succeeded
