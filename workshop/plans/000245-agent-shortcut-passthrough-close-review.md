# Boundary Review — pair#245 (whole-issue close)

| field | value |
|-------|-------|
| issue | 245 — Pass all Pair shortcuts through the agent pane |
| repo | pair |
| issue file | workshop/issues/000245-agent-shortcut-passthrough.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0ad7e1a59c990afffc0fc5e0937d1493a097de01..df2a88972cc79e3449561c250fd2a0233d8d62dd |
| command | sdlc close --issue 245 |
| reviewer | codex |
| timestamp | 2026-09-14T20:48:59-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The ownership policy, documentation, and normal-path tests are consistent with the revised plan. One reproduced framing regression blocks shipping: an incomplete escape prefix can suppress a complete reserved shortcut and retain subsequent input indefinitely. The promised CI path-filter coverage is also incomplete.

## 1. Strengths

- Agent reservations derive from shared binding metadata, without agent-specific exceptions.
- Console authorizes candidates after delivering their prefix, preserving focus-change ordering.
- Acceptance transports actual Console output into the production wrapper; README and atlas explain the revised ownership accurately.

## 2. Critical findings

**Incomplete prefixes can suppress complete reserved shortcuts.**
[wrap.go:1671](/Users/xianxu/workspace/pair/cmd/internal/wrapcmd/wrap.go:1671), interacting with line 1594.

Without Return adaptation, input `text\x1b\x1b[84;4u` should deliver `text` plus Escape and execute the reserved tab action. Instead, suffix handling retains Escape, and the outer loop appends the complete chord to pending. EOF/timeout forwards everything with **zero actions**. Further reads before timeout repeat this condition and grow pending without the declared bound.

A scratch regression test **failed at HEAD and passed when only the previous holdback logic was restored**.

Fix framing against the entire available stream: subsequent bytes can disambiguate a prefix before a recognized chord. Test stray Escape and partial CSI before each reservation, both adaptation modes, subsequent reads, and timeout/EOF. **ARCH-PURPOSE, ARCH-CONSTRAINTS, ARCH-ORDER.**

## 3. Important findings

**Conformance workflow omits promised source triggers.**
[couch-zellij-conformance.yml:23](/Users/xianxu/workspace/pair/.github/workflows/couch-zellij-conformance.yml:23), also line 64.

Both event filters omit wrapper sources, Couch `keys.go`/`console.go`, and Neovim sources, despite Task 5 marking that coverage complete and its revision retaining the requirement. Changes confined to those paths do not trigger this workflow.

Add the missing selectors to both filters and verify representative paths match. **ARCH-PURPOSE.**

## 4. Minor findings

None.

## 5. Test coverage notes

- All six reviewed Go packages passed: `workbenchshortcut`, `wrapcmd`, `couchtty`, `keyhelp`, `couchcmd`, and `termcmd`.
- Pinned-range `git diff --check` passed.
- Existing partition tests miss incomplete-prefix-plus-complete-chord combinations.
- Live Zellij conformance was inspected, not rerun. Operator smoke remains explicitly pending.

## 6. Architectural notes

| Principle | Assessment |
|---|---|
| ARCH-DRY | Pass — reservations and generated consumers reuse shared declarations. |
| ARCH-PURE | Pass — policy and framing remain deterministic. |
| ARCH-PURPOSE | Flag — reserved-action delivery and promised CI triggers have gaps. |
| ARCH-MOCK | Pass — stateful terminal seams and separate live conformance are present. |
| ARCH-CONSTRAINTS | Flag — pending input can exceed the declared finite bound. |
| ARCH-SECURE | Pass — inspected command execution uses structured argv; paste coverage is present. |
| ARCH-ORDER | Flag — an unresolved prefix repeatedly postpones an already-complete chord. |
| ARCH-FUNERAL | Pass — new fixture resources have cleanup paths. |

## 7. Plan revision recommendations

Append a dated revision documenting whole-stream prefix disambiguation and the expanded regression matrix. Record completion of the missing CI selectors after verification.

