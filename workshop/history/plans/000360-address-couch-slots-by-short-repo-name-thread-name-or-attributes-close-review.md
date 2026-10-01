# Boundary Review — pair#360 (whole-issue close)

| field | value |
|-------|-------|
| issue | 360 — Address couch slots by repository prefix, alias, or agent |
| repo | pair |
| issue file | workshop/issues/000360-address-couch-slots-by-short-repo-name-thread-name-or-attributes.md |
| boundary | whole-issue close |
| milestone | — |
| window | 5d3ab19488efe71e27961132ecbed41c66006f21..4a51373bcd65741a5db1df153a31c6b6924cb933 |
| command | sdlc close --issue 360 |
| reviewer | claude |
| timestamp | 2026-10-01T12:03:24-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

This review is blocked by one Critical finding: `--send-to` can deliver a message to the wrong repository. Message routing resolves names only against repositories that currently have a live slot. So when an enrolled repository has no live slot, its exact name falls through to the prefix rule and lands on any other live repository whose name starts with it. I reproduced this against the head commit in a scratch copy. With only `brainstorm:0` live, `ResolveRecipient(Route{Target:"brain:0"})` returned `brainstorm:0` with no error, and the same happened for `brain`. The operation path already guards against this case: the plan's Decisions section says a broken sibling keeps its own error "instead of silently routing elsewhere". The message path has no equivalent guard. The fix is small and local. The rest of the work is solid and traces cleanly to the Spec, the Done-when list and the Revisions.

**Strengths**
- **One shared resolver.** `couchcore/repositoryname.go` (`ResolveRepositoryName`) counts matches per `Key`, so a repository's own directory name and alias never compete. Two repositories that share a name are refused instead of one being picked (#353 BR-5). Both the operation and message paths use it.
- **Alias store follows the documented limits.** It is a separate root-store file, so older builds can still read the strictly decoded manifest. It is validated under the same lock as the write. An alias that a later enrollment would shadow is withheld when the file is read. Its lifecycle comment covers ARCH-FUNERAL.
- **`--actors` reads memory only.** It uses explicit staleness limits (`ObservationStaleAfter`/`RestingStaleAfter`). The 10s verification window applies only to paths that observe; Reserve and Deliver always re-check. The tests pin this: `TestMessageEndpointObserveReusesRecentCheckButReserveNever`, `TestActorsReadsMemoryWithoutProbing`, `TestMessageReconcileForgetsDeadBindings`.
- **Smoke-found label bug has a regression test.** `TestAliasBeatsThreadNameAndPaneLabelWithoutSlots` goes through the tab-bar model, not just `Label()`.
- **Every surface is documented.** README, atlas, `SKILL.md` and `couch --help` all describe prefixes, aliases and `--agent`.

**Critical**
- `cmd/internal/couchmessage/routing.go` (`canonicalTarget`): the names it resolves against are built only from live candidates. An enrolled repository with no live slot therefore loses its exact-match claim, and a prefix match silently redirects the send to a different live repository (`brain:0` → `brainstorm:0`).
  - Fix sketch: have the alias source (`messageAuthority.aliases`, which already reads `RepositoryNamesContext`) also return the enrolled message families. Add those to the names in `canonicalTarget`, so an enrolled family matches exactly and then misses with `ErrUnavailable`/`ErrNoRecipient`, listing the live slots.
  - Add a routing test with `brain` enrolled but offline and `brainstorm` live. Both `brain` and `brain:0` must be refused.
  - Class sweep (ARCH-PURPOSE): this is the rule "prefix resolution must not run over a candidate set narrower than the exact-name namespace". The operation path is covered (enrolled set plus the sibling check), and the switcher filter is exempt because it lists every match. Messaging is the only instance.

**Important**
- None.

