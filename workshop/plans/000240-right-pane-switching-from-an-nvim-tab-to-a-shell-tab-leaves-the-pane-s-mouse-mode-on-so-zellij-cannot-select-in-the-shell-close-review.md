# Boundary Review — pair#240 (whole-issue close)

| field | value |
|-------|-------|
| issue | 240 — right pane: switching from an nvim tab to a shell tab leaves the pane's mouse mode on, so zellij cannot select in the shell |
| repo | pair |
| issue file | workshop/issues/000240-right-pane-switching-from-an-nvim-tab-to-a-shell-tab-leaves-the-pane-s-mouse-mode-on-so-zellij-cannot-select-in-the-shell.md |
| boundary | whole-issue close |
| milestone | — |
| window | 15981175f839ef768e2d9dfdd6b30a47c000b25a..d2eb2695f9b08153558bd6d9d43b75edf2cc9833 |
| command | sdlc close --issue 240 |
| reviewer | claude |
| timestamp | 2026-09-12T19:39:00-07:00 |
| verdict | unknown |

## Review

You've hit your session limit · resets 8:50pm (America/Los_Angeles)

---

## Re-review — 2026-09-12T23:07:38-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 240 — right pane: switching from an nvim tab to a shell tab leaves the pane's mouse mode on, so zellij cannot select in the shell |
| repo | pair |
| issue file | workshop/issues/000240-right-pane-switching-from-an-nvim-tab-to-a-shell-tab-leaves-the-pane-s-mouse-mode-on-so-zellij-cannot-select-in-the-shell.md |
| boundary | whole-issue close |
| milestone | — |
| window | 15981175f839ef768e2d9dfdd6b30a47c000b25a..919cbcf0f3785def50a125d3bf962994d2f37311 |
| command | sdlc close --issue 240 |
| reviewer | claude |
| timestamp | 2026-09-12T23:07:38-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The fix is real, reachable, and pinned: reverting the reconcile prefix in a scratch copy turns both mux tests red (switch-to-shell and closed-tab), the 8×8 `mouseReconcile` product passes with `ptychild.Screen` as oracle, and running `probes/mousemodesmoke` live against a fresh build of head reproduces the Done-when bytes exactly (`?1002l ?1006l` on the way to the shell, the DECSET twice on the way back, DECRST again on Alt+Right). The prior open finding BR-1 is addressed: both stale hostScan comments now point at the reconcile prefix. Two things keep this from a clean SHIP, neither blocking: the Done-when's live operator check is not evidenced anywhere in the Log, and a wider-regex run of the same probe shows the class this issue belongs to ("the pane's DEC private modes follow the active child") has measured non-mouse siblings — `?1004h` focus events, `?1h` DECCKM and `?2004h` bracketed paste all stay on the pane after the switch to the shell — while the Spec's own sentence "the pane's modes must equal the active child's" over-claims what shipped. Record the enumeration and a tracked follow-up; the mouse mechanism itself is the right shape to extend.

**1. Strengths**

- `cmd/internal/termcmd/run.go:1007` reads `held` from `hostScan` before the reset and feeds the composed prefix back at `run.go:1061` — the pane's state has one source of truth and the next takeover's baseline is correct by construction (PQ-1 delivered as designed, ARCH-DRY/ARCH-ORDER).
- `cmd/internal/ptychild/screen.go:134-174` models tracking as one tagged slot plus an SGR bit, and the `?1002h` then `?1000l` → off case is pinned in `screen_test.go` — this is exactly what xterm and zellij do, and a set model would have emitted wrong bytes.
- `mousemode_test.go:12-49` uses the terminal model as oracle over the full held×want product rather than a hand-written expected-bytes table, and additionally asserts the one-write-per-axis budget.
- `hostty/control.go:100` keeps the shared primitive policy-free; the `EnableMouseClicks` equality pin in `privatemodes_test.go:21-24` stops the const and the formatter drifting.
- `tests/paint-gate-consumers-test.sh:23-26` extends the consumer guard with a stated reason rather than silently loosening the regex.

**2. Critical findings** — none.

**3. Important findings**

