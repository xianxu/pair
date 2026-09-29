# Boundary Review — pair#346 (whole-issue close)

| field | value |
|-------|-------|
| issue | 346 — Stale temporary store blocks Couch startup |
| repo | pair |
| issue file | workshop/issues/000346-stale-temporary-store.md |
| boundary | whole-issue close |
| milestone | — |
| window | b622052f6f85db7f1585de823a8fa591b6d7e2d3..80d611897eff2ed2d7440db4d630237638cdc71f |
| command | sdlc close --issue 346 |
| reviewer | claude |
| timestamp | 2026-09-29T13:57:02-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

This is a whole-issue close review of `b622052f..80d61189`. M1 and M2 each already passed their own milestone review with a SHIP verdict; I did not re-review their full code line by line, only the pieces summarised below. Since the last review round, only two commits touch non-issue files. `817ba48d` adds a lesson to `workshop/lessons.md`. `fd8c5cc3` moves a doc comment from `refuseResolvedBinding` back onto `refuseBinding`, where it belongs, with no behaviour change. Neither changes what the earlier rounds approved.

The three open findings (BR-12, BR-15, BR-16) are all Minor. None is fixed in the code: `sessionwatch/run.go` still keeps the watcher phases in separate fields, and `couchcore/resume.go:314` still says "retry when its storage can be listed". Each one is explicitly moved to #350, which is committed at HEAD. #350's Done-when criteria cover the rule behind each finding: one typed state that the launcher and Couch both read, and refusal messages that name the failing root or entry plus an explicit fresh-start command. Minor findings never block a boundary, so nothing blocks this close.

I checked HEAD directly:
- **Build and lint:** `go build ./...` passes, and `gofmt -l` and `go vet` are clean on every changed Go file.
- **Tests:** sessioninventory, sessionledger, storagegc, sessionwatch and gccmd pass. In launcher, only `TestCreateLayoutWrapperPreservesAgentCommand` fails, with "operation not permitted". That is the sandbox blocking the PTY child process, not a product bug. The issue Log records a full unsandboxed `go test ./...` passing it.

1. **Strengths**
   - The doc-comment fix (`couchcore/resume.go:312-327`) puts each comment back on its own declaration; `refuseResolvedBinding` is just a thin wrapper over the same diagnostic code.
   - The deferred items went into a real follow-up issue, #350, with testable Done-when criteria. That issue also restates what #346 guarantees, so the follow-up cannot quietly weaken it: file presence is enough for a chosen ID, only confirmed absence permits a fresh UUID, and an unknown state refuses before any side effects.
   - The lesson in `workshop/lessons.md` states the rule for the whole class of problem, not just the one place it showed up.
   - README and atlas (`couch.md`, `index.md`, `session-identity.md`) are updated in the window. TTY scope was moved to #349 with operator approval, and the Plan says so instead of claiming it.

2. **Critical findings:** none.

3. **Important findings:** none.

4. **Minor findings:** BR-12, BR-15 and BR-16 stay open and are tracked in #350 (dispositions below).

5. **Test coverage notes**
   - The one known full-suite failure is `artifactpath.TestProductionArtifactReferencesAreExactlyClassified`. The Log records the same failure at the merge-base, and it is tracked as #348, so it is not caused by this issue.
   - The launcher PTY failure appears only inside the sandbox.

6. **Architecture notes**
   - **ARCH-DRY:** pass.
   - **ARCH-PURE:** pass.
   - **ARCH-PURPOSE:** pass. Every Done-when item is covered by the M1/M2 reviews and the Log, and the TTY work was moved out with operator approval.
   - **ARCH-MOCK:** pass (accepted at earlier rounds).
   - **ARCH-CONSTRAINTS:** pass.
   - **ARCH-SECURE:** pass; BR-11 validates IDs before they reach argv.
   - **ARCH-ORDER:** flag, Minor. The watcher phases and the chosen-ID "present / absent / unknown" state are still spread across separate fields; that is BR-12 and BR-15, handed to #350. Behaviour is correct: an uncertain result is never treated as absence.
   - **ARCH-FUNERAL:** pass. `gc --forget-missing-store` gives a registered store a defined way to be removed.

7. **Plan revisions:** none. The Plan already records that M3 moved to #349 and that no M3 work is claimed.

```findings
dispose:
  - id: BR-12
    disposition: not-addressed
    note: |
      Code unchanged (watcher phases still separate fields); explicitly deferred to #350 with a typed-state Done-when. Minor, non-blocking.
  - id: BR-15
    disposition: not-addressed
    note: |
      Unknown materialization still re-derived by consumers; deferred to #350 (single typed authority consumed by launcher and Couch). Minor, non-blocking.
  - id: BR-16
    disposition: not-addressed
    note: |
      resume.go:314 still says only "retry"; deferred to #350, whose Done-when requires naming the failed root/entry plus an explicit fresh-start action. Minor, non-blocking.
```
