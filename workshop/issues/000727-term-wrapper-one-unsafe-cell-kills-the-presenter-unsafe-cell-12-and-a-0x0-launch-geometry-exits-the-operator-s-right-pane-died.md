---
id: 000727
status: working
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours: 1.98
collaboration-mode: collaboration
card_mirror: '6d2a099827fb7c68299012a4a3a410e71e710cc1' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T08:47:04-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# term wrapper: one unsafe cell kills the presenter ('unsafe cell 12'), and a 0x0 launch geometry exits: the operator's right pane died

## Problem

10-20 ~09:00, the operator's ops:0 right pane (nvim on the TL session file, launched as `v teams/devinfra/tl-status-xian-xu.md`
through pair's `term` wrapper, supervisor on the staged pair:0 build f3c597f8) crashed, and relaunching it failed:

- `term: terminal: unsafe cell 12` then `terminal: presenter unavailable` (screenshot: ops
  `teams/devinfra/assets/tl-status-xian-xu/2026-10-20-term-unsafe-cell.png`);
- `term: terminal: invalid geometry 0x0 (maximum 262144 cells)` (screenshot `…/2026-10-20-term-geometry-0x0.png`).

`unsafe cell` is `validateCell` (cmd/internal/terminal/frame.go:119): a cell whose content isn't plain text (a control
rune or invalid UTF-8), exceeds MaxClusterBytes, or has an invalid link. The session file itself has no control
characters (checked), so the offending cell comes from nvim's rendered screen (sign column, virtual text, an OSC 8
link, or a grapheme cluster). One bad cell ends the whole presenter, so the operator loses their pane; and a 0x0
geometry at launch (pane measured before it had a size) refuses outright instead of waiting for a real size.

## Spec

- An unsafe cell degrades that cell (render a replacement character, log once with the codepoints), never kills the
  presenter. The safety intent (no control sequences reach the parent terminal) is kept by the replacement.
- A 0x0 or transient geometry at launch waits briefly for a real size (or uses a minimal default) instead of exiting.
- Capture the offending cell's content in the diagnostic so the trigger is identifiable.

## Done when

- A frame with a control rune in one cell renders with a replacement and the presenter stays up (test).
- Launch with a 0x0 geometry that becomes 80x24 shortly after succeeds (test).
- The operator's `v teams/devinfra/tl-status-xian-xu.md` opens again (live).

- A mid-session resize with either dimension zero preserves the last good geometry, child, presenter and input (test).
- An unexpected wrapper error restores terminal ownership and leaves an interactive plain shell in the same pane; normal exit and host termination do not restart a shell (tests).
- Alt+Shift+T from a surviving Pair-aware pane recreates a missing right terminal after confirmed same-tab inventory; unknown inventory never creates a pane (tests and live).

## Plan

- [x] Execute [the single-pass plan](../plans/000727-term-wrapper-one-unsafe-cell-kills-the-presenter-unsafe-cell-12-and-a-0x0-launch-geometry-exits-the-operator-s-right-pane-died-plan.md): safe backend-cell projection and bounded diagnostics; geometry/recovery lifecycle; missing-terminal shortcut recovery.
- [ ] Run the recorded full suite, close review, coordinated operator acceptance, then land through SDLC.

## Log

### 2026-10-20
- 2026-10-20: closed — BR-1 decoder/read semantic reds fixed with typed pump outcomes and EOF/cancel/disconnect/overlap controls. BR-2 focus red fixed with no-focus. BR-3 native Zellij conformance passed 0.54s, including actual PTY focus and missing-no-focus negative control; native tab-id/near-current-pane conflict removed. Final complete termcmd/layoutcmd/zellijtest race packages passed 6.892s/7.490s/8.095s; terminal, hostty, ptychild, capture, qualify and artifactpath previously passed. Python scheduled-native execution gate 8 tests passed. Corrected code 97c32eaad2245af63532d100ac67e53283b268eb is pushed; its recorded suite is pending, no full pass claimed. Review overlaps suite per tl-worker; unchecked final checklist awaits suite and landing. TL owns project. Approved live check deferred to next natural relaunch, no pair:0 stage.; review verdict: SHIP


## Revisions

### 2026-10-20 — include pane survival and recovery (hub updates)

The operator reported that wrapper exit removed the Zellij pane, leaving no shell, and
Alt+Shift+T could not recreate it. Add the three Done-when rows above. Preserve the
last valid geometry during zero-sized resizes; wait up to five seconds for a nonzero
launch size, then fall back to a plain shell rather than kill the pane. Unexpected
wrapper failure restores/joins all wrapper ownership before replacing this process
with the configured interactive shell. Normal child exit and terminal termination
remain exits. Alt+Shift+T confirms same-tab inventory and creates one right terminal
only on positively observed absence; malformed/empty replies remain refusal.
The original cell's precise trigger is unknown: retain bounded codepoints/invalid-byte
hex from the first degradation per endpoint, not whole frames or raw link addresses.

### 2026-10-20 — investigation checkpoint

