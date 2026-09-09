# Boundary Review — pair#213 (whole-issue close)

| field | value |
|-------|-------|
| issue | 213 — couch strips ctrl from wheel events so scroll does not resize panes |
| repo | pair |
| issue file | workshop/issues/000213-couch-strips-ctrl-from-wheel-events-so-scroll-does-not-resize-panes.md |
| boundary | whole-issue close |
| milestone | — |
| window | f6904c48113c0980095f918930966607b8504569..ac0fbb3bd606f478eaaa4aa97273e49f89d72071 |
| command | sdlc close --issue 213 |
| reviewer | claude |
| timestamp | 2026-09-09T08:13:37-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The implementation is correct, well-narrowed, and — unusually — actually proven wired. I ran both mutations the review contract asks for: discarding the `stripWheelResizeModifier` result in `onMouse` reddens `TestForwardStripsCtrlFromWheelReports` (times out waiting for the stripped report), and reverting the predicate to the raw button reddens 4 table cells plus the wiring test with `button 80 -> 80, want 64`. The fuzz property holds at 2.7M execs over 20s. What blocks a clean SHIP is not the code: (a) the diff introduces `mouseinput.BaseButton` as the single source for "is this the wheel, ignoring modifiers" and then leaves the one pre-existing consumer — `termcmd/run.go:459,465` — comparing the raw button, which I confirmed empirically drops shift/alt+wheel into `default:` and writes raw SGR bytes to a child with `appMouseMode()` false instead of scrolling; and (b) the Plan's `- [x] Manual` box is checked while the issue's own Log says the operator check has not happened yet, and that check *is* Done-when #1.

## 1. Strengths

- `mouseinput.WithButton` (`cmd/internal/mouseinput/mouseinput.go:65`) is the right shape: a splice that re-parses first and refuses anything `Parse` refuses, so a malformed report can never become a well-formed-looking one. It sidesteps the re-encoder the package doc explicitly rejects, and the fuzz property (`Parse(WithButton(raw,b)) == Parse(raw)` with only `Button` differing, plus byte-identity from the first `;`) is the correct invariant for a splice — it covers absurd button arguments a hand-enumeration is blind to.
- `TestForwardStripsCtrlFromWheelReports` (`console_mouse_test.go:544`) is a genuine wiring test, not a restatement. Verified by mutation: `_, _ = stripWheelResizeModifier(...)` compiles, leaves every unit test green, and reddens exactly this one.
- The predicate on `BaseButton` (`couchtty/mouse.go:108`) is the bug this issue was most likely to ship, caught at the plan gate and confirmed by mutation. The doc comment at `mouse.go:103-106` says *why* in the code, where the next reader is.
- The failure path in `stripWheelResizeModifier` (`mouse.go:116-119`) degrades visibly-forward rather than dropping the report — forwarding the original beats losing a wheel tick, and the comment states the reason it is unreachable.
- Deletion trigger is named in three places (code, atlas, issue) with the superseding option's exact name (`mouse_scroll_resize`). That is what stops it calcifying.
- `atlas/couch.md:353-373` records the narrowing on all three axes *and* the masked-predicate trap, which is the part a future reader would otherwise re-derive.

## 2. Critical findings

None.

## 3. Important findings

**`cmd/internal/termcmd/run.go:459` — the one pre-existing site of the class the fix names does not derive from the new source (ARCH-DRY, ARCH-PURPOSE).**

`case event.Button == mouseinput.WheelUp:` is exactly the raw-button comparison the new `mouseinput` doc comment now warns against ("comparing a raw Button against WheelUp misses every modified wheel tick"). Confirmed empirically against the real `pumpStdin` path with `appMouse: false`:

```
"\x1b[<68;8;5M" (shift+wheel-up) -> mux="write:\x1b[<68;8;5M" rt=""
"\x1b[<72;8;5M" (alt+wheel-up)   -> mux="write:\x1b[<72;8;5M" rt=""
"\x1b[<80;8;5M" (ctrl+wheel-up)  -> mux="write:\x1b[<80;8;5M" rt=""
"\x1b[<64;8;5M" (plain wheel-up) -> mux=""                    rt="scroll-up"
```

So a modified wheel tick in `pair wrap` neither scrolls the zellij viewport nor is withheld — it falls into `default:` and writes SGR bytes into a child that never enabled mouse reporting, which is the "typeahead garbage" failure `couchtty`'s swallow rule exists to prevent. Fix sketch: `base := mouseinput.BaseButton(event.Button)` at the top of the `mouseOK` block, switch on `base`, and add three rows to the `run_test.go:169` table (shift/alt/ctrl + wheel-up → `scroll-up`). If the operator judges this outside #213's declared scope, filing it as its own issue is a legitimate disposition — but leaving it silent is not, because `BaseButton` now exists precisely to make it a two-line change.

