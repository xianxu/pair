# Boundary Review — pair#138 (whole-issue close)

| field | value |
|-------|-------|
| issue | 138 — Claude Return rewrite only in composer |
| repo | pair |
| issue file | workshop/issues/000138-claude-return-rewrite-only-in-composer.md |
| boundary | whole-issue close |
| milestone | — |
| window | 8718f50be0c75da57c3306d97d112c8bd0dc289d..HEAD |
| command | sdlc close --issue 138 |
| reviewer | claude |
| timestamp | 2026-08-20T09:56:44-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The core of this boundary is right and well-built: extracting `ruledBoxComposerActive` instead of writing a third near-copy of Muse's recognizer was the correct call (it keeps pair#139's two hard-won Muse fixes in one place), the Muse migration is genuinely behavior-preserving (commit `42c1a06` touches zero test rows, so the frozen differential really is an unmodified oracle), and the structural glyph-/colour-agnostic Claude spec is the right answer to the bash-mode evidence. `go build`, `go vet`, and `go test ./... -count=1` all pass clean. What keeps this off SHIP is a set of cheap, concrete gaps: a 20-row height ceiling that now makes plain Return **submit** a long draft on Pair's default agent (a real regression, not just a theoretical one); an atlas sentence in the very paragraph this diff edited that still says all four harnesses "rewrite to LF" — the exact PQ-1 error, one paragraph up from where PQ-1 was fixed; no Claude analogue of the `composer inactive → bare CR` routing test that codex/muse/agy each have; and a Task 5 step ticked `[x]` whose required evidence (terminal size, theme, version; permission-prompt confirm) is not in the Log.

## 1. Strengths

- **`cmd/internal/wrapcmd/composer_recognizers.go:100-176` — the shared-spec extraction (ARCH-DRY).** `ruledBoxComposerSpec` + `ruledBoxComposerActive` is the right seam: Muse and Claude differ only in `promptOK`/`ruleAt`/`rulesMatch`, and the multi-line and cursor-on-rule fixes now live once. `git show 42c1a06 --stat` confirms the refactor commit modified **no** test file, so the thirteen Muse differential rows and `muse/0.1.0-R708.1/composer.raw` are a real oracle, not a co-edited one. The only behavioral delta is the added `promptY-1 < 0 → continue` guard, which is a no-op because `terminalSnapshot.CellAt` (`terminal_model.go:57-65`) already returns `nil` for negative `y`.
- **`harness_tty_fixture_test.go:301-306` + `:369` — the LF assumption is now derived, not restated.** `composerReturnBytes` reads `profile.keymap.plainCR`, so Claude's fixture fails on *recognition* rather than on an unrelated hardcode. This is exactly what PQ-1 asked for.
- **`harness_tty_test.go:54-63` — the recognizer expectation now derives from the gate** (`wantRecognizer := want.gate == composerGatePositive`) instead of restating the harness list, and the pointer identity check generalized from agy-only to all four. A positively gated profile with a nil recognizer can no longer slip in.
- **Fail-closed wiring is genuinely fail-closed.** `wrap.go:2338-2343`: a `newTerminalModel` failure now kills the child and errors out rather than silently downgrading Claude to no-remap — important, because building a terminal model for Claude is new in this diff.
- **`harness_tty_fixture_test.go:316-328` — the split-replay is doing real work.** Replaying `claude/2.1.237/composer.raw` and `bash-mode.raw` at every prefix split proves the gate's answer is stable across PTY chunk boundaries, including mid-paint cursor parking. Both fixtures' SHA-256 digests verify against the checked-in bytes.
- **`bash-mode.raw` is the right second positive.** It repaints both the glyph (`!`) and the rule colour (pink `253;93;177`) — pinning it as `wantComposer: true` is precisely the fixture that catches a future colour- or glyph-pinned regression.

## 2. Critical findings

None.

## 3. Important findings