- Claimed/start-plan in pair:7. start-plan's refresh generated the two pending #699 CI
  workflows; inspected and reverted only those generated changes before retrying.
- Git comparison e712d8335f3e77a0694d6bd8869f5e3bf69b033f to
  f3c597f8dab1955b8b81c304e0cf2e99b935071b has no changes under terminal,
  termcmd, hostty, ptychild or layoutcmd. #722 changes Couch drain/status rows.
  Restart timing is not evidence that #722 introduced this renderer problem.
- runShellOnHost samples launch size once. inheritSize already ignores zero dimensions
  mid-session. Presenter/render validation is strict and a failed publication ends the
  wrapper, whose deferred closeAll closes the child. Both pane launch commands exec
  pair term, so returning from it leaves no shell. Screenshots confirm these messages,
  but contain no offending-cell payload.
- Shared SwitchRightTerminalTabContext explicitly returns nil when no terminal exists.
  New-terminal recovery must use confirmed inventory, including when stale pane IDs
  remain in sidecars, rather than interpret a missing registration as absence.
- A read-only internal subagent independently traced geometry, teardown and shortcut
  paths; no live process, native pane or pair:0 was touched.

### 2026-10-20 — approved rollout and full-flow checkpoint

Hub approved the full design (delivery 42487af6-bd3c-4a38-91f3-2ef1cdf804d0).
No staging: pair:0 retains #722 for its live drain check. Land on deterministic
tests/suite/review, then the operator's next relaunch on rebuilt main is live
acceptance, with diagnostic evidence to identify the still-unknown original cell.
This supersedes the plan's coordinated-staging step. Initial change-code inferred
quick from design length alone; pin full before code because the planned combined
production changes exceed the 100-line shell. Canonicalized the durable plan filename.

### 2026-10-20 — plan-quality round 1 refinements

PQ-1 requires named-function adversarial strategies instead of prose case inventories;
PQ-2 requires explicit recovery transition authority for termination during teardown.
Updated the durable plan with the function table and one pure RecoveryTransition owner,
real signal values drained after OSHost.Close, and normal process signal delivery from
signal.Stop through exec. A cleanup-induced empty closed channel is not termination.
No implementation started; plan review will be rerun. Removed the obsolete short-named
plan after canonicalization so the committed tree has one durable plan.

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against
baseline-v3.1.md. Method A only. Calibration is marked stale; values are provisional.
The v2 primitives below use thorough-spec design discount x0.2, existing library
reuse where applicable, implementation x0.40 once, familiar stack x1.0, and +15%
design buffer. Tests are included in each implementation item, not costed twice.

| Work | Primitive | Derivation (design; implementation) |
|---|---|---|
| Spec and operator recovery choices | issue-spec | 0.5 x0.2=0.10; 0.2 x0.4=0.08 |
| Cell repair/evidence | greenfield-go-module | 0.75 x0.5 existing VT/validator reuse x0.2=0.075; 0.8 x0.4=0.32 |
| Geometry and shell recovery lifecycle | tui-screen | 1.0 x0.2=0.20; 1.0 x0.4=0.40 |
| Shared shortcut recovery | smaller-go-module | 0.2 x0.2=0.04; 0.5 x0.4=0.20 |
| Existing buffered diagnostic integration | smaller-go-module | 0.15 x0.2=0.03; 0.4 x0.4=0.16 |
| README/atlas | atlas-docs | 0.1 x0.2=0.02; 0.15 x0.4=0.06 |
| One close boundary | milestone-review | 0.1 x0.2=0.02; 0.5 x0.4=0.20 |

