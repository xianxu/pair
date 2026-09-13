# Boundary Review — pair#234 (whole-issue close)

| field | value |
|-------|-------|
| issue | 234 — A bare ESC in the right pane is held until the next keystroke: pair term treats it as an unfinished Alt chord and has no ambiguity timer outside rename |
| repo | pair |
| issue file | workshop/issues/000234-right-pane-bare-esc-held-without-timer.md |
| boundary | whole-issue close |
| milestone | — |
| window | a304d55b52cc8648cad416ceb5c7ca665756dd2c..6d19102226deb4aceaa10d49c505d434975e6d35 |
| command | sdlc close --issue 234 |
| reviewer | claude |
| timestamp | 2026-09-12T16:11:23-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The fix is correct and the ordering is sound. I read the whole pump loop rather than the commit message: the plain tail arms for any pending prefix and stops otherwise, a rename that begins from a held ESC hands the timer to `applyRename` whose own tail stops it, and a tick that fired while a read was being processed is drained by the tail's `Reset`/`StopAndDrain`, so no double flush is reachable. The constant has exactly four consumers and no stray 35/50 ms literal survives in the tree. The three packages pass with the race detector outside the sandbox, vet and gofmt are clean, and a scratch copy with the expiry branch deleted turns two tests red. What keeps this off SHIP is name drift between the artifacts and the code, plus an unverified live Done-when.

**Strengths**
- `cmd/internal/termcmd/run.go:521-524` → `applyRename` tail: the "rename begins from a held ESC" ordering the plan named as most likely mishandled is handled, and I could confirm it by reading, not by trusting the Log.
- `run.go:556-566`: the tail arm/stop rule is one place, and it drains a stale tick as a side effect of `Reset`, which is what makes the read-wins race safe.
- `escape_deadline_test.go:84-91`: the generated table guards its own axis (asserts `q` is neither a prefix nor a chord), so a future chord on `q` fails loudly.
- `escape_deadline_test.go:139-175`: the real-timer conformance test asserts only a lower bound and a 1 s ceiling, so it cannot flake and still proves `newRealEscapeTimer` ticks from its stopped state.
- Plan-gate PQ-4's sibling sweep is done: the leaky split test is deleted and the tab-actions table rows moved onto the fake timer.

**Critical findings**
None.

**Important findings**
1. **Plan/atlas name a type that does not exist.** The plan's Core concepts table lists `EscapeTimer` as modified in `run.go`, Task 3 Step 1 renames `RenameTimer` → `EscapeTimer` with a doc comment, and `atlas/architecture.md:455` says "the pump's single `EscapeTimer`". The code at `run.go:354` still reads `type RenameTimer interface` with no doc comment; only the concrete `realEscapeTimer` and the test fake were renamed. This is the table/code contradiction the checklist calls out. I grade it Important rather than Critical because it is name-only with zero behavioral effect and a two-line fix: rename the interface, add the planned doc comment, and the atlas becomes true. If you choose not to rename, the atlas line and the plan table both need correcting instead.
2. **Done-when bullets 1–2 have no live evidence yet.** "One ESC leaves insert mode in nvim" and "ESC then j after the deadline reaches nvim as two keys" are pane-level claims; the pump tests prove the mux forwards after the deadline, not that zellij hands `pair term` a legacy `\x1b`. The Plan's manual step is unticked, honestly. The close should carry the operator's smoke-test result in `--verified` (new split, fresh `pair term`, nvim: ESC, `i` ESC `j`, Alt+j, Alt+t), not waive the unchecked item with `--no-plan-check`.

**Minor findings**
- `escape_deadline_test.go:111-124` and `:17-32`: case (b) and the bare-ESC regression prove the arm, not the flush; with `autoFire` the fake tick and `splitReader`'s EOF race in the select and both branches write `head`. Scratch revert confirmed: deleting the expiry branch leaves all 546 generated cases and `TestBareEscapeIsForwardedAfterTheDeadline` green; only the two gated tests go red. Either gate the EOF (a `gatedChunksReader` with an empty trailing chunk, releasing after `wrote` fires) or rename the case so it claims what it proves. PQ-5's suggested `resets` assertion was applied and this is the residual of that family.
- `shortcut.go:390-391` and plan "Latency budget": 35 ms and nvim's 50 ms `ttimeoutlen` add (worst case ≈85 ms to normal mode), they don't nest, so "never sees a slower ESC than it budgets for" is the wrong sentence. The number is fine; the justification isn't.
- A torn SGR mouse prefix (`\x1b[<0;10;`) now reaches the child as raw bytes after 35 ms. The plan calls this deliberate; no test exercises it (the table covers chord prefixes only). One explicit case would document the choice.

**Test coverage notes**
Race-clean ×2 outside the sandbox. The two gated tests are the ones that actually pin the expiry branch; the generated table pins the arm rule and the chord/non-chord resolution. Sandbox runs fail only on the documented `ptychild` class.

