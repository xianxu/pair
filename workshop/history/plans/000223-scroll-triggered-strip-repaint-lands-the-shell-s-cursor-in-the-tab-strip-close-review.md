# Boundary Review — pair#223 (whole-issue close)

| field | value |
|-------|-------|
| issue | 223 — scroll-triggered strip repaint lands the shell's cursor in the tab strip |
| repo | pair |
| issue file | workshop/issues/000223-scroll-triggered-strip-repaint-lands-the-shell-s-cursor-in-the-tab-strip.md |
| boundary | whole-issue close |
| milestone | — |
| window | 81a8c88eac2b3c531a1f6636676f453c15f65430..ef8cdb34d8226a111eadf1062f3cb8493b118e32 |
| command | sdlc close --issue 223 |
| reviewer | claude |
| timestamp | 2026-09-10T10:00:51-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The core of #223 holds up. The mechanism was measured directly rather than guessed. The fix goes where the bug lives: zellij ≥ 0.45.0, zellij#5357. The pair-side change is small and correct: `zellij/config.kdl` now states `pane_frame_style "full"`, and `zellij --config zellij/config.kdl setup --check` on the installed 0.45.1 reports "Well defined". The claim that `probes/zellijwrapmargin` runs in `make test-smoke` holds: the wholesale `probes/*/` loop picks it up (`Makefile.local:60-67`), and `go vet` is clean. Most of this window is #209's code. #209 shares the branch and was reviewed at its own close (FIX-THEN-SHIP, archived). I confirmed that no `cmd/` code changed after #209's close (`3435486c`) and that both merge commits are tree-identical. So only #223's work is under review here. Three cheap problems stand between this and SHIP:
- **Probe verdict:** the new probe prints the "fixed" verdict when its reading can't be parsed.
- **README:** it doesn't mention the new zellij version floor.
- **Repro tag:** the atlas cites a tag that exists only in this checkout.

**1. Strengths**
- **Measured, not guessed.** The probe asks zellij for its own cursor position (DSR) instead of reading it off the rendered screen (`probes/zellijwrapmargin/probe.sh:8-12`). The A/B against the pre-#209 binary and the top-edge table both settled their questions by experiment.
- **The top-edge alternative was disproved, not argued away.** The `childregion` and `nvim` modes show DECSTBM parameters are absolute even under origin mode, which is a strong reason to keep the strip at the bottom.
- **The shared harness is reused.** `zellijprobe.Start/Scrub/WriteLayout/WaitUntilListed` are used as intended. The session is named and deleted by name, so the probe can't touch an operator session, and a missing session is reported as PROBE-INCONCLUSIVE.
- **The config comment explains itself.** `zellij/config.kdl:41-53` says why the key exists, what was measured (29×49 vs 28×48), and that 0.44.x ignores it.
- **The unexplained glitch is recorded as unexplained,** with the triggers that were ruled out, rather than attributed to a guessed cause.

**2. Critical findings**
None.

**3. Important findings**
- **I-1: the probe reports "fixed" when it measured nothing** (`probes/zellijwrapmargin/main.go:190-213`, ARCH-SECURE).
  - `wrapEscaped` ignores `Sscanf` failures, and every value it can't read ends up in the `default:` branch, which prints "WRAP SCROLLS THE REGION — the #223 mechanism is not present".
  - I copied the function into a scratch program and fed it bad readings. An empty cursor reply (`row= col=`), a failed `tput` (`pane rows=`), a missing wrap line, and a wrap row *above* the margin all produce the "fixed" verdict.
  - This probe is the issue's only regression check, and it exits 0 on every verdict, so the printed verdict is its entire signal.
  - Sibling probes explicitly refuse this: `zellijscrollregion/main.go:130-147` and `cursorsaveslots/main.go:159-161` return `(int, bool)` and route unexpected readings to PROBE-INCONCLUSIVE (exit 2). This probe's own header cites the same rule (pair#208).
  - Fix: return `(wrapRow, bottom, ok)`. Then `!ok` → INCONCLUSIVE, exit 2; `wrapRow == bottom` → SCROLLS; `wrapRow > bottom` → ESCAPES; anything else → UNEXPECTED, exit 2. Add a `main_test.go` table test: the function is pure, so it needs no zellij (ARCH-PURE).
