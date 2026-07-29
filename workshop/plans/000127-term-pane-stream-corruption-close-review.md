# Boundary Review — pair#127 (whole-issue close)

| field | value |
|-------|-------|
| issue | 127 — right terminal pane corrupts the input stream |
| repo | pair |
| issue file | workshop/issues/000127-term-pane-stream-corruption.md |
| boundary | whole-issue close |
| milestone | — |
| window | 70f6ac0651dccf0424527c13ce7730885eaabaec..HEAD |
| command | sdlc close --issue 127 |
| reviewer | claude |
| timestamp | 2026-07-28T17:17:55-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The diagnosis and the seam choice are both right, and the mouse-terminator half of the issue is cleanly delivered: `sgrMouseTerminators` is a genuine single source (I grepped — no residual hardcoded `'M'`/`'m'` framing anywhere in `cmd/`), the regression tests at `run_test.go:176-178` provably fail against the old code, and moving the query filter to the replay rather than the append really does delete the tail-carry/straddle problem the earlier plan revision was paying for. The blocker is one line in the new pure function: `isParameterizedOSCQuery` slices `body[4 : len(body)-2]` after a prefix check (4 bytes) and a suffix check (2 bytes) that can **overlap**, so the input `\x1b]4;?` + BEL/ST panics with a slice-bounds error — a crash on unexpected child output that takes down `pair term` and every shell in the pane, on the tab-switch path this issue just added. That is a one-line guard plus one test row, but it is exactly the "panics on unexpected input" class, so it must land before the boundary. Everything else is Important-or-below.

## 1. Strengths

- **The seam is right, and provably so.** `tab.buffer` has exactly the two touch points the plan claims (verified: `run.go:748-750` append, `run.go:1104` snapshot — nothing else reads it), so filtering at the replay really does make the buffer contiguous at filter time. The O(128 KiB) scan moved off the per-PTY-chunk path onto the tab-switch path.
- **`sgrMouseTerminators` is a real single source, not a documented one.** All four consumers derive: `run.go:581` (`strings.Contains`), `run.go:596`, `run.go:622`, `rename_input.go:180`. The `sgrMouseSize` rewrite is behavior-identical to the old `i >= 3` loop, so the rename path's split-at-every-offset tests still mean what they meant.
- **The snapshot design avoids a worse invariant.** `redrawTab(buf []byte)` + `bufferSnapshotLocked` (nil-safe) genuinely removes mutex knowledge from the IO shell instead of adding a "no caller may hold `m.mu`" rule whose violation mode is a deadlock. All three call sites already held the lock; the change is three lines.
- **`TestStripTerminalQueriesPreservesLegitimateSequences` (queries_test.go:45)** is the right backstop and is where most of the review value is. It pins DECSET `\x1b[?1006h` (which `updateMouseMode` parses), the Kitty push/pop, DECSCUSR `\x1b[5 q`, and — most importantly — the *reply* forms (`\x1b[?62;4;52c`, `\x1b[?2026;2$y`, `\x1b[?0u`), which is the direction a greedy rule would break silently.
- **The side-quest test fix is correct, not a papering-over.** I verified both greps against the tree: `nvim/scrollback.lua:35` calls `workbench_route.install_global_maps(false)` and `nvim/workbench_actions.lua:4` carries `["<M-x>"]`. Asserting the wiring + the generated table instead of a literal in one viewer is the right level.

## 2. Critical findings

**C1 — `isParameterizedOSCQuery` panics on `\x1b]4;?` (`cmd/internal/termcmd/queries.go:178`).**

The prefix check consumes indices 0-3 and the suffix check consumes the last two; for a 5-byte body they overlap at index 3, so both pass and the slice bound inverts.

Trace for input `\x1b]4;?\x07`:
- `terminalSequenceAt` → `buf[1] == ']'` → `oscEnd` returns 6 → `seq = "\x1b]4;?\x07"`.
- `bytes.HasPrefix(seq, "\x1b]4;")` → true.
- `body = "\x1b]4;?"` (len 5); `bytes.HasSuffix(body, ";?")` → true (`body[3]==';'`, `body[4]=='?'`).
- `digits := body[4 : len(body)-2]` → `body[4:3]` → **`panic: slice bounds out of range [4:3]`**.