**I1 — `claudeComposerMaxRows = 20` makes plain Return submit a draft longer than 20 lines, on Pair's default agent** (`cmd/internal/wrapcmd/composer_recognizers.go:184`, enforced at `:127` and `:153`).

The loop condition is `snapshot.Cursor.Y - promptY < spec.maxRows`, so the prompt may sit at most 19 rows above the cursor. With the cursor on the 21st line of a draft, no box is found, the gate declines, and `decidePlainReturn` emits bare CR — which Claude reads as **submit**. Before this diff (`composerGateLegacy`) that same keystroke inserted a newline. This is the exact "lost draft" failure the plan's own blast-radius table names as the expensive direction, and multi-line drafting is the entire reason the Return remap exists. The code comment concedes the bound is "Inherited from the box-structure derivation rather than measured against Claude" and argues "the enclosing rules already prevent pairing distant chrome" — which is correct, and is the argument for not having the bound at all: the top rule must be *immediately* above the prompt, and `ruledBoxBottomRule` takes the *first* painted column-0 row below it, so nothing distant can pair up regardless of `maxRows`.

Fix sketch: raise Claude's bound to the snapshot height (or drop the `maxRows` guard from both loops for specs that set `rulesMatch`/immediate-top-rule anchoring), and update the `"draft taller than the height ceiling"` row in `composer_recognizers_test.go:340-348` to pin the new behavior. Muse carries the same 20-row bound (`:98`) and would benefit identically, though that is pre-existing.

**I2 — `atlas/architecture.md:519` still says all four harnesses "rewrite to LF", contradicting Claude's `plainCR` two sentences earlier.**

The paragraph reads: *"For `claude` plain Enter becomes `\<CR>` … All four harnesses are positive-gated …: plain Enter rewrites to **LF** only when the harness's recognizer confirms a live composer."* Claude's `keymap.plainCR` is `{'\\','\r'}` (`harness_tty.go:26`). This is the same LF-restatement error PQ-1 raised; the Conformance paragraph at `:529` was correctly fixed to say "the harness's own `keymap.plainCR`", but `:519` — in the paragraph this diff also edited — was not. ARCH-DRY/ARCH-PURPOSE: the profile registry is the source, and this line is a hand-maintained restatement that now disagrees with it. Fix: reword to "rewrites to the harness's `plainCR`".

**I3 — `atlas/how-to-bring-up-a-new-harness-cli.md:54` still carries the stale Muse claim PQ-7 flagged, on the line this diff edited.**

The bring-up guide says *"**For `muse`:** a non-faint `⟩` at column 0 **within one row of the cursor**, between faint `─` rule rows."* That description was superseded by pair#139's enclosing-rule implementation and is now flatly wrong — `museComposerActive` anchors on the enclosing rules and permits up to `museComposerMaxRows` between prompt and cursor. `atlas/architecture.md:527` was corrected in this diff ("with the cursor anywhere in the box"); the guide's copy of the same sentence, sitting immediately after the newly inserted Claude paragraph, was not. Fix: mirror the architecture.md wording.

**I4 — no Claude `composer inactive / unknown → bare CR` routing test.**

`codex_return_test.go:17`, `muse_return_test.go:22` and `:42`, and `agy_return_test.go:17` each pin their harness's declining path through `emitPlainCR`. There is no `claude_return_test.go` and no equivalent assertion anywhere (`grep 'func TestEmitPlainCR' cmd/internal/wrapcmd/*_test.go`). The one Claude test that *sounds* like it covers this — `codex_return_test.go:38 TestEmitPlainCR_NonCodexKeepsExistingRemap` — now asserts the opposite (it passes only because `claudeProxy()` paints a composer), and its name is stale post-flip. This is the harness whose gate this issue flips, and it is the default agent; the declining branch is the newly reachable one. The `newHarnessSessionFake` seam makes it ~6 lines:

```go
func TestEmitPlainCR_ClaudeComposerInactiveSendsBareCR(t *testing.T) {
	f := newHarnessSessionFake(t, "claude", true)
	t.Cleanup(f.close)
	if got := f.proxy.emitPlainCR(nil); !bytes.Equal(got, []byte{'\r'}) {
		t.Fatalf("got %q, want bare CR without an active Claude composer", got)
	}
}
```

Also rename `TestEmitPlainCR_NonCodexKeepsExistingRemap` to say what it now checks (Claude remaps *while its composer is recognized*).

**I5 — Claude's slash-menu state was captured during design but not checked in, leaving `ttyFixtureDiscriminationGaps["claude"]` resting on an unpinned observation** (`harness_tty_fixture_test.go:157`).

The gap entry asserts *"its slash menu renders below the box and leaves column 0 blank"* and immediately concedes *"that is an observation, not a proof."* Agy pins exactly this property as a checked-in fixture (`ttyFixtureExpectation["agy"] = {"menu.raw": true}`, driven scenario at `harness_tty_live_test.go:705-706`). Unlike Claude's permission prompt, the slash menu needs **no tool call and no permission mode** — it is reachable with `send: "/"`, so the child-session auto-approve blocker does not apply. Adding it is a 4-line driven scenario plus one `ttyFixtureExpectation` row, and it converts the load-bearing observation into evidence. Fix sketch:

```go
{name: "slash menu", send: "/", until: "Navigate", wantComposer: true, file: "menu.raw"},
```
plus `"claude": {"bash-mode.raw": true, "menu.raw": true}`.

**I6 — Task 5 is ticked `[x]` but two of its required verifications are absent from the Log** (`workshop/plans/000138-...-plan.md` Task 5 Steps 1 and 3; issue `## Log` 2026-08-20).

- Step 1 item 4 — *"plain Return **confirms** a real permission prompt rather than inserting"* — is not reported anywhere. The Log instead records that the prompt was unreachable. So the issue's central safety property is verified by **neither** fixture nor dogfood, while the plan checkbox claims it was. Step 1 items 2 (`#` memory mode) and 5 (draft-pane Alt+Return unchanged) are likewise unreported.
- Step 3 explicitly requires naming **terminal size, theme, and Claude version**, *"since the recognizer's colour-matching invariant is theme-sensitive"* (`sameForeground` at `composer_recognizers.go:216`). The Log names none of the three.

This is a traceability finding, not a demand to re-do the dogfood: either record the missing facts and the memory-mode/draft-pane results, or un-tick the sub-items and say plainly in the Log what was and was not exercised. (For what it's worth, structural reasoning says the recognizer *does* decline on Claude's permission dialog — it paints `╭`/`│`/`╰` at column 0, which `ruleAt` rejects since it requires exactly `─` — but that is inference, not the evidence the plan committed to.)

## 4. Minor findings