**`workshop/issues/000213-….md:196` — `- [x] Manual` is checked, but the Log says the check has not been run.**

The Plan's last item claims "Ctrl+scroll in a live couch session scrolls and pane sizes hold" is done; the Log's own Evidence paragraph says "The gesture itself still wants one operator check after a couch restart, since a running couch is on the old binary." That is Done-when #1, and it is the one claim automation cannot make. Fix sketch: either run it and record the observation in `## Log`, or uncheck the box and state in `--verified` that the manual gesture check is outstanding with the reason (running couch is on the old binary). A checked box the same file contradicts two paragraphs later is the traceability defect, independent of which way it resolves.

## 4. Minor findings

- `README.md:406` still says couch's withholding means "mouse selection and scroll inside an attached Pair session are unaffected" — now true except for ctrl+wheel, which couch rewrites. One clause would keep the user-facing sentence honest.
- No automated signal ties this filter to zellij 0.44.3. When the upgrade lands, the `DELETE THIS` comment is the only trigger, and nothing goes red. The repo already documents "0.44.3" in a dozen prose sites with no pin, so this is pre-existing, not introduced here.
- `mouseinput.go:75-78`: the `sep < 0` guard is unreachable once `Parse` has succeeded (a parseable report always has a `;`). Harmless, but it reads as a live failure mode.
- `workshop/lessons.md` gets no entry for the class PQ-1 caught ("a raw SGR button compared against a bare button constant misses every modified event"). Per AGENTS.md §4 that is exactly the rule shape lessons.md is for, and this round produced two independent instances of it.

## 5. Test coverage notes