```findings
findings:
  - id: new
    severity: Critical
    family: bounded-incremental-input-framing
    title: |
      An incomplete prefix suppresses complete reserved shortcuts and permits unbounded pending input
    detail: |
      cmd/internal/wrapcmd/wrap.go:1671 and :1594 retain a prefix together with an already-complete chord. A scratch test using text\x1b\x1b[84;4u fails at HEAD with zero actions and passes with only the previous holdback logic restored. Resolve prefixes against all available bytes and test bounded pending state across subsequent reads, timeout, and EOF in both adaptation modes. ARCH-PURPOSE, ARCH-CONSTRAINTS, ARCH-ORDER.
  - id: new
    severity: Important
    family: conformance-source-trigger-coverage
    title: |
      Both conformance event filters omit source paths promised by the completed plan
    detail: |
      .github/workflows/couch-zellij-conformance.yml:23 and :64 omit wrapper, Couch keys.go/console.go, and Neovim paths. Add these selectors to both pull_request and push filters and verify representative paths match, as Task 5 requires. ARCH-PURPOSE.
```

---

## Re-review — 2026-09-14T20:59:54-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 245 — Pass all Pair shortcuts through the agent pane |
| repo | pair |
| issue file | workshop/issues/000245-agent-shortcut-passthrough.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0ad7e1a59c990afffc0fc5e0937d1493a097de01..9e9d9ee182c321d99012c427edffb9ca248e056d |
| command | sdlc close --issue 245 |
| reviewer | codex |
| timestamp | 2026-09-14T20:59:54-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation matches the revised shortcut ownership policy, and BR-1 is verified fixed. BR-2’s selectors are corrected, but its required regression test is missing. That evidence gap is the only remaining blocker; no new behavioral defects were found.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The incomplete-prefix regressions pass at HEAD, including race coverage. Removing the prefix-release fix through a temporary Go overlay makes TestReservedShortcutAfterIncompletePrefixEOF fail with zero actions, establishing regression sensitivity.
  - id: BR-2
    disposition: not-addressed
    note: |
      Both workflow filters now match the promised source families. An independent before/after check confirms five missing representative paths become covered in each event. However, no committed regression test preserves this check; the plan records only an ad hoc run. Under the executable-configuration evidence requirement, add a reproducible test covering both event filters that fails when the added selectors are removed.
```

## 1. Strengths

- Wrapper regressions exercise malformed prefixes, every reserved encoding, read partitions, timeout, and EOF in both adaptation modes.
- Couch routes preceding input before authorizing the following shortcut against current focus.
- Reservation metadata drives routing and help; README and atlas explain the changed ownership.
- Native Neovim tests verify preserved help/changelog geometry and visible launch failures.

## 2. Critical findings

None remaining. BR-1 is addressed.

## 3. Important findings

**BR-2 — regression evidence remains incomplete.**
[Workflow selectors](/Users/xianxu/workspace/pair/.github/workflows/couch-zellij-conformance.yml:25) and the corresponding push selectors at line 70 are correct. Preserve the representative-path check as an automated regression covering **both** event filters and all promised source families. Verify it fails without the selector additions. **ARCH-PURPOSE.**

This is the existing finding’s outstanding verification requirement, not a new finding.

## 4. Minor findings

None.

## 5. Test coverage notes

Passed:

- Seven affected Go packages, including wrapper, Couch, terminal routing, and help.
- Native Neovim routing tests.
- BR-1 focused race tests.
- BR-1 mutation check: removing the fix makes the regression fail.

The pinned-range whitespace check reports two trailing-space lines in the previous review artifact. Live Zellij conformance was inspected but not rerun. Operator smoke remains explicitly pending.

## 6. Architectural notes

| Principle | Result |
|---|---|
| ARCH-DRY | Pass — routing/help reuse reservation declarations. |
| ARCH-PURE | Pass — framing and policy remain separate from execution. |
| ARCH-PURPOSE | Flag — BR-2 lacks preserved regression evidence. |
| ARCH-MOCK | Pass — portable composition uses existing terminal seams; live conformance has a separate fixture. |
| ARCH-CONSTRAINTS | Pass — prefix progress and bounded retention are exercised. |
| ARCH-SECURE | Pass — pasted input remains literal; view commands use structured argv. |
| ARCH-ORDER | Pass — post-prefix authorization and timeout/EOF behavior are covered. |
| ARCH-FUNERAL | Pass — obsolete bindings are removed; fixture resources have cleanup paths. |

## 7. Plan revision recommendations

Append a dated `## Revisions` entry naming BR-2’s regression test and its red/green evidence once added. The revised concept inventory otherwise matches the implementation.