- **I-2: README doesn't state the zellij ≥ 0.45.0 requirement** (`README.md:255`, docs gate).
  - The atlas (`architecture.md:582`) and the config comment both say `pair term`'s strip needs zellij ≥ 0.45.0. The README Install section says only that brew installs zellij "if [it isn't] already present".
  - A zellij already on the machine, including one installed outside brew or pinned, stays on 0.44.x. Nothing tells the user, because no doctor check enforces the floor (the operator declined one) and the probe exits 0.
  - Fix: one sentence in Install, e.g. "`pair term`'s tab strip needs zellij ≥ 0.45.0." This documents the floor; it doesn't enforce it, so it doesn't reopen the doctor decision.
- **I-3: the atlas cites a repro tag that only exists locally** (`atlas/architecture.md:590-591`).
  - `repro/223-bottom-strip-zellij-wrap` is an annotated tag in this checkout. `git ls-remote --tags origin` lists 48 tags and none match `repro`. `push.followTags` is unset and no sdlc verb pushes tags.
  - The commit itself (`0baacfa7`) is on `origin/main`, so the fix is either to push the tag (outward-facing, so the operator should confirm) or cite the SHA next to the tag name.

**4. Minor findings**
- **M-1: older EdgeTop comments are now incomplete** (ARCH-PURPOSE shadow sweep).
  - `hostty/reserve.go:34-36` and its error message at `:67` (`"a top strip needs DECOM arbitration"`), `reserve_test.go:43,46`, and `atlas/architecture.md:569-572` all say origin-mode (DECOM) arbitration is what a top strip needs. #223 measured that it is not enough: the child's DECSTBM would also need rewriting in flight.
  - The new prose at `architecture.md:597-598` and `main.go:47-48` says reserve.go "records" this asymmetry; it records only the weaker premise.
  - Bring reserve.go's comment, error message, and test into line with the measurement.
- **M-2: nothing pins `pane_frame_style "full"`.** Deleting it silently brings back the "first line vanished" bug on 0.45. `TestEveryTerminalPaneRungIsBorderless` already pins layout frame attributes, so a sibling assertion on `zellij/config.kdl` would be cheap.
- **M-3: the upgrade trips a "DELETE THIS" marker.** `couchtty/mouse.go:92-95` says to delete the ctrl+wheel filter once zellij carries `mouse_scroll_resize`. `zellij setup --dump-config` on 0.45.1 shows `// mouse_scroll_resize false`. Deleting the filter depends on 0.45 becoming a real floor, so file a follow-up rather than acting now.
- **M-4: stale probe comments.**
  - `main.go:99` lists `top:<wrap|plain|decom>` but the script also takes scroll, decom2, childregion and nvim.
  - `layout.kdl:4` says "see writeLayout there", but the function is now `zellijprobe.WriteLayout`. The comment was copied from `zellijscrollregion`, which has the same stale line.
  - The recorded 0.45.1 reading "(region 1..23)" predates the full-frame config, so the probe now reports region 1..21.
- **M-5: file modes differ.** `probe.sh` is mode 100644 and `probe_top.sh` is 100755. Both are run via `bash <path>`, so this is harmless, just inconsistent.

**5. Test coverage notes**
- The only #223 test is the probe itself, which reports and never fails. That is the accepted design, and consistent with `zellijscrollregion`. It makes I-1 more important, because the printed verdict is the only oracle.
- The config tests still pass: `runtimebundle`, `runtimebundlegen`, `keyhelp` and `artifactpath` all pass.
- `hostty` failed only on `pty.Open: operation not permitted` from the sandbox. It is unchanged by #223.
- I did not run the live probe (it starts a zellij server). I relied on the issue's recorded readings.