**Minor**
- **Alias read failure blocks every send.** `Broker.admit` reads aliases before resolving, so a corrupt `repository-aliases.json`, or the store staying busy for 500ms, refuses all sends, including exact-name sends that no alias could change. Failing closed is defensible, but the error should say plainly that aliases are the cause and how to recover.
- **Alias overrides all non-slot labels in the repository.** `ApplyRepositoryAliases` gives the alias to every non-slot row in the repository scope, and `Label()` then lets it override any thread name. Any second non-slot thread in the same primary checkout (parked, archived) would also show as `blog`.
- **Shadow check uses a different directory than the resolver.** `Couch.RepositoryAlias` checks for a shadowing directory beside the repository's primary root, but `repositoryPrimary` checks beside the caller's fleet root. They agree in the usual single-fleet layout and can drift otherwise.
- **Version allowlist removal is outside the Spec.** Dropping the exact-version check in `wrapcmd/peer_runtime.go` was operator-directed, logged, documented, and #368 is filed. It does widen which agent versions receive automatic input (ARCH-SECURE); the composer-state checks remain the safety net.

**Test coverage**
- The pure resolver, store, slot resolution, routing (prefix, alias, agent, ambiguity, miss) and switcher are well covered. The heartbeat window tests inject a clock.
- Missing: the offline-exact-repository case above.
- I ran the #360-focused tests in `couchcore` and `couchmessage` in a scratch copy, and they pass. The wider package failures I saw there come from the scratch copy and sandbox (`mkdir /tmp` is denied, and there is no git checkout). They are not regressions; the implementor logged full-suite evidence on pair:0.

**Architecture**
- **ARCH-DRY: pass.** The duplication is minor: `liveCandidates` repeats the bounded-list logic in `FormatRepositoryCandidates`. A shared `boundedList(rows)` helper would remove it.
- **ARCH-PURE: pass.** The resolver, validation and `ResolveRecipient` are pure; store and broker IO stay at the edges.
- **ARCH-PURPOSE: flag.** This is the Critical finding: messaging resolution doesn't fully deliver the Spec's "exact name wins" purpose.
- **ARCH-MOCK: pass.** Existing fakes are used and no new external seam is added.
- **ARCH-CONSTRAINTS: pass.** The resolver has explicit bounds (12 entries / 1 KiB, a 500ms store wait, a 250ms alias read for listings, a 10s verification window), and the measured `--actors` timeout is fixed.
- **ARCH-SECURE: pass with note.** The alias file is decoded strictly and fails visibly. Observation paths now rely on a full check from up to 10s earlier, but Reserve and Deliver still re-check. The allowlist note is above.
- **ARCH-ORDER: pass.** Verification freshness is one timestamp per binding, and the window tests inject a clock.
- **ARCH-FUNERAL: pass.** Alias entries are bounded by enrollment, and the `verified`/`workspaces` maps are cleaned up when reconciliation finds a binding dead.

**Plan revision recommendations**
- After the fix, add a Revisions entry: "Message-side name resolution covers the enrolled families plus the live ones, so an offline exact repository refuses instead of prefix-routing to another (review finding)."

```findings
findings:
  - id: new
    severity: Critical
    family: prefix-resolution-over-narrowed-namespace
    title: |
      --send-to an enrolled-but-offline repo silently prefix-routes to a different live repo
    detail: |
      canonicalTarget builds the names it resolves against from live bindings only, so ResolveRecipient(Route{Target:"brain:0"}) with only brainstorm:0 live returns brainstorm:0 (reproduced at head). Feed enrolled families (already read by messageAuthority.aliases via RepositoryNamesContext) into the resolver's name set so an exact enrolled name misses instead of rerouting; add the regression test.
  - id: new
    severity: Minor
    family: display-only-input-blocks-core-path
    title: |
      An unreadable or busy alias store fails every send, including exact-name sends
    detail: |
      Broker.admit returns readAliases errors before resolving. Failing closed is defensible, but the error should name the alias file and how to recover.
  - id: new
    severity: Minor
    family: alias-label-scope
    title: |
      Alias overrides labels of every non-slot row in the repository scope
    detail: |
      ApplyRepositoryAliases sets RepositoryAlias by RepoScope, and Label() returns it ahead of the thread name, so other non-slot threads in the primary checkout all render as the alias.
  - id: new
    severity: Minor
    family: shadow-check-anchor-mismatch
    title: |
      Alias shadow check anchors on the primary root's parent, while the resolver checks the caller's fleet root
  - id: new
    severity: Minor
    family: bounded-list-duplication
    title: |
      liveCandidates duplicates FormatRepositoryCandidates' bounded-list logic (ARCH-DRY)
```