**Architectural notes**
- ARCH-DRY pass: one constant, four readers, grep-verified. ARCH-PURE pass: the pump stays IO glue over pure `FindChord`/`IsChordPrefix`. ARCH-PURPOSE pass: shadow-sweep of consumers complete; the inside-35 ms residual is declared and belongs to #227. ARCH-MOCK pass: reader, timer and mux are injected, with a real-timer conformance test. ARCH-CONSTRAINTS pass with the wording nit above. ARCH-SECURE N/A: local pty stdin, no new parse surface, no secrets. ARCH-ORDER pass: the ownership table matches the code; flagged only the oracle weakness above.
- For #227, the passthrough decision will land between `FindChord` and the tail arm; keep the tail as the single place the timer is touched.

**Plan revision recommendations**
- If the interface is renamed as planned: none. Otherwise a `## Revisions` entry: "`RenameTimer` kept its name; Core concepts row `EscapeTimer` → `RenameTimer`, Task 3 Step 1 not done", and fix `atlas/architecture.md:455` to match.

```findings
findings:
  - id: new
    severity: Important
    family: artifact-names-real-symbol
    title: |
      Plan table and atlas name `EscapeTimer`, but run.go still declares `RenameTimer` with no doc comment
    detail: |
      Core concepts row and Task 3 Step 1 promise the interface rename plus a doc comment; atlas/architecture.md:455 already cites `EscapeTimer`. Only realEscapeTimer and the test fake were renamed. Rename the interface (two lines) or revise plan + atlas to the real name.
  - id: new
    severity: Important
    family: operator-smoke-test-before-done
    title: |
      Done-when bullets 1-2 are pane-level claims with no live evidence; the manual Plan step is unticked
    detail: |
      Pump tests prove the mux forwards after the deadline, not that the right pane's nvim sees one ESC. Carry the operator's smoke test (fresh split, nvim ESC / ESC j / Alt+j / Alt+t) in --verified rather than waiving the unchecked step.
  - id: new
    severity: Minor
    family: oracle-distinguishes-mechanism
    title: |
      Generated case (b) and the bare-ESC regression prove the arm, not the expiry flush
    detail: |
      Scratch revert of the expiry branch leaves all 546 generated cases and TestBareEscapeIsForwardedAfterTheDeadline green; only the two gated tests go red. Gate the EOF behind the observed write, or rename the case to claim what it proves.
  - id: new
    severity: Minor
    family: unsupported-performance-claim
    title: |
      35 ms and nvim's 50 ms ttimeoutlen add rather than nest; the "never slower than it budgets for" sentence is wrong
    detail: |
      shortcut.go:390-391 and the plan's latency budget. The number is fine; state the combined worst case (~85 ms) instead.
  - id: new
    severity: Minor
    family: enumerable-siblings-swept
    title: |
      Torn SGR mouse prefix now flushes raw after 35 ms with no test pinning that deliberate choice
    detail: |
      The generated table covers chord prefixes only; one explicit mouse-prefix case would document the plan's "any held prefix" decision.
```

---

## Re-review — 2026-09-12T16:31:34-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 234 — A bare ESC in the right pane is held until the next keystroke: pair term treats it as an unfinished Alt chord and has no ambiguity timer outside rename |
| repo | pair |
| issue file | workshop/issues/000234-right-pane-bare-esc-held-without-timer.md |
| boundary | whole-issue close |
| milestone | — |
| window | a304d55b52cc8648cad416ceb5c7ca665756dd2c..113666b6d2b0bae221dd1df757f34a02004be39a |
| command | sdlc close --issue 234 |
| reviewer | claude |
| timestamp | 2026-09-12T16:31:34-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

All seven prior findings are disposed as addressed, and each was checked against the code and by running things, not against the Log. The interface is really `EscapeTimer` now with the planned doc comment, and the old name greps to zero outside prose. Scratch reverts in a throwaway worktree turn the right tests red for both halves of the fix. Deleting the expiry branch fails the generated deadline-first case, the torn-mouse test, the ESC-then-j regression and the real-timer test. Deleting the plain tail arm fails the bare-ESC regression, the split-chord test, the mouse test and both gated tests. The live claim holds independently: I built pair from the head commit and from the base commit, ran `probes/escsmoke` against each outside the sandbox, and got 6/6 on head versus 3/6 on the control, failing exactly the three steps the Log names. Both packages pass with the race detector outside the sandbox. Nothing blocks SHIP. The only open item is the operator's in-zellij confirmation, which the close text already declares honestly as the one step not run.

**Strengths**
- `cmd/internal/termcmd/run.go:468-476` and `:561-571`: the timer is touched in exactly two places outside the rename decoder, the owner rule in the doc comment matches what the code does, and the "rename begins from a held ESC" ordering hands the timer to `applyRename` whose own tail drains the plain arm's tick.
- `cmd/internal/termcmd/escape_deadline_test.go:132-160`: `forwardedOnTheDeadline` gates EOF behind the observed write and asserts the arm, so case (b) now distinguishes the expiry flush from the EOF flush. Confirmed by revert.
- `cmd/internal/termcmd/escape_deadline_test.go:167-183`: the mouse-prefix test guards its own premise (mouse prefix, not chord prefix) before asserting, the same self-checking shape the generated table uses for `q`.
- `probes/escsmoke/main.go`: the oracle is nvim's own RPC (`mode()`, `line('.')`), not a screen scrape, and it follows the repo's probe shape (`main` is `os.Exit(run())`, cleanup in defers) so the existing shape test covers it. It sits under `probes/` with the same `./bin/pair` default as `termsmoke`, so `make test-smoke` picks it up with no list edit and it skips cleanly without nvim.
- `workshop/lessons.md`: the BSD-sed `\b` lesson is the right generalisation of BR-3 (grep the old name to zero after any scripted rename).

