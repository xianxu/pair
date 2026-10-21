---
id: 000731
status: codecomplete
deps: [pair#727]
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'cea5c2df3cbccf52ac09f429e7734e5953bc4095' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T20:32:00-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
actual_hours: 1.01
---

# A pane that fell back to a native shell still classifies as a Pair terminal

## Problem

pair#727 improves wrapper failures by restoring the terminal and execing a native interactive shell, so the operator keeps a usable pane. The fallback shell can still be classified as a live Pair terminal.

Source findings in pair#727 (2026-10-20):

- termcmd.OSRuntime.RegisterTerminalPane records the wrapper PID; syscall.Exec preserves that PID.
- workbenchshortcut.TerminalPaneRegistry.LiveIDs checks only PID liveness, so the wrapper registration survives the handoff.
- RoleForPaneWith first calls RoleForPane, which accepts a startup TerminalCommand containing pair term or a title equal to terminal / prefixed terminal. Removing a registry record alone therefore does not prove the pane has lost its Pair role.
- TerminalPaneRegistry currently has no serialization lock: Register is plain append. A lock on the data inode would not serialize an atomic rename. FullscreenReturnStore has a separate stable lock, but borrowing it would couple registration to native fullscreen operations.

No native before/after capture of template, title and pane_command across exec has been made. These are source findings; pair#727's disposable native placement/focus conformance does not establish exec metadata behavior.

## Spec

Measure the native metadata across a wrapper-to-shell exec in a disposable session, then choose a bounded lifetime/role fix. A plain native fallback shell must not be treated as a Pair-aware terminal for global shortcut delivery. Keep the working shell even when bookkeeping fails. Preserve other panes and unknown registry records; avoid a new native settings resolver or heuristic expansion.

## Done when

- A disposable native regression captures the PID, initial command, title and actual pane_command before/after the handoff and proves the selected classification behavior.
- LiveOwnerIDs and RoleForPaneWith reject confirmed shell argv and identity replacement after fallback; ordinary active Pair terminals retain ownership. Unavailable identity/argv inspection preserves the live registry answer and records bounded reason counts; a diagnostic-write failure does not alter classification.
- Another pane's registration survives, including concurrent registration/retirement if the chosen design mutates that registry; unknown records remain intact.
- A bookkeeping failure is visible and leaves the native shell usable.

## Plan

- [x] Confirm the measured owner/role design with the hub; durable plan: [terminal-owner-role-plan](../plans/000731-terminal-owner-role-plan.md).
- [x] Implement and qualify the shared owner observation and role projection under the approved plan, then close and land through SDLC.

## Log

Deferred from pair#727 by the hub on 2026-10-20 (delivery 7225aa16-eefc-46c0-97b6-49c250d1ca29): land the reviewed resilience improvement, which leaves a working shell instead of a dead pane. No registry/role change belongs in #727. The earlier retirement-only proposal was withdrawn before implementation when the role and lock assumptions proved incomplete.

### 2026-10-20 — claim, native measurement and proposed plan
- 2026-10-20: closed — Recorded full attempt4e0911c5874d0fb462a6898c6d9e8a8e exposed the shell fixture missing registered ownership (log SHA9387cb0035b1bc27d4026664897f2327ca9aa8106ae442a33342caf7bc2b52be). Updated only test fixture and docs: real shared Pair wrapper on a private PTY, acknowledged shell and joined teardown; unchanged positive shortcut assertions and new stale-metadata refusals all pass. Production unchanged; fresh full recorded check follows this review before landing. (fixed-then-ship); review verdict: SHIP
- 2026-10-20: closed — BR-1 acceptance text now names LiveOwnerIDs and explicitly pins confirmed mismatch denial versus unavailable-inspection retention and bounded diagnostics. Seven affected packages, owner/focus race tests, 13 reader tests and private native exec conformance passed. One recorded suite attempt0aa0002014a10c3ebaddb3b1fe334597 remains queued on the unchanged code; recorded green is required before landing.; review verdict: SHIP
- 2026-10-20: flow upgraded quick → full — 138 added lines in code files (limit 100); an earlier round of this close already ran the full review

- Hub dispatched #731 after #706 landed; verified unowned, claimed, and ran start-plan from main. No #706 work remains in flight. Bare start-plan caught up the sibling Ariadne build, as required by the fleet tool rule.
- Hub d78ab148-d0d9-49e4-9255-f5983c3ee9a6 approved bounded disposable native diagnosis. Installed Zellij0.45.1 private session p731-31448 used private HOME, config/data/cache/runtime and ZELLIJ_SOCKET_DIR. A controlled registered pair term executable gated syscall.Exec into /bin/sh. PID31490 and birth remained unchanged; startup command and terminal title survived; pane_command changed to sh -i. LiveIDs and both existing role projections still granted RightTerminal. A command executed by the interactive shell acknowledged the same PID. Cleanup closed the exact session/client; private list-sessions reported none, and the exact owned PIDs were gone. Capture: the inline native metadata capture below. This is native metadata evidence, not a production fault-recovery test. Temporary probe source was removed from cmd; retained owned diagnostic source/binary are under /Volumes/TestScratch/tmp/pair731.p79ksE.
- Found a registry-only cached-draft fast path in termcmd and a direct registry reader in Couch focus admission, alongside layout/clipboard consumers. A pane-command-only fix misses that path. Proposed plan uses existing identity-fenced native per-PID argv evidence plus mandatory registered ownership in RoleForPaneWith. It leaves the append-only registry format and all unknown rows intact, avoiding retirement rewrites/new locks and new lifetime artifacts (ARCH-DRY, ARCH-EVOLVE, ARCH-FUNERAL). No production/test implementation has begun; awaiting hub plan approval before change-code.
- Hub d495d55d-e94e-4033-8b07-6095ee2c618a approved the plan with one compatibility requirement: confirmed shell argv denies; unavailable/unreadable native inspection retains today's liveness-based registry result and records why. Revised the current design/state/test tables and appended its history under Revisions; both policies and diagnostic failure are mandatory tests. No implementation yet; entering change-code next.

### 2026-10-20 — recorded suite fixture defect and correction

Final-head affected attempt4e0911c5874d0fb462a6898c6d9e8a8e fell back to the full suite, acquired capacity after1819s, then failed in4m7s at term-pane-shortcuts-test.sh. Log SHA-2569387cb0035b1bc27d4026664897f2327ca9aa8106ae442a33342caf7bc2b52be. This is a reproducible branch fixture defect, not a flake: the shell integration supplied terminal metadata without a registered native wrapper, unlike the Go fixtures already updated. Go packages were not reached by that suite, so a fresh full run remains required.

The fixture now holds the shared production Pair image on an owned PTY, with a private interactive shell readiness acknowledgement and joined teardown; no native Zellij session is started. Existing split/tab/focus assertions pass unchanged. Added explicit unregistered-metadata refusal cases for split, delivered tab and focus, including focus's nonzero result. The final focused shell run passed all cases and cleanup. Audited other shell consumers: the remaining route fixture substitutes its CLI and does not claim native terminal ownership. No production code, timeout or assertion was weakened. Entering narrow fixed-then-ship review, then one fresh recorded check on its final head.

## Revisions

### 2026-10-20 — approved inspection-unavailable compatibility

The original native owner proposal denied unavailable inspection. The hub's approved contract distinguishes it from confirmed different argv: retain today's registry answer and bounded diagnostic reason when native inspection is unavailable. This accepts possible stale ownership in restricted contexts to preserve working terminal panes. Confirmed wrapper-to-shell exec still loses the role; registry bytes/schema remain unchanged.

### 2026-10-20 — implementation checkpoint

- change-code accepted the design (quick flow inferred). Implemented shared LiveOwnerIDs with identity-bracketed argv, once per PID per registry read, plus mandatory registration in RoleForPaneWith. The env adapter covers term/layout/clipboard; Couch injects the same procutil.Source through its focus probe. Confirmed shell and replaced identity deny; unavailable inspection retains liveness and emits one bounded scoped diagnostic. No registry rewrite or new lock.
- Owner and role regression tests failed before implementation, then passed. Stateful N/2N tests count one argv and two identity reads per unique live PID, including duplicates and rejected shells; no process-table enumeration. Added the pair#731 local lint rule after its four forbidden-reader cases failed; its regression now passes. Corrected an artifact manifest duplicate by reusing the existing scoped-adapt binding.
- Disposable installed-Zellij regression passed with two registered helper panes: wrapper-to-shell exec preserves PID/birth and metadata, but loses admitted ownership while the control remains registered; shell input still works and unknown registry bytes survive. Exact owned session/helpers were collected. This remains a controlled native ownership test, not a production recovery fault injection.
- Updated healthy terminal fixtures to register ownership explicitly; added stale unregistered terminal refusals across admission paths. Focused Couch tests passed. Term/layout agents report package passes; independent combined verification, diagnostic failure coverage, reader preflight, documentation and normal close/landing remain pending. No suite or merge runner is active at this checkpoint.

### 2026-10-20 — qualification and source-reader audit

- Combined uncached tests passed for workbenchshortcut, termcmd, layoutcmd, clipcmd, couchcmd, artifactpath and repolint (Couch53.542s). Focused owner/focus tests also passed with -race. Diagnostic adapter tests retain ownership when its scoped log path is blocked, and otherwise emit one reason without raw native errors; dead/invalid PIDs never reach native inspection.
- Final controlled native conformance passed0.459s: PID67096 kept birth1792555508.235884 while argv changed pair term to sh -i; startup command/title remained unchanged, admitted IDs changed from [2 1] to [2], and registry bytes were unchanged. Session to-aab633161e5d and both helper generations were collected. A first root command used an incorrect test regex and ran no tests; it is excluded from evidence.
- Audited changed reader files: clip/layout/Couch fixtures still read only private or literal inputs; term admission/fullscreen retain their non-Go full-fallback ownership; term run and shortcut tests retain their source scanners, with the new role test added to the fixed preflight. New owner fixtures consume private registries and the owned executable, not repository source. Refreshed hashes through the inventory's supported record command. Fast inventory check passes. The seeded detached-checkout inventory test still sees the last committed inventory until this checkpoint is committed; rerun it afterwards before publishing.
- Updated architecture and local lint documentation. The previous recovery screen is unavailable; workflow state, approved plan, current branch and implementation checkpoint are recovered from SDLC/Git and this Log. No stale screen instruction was used as new authorization.

- Committed-head reader qualification passed all13 tests, including seeded violations through the real detached-checkout affected runner. PR503 is open. Recorded suite attempt0aa0002014a10c3ebaddb3b1fe334597 is queued on code head32ef80aa856c51b5d1a9523a953ac202937a84ea; no duplicate check is queued. Entering the permitted close-review overlap; a recorded pass still precedes landing.

### 2026-10-20 — BR-1 contract alignment

The full close review passed implementation and native behavior but blocked on BR-1: Done when still named generic LiveIDs and implied unconditional rejection. Reconciled the acceptance criteria with the already approved LiveOwnerIDs contract: confirmed mismatch/replacement denies; unavailable inspection retains and logs, with logging failure unable to change classification. No code or policy change. CI run39542144320 also misclassified the .side capture directory as another issue731; preserving that small text-only capture inline below avoids an attachment dependency without weakening the gate.

## Native metadata capture

```json
{
  "scope": "Controlled registered executable named pair term using production registry/projection and gated syscall.Exec; this measures native exec metadata, not the full production recovery path.",
  "result": {
    "version": "zellij 0.45.1",
    "session": "p731-31448",
    "pid_before": 31490,
    "pid_after": 31490,
    "ps_before": "31490 31485 Tue Oct 20 20:35:00 2026     /Volumes/TestScratch/tmp/pair731.p79ksE/pair term\n",
    "ps_after": "31490 31485 Tue Oct 20 20:35:00 2026     sh -i\n",
    "before": [
      {
        "live_ids": [
          "0"
        ],
        "pane": {
          "ID": "0",
          "TabID": 0,
          "TabPosition": 0,
          "TerminalCommand": "",
          "PaneCommand": "",
          "PaneCwd": "",
          "Title": "(.) - zellij:link",
          "IsFocused": false,
          "IsPlugin": true,
          "IsFloating": false,
          "IsFullscreen": false,
          "X": 25,
          "Y": 8,
          "Columns": 50,
          "Rows": 15
        },
        "role_with_registry": 0,
        "role_without_registry": 0
      },
      {
        "live_ids": [
          "0"
        ],
        "pane": {
          "ID": "0",
          "TabID": 0,
          "TabPosition": 0,
          "TerminalCommand": "/Volumes/TestScratch/tmp/pair731.p79ksE/pair term",
          "PaneCommand": "/Volumes/TestScratch/tmp/pair731.p79ksE/pair term",
          "PaneCwd": "/Users/xianxu/workspace/worktree/pair-slot6/pair",
          "Title": "terminal",
          "IsFocused": true,
          "IsPlugin": false,
          "IsFloating": false,
          "IsFullscreen": false,
          "X": 0,
          "Y": 0,
          "Columns": 100,
          "Rows": 30
        },
        "role_with_registry": 3,
        "role_without_registry": 3
      }
    ],
    "after": [
      {
        "live_ids": [
          "0"
        ],
        "pane": {
          "ID": "0",
          "TabID": 0,
          "TabPosition": 0,
          "TerminalCommand": "",
          "PaneCommand": "",
          "PaneCwd": "",
          "Title": "(.) - zellij:link",
          "IsFocused": false,
          "IsPlugin": true,
          "IsFloating": false,
          "IsFullscreen": false,
          "X": 25,
          "Y": 8,
          "Columns": 50,
          "Rows": 15
        },
        "role_with_registry": 0,
        "role_without_registry": 0
      },
      {
        "live_ids": [
          "0"
        ],
        "pane": {
          "ID": "0",
          "TabID": 0,
          "TabPosition": 0,
          "TerminalCommand": "/Volumes/TestScratch/tmp/pair731.p79ksE/pair term",
          "PaneCommand": "sh -i",
          "PaneCwd": "/Users/xianxu/workspace/worktree/pair-slot6/pair",
          "Title": "terminal",
          "IsFocused": true,
          "IsPlugin": false,
          "IsFloating": false,
          "IsFullscreen": false,
          "X": 0,
          "Y": 0,
          "Columns": 100,
          "Rows": 30
        },
        "role_with_registry": 3,
        "role_without_registry": 3
      }
    ]
  },
  "cleanup": {
    "client_returncode": 0,
    "list_returncode": 1,
    "stdout": "",
    "stderr": "No active zellij sessions found.\n"
  },
  "probe_sha256": "1dac624b0b5c9413f9d24ff5a747697d50be346dae355fdc88fa2255f5e61474"
}
```
