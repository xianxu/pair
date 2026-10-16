# Boundary Review — pair#583 (milestone M1)

| field | value |
|-------|-------|
| issue | 583 — Orphaned embedded nvim (22 GB, 13 days) escaped the reaper: untagged 'nvim --embed' children; tag them and sweep orphans |
| repo | pair |
| issue file | workshop/issues/000583-orphaned-embedded-nvim-22-gb-13-days-escaped-the-reaper-untagged-nvim-embed-children-tag-them-and-sweep-orphans.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 758c4b72216ed740e7167f241e8287d2d6db25c6..95db2c14bf53651a4f6d9c7d249480808de89ec0 |
| command | sdlc milestone-close --issue 583 --milestone M1 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-16T13:44:04-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Now the core code.

Tests pass. Now I'm checking whether the live-tag/launchd-reparent rule can actually be reached, with a mutation test.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

M1 delivers most of what it set out to do. Both name-based `pkill -9 -f` kills and the subprocess `kill -9` calls are gone. Every reap now goes through one pure decision (`ClassifyNvimReap`) and one identity-fenced `Signal` path, and the kind set is defined once in `artifactpath.NvimKinds()`. The changelog viewer now has an artifact-keyed pid record, which avoids the (tag, kind) collision. I built the tree and ran tests at the pinned head: `go build ./...`, `go vet` and `go test` all pass for `launcher`, `artifactpath`, `opener` and `procutil`, and `make runtimebundle-generate test-nvim-pidfile` passes all 20 assertions.

One problem should be fixed before the boundary. The plan says an embed whose tag is still live but which has been reparented to launchd must be killed (Task 2, and Task 5 sets `OwnerLive` to `live[tag]`). The code never builds that row for draft, scrollback or review: the sweep drops live tags before deciding. So `reapReasonReparented` only fires for changelog viewers and for test-built targets. Either wire live tags into the sweep or add a plan revision saying the case is out of scope.

1. **Strengths**
   - `nvim_reap.go:71-106`: the decision is pure, and every skip returns a named reason. Argv is matched field by field, never by substring (`argvCarriesField`). Fake tests drive one predicate at a time.
   - `nvim_reap_os.go:283-344`: a pid is decided once from one shared snapshot. The kernel start stamp is re-read just before `SIGKILL`. `Killed` only reports a signal the kernel accepted. A record is kept when evidence is missing and removed once its outcome is decided.
   - `lifecycle.go:410-416`: there is now one derivation (`editorPathsOf` → `nvimSidecarsFor`) instead of two copies pinned to each other. This removes the kind of drift that let `review` leak.
   - `paths.go`: `changelogSiblings`, `ChangelogNvimPIDGlob` and `ChangelogSetFromNvimPID` share one suffix constant. The parser rejects paths outside the scope and `../` tricks, and GC retention covers `.nvim-pid`.
   - `tests/nvim-pidfile-lifecycle-test.sh` checks all four Lua configs for the same four properties, including the guard against deleting a record that was republished. It passes.

2. **Critical:** none.

3. **Important**
   - `nvim_reap_os.go:159` and `:232`: both sweep sources skip live tags (`!live[tag]` / `live[tag] → continue`), and the quit path passes `OwnerLive:false`. In production, no draft, scrollback or review row ever has `OwnerLive:true`. As a result, an embed of a still-live tag that has been reparented to ppid 1 is not reaped until the tag dies. The plan calls this "the closed-pane case the 22 GB orphan is an instance of".
     - The test "a live owner's embed that reparented to launchd" (`nvim_reap_os_test.go:170`) builds its target by hand, so it doesn't exercise what production does.
     - **Fix:** in `deadTagSidecars` and `argvOnlyEmbeds`, also emit rows for live tags with `OwnerLive:true`. The classifier already requires ppid 1 plus the full argv attribution before killing, so this is safe. Add a sweep case where a live tag's embed has ppid 1 and expect one SIGKILL. Alternatively, add a `## Revisions` entry withdrawing the claim. (ARCH-PURPOSE, ARCH-ORDER)

4. **Minor**
   - **Quit path lost its argv fallback.** `ReapNvim` now reads only pid records. A second scrollback viewer whose shared per-tag sidecar was overwritten is no longer reaped at quit. It waits for the next launch sweep after the tag dies, whereas the old pattern kill caught it at quit. This is arguably acceptable given revision (d), but it isn't written down anywhere.
   - **A failed openlock write lets another session's sweep kill a live changelog viewer.** `openerLive` treats an absent `.openlock` as a dead owner, but `opener/run.go` ignores the error from `rt.WriteFile(openlock, …)`. A viewer started after a failed lock write can be killed by any other session's sweep (ARCH-SECURE: missing evidence treated as proof of death). Consider also checking that the viewer's grandparent pid is the opener.
   - **Repeated work (ARCH-DRY):** `byPID` is built twice per sweep (`sweepOrphanNvim:133` and `reapNvimRows:287`), and `Command(pid)` is read twice for each argv-only row (`argvOnlyEmbeds:227` and `decideNvimSidecar:355`). Only a few rows are affected, and it's not on a hot path.
   - **Duplicated Lua:** the `VimLeave` block is copied four times. This was a deliberate choice and is covered by the behavioural test, so it's acceptable.