---

## Re-review — 2026-10-01T12:22:10-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 360 — Address couch slots by repository prefix, alias, or agent |
| repo | pair |
| issue file | workshop/issues/000360-address-couch-slots-by-short-repo-name-thread-name-or-attributes.md |
| boundary | whole-issue close |
| milestone | — |
| window | 5d3ab19488efe71e27961132ecbed41c66006f21..5f6efb1759a41d62d7bc3c0611404b446393bc5e |
| command | sdlc close --issue 360 |
| reviewer | claude |
| timestamp | 2026-10-01T12:22:10-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

All five round-1 findings are fixed at `5f6efb17`, and nothing blocks the close. The Critical one, BR-1 (an offline repository's exact name rerouting to a longer live one), now has a regression test that fails without the fix. I checked this in a scratch copy from `git archive`: with the enrolled-families loop removed from `canonicalTarget`, `TestRoutingOfflineEnrolledNameNeverPrefixRoutes` fails at `routing_short_test.go:178` (`brain:0: <nil>`). The routing, alias, repository-name and slot-resolution tests pass in the sandbox. Tests that need a pty or a fixed `/tmp` path fail there with "operation not permitted", which is the known sandbox limit, not a regression. The full-suite run recorded in the issue Log was at `b8ca068c`, before the round-1 commit. I only raise Minor nits below, plus one point to re-smoke: the round-1 change to how the `:0` row gets its alias label has not been seen live.

1. **Strengths**
   - `couchmessage/routing.go:66-75`: enrolled families now join the names that routing resolves against. A prefix shared by an offline and a live repository is refused as ambiguous instead of picking the live one, and the test has a control case (no enrolled set means the old reroute), so the test shows exactly what the fix guards.
   - `couchcmd/message_service.go:510`: `messageFamilies` keeps a directory name shared by two enrolled repositories known (so it can't prefix-route elsewhere) but publishes no alias for it. The test was updated to match.
   - `couchcore/repositoryname.go:103`: `FormatBoundedList` is now the one bounded candidate-list renderer, so the routing copy and its separate constant are gone (ARCH-DRY).
   - `couchcore/repositoryalias.go:39-49`: the alias shadow check now looks in the fleet root, the same place `repositoryPrimary` gives sibling directories precedence, and `TestAliasShadowCheckUsesFleetRoot` pins this.
   - BR-3 has a regression test: the added `subdir` row in `TestApplyRepositoryAliasesMatchesScopeNotPath` would come out as `blog` under the old scope-wide labelling.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - `couchcore/actionableinventory.go:708`: the `:0` row gets its alias only when `filepath.Clean(StartingPath) == filepath.Clean(primary root)`. The primary root comes from git identity (a physical path), while a `:0` record's `StartingPath` comes from the launch cwd. If those spell the same directory differently (a symlink, `/private` on macOS), the alias silently stops showing on the very row the operator's smoke test checked. That smoke test ran before this change. This is the 2nd finding in family `alias-label-scope`. The rule that covers both: a row carries the alias exactly when it stands for that repository's slot or its `:0` checkout, decided by canonical path identity rather than string equality. Fix: canonicalize both sides (EvalSymlinks, as `canonicalFamilyPath` does) and re-smoke the `blog` tab label on pair:0.
   - `couchmessage/routing.go:53-55`: the `canonicalTarget` doc comment still says it rewrites to "a live family" and that the names are "every live binding's family plus its alias"; it now also covers enrolled families.
   - `couchmessage/routing.go:56`: the `known` bool that `canonicalTarget` returns is no longer used (`ResolveRecipient` discards it), so it should be removed.

5. **Test coverage**
   - Routing (BR-1) and labelling (BR-3) behaviour is covered by tests that fail without the fixes; I mutation-checked BR-1.
   - The BR-2 change is error text only. It now names `repository-aliases.json` and how to recover, and needs no test.
   - Pty and `/tmp` tests could not run in this sandbox. The full-suite evidence is the implementor's run at `b8ca068c` (the issue Log).

6. **Architecture**
   - ARCH-DRY: pass (BR-5 consolidated).
   - ARCH-PURE: pass. `ResolveRecipient` and `canonicalTarget` stay pure; the store read sits behind the injected `families` source, outside the broker lock.
   - ARCH-PURPOSE: pass. The Done-when items are delivered. Two pieces of scope beyond the Spec are logged as operator-directed: the in-memory `--actors` listing with bounded liveness re-checks, and removing the exact agent-version allowlist (#368).
   - ARCH-MOCK: pass. Routing tests run against the broker and a fake slot catalog.
   - ARCH-CONSTRAINTS: pass. `--actors` no longer does per-item IO, the alias read in the listing has a 250ms bound, and the routing store read stays inside `AdmissionTimeout`.
   - ARCH-SECURE: pass. The alias file is parsed strictly, an invalid one fails with a recovery hint, and a stored alias that would shadow another repository's name is withheld when read.
   - ARCH-ORDER: pass with a note. The 10s verification window is consulted only on paths that just observe; Reserve and Deliver always re-run the full check, so a stale cached check cannot authorize a delivery.
   - ARCH-FUNERAL: pass. `verified` and `workspaces` entries are dropped when reconciliation finds a binding dead, and alias entries for un-enrolled repositories are dropped on the next write.

7. **Plan revisions:** none needed. The plan already records the round-1 deltas.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      canonicalTarget now resolves over enrolled families too; reverting that loop makes TestRoutingOfflineEnrolledNameNeverPrefixRoutes fail (checked in a scratch copy).
  - id: BR-2
    disposition: addressed
    note: |
      readFamilies error now names repository-aliases.json and the recovery steps (retry, or remove the file to reset aliases); failing closed is kept.
  - id: BR-3
    disposition: addressed
    note: |
      Only slot rows and the :0 row (repository scope starting at the primary root) get the alias; the subdir row in the test pins this.
  - id: BR-4
    disposition: addressed
    note: |
      Shadow check now uses identity.FleetRoot, the same root repositoryPrimary checks; TestAliasShadowCheckUsesFleetRoot covers it.
  - id: BR-5
    disposition: addressed
    note: |
      liveCandidates now calls couchcore.FormatBoundedList; the duplicate loop and maxRouteCandidates are removed.
findings:
  - id: new
    severity: Minor
    family: alias-label-scope
    title: |
      The :0 row gets its alias by path-string equality between StartingPath and the primary root, not canonical identity
    detail: |
      2nd finding in this family. The rule: a row carries the alias exactly when it stands for that repository's slot or :0 checkout, decided by canonical path identity. The primary root is a physical git path, while StartingPath comes from the launch cwd, so a symlink-spelled cwd silently loses the label. EvalSymlinks both sides and re-smoke, since this change came after the operator's smoke test.
  - id: new
    severity: Minor
    family: stale-contract-after-fix
    title: |
      canonicalTarget's doc comment still says "live family" and its known bool return is now unused
    detail: |
      routing.go:53-56 describes live-only resolution after the BR-1 change, and ResolveRecipient discards the second return value; update the comment and drop the return.
```