**Critical findings**
None.

**Important findings**
None.

**Minor findings**
- The plan has no `## Revisions` section recording the post-approval delta: the `probes/escsmoke` integration artifact, the gated case (b) helper, the mouse-prefix test, and the "add, not nest" latency wording. CLAUDE.md asks for an appended Revisions entry when a plan artifact changes mid-stream. Name-only, no code impact.
- `gofmt -l` flags `cmd/internal/couchcore/detachedsessions.go`. It is outside this window and untouched by the diff. Noted for whoever touches that file next, not raised as a finding here.

**Test coverage notes**
- The generated table is 546 cases and the whole termcmd package runs in under 3 s green. Under the expiry-branch revert it takes about 3 minutes because each red deadline-first case waits its 1 s bound, which is acceptable for a failure mode.
- Remaining real-timer callers in `run_test.go` (lines 358, 389, 404, 229, 343, 377, 416, 436, 451, 980) either deliver one read or split only inside a rename session with an empty pending buffer, so none can race the 35 ms wall clock. BR-1's class is empty.
- `TestBareEscapeIsForwardedAfterTheDeadline` still passes without the expiry branch. Its comment now states plainly that the arm is its evidence, which was BR-5's alternate remedy; the flush is pinned by its gated siblings.

**Architectural notes**
- ARCH-DRY pass: one constant, six call sites across four framers, no stray 35 or 50 ms literal.
- ARCH-PURE pass: the pump remains IO glue over pure `FindChord` / `IsChordPrefix` / `findSGRMousePress`; the new logic is two branches, both driven by injected seams.
- ARCH-PURPOSE pass: the shadow-sweep of `EscapeAmbiguity` consumers is complete; the inside-35 ms residual is declared in Done-when and belongs to #227.
- ARCH-MOCK pass: reader, timer and mux are injected; the real-timer test and the committed probe are the live conformance layer, and the probe runs under the existing `test-smoke` target.
- ARCH-CONSTRAINTS pass: the additive worst case is now stated correctly at the constant and in the plan; the real-timer test asserts a lower bound and a 1 s ceiling.
- ARCH-SECURE N/A: local pty stdin, no new parse surface, no secrets; the probe's socket is per-pid in the temp dir and removed on exit.
- ARCH-ORDER pass: the state table in the plan matches the code; `autoFire` plus the gated readers are a real interleaving seam, and both orderings were forced and verified.
- For #227: keep the tail at `run.go:561-571` as the single place the plain path touches the timer; the passthrough decision belongs between `FindChord` and that tail.

**Plan revision recommendations**
- Append a `## Revisions` entry dated 2026-09-12: "Close round 1 (FIX-THEN-SHIP): added `probes/escsmoke` as the live oracle for Done-when 1-2; case (b) now gates EOF behind the observed write; added the torn-SGR-mouse-prefix test; latency budget corrected to additive (~85 ms)."

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Leaky split test deleted; tab-actions rows use beforeDeadline(); every remaining pumpStdin caller is single-read or rename-mode with empty pending, so no plain prefix crosses reads on the real timer.
  - id: BR-2
    disposition: addressed
    note: |
      forwardedOnTheDeadline asserts timer.resets >= 1 and gates EOF behind the observed write.
  - id: BR-3
    disposition: addressed
    note: |
      run.go:354 declares EscapeTimer with the planned doc comment; RenameTimer greps to zero in Go source.
  - id: BR-4
    disposition: addressed
    note: |
      Reproduced independently: escsmoke 6/6 on a head build, 3/6 on a base-commit control, failing the same three steps. Operator in-zellij confirmation remains requested and is declared as not run in the close text.
  - id: BR-5
    disposition: addressed
    note: |
      Scratch revert of the expiry branch now fails the generated deadline-first case, the mouse test, ESC-then-j and the real-timer test; the bare-ESC test's comment states the arm is its evidence.
  - id: BR-6
    disposition: addressed
    note: |
      shortcut.go:390-393 and the plan's latency budget both state the additive ~85 ms worst case.
  - id: BR-7
    disposition: addressed
    note: |
      TestATornMousePrefixMeetsTheSameDeadline covers both sides of the deadline and goes red under either scratch revert.
findings:
  - id: new
    severity: Minor
    family: plan-records-post-approval-delta
    title: |
      Plan lacks a Revisions entry for the probe, the gated case (b) helper and the mouse test added after approval
    detail: |
      CLAUDE.md asks for an appended Revisions section when a plan artifact changes mid-stream; the Core concepts and Task lists predate probes/escsmoke and the round-1 test changes.
```
