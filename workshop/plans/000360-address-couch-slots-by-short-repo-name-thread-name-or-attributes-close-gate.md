---
gate: boundary-review
issue: 360
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-01T12:03:25-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Critical
          title: --send-to an enrolled-but-offline repo silently prefix-routes to a different live repo
          detail: canonicalTarget builds the names it resolves against from live bindings only, so ResolveRecipient(Route{Target:"brain:0"}) with only brainstorm:0 live returns brainstorm:0 (reproduced at head). Feed enrolled families (already read by messageAuthority.aliases via RepositoryNamesContext) into the resolver's name set so an exact enrolled name misses instead of rerouting; add the regression test.
          family: prefix-resolution-over-narrowed-namespace
          round: 1
        - id: BR-2
          severity: Minor
          title: An unreadable or busy alias store fails every send, including exact-name sends
          detail: Broker.admit returns readAliases errors before resolving. Failing closed is defensible, but the error should name the alias file and how to recover.
          family: display-only-input-blocks-core-path
          round: 1
        - id: BR-3
          severity: Minor
          title: Alias overrides labels of every non-slot row in the repository scope
          detail: ApplyRepositoryAliases sets RepositoryAlias by RepoScope, and Label() returns it ahead of the thread name, so other non-slot threads in the primary checkout all render as the alias.
          family: alias-label-scope
          round: 1
        - id: BR-4
          severity: Minor
          title: Alias shadow check anchors on the primary root's parent, while the resolver checks the caller's fleet root
          family: shadow-check-anchor-mismatch
          round: 1
        - id: BR-5
          severity: Minor
          title: liveCandidates duplicates FormatRepositoryCandidates' bounded-list logic (ARCH-DRY)
          family: bounded-list-duplication
          round: 1
      recipe: milestone-review
      blocked: true
    - "n": 2
      timestamp: "2026-10-01T12:22:10-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: canonicalTarget now resolves over enrolled families too; reverting that loop makes TestRoutingOfflineEnrolledNameNeverPrefixRoutes fail (checked in a scratch copy).
          round: 2
        - id: BR-2
          disposition: addressed
          note: readFamilies error now names repository-aliases.json and the recovery steps (retry, or remove the file to reset aliases); failing closed is kept.
          round: 2
        - id: BR-3
          disposition: addressed
          note: Only slot rows and the :0 row (repository scope starting at the primary root) get the alias; the subdir row in the test pins this.
          round: 2
        - id: BR-4
          disposition: addressed
          note: Shadow check now uses identity.FleetRoot, the same root repositoryPrimary checks; TestAliasShadowCheckUsesFleetRoot covers it.
          round: 2
        - id: BR-5
          disposition: addressed
          note: liveCandidates now calls couchcore.FormatBoundedList; the duplicate loop and maxRouteCandidates are removed.
          round: 2
      findings:
        - id: BR-6
          severity: Minor
          title: The :0 row gets its alias by path-string equality between StartingPath and the primary root, not canonical identity
          detail: '2nd finding in this family. The rule: a row carries the alias exactly when it stands for that repository''s slot or :0 checkout, decided by canonical path identity. The primary root is a physical git path, while StartingPath comes from the launch cwd, so a symlink-spelled cwd silently loses the label. EvalSymlinks both sides and re-smoke, since this change came after the operator''s smoke test.'
          family: alias-label-scope
          round: 2
        - id: BR-7
          severity: Minor
          title: canonicalTarget's doc comment still says "live family" and its known bool return is now unused
          detail: routing.go:53-56 describes live-only resolution after the BR-1 change, and ResolveRecipient discards the second return value; update the comment and drop the return.
          family: stale-contract-after-fix
          round: 2
      recipe: milestone-review
      blocked: false
---

# Gate ledger — pair#360 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-01T12:03:25-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Critical] `prefix-resolution-over-narrowed-namespace` --send-to an enrolled-but-offline repo silently prefix-routes to a different live repo
  canonicalTarget builds the names it resolves against from live bindings only, so ResolveRecipient(Route{Target:"brain:0"}) with only brainstorm:0 live returns brainstorm:0 (reproduced at head). Feed enrolled families (already read by messageAuthority.aliases via RepositoryNamesContext) into the resolver's name set so an exact enrolled name misses instead of rerouting; add the regression test.
- **BR-2** [Minor] `display-only-input-blocks-core-path` An unreadable or busy alias store fails every send, including exact-name sends
  Broker.admit returns readAliases errors before resolving. Failing closed is defensible, but the error should name the alias file and how to recover.
- **BR-3** [Minor] `alias-label-scope` Alias overrides labels of every non-slot row in the repository scope
  ApplyRepositoryAliases sets RepositoryAlias by RepoScope, and Label() returns it ahead of the thread name, so other non-slot threads in the primary checkout all render as the alias.
- **BR-4** [Minor] `shadow-check-anchor-mismatch` Alias shadow check anchors on the primary root's parent, while the resolver checks the caller's fleet root
- **BR-5** [Minor] `bounded-list-duplication` liveCandidates duplicates FormatRepositoryCandidates' bounded-list logic (ARCH-DRY)

## Round 2 — 2026-10-01T12:22:10-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — canonicalTarget now resolves over enrolled families too; reverting that loop makes TestRoutingOfflineEnrolledNameNeverPrefixRoutes fail (checked in a scratch copy).
- BR-2 — addressed — readFamilies error now names repository-aliases.json and the recovery steps (retry, or remove the file to reset aliases); failing closed is kept.
- BR-3 — addressed — Only slot rows and the :0 row (repository scope starting at the primary root) get the alias; the subdir row in the test pins this.
- BR-4 — addressed — Shadow check now uses identity.FleetRoot, the same root repositoryPrimary checks; TestAliasShadowCheckUsesFleetRoot covers it.
- BR-5 — addressed — liveCandidates now calls couchcore.FormatBoundedList; the duplicate loop and maxRouteCandidates are removed.

### Raised

- **BR-6** [Minor] `alias-label-scope` The :0 row gets its alias by path-string equality between StartingPath and the primary root, not canonical identity
  2nd finding in this family. The rule: a row carries the alias exactly when it stands for that repository's slot or :0 checkout, decided by canonical path identity. The primary root is a physical git path, while StartingPath comes from the launch cwd, so a symlink-spelled cwd silently loses the label. EvalSymlinks both sides and re-smoke, since this change came after the operator's smoke test.
- **BR-7** [Minor] `stale-contract-after-fix` canonicalTarget's doc comment still says "live family" and its known bool return is now unused
  routing.go:53-56 describes live-only resolution after the BR-1 change, and ResolveRecipient discards the second return value; update the comment and drop the return.

## Open findings

- **BR-6** [Minor] `alias-label-scope` The :0 row gets its alias by path-string equality between StartingPath and the primary root, not canonical identity
- **BR-7** [Minor] `stale-contract-after-fix` canonicalTarget's doc comment still says "live family" and its known bool return is now unused