5. **Test coverage**
   - The pure classifier and the fake-table sweep (15 cases) are thorough on mis-attribution: prefix tags, recycled identity, unreadable argv or identity, a bare embed, two records naming one pid, and malformed records.
   - The live conformance test is env-gated and mutation-proven according to the log. I did not run the gated live test.
   - Gap: no production-path test covers a live tag with a reparented embed (see the Important finding).
   - Gap: no test covers a changelog viewer whose openlock is absent while its opener is alive.

6. **Architecture**
   - **ARCH-DRY:** pass, with the Minor notes above. The kind set is defined once and the kind-to-attribution mapping exists once.
   - **ARCH-PURE:** pass. The decision has no IO, and the IO shell is a thin `ProcessTable` plus `nvimSidecarIO`.
   - **ARCH-PURPOSE:** flag (the Important finding). Everything else in M1 is delivered.
   - **ARCH-MOCK:** pass. `fakeProcessTable` keeps state across calls and records signals, it sits behind the same seam production uses (`OSProcessTable{Source}`), and there is a live conformance check against real nvim.
   - **ARCH-CONSTRAINTS:** pass. Pattern-kill subprocesses are replaced by in-process reads, with one argv read per nvim row only on quit and attach.
   - **ARCH-SECURE:** pass, with one Minor. Pidfiles are parsed into a positive int, and a pid carries no authority without kernel evidence. The exception is the openlock-absence edge above.
   - **ARCH-ORDER:** pass. The window between observation and signal is fenced, and the record rule is stated and tested. M2 should reuse this fence in `reap-orphans`.
   - **ARCH-FUNERAL:** pass. `.nvim-pid` is removed by the reaper, by `VimLeave` and by storage GC, and the review sidecar's end is now named.
   - **For M2:** the `UnattributedCandidate` extraction should keep the same exact-field `--embed` rule as `argvCarriesField`, so the two classifiers can't diverge.

7. **Plan revisions**
   - Either update Task 5 Step 3 to match the code ("live tags are not candidates; `reapReasonReparented` is reached only through changelog records"), or implement the live-tag ppid-1 rows. Pick one.
   - Add a sentence noting that the quit path no longer has an argv fallback for overwritten scrollback sidecars, and that the next launch sweep covers them.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: plan-claim-unreachable-in-production
    title: |
      The rule "live owner but reparented to launchd → kill" is never reached for tag-keyed kinds; the sweep drops live tags
    detail: |
      deadTagSidecars (nvim_reap_os.go:159) and argvOnlyEmbeds (:232) both drop live tags, and the quit path passes OwnerLive:false, so no draft/scrollback/review row ever has OwnerLive:true and reapReasonReparented only fires for changelog viewers and the hand-built test target at nvim_reap_os_test.go:170. Plan Task 2 names this "the closed-pane case" and Task 5 says OwnerLive is live[tag]. Fix: emit live-tag rows with OwnerLive:true (the classifier still requires ppid 1 and full argv attribution) and add a sweep case, or add a plan revision withdrawing the claim.
  - id: new
    severity: Minor
    kind: defect
    family: absent-evidence-read-as-owner-death
    title: |
      A failed openlock write lets another session's sweep reap a live changelog viewer
    detail: |
      openerLive treats an absent .openlock as a dead owner, but opener/run.go ignores the WriteFile error on the openlock. Consider also checking that the viewer's grandparent pid is the opener before killing.
  - id: new
    severity: Minor
    kind: defect
    family: quit-path-coverage-narrowed
    title: |
      Quit-path ReapNvim lost its argv fallback; an overwritten scrollback sidecar's viewer waits for the next launch sweep
    detail: |
      The old pattern kill caught every scrollback viewer of the tag at quit (and over-matched); the new path reads only pid records. Document this in the plan's Non-goals or Revisions.
  - id: new
    severity: Minor
    kind: defect
    family: repeated-per-pass-work
    title: |
      byPID is built twice per sweep and Command(pid) is read twice for each argv-only row
    detail: |
      sweepOrphanNvim:133 and reapNvimRows:287 each build the map, and argvOnlyEmbeds:227 and decideNvimSidecar:355 each read argv. Pass the map through and keep the argv already read.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"495b4eaf-50ee-4ed6-bbf5-df138e05b085","repository":"github.com/xianxu/pair","issue_id":"000583","recipe":"milestone-review","round":1,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/96dbfd4db73a2e74","reviewed_head":"95db2c14bf53651a4f6d9c7d249480808de89ec0","slot":"pair:4","requested_at":"2026-10-16T20:40:18.634276Z"},"phase":"finished","pid":30392,"started_at":"2026-10-16T20:40:18.768151Z","finished_at":"2026-10-16T20:44:03.690059Z","session_id":"26785c50-cd8e-43da-9092-614db9ec4874","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.711956042s
head: "95db2c14bf53651a4f6d9c7d249480808de89ec0"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