- **Done-when "Live: the operator selects text…" has no evidence in the Log.** The Plan row says "the in-pane check is the operator's step after install", but the Done-when lists it as a close criterion and the Log records only probe runs. Fix: run the live check (or have the operator confirm) and record the outcome in `## Log` / `--verified` before the close finalizes; if it is deliberately deferred past close, say so in a Revisions entry so the Done-when stops claiming it.
- **The class is wider than mouse, and the Spec claims the class.** Measured with the in-tree probe's regex widened to every `?…h/l`: on Alt+t to the shell tab the pane receives `?1002l ?1006l` and nothing else, so nvim's `?1049h ?1h ?2004h ?1004h` remain in zellij's view of the pane. `?1004h` is the visible one: zellij forwards focus in/out to a pane that asked, so clicking between panes delivers `ESC[I`/`ESC[O` to the shell tab. The fix as shipped is the right mechanism for the class (Screen models a mode → takeover reconciles it), so the deliverable here is the enumeration and a tracked follow-up, not code in this round: list which of {1, 1004, 2004, 25, 1049 (deliberately excluded per `repaint.go:55`)} the takeover should reconcile and why the rest are excluded. Spec bullet 1's "the pane's modes must equal the active child's" should be narrowed to mouse modes with the follow-up named (ARCH-PURPOSE).

**4. Minor findings**

- `run.go:1471-1486` — `removeTab` with a rename open skips the takeover, so a dead child's mouse modes stay on the pane until the next switch; `finishRename` (`run.go:1373-1376`) only repaints the strip. Rare (input goes to the rename editor, so the child must exit on its own), same family as above — dispose via the enumeration, not a local patch.
- `screen_test.go:952-961` calls `feedWhole(tt.data)` three times per case; compute once.
- Live probe tail shows a same-tab takeover after `:q!` replaying `?1002h…?1002l` from the ring (net state correct, transient flicker) — pre-existing replay behaviour, noted only so it isn't mistaken for a reconcile bug.

**5. Test coverage notes**

