# Boundary Review — pair#214 (whole-issue close)

| field | value |
|-------|-------|
| issue | 214 — racing launch operations on one thread leave it unresumable |
| repo | pair |
| issue file | workshop/issues/000214-racing-launch-operations-on-one-thread-leave-it-unresumable.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6986321e216f9da591464e5991bcb03a5675fdf4..cdbf8242daee65a885ad5f404c149e08fc89aba3 |
| command | sdlc close --issue 214 |
| reviewer | claude |
| timestamp | 2026-10-07T15:38:53-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

Verdict: **REWORK**. Half of the work is done well, but the named binding failures — the first item in Done-when — never reach a real row. The ledger fallback (`PreviousEstablished` plus the owner query falling back on `FreshRequired`) works and is tested. The IO-error fix also works. The problem is in the evidence pass. The real resolver (`resume.go:405`) and its fake (`artifactcollision_fake.go:233`) both return the populated resolution *together with* a typed refusal error. That sends every refusal down the `resolveErr != nil` branch at `actionableinventory.go:1017-1023`. That branch only records `ResumeBindingAmbiguous`, and its comment wrongly says the refusal "carries no resolution". The `default:` branch, which does call `provenBindingRefusal`, is unreachable for any refusal. So `no-turn` and `unconfirmed` never show in production; those rows still read `session-gone` or `binding-lost`.

I confirmed this in a scratch copy at the pinned head. A thread whose resolver returns `{Provisional, FreshRequired, RequestedNativeID}` plus `refuseResolvedBinding(r)` (exactly what the real resolver does) projects `unusable/session-gone`. Replacing the branch body with `item.ParkedRefusal = provenBindingRefusal(ResumeDiagnosticOf(resolveErr), binding)` makes it project `no-turn`. The existing classify, reason, parked and binding tests stay green with that change. The full `couchcore` package hit the 10-minute Go timeout in the sandbox, which looks like the known pty-test limit and not this change.

**1. Strengths**
- `sessionledger.PreviousEstablished` / `generation` (`record.go:571-628`): a clean pure extraction. `CurrentLaunch` reuses the same join (ARCH-DRY), and it never reaches past the nearest earlier launch, which is the right choice.
- The fallback is limited to `FreshRequired` in `QueryResumeTargetContext` (`query.go:444`). It doesn't trigger on missing evidence (an incomplete listing stays provisional), and `ResumeTargetForRuntimeLaunch`, which the restart path uses, is untouched. The test covers all four branches.
- The IO-error hole is closed: a resolver error with no code leaves the proof unresolved. `TestAResolverIOFailureProjectsUnknownNotABindingReason` runs end-to-end through the inventory.
- `provenBindingRefusal` keeps "named only when proven" as a pure, table-tested function, and `IsBindingFailure` brings the four reason sites under one class rule.

**2. Critical**
- `actionableinventory.go:1017-1023`: the typed-refusal branch drops the resolution, so `ReasonNoTurn` and `ReasonUnconfirmed` are dead in production. Fix: call `provenBindingRefusal(ResumeDiagnosticOf(resolveErr), binding)` there, fix the comment, and either delete the now-unreachable `bindingResumeDiagnostic` arm in `default:` or mark it as the no-error path. Add an end-to-end inventory test (like the scratch one) using a resolver that returns resolution and refusal together, for both no-turn and unconfirmed.

**3. Important**
- **Missing coverage for this bug class.** The reasons are tested only by feeding hand-built `ThreadEvidence{ParkedRefusal: …}` into `ClassifyThread`, plus the pure `provenBindingRefusal`. Nothing runs the production path from resolver to evidence to classification for a named reason, which is how the dead branch shipped green. The "every new reason is produced by a test shape" guard passes without any production path producing those reasons.
- **The `unconfirmed` wording doesn't match its proof.** The reason is now only produced when `ObservationIncomplete` is set, meaning native storage couldn't be listed. Yet the label (`threadreason.go:161`) and menu notice (`menu.go:1283`) say "retry after a turn", and a parked or detached thread can't take a turn. The resolver's own message for this case (`resume.go:337`) says "retry when its storage can be listed". The slot actions for this reason are reboot only (`actor_actions_test.go:186`), so the label also points at a retry the row doesn't offer. Fix: reword to the storage condition, or revisit which evidence counts as "unconfirmed".

