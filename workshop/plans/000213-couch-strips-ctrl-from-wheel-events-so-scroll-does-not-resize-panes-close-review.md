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

---

## Re-review — 2026-09-09T08:29:31-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 213 — couch strips ctrl from wheel events so scroll does not resize panes |
| repo | pair |
| issue file | workshop/issues/000213-couch-strips-ctrl-from-wheel-events-so-scroll-does-not-resize-panes.md |
| boundary | whole-issue close |
| milestone | — |
| window | f6904c48113c0980095f918930966607b8504569..d58c870a3d45e5bee76d4221710a483e7c2e42a5 |
| command | sdlc close --issue 213 |
| reviewer | claude |
| timestamp | 2026-09-09T08:29:31-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Both round-1 Important findings are genuinely closed, and I confirmed each by mutation rather than by reading the commit message: reverting `termcmd/run.go:459` to the raw-button comparison reddens four table rows *and* makes the new tree-scan rule test name both offending lines by file:line; deleting the `stripWheelResizeModifier` call from `console.go:1590` makes the wiring test time out; reverting the strip's predicate to the raw button reddens six cells of the cross-product table. I independently re-ran the class enumeration (`grep '\.Button' cmd/ probes/`) and it yields exactly three non-`mouseinput` comparison sites — the two termcmd wheel arms (now derived) and `mouse.go:60`'s deliberately-raw `Button == 0`, which now documents why. `go build ./...` and `go vet` clean; `mouseinput` fully green; `couchtty`'s only failure is `TestNotificationPTYConformance` and every other failing package in `go test ./...` fails on `ptychild: operation not permitted` / `mktemp` EPERM — the documented environment class, none of it in this diff's blast radius. Nothing blocks the boundary: the three findings that remain open (README, zellij conformance, `lessons.md`) are all Minor and two of them are one-line edits.

## 1. Strengths

- **`mouseinput.WithButton` is a splice, not a re-encoder** (`cmd/internal/mouseinput/mouseinput.go:64`). It cannot drift on the bytes it does not touch, which is exactly the property `keys.go` and `console.go` demand of the forward path, and `FuzzWithButtonChangesOnlyTheButton` (`mouseinput_test.go:107`) asserts that property directly rather than restating the implementation.
- **The wiring test earns its place** (`console_mouse_test.go:545`). Both halves of the change are pure, so a correct-but-uncalled implementation would leave every unit test green. I removed the call and the test timed out — this is a real oracle, not a decorative end-to-end.
- **The rule test converts a review finding into an executable oracle** (`mouseinput_test.go:153`), and it reports offenders by file:line rather than just failing. That is a better answer to BR-1 than fixing the one site.
- **The third site of the class says why it stays raw** (`couchtty/mouse.go:54-60`). Documenting the deliberate exception is what stops the next reviewer re-raising it as a fourth oversight.
- **The workaround is dateable.** `mouse_scroll_resize` is named in the code comment and in `atlas/couch.md:363-371`, along with *why* setting it on 0.44.3 silently does nothing.

## 2. Critical findings

None.

## 3. Important findings

None. BR-1 and BR-2 are both disposed `addressed` below.

One process note for the close, not a finding: the Manual row is correctly left unticked, so `sdlc close` will trip the `plan-unchecked` gate. That wants `--no-plan-check` with the operator-pending reason carried in `--verified`, not `--force`.

## 4. Minor findings

- The rule oracle is narrower than the rule it names (new finding, below).
- BR-3, BR-4, BR-5 remain open — see dispositions.

## 5. Test coverage notes

