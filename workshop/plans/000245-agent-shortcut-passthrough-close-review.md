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