Reachability: the terminator does not have to be adjacent — `oscEnd` scans forward to the first BEL/ST anywhere in the buffer, so any `\x1b]4;?` in child output with a later `\a` triggers it. `cat`ing a binary file or any app emitting a malformed OSC 4 is enough. The panic then fires on the next `Alt+t` / `Alt+←/→` (pump goroutine) or tab exit (`removeTab`, output goroutine) — unrecovered, so the process dies and every tab's PTY closes.

Fix sketch:

```go
	body := bytes.TrimRight(seq, "\x07")
	body = bytes.TrimSuffix(body, []byte("\x1b\\"))
	if len(body) < 6 || !bytes.HasSuffix(body, []byte(";?")) {
		return false
	}
```

`len(body) >= 6` makes `4 <= len(body)-2` hold unconditionally. Add two rows to `TestStripTerminalQueriesPreservesLegitimateSequences` asserting `\x1b]4;?\x07` and `\x1b]4;?\x1b\\` pass through unchanged (they are malformed, not queries).

For the record, `isParameterizedCSIQuery` (`queries.go:156`) is *not* vulnerable: its prefix is 3 bytes and suffix 2, and `\x1b[?` + `$p` cannot overlap (index 2 would have to be both `?` and `$`), so the minimum matching length is 5 and `seq[3:3]` is the empty-slice case, correctly rejected.

## 3. Important findings

**I1 — `TestReplyGoesToActiveTabNotTheQueryingTab` (`queries_test.go:155`) does not test what it claims; the "pin the accepted residual" plan item is ticked but not delivered.**

The test creates one `fakeMux`, never creates a second tab, and never switches. Its assertion is byte-for-byte the first subcase of `TestPumpStdinForwardsRepliesToChild` (`queries_test.go:140`, same input `\x1b[?62;4;52c`, same expected `write:...`). It therefore pins "a reply reaches the active tab" — which the other test already pins — not "a reply from tab A's query lands on tab B". Concretely: if a later change *fixed* the residual by routing replies to the querying tab, this test stays green, so it is not a pin at all. Also duplicate coverage (`ARCH-DRY`).

Fix: either (a) drive the residual through a real `terminalMux` with two tabs backed by `os.Pipe` PTYs, switch, then assert the reply arrived on the *new* tab's pipe; or (b) if that's more than the residual is worth, delete the test and move its comment onto `TestPumpStdinForwardsRepliesToChild`, and revise the plan item from "add a test asserting exactly that" to "record in `## Log` + atlas" (which the diff does do).

**I2 — no robustness coverage for `stripTerminalQueries` over arbitrary bytes; this is precisely the gap that let C1 ship.**

Every existing case is a hand-authored well-formed sequence. The function's input is unfiltered child output, which includes binary garbage. A five-line fuzz target would have found C1 in seconds and is the natural guard for the whole CSI/OSC framing surface:

```go
func FuzzStripTerminalQueries(f *testing.F) {
	f.Add([]byte("before\x1b[c after"))
	f.Add([]byte("\x1b]4;?\x07"))
	f.Fuzz(func(t *testing.T, b []byte) { _ = stripTerminalQueries(b) })
}
```

Worth strengthening to the real invariant while you're there — the output must be `b` with only whole matched sequences removed, i.e. every non-escape byte survives in order. That property test also covers the "never drop-to-end" rule.

## 4. Minor findings