- All three claimed fixes are mutation-verified; the working tree was restored and `git status` is clean.
- `TestAReattachedChildKeepsItsTrackingMode` (#196's reattach test) is untouched by the diff and passes — the evidence the mode-belief behaviour did not shift.
- The modifier cross-product table covers {plain, shift, alt, ctrl, ctrl+shift} × {wheel-up, wheel-down, horizontal, left-press} and asserts raw bytes *and* the decoded event, plus a release row. That is the right shape: an `Event`-only assertion would pass with the splice broken.
- `FuzzWithButtonChangesOnlyTheButton` exercises its 7 seeds under a normal `go test`; there is no committed `testdata/fuzz` corpus and no `make` fuzz target, so the Log's "7.7M execs" is a one-off local measurement the repo cannot re-run. Standard Go practice, not a defect — but the standing coverage is the seed set, and the seeds are the enumeration that matters.
- Full-suite failures (10 packages) are all `ptychild`/`mktemp` permission-class, consistent with the documented sandbox limitation.

## 6. Architectural notes

- **ARCH-DRY — pass.** Modifier bits, `BaseButton` and the byte splice all landed in `mouseinput`, the one package that owns the wire format; both consumers derive from it and there is no second splice.
- **ARCH-PURE — pass.** `BaseButton`, `WithButton` and `stripWheelResizeModifier` are pure and tested with no IO. The IO seam is exactly one line at `console.go:1590`, and it has its own test.
- **ARCH-PURPOSE — pass on the class.** The shadow-sweep holds: three comparison sites, all either derived or documented-raw, and no second SGR parser anywhere in `cmd/` or `probes/`. Done-when #1 remains operator-pending, but it is honestly labelled rather than claimed (see BR-2).
- **ARCH-MOCK — flag,** carried by BR-4. Note for whoever picks it up: the seam already exists — `Makefile.local:69-84` has `test-live` / `PAIR_LIVE_COUCH=1`, so a conformance check here is cheaper than round 1 assumed.
- **ARCH-CONSTRAINTS — pass.** Keystroke-class path; the strip early-returns without allocating on every non-ctrl report, so the one small allocation and the redundant second `Parse` inside `WithButton` occur only on ctrl+wheel ticks. Negligible at human scroll rates.
- **ARCH-SECURE — pass.** `WithButton` refuses anything `Parse` refuses and refuses a negative button, so a caller cannot launder a malformed report into a well-formed-looking one; the failure path forwards the original rather than fabricating or dropping a report.
- **ARCH-ORDER — pass.** `stripWheelResizeModifier` holds no state between events — it is `(event, raw) -> (event, raw)` with no carried fields, so there is no transition set to enumerate. The wiring test observes the real input-loop interleaving through `waitFor` on the child's writes rather than a single hand-picked ordering.

## 7. Plan revision recommendations

One `## Revisions` addition is worth making — the Spec sentence is now narrower than what shipped:

> **Delta.** The Spec's "Known limitation, by construction — this fixes couch only; standalone pair is unaffected" is true of the *ctrl+wheel resize gesture* (zellij consumes it above `pair term`) but no longer true of the diff as a whole: BR-1's fix changed standalone `pair term`'s handling of every modified wheel tick — shift+wheel and alt+wheel now scroll the zellij viewport instead of writing raw SGR bytes into the shell. The termcmd fix also has no `## Plan` row; add a checked one so the Plan is a complete inventory of the boundary's deliverables rather than of its originally-designed ones.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Mutation-verified: reverting to `event.Button ==` reddens 4 run_test.go rows and the new rule test names both lines.
  - id: BR-2
    disposition: addressed
    note: |
      Row unticked and labelled OPERATOR-PENDING with the reason; close will need --no-plan-check and the reason in --verified.
  - id: BR-3
    disposition: not-addressed
    note: |
      README.md is untouched in this window; line 406 still reads "scroll inside an attached Pair session are unaffected" while couch now rewrites one report class, and standalone `pair term` now translates modified wheel ticks too.
  - id: BR-4
    disposition: not-addressed
    note: |
      No pin or conformance check added. Cheaper than round 1 assumed: Makefile.local:69-84 already has test-live / PAIR_LIVE_COUCH as the seam.
  - id: BR-5
    disposition: not-addressed
    note: |
      workshop/lessons.md has no entry. The tree-scan test is a stronger oracle for the enumerable shape, but the general rule AGENTS.md section 4 asks for is still uncaptured.
  - id: BR-6
    disposition: addressed
    note: |
      The `sep < 0` guard now says it is not a reachable failure mode; the same shape was applied preemptively to stripWheelResizeModifier's `!ok` branch.
findings:
  - id: new
    severity: Minor
    family: oracle-narrower-than-its-rule
    title: |
      The raw-button rule test scans cmd/ only and matches one syntactic shape, so it under-enforces the rule its own comment states
    detail: |
      cmd/internal/mouseinput/mouseinput_test.go:153-155. The comment says it "scans the tree because this is a
      cross-package rule", but filepath.Walk("../..") resolves to cmd/ — probes/ holds real Go programs and is
      never visited. The match is also line-local: it requires ".Button ==" or ".Button !=" on the same line as
      "Wheel", so `switch event.Button { case mouseinput.WheelUp:` across two lines, or a raw button assigned to
      a variable first, evades it entirely. That is the same class shape BR-1 named, one refactor away. Both are
      cheap: walk from the module root, and match a raw Button reaching a Wheel constant by any route (a
      go/ast pass over each file, or additionally flagging `switch .*\.Button` blocks). Verified: the current
      oracle does redden on the exact BR-1 revert, so it pins the regression — it just does not cover the rule.
```