**4. Minor**
- `readOwnerLaunch` now returns five positional values; a small result struct would read better.
- There are now three wording tables for the same codes: `Label()`, `unusableThreadNotice`, and `bindingRefusalDiagnostic` (ARCH-DRY). Consider deriving the notices from one place.
- Done-when asks for a test that reproduces the 2026-09-08 ledger shape. The full 26/29/31 shape exists only in the `PreviousEstablished` table; the query-level test uses a simpler one-binding shape.
- The fallback overwrites the diagnostics with `result.Diagnostics`. That's fine, but `FellBackFrom` isn't surfaced in any diagnostic, so no operator or log will see it.

**5. Test coverage notes**
- The fallback layer is well covered. The couch layer needs one inventory-level test per named reason that goes through a resolver returning (resolution, refusal), which is the real seam's contract (ARCH-MOCK: the fake already models it, but no test drives it).

**6. Architecture**
- **ARCH-DRY:** passes overall (the `generation` extraction); one minor flag on the triplicated wording.
- **ARCH-PURE:** passes. The new logic is pure (`PreviousEstablished`, `provenBindingRefusal`, `bindingFailureReason`) and the evidence pass is thin glue.
- **ARCH-PURPOSE:** **flagged.** The named-failure item, the point of Done-when bullet 1, isn't delivered at runtime. That is the Critical finding.
- **ARCH-MOCK:** passes for sessioninventory, which uses the fake runtime. For couch, the stateful fake exists but no test exercises its refusal-with-resolution behaviour.
- **ARCH-CONSTRAINTS:** passes. Still one ledger read, and the fallback is O(records).
- **ARCH-SECURE:** passes; no new trust boundary. The ledger parsing is unchanged.
- **ARCH-ORDER:** passes. No state is held between events; the fallback and the mapping are functions of a single read.
- **ARCH-FUNERAL:** passes; nothing durable is created.

**7. Plan revisions**
- Add a `## Revisions` entry recording that the evidence pass must read the resolution that comes with a typed refusal (the resolver contract is "resolution + refusal"). Also update the plan's evidence-pass description, which says the code is recorded "from a refusal code, or a nonempty `bindingResumeDiagnostic`", to match the fixed code.
- Revisit the `unconfirmed` label and repair column in the plan's reason table, since the proven condition is an incomplete listing, not a missing turn.

```findings
findings:
  - id: new
    severity: Critical
    family: refusal-drops-resolution
    title: |
      Evidence pass drops the resolution on a typed refusal, so no-turn and unconfirmed are never produced
    detail: |
      Real and fake resolvers return (resolution, ResumeRefusal) together; actionableinventory.go:1017-1023 records only Ambiguous on that branch and the default-branch provenBindingRefusal is unreachable for refusals. Scratch test at head: FreshRequired refusal projects session-gone; calling provenBindingRefusal(ResumeDiagnosticOf(resolveErr), binding) there yields no-turn.
  - id: new
    severity: Important
    family: test-bypasses-production-seam
    title: |
      Named reasons are tested only via hand-built ThreadEvidence, never through resolver to evidence to classify
    detail: |
      Add inventory-level tests using a resolver returning resolution plus refusal (the real contract) for no-turn and unconfirmed; the existing shapes stayed green over the dead branch.
  - id: new
    severity: Important
    family: label-matches-proof
    title: |
      unconfirmed label says retry after a turn, but it is proven only by an incomplete storage listing
    detail: |
      provenBindingRefusal names provisional only when ObservationIncomplete is set; the resolver message says retry when storage can be listed, and slot actions offer reboot only. Reword the Label and menu notice, or change the proof condition.
  - id: new
    severity: Minor
    family: wording-single-source
    title: |
      Binding-code wording now lives in three tables (Label, unusableThreadNotice, bindingRefusalDiagnostic)
  - id: new
    severity: Minor
    family: done-when-test-shape
    title: |
      Query-level fallback test uses a simplified ledger; the 2026-09-08 26/29/31 shape exists only in the pure table
```