**6. Architectural notes for upcoming work**
- **ARCH-DRY: pass.** The harness is reused. The duplicated shell `q()` helper across the two probe scripts is trivial.
- **ARCH-PURE: pass**, with I-1: `wrapEscaped` is pure but untested and fails open.
- **ARCH-PURPOSE: flag (Minor, M-1).** The Done-when rows are met on zellij 0.45 and were revised in the Log. The new top-edge fact wasn't propagated into reserve.go, reserve_test.go or the older atlas paragraph.
- **ARCH-MOCK: pass.** pair doesn't model the cursor, so there's nothing to fake. The probe is the live conformance check, run from `test-smoke`.
- **ARCH-CONSTRAINTS: pass / N/A.** The probe's waits are bounded (15 s + 15 s). The "~20s" estimate in the `Makefile.local` comment for `test-smoke` is probably stale now that another zellij-session probe runs there.
- **ARCH-SECURE: flag (I-1).** The pane writes readings to a file across a process boundary, and when that input is bad the probe prints a "fixed" verdict instead of failing visibly.
- **ARCH-ORDER: pass.** The probe holds no state between events beyond a sequential script. The DONE sentinel is the only ordering it needs, and it guards against partial reads of the readings file.
- **Going forward:** several atlas lines are scoped to "zellij 0.44.3": `architecture.md:378` (floating-pane drag), `:477` (no repaint action — I checked 0.45.1 and it still has none), and `couch.md:367` (ctrl+wheel). If 0.45 becomes the floor, each should be re-measured rather than carried forward.

**7. Plan revision recommendations**
- None needed for #223's Plan: the `## Log` "done-when rows, against what happened" entry already reconciles it. If I-1 is fixed, add a `## Revisions` line saying an unparseable reading now reports PROBE-INCONCLUSIVE rather than a verdict. The Spec calls this probe the regression check, and that line is part of its contract.

```findings
findings:
  - id: new
    severity: Important
    family: probe-unparsed-reading-is-a-verdict
    title: |
      zellijwrapmargin reports WRAP SCROLLS (fixed) when its DSR/rows reading fails to parse
    detail: |
      wrapEscaped (main.go:202-213) ignores Sscanf failures and the switch default (main.go:190-197) prints the fixed verdict; scratch-verified for an empty CPR reply, a failed tput, a missing wrap line and a wrap row above the margin. Return (wrapRow, bottom, ok), route !ok or unexpected rows to PROBE-INCONCLUSIVE exit 2 as zellijscrollregion does, and add a table test.
  - id: new
    severity: Important
    family: readme-install-surface-gate
    title: |
      README Install omits the new zellij 0.45.0 floor that pair term's strip requires
    detail: |
      atlas/architecture.md:582 and zellij/config.kdl:51 state the requirement, but README.md:255 only says brew installs zellij if absent; an existing 0.44.x zellij is kept and nothing warns (no doctor floor, and the probe exits 0).
  - id: new
    severity: Important
    family: durable-doc-cites-unpublished-ref
    title: |
      atlas cites repro/223-bottom-strip-zellij-wrap, a tag that exists only in the local checkout
    detail: |
      git ls-remote --tags origin lists 48 tags and none match repro; push.followTags is unset. Push the tag (operator-confirmed) or cite commit 0baacfa7, which is on origin/main, at atlas/architecture.md:590-591.
  - id: new
    severity: Minor
    family: superseded-premise-sweep
    title: |
      reserve.go, reserve_test.go and atlas:569-572 still say DECOM arbitration is what a top strip needs
    detail: |
      #223 measured that DECSTBM params are absolute even under DECOM, so a top strip also needs the child's DECSTBM rewritten in flight; the new atlas text says reserve.go records this, but reserve.go:34-36 and its error message at :67 carry only the older premise.
  - id: new
    severity: Minor
    family: config-invariant-unpinned
    title: |
      No test pins pane_frame_style full in zellij/config.kdl
    detail: |
      Removing it silently brings back the first-line-vanished bug on zellij 0.45; a sibling of TestEveryTerminalPaneRungIsBorderless would pin it cheaply.
  - id: new
    severity: Minor
    family: version-scoped-workaround-tripwire
    title: |
      The zellij 0.45 upgrade trips the DELETE THIS marker on couchtty stripWheelResizeModifier
    detail: |
      zellij 0.45.1 dump-config carries mouse_scroll_resize (mouse.go:92-95 names it as the deletion trigger); deleting the filter depends on 0.45 becoming a real floor, so file a follow-up.
  - id: new
    severity: Minor
    family: stale-comment
    title: |
      Stale probe comments - PAIR_PROBE_EDGE mode list, writeLayout reference, 0.45.1 region reading
    detail: |
      main.go:99 lists only wrap, plain and decom modes; layout.kdl:4 names writeLayout, now zellijprobe.WriteLayout; the recorded 0.45.1 region 1..23 predates the full-frame config. Also probe.sh is mode 100644 while probe_top.sh is 100755.
```

