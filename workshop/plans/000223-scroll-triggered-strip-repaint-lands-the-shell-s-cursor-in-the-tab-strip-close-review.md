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