- Revert check performed: with `composed := hostty.RepaintFor(child, replay)` restored in a scratch copy of head, `TestSwitchingTabsReconciles…` and `TestClosingATabReleases…` fail; `TestMouseReconcile…` stays green (pure, as expected).
- `go test` on termcmd, ptychild, hostty green unsandboxed; sandboxed runs fail only on pty open (environment, not code). Paint-gate consumer script and `TestNoProbeExitsPastItsOwnCleanup` (which enumerates `probes/*` and so covers the new probe) green.
- Covered: the three transitions, closed-tab release, RIS clearing, grouped DECSET, one-slot semantics. Not covered: the rename-open removal path (Minor above); a `?1000l` written by the *foreground* child after a takeover-asserted `?1002h` (works by inspection since hostScan is fed live bytes, but no test pins that hostScan's live feed is what the next takeover diffs against).

**6. Architectural notes**

- ARCH-DRY pass — one formatter (`PrivateModes`), one model (`Screen`), no parallel tables.
- ARCH-PURE pass — `mouseReconcile`/`splitMouseModes`/`childMouseModes` are pure; the IO seam is the existing `applyTakeover`.
- ARCH-PURPOSE flag — see Important 2; the instance is fixed, the class is measured and unenumerated.
- ARCH-MOCK pass — the terminal model is the fake and the live probe is the conformance check; keep running the probe when the zellij pin moves.
- ARCH-CONSTRAINTS pass — at most two extra CSI writes per takeover, keystroke path unaffected.
- ARCH-SECURE pass — N/A: no untrusted input beyond child bytes the scanner already frames; no secrets.
- ARCH-ORDER pass with one gap — held/want read on the writer goroutine at compose time; queued chunks converge because live bytes are written verbatim after the takeover. The rename-open skip is the one event where the pane's modes and the active child's diverge with no later reconciliation until an unrelated switch.
- ARCH-FUNERAL pass — nothing durable created; the probe binary is git-ignored.
- Lessons entry suggested (I have no write access here): "when a fix reconciles one member of a mode family at a boundary, run the probe with the family-wide regex before closing, and record the siblings."

**7. Plan revision recommendations**

- Add a `## Revisions` entry narrowing Spec bullet 1 from "the pane's modes must equal the active child's" to mouse modes, and recording the measured sibling list (`?1`, `?1004`, `?2004`, `?25`, `?1049`-excluded) with the follow-up issue reference.
- If the live operator check is deferred past close, move it out of Done-when into the follow-up in the same entry.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      run.go:796-803 and run.go:1053-1058 now describe the takeover prefix as the one suspension and point at applyTakeover.
findings:
  - id: new
    severity: Important
    family: done-when-evidenced
    title: |
      Done-when's live operator selection check is not recorded anywhere in the Log
    detail: |
      The Plan row defers it to "the operator's step after install" while Done-when lists it as a close criterion; record the live outcome in the Log or --verified, or move the bullet to the follow-up via a Revisions entry.
  - id: new
    severity: Important
    family: pane-modes-follow-active-child
    title: |
      Spec claims the pane's modes equal the active child's, but only mouse modes are reconciled; measured siblings 1004, 1, 2004 stay on the pane
    detail: |
      Widening the probe's regex to every DEC private mode shows the switch to the shell writes only ?1002l ?1006l, leaving nvim's ?1004h (zellij then forwards focus in/out to the shell), ?1h and ?2004h. Write the enumeration (which modes reconcile, which are excluded and why, 1049 per repaint.go:55) and track the sweep as a follow-up; narrow Spec bullet 1 to mouse. ARCH-PURPOSE.
  - id: new
    severity: Minor
    family: pane-modes-follow-active-child
    title: |
      removeTab with a rename open skips the takeover, so a dead child's mouse modes outlive it until the next switch
    detail: |
      run.go:1471-1486 takes the paintStripInline branch and finishRename (run.go:1373-1376) only repaints the strip. This is the 2nd finding in family pane-modes-follow-active-child; do not patch the site, dispose it through the events axis of the same enumeration.
```

---

## Re-review — 2026-09-12T23:12:51-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 240 — right pane: switching from an nvim tab to a shell tab leaves the pane's mouse mode on, so zellij cannot select in the shell |
| repo | pair |
| issue file | workshop/issues/000240-right-pane-switching-from-an-nvim-tab-to-a-shell-tab-leaves-the-pane-s-mouse-mode-on-so-zellij-cannot-select-in-the-shell.md |
| boundary | whole-issue close |
| milestone | — |
| window | 15981175f839ef768e2d9dfdd6b30a47c000b25a..9159456528b02c42266b447e8f407c26a9cf0585 |
| command | sdlc close --issue 240 |
| reviewer | claude |
| timestamp | 2026-09-12T23:12:51-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The code range is byte-identical to the one the last round reviewed (no non-workshop changes between the round-3 head and this head), and I re-verified it independently: the targeted hostty/termcmd/ptychild tests and the paint-gate consumer guard are green at head, and reverting the reconcile prefix in a scratch worktree turns both mux tests red (`TestSwitchingTabsReconciles…` at mousemode_test.go:91 and `TestClosingATabReleases…` at mousemode_test.go:142), so the fix is reachable and pinned. All three open findings are disposed by artifacts rather than by claims: the Log's close entry quotes the operator's live confirmation (BR-2), the Spec header is narrowed to mouse and the reconcile/exclude enumeration is written into #241's Spec with the reason for the excluded set (BR-3), and #241's "paths" axis names the `removeTab`-with-rename-open branch explicitly instead of patching the site (BR-4). What remains is two Minor process gaps, neither blocking.

**1. Strengths**

- `run.go:1004-1008` reads `held` from `hostScan` before the reset and `run.go:1061` feeds the composed prefix back, so the pane's mode state has one source of truth and the next takeover's baseline is correct by construction (PQ-1 as designed; ARCH-DRY, ARCH-ORDER).
- `screen.go:134-174` models tracking as one tagged slot plus an SGR bit; `screen_test.go` pins the `?1002h` then `?1000l` → off case, which a set model would get wrong against xterm and zellij (ARCH-ORDER, PQ-2).
- `mousemode_test.go:17-49` uses the terminal model as the oracle over the full 8×8 held×want product and asserts the one-write-per-axis budget, rather than a hand-written expected-bytes table (ARCH-PURE).
- `TestRISClearsEveryModeTheChildCanSet` was extended (`screen_test.go:883-885`) so the regex derives the new non-boolean `tracking` field too; the reflect-based RIS check still covers every mode arm rather than passing vacuously.
- `hostty/control.go:100-114` keeps the shared formatter policy-free, and `privatemodes_test.go:21-24` pins `EnableMouseClicks` to it so the constant and formatter cannot drift.

**2. Critical findings** — none.

**3. Important findings** — none.

**4. Minor findings**

- The Spec narrowing for BR-3 was applied in place (Spec paragraph rewritten) with the delta recorded in the Log's close entry, not under `## Revisions`. AGENTS.md §1 says revise plan artifacts by appending a Revisions entry, not overwriting; the existing `## Revisions` section already has the plan-quality entry, so add a second one (timestamp, reason "close review BR-3/BR-4", delta "Spec bullet 1 narrowed to mouse; siblings and paths → #241").
- `workshop/lessons.md` did not change in the window. AGENTS.md §4 asks for a rule after each review; the last round handed one over ("when a fix reconciles one member of a mode family at a boundary, run the probe with the family-wide regex before closing and record the siblings"). Land it as a follow-up commit after close.
- `screen_test.go:952-961` calls `feedWhole(tt.data)` three times per case; compute once. Style only.

**5. Test coverage notes**

- Green at head: `go test` for hostty, termcmd, ptychild (targeted Mouse/PrivateModes/RIS/Switching/Closing/NoProbeExits), `tests/paint-gate-consumers-test.sh`.
- Revert check performed this round: with `composed := hostty.RepaintFor(child, replay)` restored in a scratch worktree of head, both mux tests fail and the pure `mouseReconcile` product stays green, as expected.
- Covered: three transitions, closed-tab release through `removeTab` → `applyTakeover`, RIS clearing, grouped DECSET, one-slot semantics, `clearTab`/nil-child neutral state via `childMouseModes`. Not covered here and now owned by #241: the rename-open removal path; the non-mouse modes.

**6. Architectural notes**

- ARCH-DRY pass: one formatter, one model, no parallel tables.
- ARCH-PURE pass: `mouseReconcile`, `splitMouseModes`, `childMouseModes` are pure; the IO seam is the existing `applyTakeover`.
- ARCH-PURPOSE pass for this issue as now scoped: the Spec no longer over-claims, the class is enumerated (modes axis and paths axis) in #241, and the shipped mechanism (Screen models a mode → takeover reconciles it) is the shape #241 should generalise rather than replace. Watch that #241 generalises `PrivateModes`/`mouseReconcile` into one delta formatter over a mode set instead of adding a sibling function per mode family.
- ARCH-MOCK pass: `ptychild.Screen` is the stateful terminal model, `probes/mousemodesmoke` is the live conformance check against real `pair term` + nvim; re-run it when the zellij or nvim pin moves.
- ARCH-CONSTRAINTS pass: at most two extra CSI writes per takeover; keystroke path untouched.
- ARCH-SECURE N/A: only child bytes the scanner already frames; the tracking lookup at `screen.go:655` is reached only for the three enumerated cases, so no zero-value fallthrough.
- ARCH-ORDER pass: `held` and `want` are read on the writer goroutine at compose time and later live chunks are written verbatim and fed to `hostScan`, so the pane converges; the one divergent event (rename-open close) is tracked in #241's paths axis.
- ARCH-FUNERAL pass: nothing durable created; the probe binary is git-ignored.

**7. Plan revision recommendations**

- Append a `## Revisions` entry to the #240 issue recording the close-review narrowing (see Minor 1). No other drift between plan and code.

```findings
dispose:
  - id: BR-2
    disposition: addressed
    note: |
      Log "2026-09-12 (close)" records the operator's live confirmation of selection in the right-pane shell on this build.
  - id: BR-3
    disposition: addressed
    note: |
      Spec header narrowed to MOUSE modes; the reconcile/exclude enumeration (1004, 2004, 1 reconcile; 1049/1047/47/1048 excluded per repaint.go:55) and the sweep are tracked in #241.
  - id: BR-4
    disposition: addressed
    note: |
      Disposed through #241's paths axis, which names removeTab-with-rename-open explicitly; the site was not patched, as asked.
findings:
  - id: new
    severity: Minor
    family: revisions-appended-not-overwritten
    title: |
      Close-review Spec narrowing was applied in place with the delta in the Log, not as a Revisions entry
    detail: |
      AGENTS.md section 1 asks for an appended Revisions entry (timestamp, reason, delta) when a plan artifact changes mid-stream; the Spec paragraph was rewritten and only the Log's close entry records why. Add a second Revisions entry naming BR-3/BR-4 and the move to issue 241.
  - id: new
    severity: Minor
    family: lessons-recorded-after-review
    title: |
      workshop/lessons.md carries no rule from this review cycle
    detail: |
      AGENTS.md section 4 asks for a lessons rule after each code review; the prior round proposed one (run the probe with the family-wide DEC-mode regex before closing a one-family fix and record the siblings). Land it as a post-close follow-up commit.
```