- `composer_recognizers.go:201-204` — `ruleAt` accepts a **single** `─` at column 0 as a rule row; Claude's real rules span the pane width (118 cols in both fixtures). Agy already requires `minBorderLength = 5` (`:238`). Adding a small minimum run-length to the Claude spec would materially shrink the false-positive surface at zero false-negative cost, which is the concrete hardening `ttyFixtureDiscriminationGaps["claude"]` is asking for.
- `composer_recognizers.go:194-195` — the comment claims *"Requiring the box's two rules to share a foreground is what keeps unrelated chrome from pairing into a composer."* It only rejects *mismatched* colours; two default-foreground `─` rows pass via `sameForeground(nil, nil) == true`. Soften the claim to match.
- `harness_tty_fixture_test.go:379-397` — `TestComposerReturnExpectationMatchesProfile` says it pins that the expectation is *derived*, but it hardcodes the same four-entry `plainCR` table `TestHarnessTTYProfileRegistry` (`harness_tty_test.go:22-26`) already pins. Reverting `replayHarnessTTYFixture` to a literal `"\n"` would leave this test green; the Claude fixture is the actual guard. Either drop it as redundant (ARCH-DRY) or restate its comment.
- `harness_tty_live_test.go:545` and `:555` — the skip and fatal messages still read *"want agy, codex, or muse"* although `commands` (`:548`) now accepts `claude`. Anyone running the live check for Claude is told it is unsupported.
- `translate_stdin_test.go:70-81` — `claudeProxy()` builds a terminal model (which spawns a drain goroutine, `terminal_model.go:94-97`) at five call sites and never calls `closeTerminal`; every other helper in the package defers it. It also `panic`s rather than taking `*testing.T` and calling `t.Fatalf`, so a failure loses the test name.
- `harness_tty.go:9` — `composerGateLegacy` now has no registered consumer, and `decidePlainReturn`'s `case composerGateLegacy: return remap()` (`wrap`-side `harness_tty.go:114-115`) is a live "always remap unconditionally" branch reachable by a one-word edit, in a system whose whole design is fail-closed. The issue Log defers deletion over enum renumbering — but nothing persists these ordinals. Minimum: assert in `TestHarnessTTYProfileRegistry` that no registered profile uses it, so the observation becomes an enforced invariant.
- `README.md:109` — *"in a permission picker, **a selection menu**, or any state it doesn't recognize, Return stays a plain Enter and the dialog confirms."* This is false for selection menus on both agy (`ttyFixtureExpectation["agy"]["menu.raw"] = true` — gate deliberately stays open) and Claude (the plan's own capture shows the box intact with the cursor inside during `/`, so Return inserts a newline and dismisses the menu rather than running the command). The sentence is pre-existing, but this diff rewrote it and extended it to the default agent, where `/` and `@` menus are used constantly. Narrow it to permission pickers.
- The checked-in Claude fixtures were captured from a *child* session — both `composer.raw` and `bash-mode.raw` carry the `⚠ Transcript saving is off — inherited CLAUDE_CODE_CHILD_SESSION marker` footer and `⏵⏵ auto mode on`. Harmless for a shape-based recognizer, but worth a line in `metadata.json`/the ledger so a future recapture isn't confused by the difference.

## 5. Test coverage notes

- **Covered well:** active composer end-to-end through the production path (`replayHarnessTTYSplit` calls the real `emitPlainCR`, `harness_tty_fixture_test.go:365`), across every prefix split; mode variance via `bash-mode.raw`; overlay-beats-composer for Claude, now genuinely meaningful because `claudeProxy()` paints a live composer (`picker_overlay_test.go:123`, `:110`); malformed-snapshot safety via the adversarial guard, which correctly gained `claude` (`composer_recognizers_test.go:431`); box-integrity, multi-line growth, empty/blank continuation rows, and cursor-on-closing-rule via the new differential (`composer_recognizers_test.go:268-347`).
- **Gaps:** I4 (no Claude declining-path routing test), I5 (no `menu.raw`), and the acknowledged-but-uncaptured `overlay.raw`. Together these mean **every** Claude test in the tree asserts the gate saying *yes*; nothing asserts it saying *no* on a real Claude screen.
- **Uncommitted work in the tree:** `TestHarnessTTYIntegration_ClaudeOverlayBeatsComposer` (`harness_tty_integration_test.go:89-113`) is a good addition — fake-session-level overlay precedence with a one-shot-flag assertion — and it passes. It is not in the reviewed commit range; commit it with the fixes.

## 6. Architectural notes

- **ARCH-DRY — pass, with I2/I3 flagged.** The `ruledBoxComposerSpec` extraction and the profile-derived `composerReturnBytes`/`wantRecognizer` are model applications of the principle. The two doc findings are the residue: `atlas/architecture.md:519` and the bring-up guide both hand-restate facts the registry and the recognizers own. Trivial duplication also exists in the two specs' identical `minCursorX: 2` + comment (`:173-174`, `:209-210`) — not worth extracting.
- **ARCH-PURE — pass.** `claudeComposerActive` and `ruledBoxComposerActive` are pure `terminalSnapshot → bool`; the differential tests feed an in-memory `terminalModel` with no exec, net, or fs. Business logic (`decidePlainReturn`) stays out of the IO shell, and `configureHarnessTTY`/`handleChunk` remain the thin seam. Sole nit under this lens is the `claudeProxy()` helper doing unmanaged IO-ish setup (Minor above).
- **ARCH-PURPOSE — flag (I5, I6).** The shadow-sweep over "positively gated" consumers: all four profiles derive from `harnessTTYProfiles` ✓; the fixture harness derives its Return expectation from the profile ✓; `doctor/README.md:41` correctly derives ✓; `atlas/architecture.md:519` does **not** ✗. On purpose-delivery: the Spec's motivating sentence is *"permission prompts, pickers, and future Claude UI variants do not depend on enumerating every non-composer menu."* The mechanism ships, but the declining side rests entirely on the single exact-match OSC 777 body (`wrap.go:636`) with no fixture and no dogfood behind it, while Task 5's checkbox claims otherwise. The plan branched on this correctly (outcome 3) and both ledgers record it honestly — the remaining ask is to stop the plan/issue from claiming more than was verified, and to take the *reachable* capture (I5).
- **ARCH-MOCK — pass, with the conformance caveat already documented.** Frozen fixtures replay through the same `proxy.handleChunk` boundary production uses, and `newHarnessSessionFake` is a proper stateful double; there are no direct external calls outside the seam. The live conformance check is manual and unscheduled — correctly stated in `atlas/architecture.md:529` — but the affordance for Claude is half-wired (I5's missing menu scenario; the stale skip/fatal messages at `harness_tty_live_test.go:545`/`:555`). The `ttyFixtureNegativeGaps`/`ttyFixtureDiscriminationGaps` mechanism, which *fails the build* when an acknowledgment outlives its gap, is a genuinely good instance of this principle.

## 7. Plan revision recommendations

Append a `## Revisions` entry to `workshop/plans/000138-claude-return-rewrite-only-in-composer-plan.md`:

1. **Core concepts — `ttyFixtureReturnExpectation` row.** The table lists it as `modified` in `harness_tty_fixture_test.go`, and the Task 3 bullet says it "must also yield *what bytes* that means." The shipped code leaves `ttyFixtureReturnExpectation` (`:260-266`) untouched and adds a sibling `composerReturnBytes` (`:269-276`) that `replayHarnessTTYFixture` calls. The intent (derive from the profile) is delivered; the named entity is not the one that changed. *(Reported here as a plan-bookkeeping mismatch rather than Critical: the entity exists at the stated path, the derivation is real, and the actual shape is arguably cleaner than the one planned.)* Retitle the row to `composerReturnBytes` / `replayHarnessTTYFixture`, status `new`/`modified`.
2. **Task 5 Step 1 and Step 3 dispositions.** Record which sub-items were actually exercised (multi-line insert ✓, bash mode ✓, Alt+Return submit ✓) and which were not (permission-prompt confirm, `#` memory mode, draft-pane Alt+Return), and either supply the terminal size / theme / Claude version Step 3 demands or note explicitly that the dogfood ran through `pair wrap claude` in a bare PTY rather than a live `pair` session, and what that does not cover.
3. **Task 6 Step 1 residual.** The step named `atlas/architecture.md`'s Conformance "must remap to LF" sentence (fixed at `:529`) but missed the second LF restatement at `:519`, and named only `architecture.md` for the stale Muse "within one row of the cursor" wording while the same sentence survives in `atlas/how-to-bring-up-a-new-harness-cli.md:54`. Extend the step to both lines.
4. **Captured-evidence version.** The evidence section is headed "Claude Code 2.1.236"; the checked-in fixture directory and `metadata.json` are `2.1.237`. Note the recapture so the plan and the fixtures agree.
