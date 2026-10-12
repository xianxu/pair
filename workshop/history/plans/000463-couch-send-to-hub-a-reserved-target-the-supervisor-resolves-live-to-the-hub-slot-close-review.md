# Boundary Review — pair#463 (whole-issue close)

| field | value |
|-------|-------|
| issue | 463 — couch --send-to hub: a reserved target the supervisor resolves live to the hub slot |
| repo | pair |
| issue file | workshop/issues/000463-couch-send-to-hub-a-reserved-target-the-supervisor-resolves-live-to-the-hub-slot.md |
| boundary | whole-issue close |
| milestone | — |
| window | 425de54327f006558f857a7309f1406ae9294ee6..6e31a8028d208661a53601884a4b51bc3e1c2a2f |
| command | sdlc close --issue 463 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T00:33:40-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Checking the docs (atlas/README) and running the tests next.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The code is correct and small. A send to the reserved `hub` target is turned into the hub's exact `repo:N` before the broker's repository-name routing, so admission, allowance and receipt rules don't change. Each of the three refusals carries a code and names its reason. The two new tests check real behaviour: I disabled the rewrite in this checkout and both `TestMessageSendToHub*` tests failed, then restored it. Two things stand between this and SHIP. First, the docs gate: README.md and `atlas/couch.md` still describe `--send-to` as `repo[:N]` only. Second, `## Done when` still lists a live check that `## Revisions` moved to ariadne#336.

Done-when: stale. Clause 1 ("verified live in a running Couch") still reads as this issue's exit criterion, but the 2026-10-12 Revision moved the live check to ariadne#336. The clause was never changed to match.

1. **Strengths**
   - `cmd/internal/couchcmd/message_service.go:671-677`: the rewrite runs after the caller is proven current and before the `SendTargeted` wake. The rewritten `repo:N` contains `:`, so a dormant hub session also gets its bounded retry. That one delivery path is good reuse (ARCH-DRY).
   - `message_service.go:695-707`: `hubRecipient` reuses the existing `hubBinding()` (`wait_notice.go:252`), so the hub's identity has one source.
   - Refusal codes are sensible: "no hub" and "you are the hub" are `refused` (permanent), while "hub not connected" is `unavailable` (transient). Each has its own subtest, and the subtests check both the code and the reason text.
   - The self-send check compares `Slot`, not the whole binding, so it still catches a hub whose binding was renewed.

2. **Critical:** none.

3. **Important**
   - **Docs not updated for the new target.** `README.md:422-424` lists the `--send-to` forms without `hub`. `atlas/couch.md:185` says `--send-to repo[:N] ...`. `atlas/couch.md:143` says every `--send-to` resolves through `ResolveRepositoryName`, which is now wrong for `hub`. Fix: add a `couch --send-to hub --message TEXT` row to the README, and an atlas sentence explaining that `hub` is resolved live by the service through `hubBinding` before name resolution.
   - **`## Done when` is stale.** Change clause 1 to say the live check is delivered by ariadne#336, or split it out, so the close gate doesn't claim a check that didn't happen here.

