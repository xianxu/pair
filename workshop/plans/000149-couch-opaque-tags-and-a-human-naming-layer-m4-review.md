# Boundary Review — pair#149 (milestone M4)

| field | value |
|-------|-------|
| issue | 149 — couch: opaque tags and a human naming layer |
| repo | pair |
| issue file | workshop/issues/000149-couch-opaque-tags-and-a-human-naming-layer.md |
| boundary | milestone M4 |
| milestone | M4 |
| window | 328551b19f7d3eb2d2b0c1e7ab6c34e1f3509e4c..344214a101b234c2996bc4b4dce8c666cd272b2e |
| command | sdlc milestone-close --issue 149 --milestone M4 |
| reviewer | codex |
| timestamp | 2026-08-26T17:51:53-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The launch-profile implementation is coherent and the test suite passes, but M4 cannot cross the boundary yet. The plan’s Core concepts inventory no longer describes the implemented architecture—a repeated `core-concept-kind-contract` failure explicitly classified as Critical by the review contract. I also found a user-facing empty-value bug where `--agent=` silently launches the fallback agent.

## 1. Strengths

- Agent and argv precedence are independently modeled and defensively copied in [launchprofile.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/launchprofile.go:59).
- The Couch-to-Pair handoff is strict, tag-bound, rejects unknown fields/agents, and cross-checks provenance against `PAIR_USE_REPO_DEFAULT` in [launch_args_policy.go](/Users/xianxu/workspace/pair/cmd/internal/launcher/launch_args_policy.go:57) and [runcli.go](/Users/xianxu/workspace/pair/cmd/internal/launcher/runcli.go:155).
- Registration atomically publishes the thread profile, path preference, and manifest generation through the existing recoverable journal in [threadstore.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/threadstore.go:397).
- `AgentInventory` replaces the duplicated rename-agent enumeration and returns a defensive copy.
- README and atlas document the new command, precedence, persistence, and failure semantics.

## 2. Critical findings

- [Plan Core concepts table](/Users/xianxu/workspace/pair/workshop/plans/000149-couch-opaque-tags-and-a-human-naming-layer-plan.md:37) — **Fourth finding in family `core-concept-kind-contract` (ARCH-PURE, ARCH-PURPOSE).** The table lists only `LaunchProfileResolution` for M4, while M4 also introduces the durable `LaunchProfile` and `PathLaunchPreference`, widens `ThreadRecord` and `threadrecord.Record` despite their statuses ending at M3, and introduces the strict Couch launch-profile wire integration without an integration row. Do not patch only one row: establish and apply the rule that every milestone-added or milestone-modified architectural entity appears with its kind, path, and current status, then sweep the complete M4 diff.

## 3. Important findings

- [run.go:400](/Users/xianxu/workspace/pair/cmd/internal/couchcmd/run.go:400), [operationdispatch.go:175](/Users/xianxu/workspace/pair/cmd/internal/couchcore/operationdispatch.go:175), [launchprofile.go:61](/Users/xianxu/workspace/pair/cmd/internal/couchcore/launchprofile.go:61) — `--agent=` is recorded as a supplied empty value, but the executor loses suppliedness and `ResolveLaunchProfile` treats it as absent. Couch consequently launches the path/root fallback instead of refusing the malformed explicit selection. A bare `--agent` similarly becomes the synthetic agent `"true"`. Give value-bearing flags an explicit schema contract, reject missing/empty values before `Spawn`, and add CLI plus generic-dispatch regressions.

## 4. Minor findings

None.

## 5. Test coverage notes

`git diff --check`, the four focused packages, and `go test ./... -count=1` all passed.

Coverage is strong for precedence, cross-agent isolation, defensive copying, strict wire decoding, failure-before-registration, atomic publication, and journal recovery. No prior-round fixes required disposition. The missing regression is the actual public `--agent` path, particularly `--agent=` and bare `--agent`.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — pass:** agent inventory is centralized and the rename consumer derives from it.
- **ARCH-PURE — pass in code, tracker flag:** resolution and start transitions remain pure; filesystem and process work stay in injected/store boundaries. The plan must accurately inventory those entities.
- **ARCH-PURPOSE — flag:** runtime behavior delivers remembered per-path/per-agent profiles, but the architectural source of truth does not enumerate the full delivered surface.
- **ARCH-MOCK — pass:** no new external service or binary dependency was introduced; the Pair-owned state uses the portable data directory and production/test flows share the existing seams.

## 7. Plan revision recommendations

Append, rather than overwrite, an entry such as:

> ### 2026-08-26 — align the M4 architectural inventory with implementation  
> **Reason:** M4 introduced additional durable entities and a strict launcher integration beyond the original Core concepts rows and task file lists.  
> **Delta:** enumerate `LaunchProfile`, `PathLaunchPreference`, `LaunchProfileResolution`, the Couch launch-profile wire codec, and the modified `ThreadRecord`, `threadrecord.Record`, `ThreadStore`, and agent-default integration with accurate PURE/INTEGRATION kinds, paths, and M4 statuses. Update M4 task file inventories accordingly.

```findings
findings:
  - id: new
    severity: Critical
    family: core-concept-kind-contract
    title: |
      M4's Core concepts inventory omits and misstates implemented architectural entities
    detail: |
      This is the 4th finding in family `core-concept-kind-contract`. Earlier rounds fixed instances. Do NOT fix only one row: state the rule that every milestone-added or modified architectural entity must have a greppable row with its correct kind, path, and current status, then sweep the full M4 diff. The table at plan line 37 omits LaunchProfile, PathLaunchPreference, and the strict Couch profile-wire integration, while the ThreadRecord and threadrecord.Record statuses do not acknowledge their M4 widening (ARCH-PURE, ARCH-PURPOSE).
  - id: new
    severity: Important
    family: value-bearing-flag-contract
    title: |
      An explicitly empty agent selection silently launches the fallback agent
    detail: |
      bindArgs preserves `--agent=` as a present empty string at couchcmd/run.go:400, but CouchLiveOwnerExecutor passes only the value and ResolveLaunchProfile treats empty as no explicit selection. Reject missing or empty values for value-bearing flags before Spawn, distinguish them from boolean switches, and add public CLI plus generic-dispatch tests that fail without the fix.
```