---

## Re-review — 2026-09-10T10:19:08-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 223 — scroll-triggered strip repaint lands the shell's cursor in the tab strip |
| repo | pair |
| issue file | workshop/issues/000223-scroll-triggered-strip-repaint-lands-the-shell-s-cursor-in-the-tab-strip.md |
| boundary | whole-issue close |
| milestone | — |
| window | 81a8c88eac2b3c531a1f6636676f453c15f65430..b4e23f15d2def09b5ce004af3dc190681454f711 |
| command | sdlc close --issue 223 |
| reviewer | claude |
| timestamp | 2026-09-10T10:19:08-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

All three Important findings from round 1 are fixed, and I checked each one against the code rather than the commit message. BR-1's new table test fails against the old parser (6 of 8 subtests) and against a variant with no default branch (2 subtests). BR-5's new check fails when the key is removed, commented out, or missing from only the runtime-bundle copy. BR-3's replacement commit `0baacfa7` is on `origin/main`. Everything left is Minor and cheap:
- **BR-7:** its four named sites are fixed, but not the copied sibling the review also named.
- **Unchecked claims:** the fix round wrote four claims that git, the Homebrew source, or the probe itself contradicts or doesn't support. One is in the README install section.
- **Duplicated fact:** the atlas now states one measurement twice.

None of these blocks the gate. I'd fix them in the close so a false install statement doesn't ship.

**1. Strengths**
- **The parser only accepts rows the setup can produce.** `verdict()` accepts `wrapRow == rows - 1` or `wrapRow == rows`, and treats everything else as inconclusive, including a row past the screen (`probes/zellijwrapmargin/main.go:228-252`). This is stricter than the `>` the round-1 fix sketch suggested.
- **The test covers failure inputs, not just the happy path** (`main_test.go:8-29`). `make test`'s `go test ./...` reaches it.
- **The config check can't be fooled by comments** (`paneframestyle_test.go:27-38`). It strips `//` comments before matching, so documentation that names the key can't satisfy it. It checks both the source config and the runtime-bundle copy.
- **#226 names its blocker instead of deleting the filter early.** Deleting `stripWheelResizeModifier` waits until the 0.45 floor is enforced.
- **The lessons are written as rules** a later round can apply: a measurement has three outcomes, and durable docs cite refs that exist on origin.

**2. Critical findings**
None.

**3. Important findings**
None.