No new external API or native library discovery: reuse VT, hostty/PTY, diagnosticlog
and current Zellij admission. No cross-repo implementation. Design subtotal 0.485h,
implementation 1.42h; 0.485 x1.15 +1.42 =1.97775h, rounded 1.98h. Queue wait excluded.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.10 impl=0.08
item: greenfield-go-module design=0.075 impl=0.32
item: tui-screen design=0.20 impl=0.40
item: smaller-go-module design=0.04 impl=0.20
item: smaller-go-module design=0.03 impl=0.16
item: atlas-docs design=0.02 impl=0.06
item: milestone-review design=0.02 impl=0.20
design-buffer: 0.15
total: 1.98
```

### 2026-10-20 — estimate arithmetic correction

The deterministic reconciliation gate caught a summed-design typo: the unchanged
itemized design values total 0.485, not 0.585. Recomputed the total as 1.97775h
(1.98 rounded), with no item back-fitting or gate bypass. Plan-quality remained clean.

### 2026-10-20 — implementation checkpoint and semantic regressions

Full change-code passed after clean plan-quality review. Implemented backend-only
row repair, once-per-endpoint bounded codepoint evidence through the registered
buffered diagnostic writer, launch-size waiting, pure recovery authority and
native shell handoff, plus confirmed-absence right-pane recreation. Frame and
panel safety remain strict. README/atlas describe recovery and evidence location.

Semantic reds: unsafe screen/history snapshots failed presentation and emitted no
fault observation; oversized evidence scanned the entire supplied content; zero
launch geometry failed immediately; a closed signal channel incorrectly ended
sessions; profile setup failure skipped shell handoff while non-TTY pipes wrongly
qualified; the missing-pane shortcut wrote to stale IDs or accepted unknown
inventory. Each now has a passing targeted regression. The native fake initially
misclassified new-pane as requiring a focused target, exposed by the foreign-tab
negative control; corrected the fixture, retaining explicit invoking-tab checks.

Evidence so far: terminal cell/strict-frame/presenter tests with the xterm oracle
and race detector passed (8.359s); recovery/geometry/real-PTY handoff and signal
race tests passed (2.872s); diagnostic admission and shortcut fixture tests passed;
artifact source inventory passed (29.506s). Worst-case sampled fault evidence is
1560 bytes against the sink's 4096-byte bound. A broader affected-package run
started during the production-wiring TDD and recorded its two expected semantic
reds; the final recovery run is green. Remaining work: final affected-package
verification, recorded full suite, close review and landing. No live acceptance
claimed; no pair:0 changes.

### 2026-10-20 — final targeted verification and recorded-suite queue

Implementation commit 55df33935c21f65e800ac483aaa7d82feed7c9ca is pushed in
PR #486. Final complete affected packages passed: terminal 93.468s,
workbenchshortcut 0.213s, zellijtest 17.989s; final termcmd/layoutcmd with race
passed 8.015s/8.698s after production wiring. hostty/ptychild passed
0.238s/0.984s. Artifact-path inventory passed. The recorded full suite is queued
normally (attempt bea3d4367048109a4e60570962f1d871); no duplicate full run.

Fresh implementation checks also cover maximal bounded evidence, strict direct
Frame rejection, exact foreign-tab absence and unknown-inventory no-effect.
The optional diagnostic writer uses its existing registration/rotation/retention;
this incident changes resilience policy, not a new mechanical API misuse class.
There is no generic lint rule that can infer which backend payload is safe to
repair or which teardown grants handoff. The deterministic transition and safety
regressions defend those decisions. Live acceptance remains owed at the next
natural relaunch per the hub, with the actual original cell trigger unknown.

### 2026-10-20 — BR-1/BR-2/BR-3 recovery boundary completeness

The first close review was REWORK, not a finalized close. BR-1 found that the
existing void input pump discarded decoder and read failures before the new
recovery owner could classify them. Preserve typed input endings through the
existing mux failure path: ordinary EOF and owner cancellation remain normal;
unexpected decoder/read failures authorize the existing fault recovery only
after cleanup; terminal-disconnection errors remain ineligible for handoff.
No new recovery state, retry or owner is added.

BR-2 found that native pane creation defaults to moving client focus despite the
existing shortcut contract. Add --no-focus and assert both invoking-pane focus
and new-pane unfocused state; a foreign tab's focus is independent. BR-3 adds a
disposable native production-routing test for same-tab visible creation and
retained client focus, required by the existing scheduled native-terminal runner.
No workflow or merge-gate configuration changes. The operator's live acceptance
still rides the next natural relaunch per the approved rollout.

The old queued suite was canceled through local-check cancel before acquisition
because these production fixes supersede its code head; this is not a flake
retry. A new recorded suite is required on the corrected code.

### 2026-10-20 — review fixes and native argv correction

BR-1 semantic reds reproduced decoder/non-EOF read failures returning nil with
zero handoffs. Typed pump outcomes now retain the fault, while ordinary EOF and
cancellation stay normal and dead transport causes suppress exec. An overlapping
invalid-UTF8 plus EIO control proves a decoder fault cannot conceal disconnection.
The targeted pump/recovery race set passed (2.561s).

BR-2 focused-state controls failed because the recreated pane became focused;
--no-focus corrects the effect. BR-3's real disposable Zellij run then caught a
native argument incompatibility that the fake accepted: --tab-id conflicts with
--near-current-pane. Keep the explicit confirmed tab plus --no-focus, omit the
conflicting flag, and make the fake reject that same invalid combination. This
changes no owner, retry, native lookup or visible recovery state. The conformance
runner requires actual execution of the new native test, not merely go-test exit
zero; its eight Python proof tests pass. The receiver shell in that fixture emits
no native query replies, so the existing producer inventory records that precise
native-only exclusion rather than inventing a synthetic envelope.

BR-3 native green: TestTerminalRecoveryZellijConformance passed 0.54s
(package 0.681s), log /Volumes/TestScratch/tmp/pair-slot7/pair727-terminal-recovery-native-final.log.
Actual PTY input proves focus remains in the invoking pane; dropping only
--no-focus in the test-only runtime makes native input move to the new receiver,
so the negative control proves the focus oracle. No operator context was used.
Final complete race packages after fixes passed: termcmd 6.892s, layoutcmd 7.490s,
zellijtest 8.095s. The initial new-code package run caught the missing native-only
fixture classification and a mismatched godoc declaration name; both corrected
without weakening their sentinels. The recorded suite will run on this fix commit.
