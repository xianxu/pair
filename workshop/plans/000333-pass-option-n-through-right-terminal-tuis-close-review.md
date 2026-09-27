# Boundary Review — pair#333 (whole-issue close)

| field | value |
|-------|-------|
| issue | 333 — Pass Option+n through to right-terminal TUIs |
| repo | pair |
| issue file | workshop/issues/000333-pass-option-n-through-right-terminal-tuis.md |
| boundary | whole-issue close |
| milestone | — |
| window | 5dab9c6fca12e1dd41d7889b2be6315dfb5c804a..acce3811d5e7540c068c3bf190bfcb3b38940864 |
| command | sdlc close --issue 333 |
| reviewer | codex |
| timestamp | 2026-09-26T22:16:12-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The routing changes satisfy the core passthrough requirement, and the affected routing suites pass. The boundary is blocked by incorrect hosted key-help text and a reproducible failure in the key-help suite.

1. **Strengths**
   - Couch queries the exact session’s inner-pane focus instead of inferring it from Zellij’s alternate screen.
   - Missing, malformed, and ambiguous focus evidence produces a notice rather than authorizing a restart.
   - Input-loop tests cover forwarding both restart chords and retaining draft confirmation.
   - README and atlas document the new routing exception.

2. **Critical findings:** None.

3. **Important findings**
   - **Incorrect hosted help and failing help regression** — `cmd/internal/workbenchshortcut/shortcut.go:188` now advertises “draft reload” for Couch-owned sessions. When reattached outside Couch, those sessions still reject restart through `CouchOwnsRestart`; the launcher regression confirms this. `TestPresenterAndHostingAreIndependent` also fails for both hosted and standalone output. Restore the hosted refusal guidance, retain the right-terminal exception, and update the standalone expectation to the intended wording. Preserve the hosting-versus-presentation distinction.

4. **Minor findings:** None.

5. **Test coverage notes**
   - Passed: `workbenchshortcut`, `termcmd`, `couchcmd`, and `couchtty` package suites.
   - Passed: `TestRunRestartRefusesACouchOwnedSessionBeforeMutation`.
   - Failed, including an uncached targeted rerun: `keyscmd/TestPresenterAndHostingAreIndependent`.
   - No repository files were changed.

6. **Architectural notes**
   - **ARCH-DRY — pass:** Reuses chord definitions, pane-role classification, and terminal registry.
   - **ARCH-PURE — pass:** Parsing and shortcut policy remain separate from external observation.
   - **ARCH-PURPOSE — flag:** Runtime routing delivers the purpose, but operator help does not accurately describe retained restart restrictions.
   - **ARCH-MOCK — pass:** Injected focus query models changing focus; console tests use fake children.
   - **ARCH-CONSTRAINTS — pass:** Discovery is limited to restart candidates with a one-second context.
   - **ARCH-SECURE — pass:** Client output is validated; uncertain evidence fails visibly.
   - **ARCH-ORDER — pass:** Routing uses the existing event loop without adding background work.
   - **ARCH-FUNERAL — pass:** Creates no new durable artifact family or long-lived handle.

7. **Plan revision recommendations:** None required; correct the help and verification evidence before closing.

```findings
findings:
  - id: new
    severity: Important
    family: key-help-matches-runtime-policy
    title: |
      Hosted restart help contradicts the retained restart guard, and the help suite fails
    detail: |
      cmd/internal/workbenchshortcut/shortcut.go:188 advertises draft reload for Couch-owned sessions reattached outside Couch, although CouchOwnsRestart still refuses it. Restore that refusal guidance alongside the right-terminal exception and update the standalone wording expectation in cmd/internal/keyscmd/keyscmd_test.go:159; TestPresenterAndHostingAreIndependent reproducibly fails in both modes. ARCH-PURPOSE.
```