**4. Minor findings**
- **N-1: four checkable claims don't match their sources** (new family `prose-claim-unchecked-against-source`). This is adjacent to BR-2 and BR-3 but a different rule: those were a missing statement and an unpublished ref; these are statements that exist but weren't checked.
  - `README.md:257-258` says "`brew install` skips a present dependency". That's false for pair's formula. It builds from source with no bottle, so there is no bottle minimum version. `dependency.rb:82-84` then treats any zellij older than the latest as unsatisfied, and `formula_installer.rb:951` upgrades it. A pinned zellij makes the install refuse loudly. The advice to check the version and `brew upgrade zellij` is still right. The real gaps are:
    - a zellij installed outside brew that comes first on `PATH`;
    - a pair install that `brew upgrade` finds already current, so the installer never runs.
  - `atlas/architecture.md:594` says `0baacfa7` "is the last one before the frame-style change". `git rev-parse 5fdaf32e^` is `4e62e04a`, and `603fc8f0` and `4e62e04a` both sit in between. Either drop the clause or cite `4e62e04a`.
  - `hostty/reserve.go:40` says nvim overwrote the strip "on its first half-page", and `atlas:573` and `:599` say "at once". The nvim mode sends six keystroke groups and then takes one `dump-screen` (`main.go:159-189`). The issue Log records only that end state (row 1 showing line `94`).
  - `paneframestyle_test.go:18-19` says it reads "the same pair of files" as `TestEveryTerminalPaneRungIsBorderless`. That test reads `zellij/layouts/main-3.kdl`, not `config.kdl`. It's the same pair of *locations*.
- **N-2: the atlas states the top-edge measurement twice, 25 lines apart** (ARCH-DRY). BR-4's sweep added it at `architecture.md:570-574`, and `:596-601` (from `ef8cdb34`) says nearly the same thing, including "real nvim overwrote it at once". Fixing that phrase under N-1 already needs two edits. Keep one copy and point to it.
- **BR-7 remainder:** `probes/zellijscrollregion/layout.kdl:3-4` still says "see writeLayout there". Round 1 named this file as where the stale line was copied from.

**5. Test coverage notes**
- **BR-1** is proved by the scratch mutation runs above. The glue in `run()` that turns inconclusive into exit 2 isn't unit-tested, but it's a three-arm switch, so that's acceptable.
- **BR-5** is proved by the three mutations above, all of which failed.
- **Other packages:** `go vet ./probes/...` is clean. `go test` passes for `./probes/...` and `./cmd/internal/runtimebundle/...`. `zellij --config zellij/config.kdl setup --check` on 0.45.1 reports "Well defined". `hostty` fails only `TestOSHostConformsToTheFakeOnSizeAndRawMode`, on `pty.Open: operation not permitted` from the sandbox; that test was not changed in this window.
- **Not run:** the live probe needs a zellij server, so I relied on the readings recorded in the issue.

**6. Architectural notes**
- **ARCH-DRY: flag, Minor (N-2).** The duplicated shell `q()` helper across the two probe scripts is still trivial.
- **ARCH-PURE: pass.** `verdict()` is pure and table-tested; `run()` is thin IO glue.
- **ARCH-PURPOSE: flag, Minor.** The BR-7 fix covered only the sites that were named; the copied sibling round 1 pointed at is still stale. The issue's purpose is delivered: the zellij 0.45 upgrade, the regression probe, and the frame style. I checked the other probes for the BR-1 pattern: `zellijscrollregion`, `cursorsaveslots`, `termctrlc`, `termrows` and `zellijrepaint` all refuse to report a verdict from nothing.
- **ARCH-MOCK: pass.** The probe is the live check of the zellij-emulator seam, and pair doesn't model the cursor.
- **ARCH-CONSTRAINTS: pass.** The probe's waits are bounded. The "~20s" estimate for `test-smoke` in `Makefile.local` is probably stale.
- **ARCH-SECURE: pass.** The readings file crosses a process boundary and is now parsed into typed values with ok flags. A bad reading fails visibly with exit 2.
- **ARCH-ORDER: pass.** `DONE` is written last, which guards against partial reads, and `verdict()` only sees a complete file. The nvim mode depends on sleeps, but it's manual-only and outside `test-smoke`.
- **Going forward:** the README now declares 0.45.0 as the floor, but many constants were measured on zellij 0.44.3. Examples: `layoutcmd/resizeplan.go:6`, `workbenchshortcut/shortcut.go:537`, `termcmd/run.go:599`, `ptychild/child.go:347`. Re-measure them when #226 enforces the floor.