- Mutation-verified both ways: call discarded → wiring test red; predicate un-masked → 4 table cells + wiring test red. The claimed fixes are pinned by tests that actually fail without them.
- Cross-product table covers {plain, shift, alt, ctrl, ctrl+shift} × {wheel-up, wheel-down, horizontal, left-press} and asserts raw bytes *and* the decoded event, which is the assertion that survives a broken splice. The release row pins the terminator.
- `TestAReattachedChildKeepsItsTrackingMode` (#196) passes unmodified; no existing test line in the three touched test files was edited — the diff is pure addition apart from two import lines. That is the evidence the mode-belief behaviour did not shift.
- Uncovered by construction and correctly argued in the Log: zellij's *response* to the stripped bytes. The argument holds (a plain wheel report is byte-identical to what an unmodified scroll already sends), so this is a documented limit, not a gap.
- Environment note: `TestNotificationPTYConformance` (couchtty) and `TestEveryStripModelMutationRepaintsTheRow` (termcmd) fail here with `ptychild: start sh: operation not permitted` — the documented pty restriction in this environment, unrelated to the diff. The Log's `make test` EXIT=0 claim could not be independently reproduced from this session for that reason; everything else in both packages passes.

## 6. Architectural notes

- **ARCH-DRY** — flag, see Important #1. Otherwise clean: modifier bits, mask and splice all landed in `mouseinput`, and `couchtty` derives rather than restating.
- **ARCH-PURE** — pass. Both new functions are deterministic and allocate-and-return; `onMouse` is the thin seam and calls the strip exactly once, at the top, before routing. The unit tests need no doubles at all.
- **ARCH-PURPOSE** — partial. The couch purpose is delivered, and the standalone-pair gap is an *accepted, argued limitation* in the Spec rather than a deferred point (the zellij upgrade retires both). The shadow-sweep of the newly-created single source is what surfaces Important #1.
- **ARCH-MOCK** — pass with the Minor above. `ptychild.FakeChild` is a stateful double (it tracks mouse-mode belief across `Feed` calls and records writes), and the wiring test runs the real console loop against it through the same boundary production uses. zellij is not faked and has no live conformance check.
- **ARCH-CONSTRAINTS** — pass. Interactive input path; the added cost is one extra `Parse` plus one small allocation, gated behind both predicates so it runs only on ctrl+vertical-wheel. No fan-out, no blocking work, no unbounded buffering.
- **ARCH-SECURE** — pass. The host tty is the untrusted producer; `WithButton` re-parses at the boundary and refuses what `Parse` refuses, so an ill-formed report cannot be laundered into a well-formed-looking one, and the `!ok` path forwards the original rather than fabricating a value downstream would read as evidence. Fuzzed against arbitrary bytes. No credentials in scope.
- **ARCH-ORDER** — pass. The change carries no state between events: it is a value transform on a copy of `MouseHit`, at one call site. The pre-existing ordering hazard (`c.mouseHit` overwritten by the next `FeedHit`) is untouched and already resolved by reading it under the mutex immediately after the hit.

## 7. Plan revision recommendations

- `## Plan` — the `- [x] Manual.` item: either record the operator's observation under a new `### 2026-09-09 — manual verification` Log heading, or revise the box to unchecked with a one-line note that it is pending a couch restart onto the new binary. The Plan and the Log currently disagree about the same fact.
- `## Revisions` — add an entry if Important #1 is taken in-round: "**Reason.** `BaseButton` was introduced as the single source for modifier-masked wheel tests; `termcmd/run.go` remained the one non-deriving consumer. **Delta.** Swept `termcmd`'s wheel predicate onto `BaseButton` and added modifier rows to the `pumpStdin` table." If it is instead filed as a follow-up, record that decision and the new issue number in `## Log` so the class is not silently closed with the instance.

```findings
findings:
  - id: new
    severity: Important
    family: format-knowledge-ownership
    title: |
      termcmd's wheel predicate still compares the RAW button, so every modified wheel tick misses scroll and leaks SGR bytes to the child
    detail: |
      cmd/internal/termcmd/run.go:459,465 use `event.Button == mouseinput.WheelUp/WheelDown` — the exact
      raw-button comparison this diff's new mouseinput doc comment warns against, and the one pre-existing
      consumer that does not derive from the newly-added BaseButton. Verified against the real pumpStdin path
      with appMouse false: "\x1b[<68;8;5M" (shift+wheel), "\x1b[<72;8;5M" (alt+wheel) and "\x1b[<80;8;5M"
      (ctrl+wheel) all fall into `default:` and produce mux="write:<raw>" with rt="", while plain 64 produces
      rt="scroll-up". So a modified wheel neither scrolls the zellij viewport nor is withheld — it writes SGR
      bytes into a child that never enabled mouse reporting. Fix: `base := mouseinput.BaseButton(event.Button)`
      and switch on it, plus three rows in the run_test.go:169 table. ARCH-DRY / ARCH-PURPOSE: the diff fixed
      the site the plan gate named and left the enumerable sibling of the same class in the tree.
  - id: new
    severity: Important
    family: unverified-claim-marked-done
    title: |
      Plan item "Manual" is checked while the Log says the operator gesture check has not been run
    detail: |
      workshop/issues/000213-….md marks `- [x] Manual. Ctrl+scroll in a live couch session scrolls and pane
      sizes hold` as delivered, but the Log's Evidence paragraph in the same file states "The gesture itself
      still wants one operator check after a couch restart, since a running couch is on the old binary."
      That manual step IS Done-when #1 and is the only claim automation cannot make. Either run it and record
      the observation in ## Log, or uncheck the box and state the outstanding check plus its reason in the
      close --verified evidence.
  - id: new
    severity: Minor
    family: docs-behavior-drift
    title: |
      README's couch paragraph still says scroll inside an attached session is unaffected
    detail: |
      README.md:406 reads "couch enables click reporting for itself and withholds every report from a child
      that never asked for one, so mouse selection and scroll inside an attached Pair session are unaffected."
      couch now REWRITES one report class. One clause noting that ctrl+scroll scrolls rather than resizing
      keeps the user-facing sentence true; atlas/couch.md already documents it fully.
  - id: new
    severity: Minor
    family: deletable-workaround-untriggered
    title: |
      Nothing automated fires when the zellij upgrade makes this filter redundant
    detail: |
      The filter is correctly documented as deletable and names mouse_scroll_resize, but no test, version pin
      or conformance check ties it to zellij 0.44.3 — after an upgrade it keeps stripping silently and nothing
      goes red. Pre-existing pattern (the repo documents 0.44.3 in prose at a dozen sites with no pin), so
      noted rather than blocked. ARCH-MOCK: behavior we depend on from an external binary has no live
      conformance check.
  - id: new
    severity: Minor
    family: lessons-not-captured
    title: |
      The raw-button-comparison class produced two instances this round and no lessons.md rule
    detail: |
      AGENTS.md section 4 asks for a lessons.md rule preventing the mistakes a review found. PQ-1 caught the
      class in the plan, and Important #1 above is a second live instance in termcmd. One line — "an SGR
      button field carries modifier bits; compare mouseinput.BaseButton(b), never the raw value" — is the
      rule shape that would have caught both.
  - id: new
    severity: Minor
    family: unreachable-guard
    title: |
      WithButton's `sep < 0` branch is unreachable once Parse has succeeded
    detail: |
      cmd/internal/mouseinput/mouseinput.go:75-78. A report that Parse accepts always contains a ';', so the
      guard can never return false there. Harmless, but it reads as a live failure mode to the next reader.
```