4. **Minor**
   - Reserving `hub` also hides prefix matches. Before this change, `--send-to hub` reached a repository named `hubble` by unique prefix; now it is always the reserved target. SKILL.md says "reserved ahead of repository names" but doesn't mention prefixes. The Spec does, so add a sentence to SKILL.md.
   - No test covers the reservation itself: a repository named `hub` should still be reachable as `hub:N`, and bare `hub` should not reach that repository's slots.
   - When `liveness == nil` (the supervisor hasn't observed slots yet), `hubBinding` reports "the hub isn't connected". Accurate enough, but it can mislead early in startup.

5. **Test coverage notes:** The hub tests pass, and reverting the rewrite makes both fail. All three refusals and the success path are covered. Not covered: `--agent` combined with `hub` (it goes through the existing agent check on exact sends, which is fine) and the reservation and prefix behaviour above. The live delivery check is deferred to ariadne#336.

6. **Architectural notes**
   - **ARCH-DRY: pass.** The change reuses `hubBinding` and the broker's single exact-slot path, with no parallel delivery logic.
   - **ARCH-PURE: pass.** `hubRecipient` is a small decision over in-memory service state, tested through the existing rig without new mocks. It sits inside the glue layer, as `hubBinding` already does.
   - **ARCH-PURPOSE: pass, with a caveat.** The purpose (a slot can reach the hub without knowing its address) is delivered on all the consumer surfaces in the diff (service, `--help`, `--skill`). README and atlas are the remaining restatements, covered by the Important docs finding. Moving the live check to ariadne#336 is an explicit TL decision, recorded in `## Revisions`.

7. **Plan revision recommendations**
   - Amend `## Done when` clause 1 to match the 2026-10-12 Revision, e.g. "live delivery check: delivered by ariadne#336's Done-when".

```findings
findings:
  - id: new
    severity: Important
    family: docs-gate-new-surface
    title: |
      README and atlas not updated for the reserved send-to hub target
    detail: |
      README.md:422-424 lists send-to forms without hub; atlas/couch.md:185 still says send-to repo[:N];
      atlas/couch.md:143 says every send-to resolves via ResolveRepositoryName, now false for hub.
      Add the hub row to README and an atlas note that hub is resolved live in the service via hubBinding before name resolution.
  - id: new
    severity: Important
    family: done-when-tracks-revisions
    title: |
      Done-when clause 1 (live check) is stale after the Revision moved it to ariadne 336
    detail: |
      Revisions moved the live delivery check to ariadne 336 but Done-when still lists it as this issue's exit criterion.
      Amend the clause to name where it is delivered.
  - id: new
    severity: Minor
    family: reserved-name-shadowing
    title: |
      Reserving hub also shadows unique-prefix matches (e.g. hubble); SKILL.md says only repository names
    detail: |
      SKILL.md reservation paragraph should say hub is reserved ahead of prefix resolution too, matching the Spec.
      No test pins that a repo named hub stays reachable as hub:N while bare hub goes to the hub slot.
```

---

## Re-review — 2026-10-12T00:36:24-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 463 — couch --send-to hub: a reserved target the supervisor resolves live to the hub slot |
| repo | pair |
| issue file | workshop/issues/000463-couch-send-to-hub-a-reserved-target-the-supervisor-resolves-live-to-the-hub-slot.md |
| boundary | whole-issue close |
| milestone | — |
| window | 425de54327f006558f857a7309f1406ae9294ee6..3c8791ab6240bb4862bec2b12802514de2d3cafd |
| command | sdlc close --issue 463 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T00:36:24-07:00 |
| verdict | SHIP |

## Review

Prior findings look addressed in the diff; running the package tests to confirm before the verdict.

Tests pass. Doing one revert check on the reservation test, then I'll write up the verdict.

```verdict
verdict: SHIP
confidence: high
```

All three prior findings are fixed, and I checked each against the code. `hub` resolves live in `messageService.handle`, before the broker routes anything, through the same `hubBinding()` the waiting notices already use (`cmd/internal/couchcmd/message_service.go:671-677`, `:691-707`). After that rewrite it becomes an ordinary exact `repo:N` send, so admission, allowance, receipts and the dormant-session wake all apply unchanged. All three hub tests pass. To build the `couchcmd` package I first had to run `make`'s `runtimebundle-generate` step. I also switched the rewrite off and reran the reservation test: it fails ("send to hub: not-dispatched"), so it guards real behaviour. I then restored the file. Nothing blocks shipping; there are three Minor notes.

1. **Strengths**
   - The rewrite sits in the service, which owns `authority.hub`, not in the pure `couchmessage` routing. That keeps one delivery path, and an `--agent` mismatch on the hub gets the existing exact-slot refusal (`routing.go:151`).
   - Each refusal is distinct and names its reason: `refused` for no hub, `unavailable` for a hub that isn't connected, `refused` when you are the hub. A test pins each one (`message_service_test.go:684-715`).
   - `TestMessageHubTargetIsReservedAheadOfRepositoryNames` checks both sides of the reservation in one fixture: bare `hub` reaches the hub slot, and `hub:3` still reaches the repository named `hub`.
   - The `--help`, README, atlas and SKILL.md wording all match one another and the code.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - `atlas/couch.md:143-144`: two parentheticals in a row, "`ResolveRepositoryName` (except the reserved bare `hub`, below) (`couchcore/repositoryname.go`)". Put the exception after the path.
   - The issue's Plan "Tests:" item lists two tests. It leaves out `TestMessageHubTargetIsReservedAheadOfRepositoryNames`, which was added in the review-fix round.
   - `SKILL.md`: "reached by its exact or full name, `hub:N`" reads as if `hubble` were also reached as `hub:N`. "`hub:N` / `hubble`" would be clearer.

5. **Test coverage**
   - Done-when clause 1 (delivery from a non-hub slot) and clause 2 (three refusals) are each covered by service tests through `handle`.
   - Clause 3 (the skill documents `hub`) is met by the SKILL.md text.
   - The reservation has a test that fails without the fix.
   - The live check in a running Couch is explicitly moved to ariadne#336, and the Revisions section records it.

6. **Architecture notes**
   - **ARCH-DRY: pass.** It reuses `hubBinding()` and the broker's exact-target path; nothing is duplicated.
   - **ARCH-PURE: pass.** `hubRecipient` is a thin lookup on service state. The routing core stays pure and unchanged.
   - **ARCH-PURPOSE: pass.** The live resolution, the refusals, the reservation and the skill docs are all delivered. The only thing deferred is the live run, which needs a supervisor restart and is tracked in ariadne#336.
   - One thing for later: `hubBinding` takes the first workspace in a map whose `isHub` is true. That's fine while there is one hub slot, but its order is undefined if the hub worktree ever hosts more than one slot. This predates the change.

7. **Plan revisions:** optionally add the third test name to the Plan "Tests:" line. No Revisions entry is needed.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      README.md:425 adds the hub row; atlas/couch.md:144 notes the exception, :186 the usage, :193-198 explain hubRecipient/hubBinding ahead of name and prefix resolution.
  - id: BR-2
    disposition: addressed
    note: |
      Done-when clause 1 now names the service test here and that the live check moved to ariadne 336, matching the Revisions entry.
  - id: BR-3
    disposition: addressed
    note: |
      SKILL.md now covers prefixes (hubble); TestMessageHubTargetIsReservedAheadOfRepositoryNames pins bare hub vs hub:3 and fails with the rewrite disabled (verified by revert).
findings:
  - id: new
    severity: Minor
    family: docs-gate-new-surface
    title: |
      atlas/couch.md:143-144 has two consecutive parentheticals; SKILL.md says hub:N where hubble is meant
    detail: |
      The window is small, so I checked every instance: these two are the only wording issues; README and the --help usage line read correctly. Move the exception after the repositoryname.go path, and write hub:N / hubble in SKILL.md.
  - id: new
    severity: Minor
    family: done-when-tracks-revisions
    title: |
      Plan Tests line omits TestMessageHubTargetIsReservedAheadOfRepositoryNames added in the fix round
    detail: |
      The rule: the Plan and Done-when must be updated in the same commit as whatever changes their claims. Here the review-fix commit added a test without updating the Plan list. This is the only instance in the window.
```