- `queries.go:126-133` (`csiEnd`), `rename_input.go:192-198` (`escapeSequenceIncomplete`), `rename_input.go:213-218` (`malformedEscapeSize`) are now three copies of the same `for i := 2; …; isTerminalFinalByte` scan **inside one package** (`ARCH-DRY`). `escapeSequenceIncomplete`'s `'['` case is exactly `csiEnd(input) < 0`, and `malformedEscapeSize`'s tail is `if e := csiEnd(input); e > 0 { return e }`. This is in-package and unrelated to the cross-package extraction correctly deferred to #128.
- `run.go:577`: `if !strings.HasPrefix(s, "\x1b[<") || s == ""` — the `s == ""` arm is unreachable (`HasPrefix` short-circuits first and already guarantees `len(s) >= 3`). Drop it or move it first if it's meant as the guard for `s[len(s)-1:]`.
- `atlas/architecture.md:396-397`: the new block starts with no blank line after the `Alt+↑ / Alt+↓` bullet, so it renders as a lazy continuation *inside* that list item rather than as its own section. Insert a blank line; consider moving it below the keybind list entirely, since stream hygiene isn't a keybind.
- Plan item 8 asked for an `n`-final derived negative ("DSR *reports* other than 6n"); the keep-list has no `n`-final row (finals covered: `h l u q m c y R` + OSC). Add `\x1b[0n` or `\x1b[5n`.
- `queries.go:44-51`: trailing comments in `terminalQueryLiterals` sit at two different columns (the six CSI rows at one, the two OSC rows one further right) — looks non-`gofmt`. No `gofmt`/`go vet` gate exists in `Makefile.local`, so nothing catches it; worth a `gofmt -w`.
- `stripTerminalQueries` appends one byte at a time for the ~99.9% non-escape case over up to 128 KiB. Correct and capacity-preallocated, so not a real cost on a tab switch, but `bytes.IndexByte(buf[i:], 0x1b)` + bulk copy would be both faster and shorter.
- `workshop/issues/000128-share-escape-framing.md:54-62` has leftover template scaffolding — a second empty `## Spec`, a second empty `## Done when`, and an empty `## Plan` checkbox — below the real content.
- Working tree carries an uncommitted `README.md` modification at this boundary (present in `git status` at session start, absent from the review window). Confirm it's unrelated before closing.

## 5. Test coverage notes

Coverage of the *specified* behavior is genuinely good: 12 strip rows, 14 negatives, 4 truncation/bisection cases, the redraw pin, both live-path invariant pins, and three new pump cases. Two things about it are worth saying precisely:

- The `run_test.go:177` case ("keystroke after mouse release is not swallowed") is a real regression test — under the old code the release plus `a` would coalesce into a single `write:\x1b[<0;8;2ma` at EOF, so the expectation `write:\x1b[<0;8;2m,write:a` fails. That matches the `## Log` claim; I verified the mechanism against `pumpStdinWithTimer`'s `held` path rather than taking the log's word.
- The gap is entirely on the **malformed-input** axis (I2), and C1 sits exactly in it. Every strip test feeds a syntactically valid sequence; nothing feeds a truncated-but-terminated, overlapping, or garbage OSC. `TestStripTerminalQueriesHandlesTruncatedSequences` covers *unterminated*, which is a different failure mode (and is handled correctly).
- `TestRedrawSnapshotIsRaceFree` pins the new shape but cannot detect the old bug — it takes the snapshot under the lock itself, so it would have passed pre-fix too if written against the old signature. That's fine as a forward pin; just don't read it as proof the race existed.

## 6. Architectural notes

- **ARCH-DRY — flag (Minor).** The headline consolidation (`sgrMouseTerminators`, four derived consumers) is exactly right and the bug *was* the divergence, so this is the correct lesson learned. The cross-package `wrapcmd` deferral is well-argued (`otherEscRe` is consumed three structurally different ways) and properly recorded in #128 + atlas — that is a separable extension, not a deferred purpose. What slipped through is the *in-package* triple of final-byte scans (Minor above) and the duplicate residual test (I1).
- **ARCH-PURE — pass.** `stripTerminalQueries` is genuinely pure and its 30 test cases run with zero IO, no fakes, no clock. `redrawTab` is down to two writes and knows nothing about the mutex. This is the cleanest part of the diff.
- **ARCH-PURPOSE — pass.** Both defects in the Problem section are actually fixed, not just the cheap one. Shadow-sweep on the constant: all four framing sites derive from `sgrMouseTerminators`; no hand-maintained restatement remains. The residual (in-flight query, tab switch) is scoped out explicitly in the Spec and Done-when, not smuggled into a follow-up — that's a legitimate narrowing, not an under-delivery. (I1 is about the *test* for it, not the scoping.)
- **ARCH-MOCK — pass, with one note for #128.** No stateful fake for the host terminal, deliberately: the design *deletes* the cross-call round trip rather than modeling it, so there's no persisted state left to fake, and the Spec argues this explicitly. Worth carrying forward, though: the deny-list's two failure directions are **not** symmetric. A missed query degrades benignly (the Spec's argument, and it holds). An **over-strip** does not — it silently removes mouse mode, Kitty encoding, or the cursor shape, and the only thing standing between the code and that is a hand-written negative list with no live conformance check. The fuzz/property test in I2 is the cheap structural guard; a conformance check against real emitters would be the expensive one, and I'd note in #128 that the shared framing helper is the natural place to attach it if that ever becomes worth building.

## 7. Plan revision recommendations

Two `## Revisions` entries for `workshop/issues/000127-term-pane-stream-corruption.md`:

1. **Correct the residual-pin claim.** Plan item "Pin the accepted residual" (line 339) is checked, but the delivered `TestReplyGoesToActiveTabNotTheQueryingTab` never constructs two tabs or switches between them and is behaviorally identical to `TestPumpStdinForwardsRepliesToChild`'s first case. Either deliver the two-tab test or revise the item to say the residual is recorded in `## Log` + `atlas/architecture.md` and pinned only as "a reply reaches the active tab."

2. **Reconcile the Estimate section with itself.** Three internal contradictions, all of which pollute velocity calibration if left standing:
   - Line 113 says "Design buffer is **+15%**, not +30%"; line 119 says "Design buffer is +30%." The ```estimate``` block uses `0.15`. Delete the stale line 119 sentence.
   - Line 134-138 ("Item mapping") describes **three** `smaller-go-module` rows including "the test suite priced on its own row"; the block at lines 121-131 contains **two**. Either add the third row (and re-total) or fix the prose.
   - The 2026-07-28 revision at line 165 states the estimate moved "1.05 → **1.47**," but the frontmatter and block both say **1.40** (which is arithmetically consistent with the block: design 0.43 × 1.15 + impl 0.91 = 1.40). Record 1.40 as the landed number, or restate why 1.47 was reduced.

---

## Re-review — 2026-07-28T17:29:33-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 127 — right terminal pane corrupts the input stream |
| repo | pair |
| issue file | workshop/issues/000127-term-pane-stream-corruption.md |
| boundary | whole-issue close |
| milestone | — |
| window | 70f6ac0651dccf0424527c13ce7730885eaabaec..HEAD |
| command | sdlc close --issue 127 |
| reviewer | claude |
| timestamp | 2026-07-28T17:29:33-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

The blocking defect from the prior pass is genuinely fixed, not papered over: `isParameterizedOSCQuery` now guards `len(body) < 6` (`queries.go:190`), and I hand-traced the three previously-panicking inputs (`\x1b]4;?\x07`, `\x1b]4;?\x1b\\`, `\x1b]4;;?\x07`) through `terminalSequenceAt` → `oscEnd` → the slice — all three now return `false` and pass through unchanged, matching the new keep-rows at `queries_test.go:68-70`. `FuzzStripTerminalQueries` (`queries_test.go:215`) closes the robustness gap that let the panic ship, the false residual-pin test is deleted with an honest `## Revisions` entry (issue:420), and `csiEnd` is now the single final-byte scan for all three in-package callers. I traced every strip/keep/truncation assertion by hand against the implementation and each one holds; I also confirmed `\x1b[?...$p` cannot overlap the way OSC 4 did (min matching length 5, `seq[3:3]` → empty → rejected), so its twin is not lurking. Nothing left is blocking correctness. What keeps this off SHIP is one traceability gap — a ~110-line user-facing README rewrite rides in this window with no record in the issue's `## Log`, while the two genuine side-quests *were* recorded — plus four cheap Minors that survived the last pass. **Confidence is medium, not high, for one reason I have to state plainly: `Bash` is unavailable in this environment (EPERM on the harness session-env), so I could not run `go test -race ./cmd/internal/termcmd/`, `make test`, `gofmt`, or any `git` command.** Every claim below is from reading the tree, not from executing it; the `## Log`'s "green / exit 0" claim is unverified by me.

## 1. Strengths

- **The C1 fix is minimal and correctly placed.** `queries.go:185-192` — the guard is on `len(body)`, before the slice, with a comment that explains *why* six (prefix and suffix overlap) rather than restating the code. `isParameterizedCSIQuery` was correctly left alone; I verified it is not vulnerable to the same overlap.
- **The negative rows are the right three.** `queries_test.go:68-70` pins both terminator forms of the malformed OSC 4 *and* the empty-index form (`\x1b]4;;?`), which exercises the `len(digits) == 0` arm rather than the length guard — a different code path, not a third copy of the same case.
- **`FuzzStripTerminalQueries` seeds are chosen from the failure class, not decoration.** `queries_test.go:216-220` includes `\x1b]4;?\x07`, `\x1b]4;\x07`, `\x1b[?$p`, and a bare `\x1bP+q\x1b\\` DCS — the exact shapes where framing and slicing interact. The `len(out) > len(in)` invariant is a real (if weak) property, not just `_ =`.
- **`csiEnd` consolidation is complete and behavior-identical.** `rename_input.go:193` (`csiEnd(input) < 0`) and `rename_input.go:208` are line-for-line equivalent to the loops they replaced, including the `'O'` case falling through to the same scan. Three copies → one, inside the package, with the divergence lesson written on the function (`queries.go:131-135`). `ARCH-DRY`.
- **The residual claim was corrected the honest way.** The duplicate test is gone, its reasoning moved onto `TestPumpStdinForwardsRepliesToChild` (`queries_test.go:145-151`), and issue:420 records that the item was ticked but not delivered. Taking option (b) and *saying so* is better than a two-PTY test written to satisfy a checkbox.
- **`sgrMouseTerminators` remains a real single source.** Re-swept: `run.go:581`, `run.go:596`, `run.go:622`, `rename_input.go:180`. No residual hardcoded `'M'`/`'m'` framing anywhere under `cmd/`.
- **The bulk-copy rewrite in `stripTerminalQueries` (`queries.go:69-78`) is correct at the loop-termination level** — `next` can never be 0 (guarded by `buf[i] != 0x1b`), and every `terminalSequenceAt` success returns `size >= 2`, so no input can spin.

## 2. Critical findings

None. C1 from the prior pass is fixed and covered.

## 3. Important findings

**I1 — Undeclared scope in the review window: `README.md` (~110 lines changed) has no record in the issue.**

`workshop/issues/000127-term-pane-stream-corruption.md:393-405` records exactly two side-quests (`tests/scrollback-open-test.sh`, the `-race` test writer) with rationale. The README rewrite — which adds the `Alt+l` / `Alt+c` / `Alt+Shift+C` feature sections, the context-meter paragraph, the Required→Optional dependency split (`jq`, `par`), the per-repo session-scoping note, the Troubleshooting section, and deletes the `pair-shell`-retired / `PAIR_DATA_DIR` runtime-extraction paragraph — is not mentioned anywhere in Spec, Plan, or Log.

I checked the content and it is *accurate*, so this is a bookkeeping finding, not a correctness one: `CHANGELOG.md`, `doctor/README.md` and the pensive link all exist; `:PairDoctor` exists (`nvim/doctor.lua`); and `jq` no longer appears in any Go code path (only in historical comments — `zellijpane` parses `list-panes --json` natively), so demoting it to Optional is right.

The cost of leaving it: `sdlc close` measures and adopts actual hours across this window, so README time gets silently attributed to a two-defect stream bugfix and pollutes velocity calibration — the exact thing the actual-gate exists to prevent. It also means the doc rewrite has no `## Log` trace for whoever later asks "when did the dependency table change and why".

Fix: add a `side-quest:` bullet under `## Log` in the same shape as the other two, naming what changed and why it rode along (the Log already notes this issue was filed "while doing v1.24 release prep" — that's the reason, it just isn't connected). Mention it in `--verified` too. If it actually belongs to a different issue, say which.

## 4. Minor findings

- `atlas/architecture.md:396-397` — **the prior pass flagged this and it was not fixed.** There is still no blank line between the `Alt+↑ / Alt+↓` bullet and `**\`pair term\` stream hygiene (#127).**`, so those two lines are a CommonMark lazy continuation of that list item: the section-header sentence renders *inside* the keybind bullet, and the `*Input.*` / `*Output, replay only.*` bullets then render as two more siblings of the keybind list. One blank line fixes it; moving the whole block below the keybind list (it is not a keybind) is better.
- `run.go:577` — **also unfixed from the prior pass.** `if !strings.HasPrefix(s, "\x1b[<") || s == ""` — the `s == ""` arm is unreachable; `HasPrefix` short-circuits first and already guarantees `len(s) >= 3` for the `s[len(s)-1:]` on the next line. Drop it, or move it first if it was meant as that guard.
- `queries_test.go:46-71` — still no `n`-final negative. Plan item at issue:321 explicitly lists "`n` → DSR *reports* other than 6n", and Done-when promises "a derived negative case per matched final byte". `\x1b[6n` is a literal so the risk is nil, but the promise is unmet by one row: add `\x1b[0n` or `\x1b[5n`.
- `workshop/issues/000127-term-pane-stream-corruption.md:262` — the item header says "a literal set + **three** narrow parameterized rules", while its own body (issue:283) says "**Two** parameterized rules" and the code has two (`isParameterizedCSIQuery`, `isParameterizedOSCQuery`). Fix the header to two.
- `workshop/issues/000127-term-pane-stream-corruption.md:339` — the "Pin the accepted residual" item is still `[x]` with text reading "Add a test asserting exactly that". The `## Revisions` entry at :420 correctly discloses it wasn't delivered, so this is compliant with the append-don't-overwrite rule, but a reader scanning `## Plan` sees a ticked claim the code doesn't back. Consider `[~]` or an inline "(revised — see Revisions)".
- `queries.go:98-108` — an OSC literal match (`\x1b]10;?` / `\x1b]11;?`) calls `oscEnd` over the *whole* remaining buffer, so an unterminated OSC query would consume forward to the next BEL/ST anywhere and strip it. Real terminals behave the same way and the input is malformed by construction, so this is a note rather than a defect — but it is the one place strip can eat arbitrary visible bytes.
- I could not run `gofmt`. For the record, the `terminalQueryLiterals` comment alignment the prior pass called non-gofmt (`queries.go:44-51`) is in fact correct — the six CSI rows and two OSC rows both land at the same column once `\x1b` is counted as four source characters. No change needed there.

## 5. Test coverage notes

- The shipped-bug class is now covered on both axes: the specific inputs (`queries_test.go:68-70`) and the structural guard (`FuzzStripTerminalQueries`). That is the right pairing — the explicit rows document the failure, the fuzz catches its siblings.
- The fuzz property is weaker than the one the prior pass suggested. It asserts *no panic* and *no growth*; it does not assert the real invariant, "every non-escape byte survives in order." That stronger property would also cover the never-drop-to-end rule and would catch an over-strip regression that a hand-written keep-list misses. Worth adding when #128 touches this code; not worth blocking on now.
- `TestRedrawTabEmitsNoQueries` (`queries_test.go:104`) is a genuine end-to-end pin of the fix — I confirmed by hand that `\x1b[c` and `\x1b[?2026$p` are removed while `\x1b[?1006h` and both visible-text fragments survive, and that the emitted `\x1b[1;1H\x1b[J` prefix does not accidentally satisfy the `strings.Contains("\x1b[c")` negative.
- `TestLiveQueryPathIsUnfiltered` and `TestPumpStdinForwardsRepliesToChild` correctly pin the invariant that justifies having no input-side filter. I traced the latter through `pumpStdinWithTimer`: none of the three reply forms matches a chord, `findSGRMousePress`, or `isSGRMousePrefix`, so each falls to `mux.writeActive(data)` — the expectations hold.
- The three new pump cases (`run_test.go:176-178`) are real regression tests. I re-derived the byte offsets for the combined `\x1b[<0;8;2M\x1b[<0;9;2mls\n` case: `IndexAny` lands on index 8 in each event, `raw` is 9 bytes, `rest` is exactly `ls\n` — the expected three-op sequence is what the code produces, and the pre-fix code would have coalesced them in `held`.
- **Unverified by me:** `go test -race ./cmd/internal/termcmd/`, `make test`, and the live-session check. All three are claimed green at issue:406-408. Given the known env-leak failure mode in this checkout, re-running `make test` with a scrubbed env before recording the close verdict is worth the thirty seconds.

## 6. Architectural notes

- **ARCH-DRY — pass.** Both consolidations verified by sweep, not by claim: `sgrMouseTerminators` has four derived consumers and zero hand-maintained restatements; `csiEnd` has three. The duplicate residual test that was the last in-diff repetition is gone. The cross-package `wrapcmd` framing duplication is deferred with a well-argued rationale (issue:199-229) and a filed, non-scaffolded follow-up (`workshop/issues/000128-share-escape-framing.md`) — a separable extension, not a deferred purpose.
- **ARCH-PURE — pass.** `stripTerminalQueries`, `terminalSequenceAt`, `csiEnd`, `oscEnd`, and both `isParameterized*` predicates are deterministic and side-effect-free; the whole `queries_test.go` suite runs with no exec, no net, no clock (the two `time` uses are polling deadlines in the two tests that intentionally drive the IO shell). `redrawTab` (`run.go:1093`) is down to two writes and knows nothing about `m.mu`; `bufferSnapshotLocked` is a nil-safe pure copy. I re-verified all three call sites snapshot inside the lock they already held (`run.go:698`, `:879`, `:926`) and that `removeTab`'s `empty` path returns before any redraw, so the nil snapshot is never written.
- **ARCH-PURPOSE — pass.** Both Problem-section defects are fixed, and the fix pass addressed the substance of every prior finding rather than the cheapest one — including the one that required *deleting* a test and admitting a ticked item wasn't delivered. The accepted residual stays scoped out explicitly in Spec/Done-when and is now recorded in `## Log` and atlas rather than falsely claimed as pinned, which is the honest form of "narrowed, not closed."
- **ARCH-MOCK — pass, with the asymmetry now carried forward.** No new external-binary seam; zellij calls stay behind `Runtime`/`fakeRuntime`. The deliberate absence of a host-terminal fake still holds: the design deletes the cross-call round trip instead of modeling it, so there is no persisted state to fake. The important addition since the last pass is that the deny-list's non-symmetric failure directions (missed query → benign; over-strip → silent loss of mouse mode / Kitty encoding / cursor shape) are now written into `workshop/issues/000128-share-escape-framing.md:61-66`, pointing at the shared framing helper as the natural attachment point for a conformance check. That is the right place for it.

## 7. Plan revision recommendations

Three `## Revisions` entries for `workshop/issues/000127-term-pane-stream-corruption.md` — all small, none blocking:

1. **Record the README side-quest.** Add a `side-quest:` bullet under `## Log` alongside the existing two, naming the `README.md` rewrite in this window (feature sections for `Alt+l` / `Alt+c` / `Alt+Shift+C`, the Required→Optional dependency split, the Troubleshooting section, the removed `pair-shell`/`PAIR_DATA_DIR` paragraph) and why it rode along with a stream bugfix. Without it, `sdlc close`'s measured actuals attribute release-prep doc time to #127.
2. **Fix the "three parameterized rules" header** at issue:262 to "two" — its own body at issue:283 and the code both say two.
3. **Note the one unmet negative.** Done-when promises "a derived negative case per matched final byte"; the `n` final has no keep-row. Either add `\x1b[0n` to `TestStripTerminalQueriesPreservesLegitimateSequences` and leave Done-when alone, or narrow the Done-when line to the finals actually covered.