**7. Plan revision recommendations**
- Add a `## Revisions` entry. `probes/zellijwrapmargin` now has a third outcome, PROBE-INCONCLUSIVE with exit 2, and `test-smoke` fails on it. So the Log's "reports rather than fails" now holds only for the two verdicts. Round 1 recommended this revision too, and it hasn't been made.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Mutation-checked in a scratch copy: the old parser fails 6 of 8 subtests, a variant with no default branch fails 2; make test's go test ./... reaches the test.
  - id: BR-2
    disposition: addressed
    note: |
      Floor stated at README.md:216 and :257; the Homebrew reason it gives is wrong, raised as the new prose-claim finding.
  - id: BR-3
    disposition: addressed
    note: |
      0baacfa7 is reachable from origin/main (the tag is still local-only, correctly not cited); the atlas's description of the commit is wrong, raised in the new finding.
  - id: BR-4
    disposition: addressed
    note: |
      reserve.go comment, error message, test name and atlas:570-574 now give the DECSTBM-absolute reason; no stale DECOM-arbitration premise left outside workshop/.
  - id: BR-5
    disposition: addressed
    note: |
      Mutation-checked: key removed, commented out, or missing from only the runtime-bundle copy - all three fail TestConfigStatesFullPaneFrames.
  - id: BR-6
    disposition: addressed
    note: |
      Filed as 226 with the blocker (0.45 not enforced) stated; grep finds no other DELETE THIS tripwire.
  - id: BR-7
    disposition: not-addressed
    note: |
      Four named sites fixed; the copied sibling round 1 named, probes/zellijscrollregion/layout.kdl:3-4, still says see writeLayout there. Rule: when fixing a copied comment, grep the copied phrase across the tree in the same commit.
findings:
  - id: new
    severity: Minor
    family: prose-claim-unchecked-against-source
    title: |
      Fix-round prose states four checkable facts that git, the Homebrew source or the probe contradicts or does not support
    detail: |
      (1) README.md:257-258 says brew install skips a present dependency; for pair's bottle-less formula Homebrew treats an outdated dep as unsatisfied (dependency.rb:82-84) and upgrades it (formula_installer.rb:951) - the real gaps are a non-brew zellij earlier on PATH and a pair that brew upgrade finds current. (2) atlas/architecture.md:594 calls 0baacfa7 the last commit before the frame-style change; 5fdaf32e^ is 4e62e04a. (3) reserve.go:40 (first half-page) and atlas:573/:599 (at once): the nvim mode takes one dump-screen after six keystroke groups (main.go:159-189), and the Log records only that end state. (4) paneframestyle_test.go:18-19 says TestEveryTerminalPaneRungIsBorderless reads the same files; it reads layouts/main-3.kdl. Rule: run the check (git, tool source, the probe's actual reading) before committing a checkable claim.
  - id: new
    severity: Minor
    family: doc-fact-single-home
    title: |
      atlas states the top-edge DECSTBM-under-DECOM measurement twice, at architecture.md:570-574 and :596-601
    detail: |
      BR-4's sweep added it to the Edge asymmetry paragraph, and ef8cdb34's strip-stays-at-the-BOTTOM paragraph already said nearly the same, including real nvim overwrote it at once, so correcting that phrase now needs two edits. Keep one copy and point to it (ARCH-DRY).
```
