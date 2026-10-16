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

---

## Re-review — 2026-10-16T14:07:24-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 583 — Orphaned embedded nvim (22 GB, 13 days) escaped the reaper: untagged 'nvim --embed' children; tag them and sweep orphans |
| repo | pair |
| issue file | workshop/issues/000583-orphaned-embedded-nvim-22-gb-13-days-escaped-the-reaper-untagged-nvim-embed-children-tag-them-and-sweep-orphans.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 758c4b72216ed740e7167f241e8287d2d6db25c6..42045692d9a2a272484f7ee9af80f882035ab67a |
| command | sdlc milestone-close --issue 583 --milestone M1 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-16T14:07:24-07:00 |
| verdict | REWORK |

## Review

Let me look at the fix commits for prior findings first.

Tests mostly pass, but the BR-2 fix broke an artifact-classification test. I'm checking whether that failure comes from the fix itself, and whether the BR-1 test fails without its fix.

```verdict
verdict: REWORK
confidence: high
```

The four prior findings are all handled. BR-1 and BR-2 each have a regression test, and I confirmed both tests fail when their fix is reverted. BR-3 is now recorded in the plan's Non-goals. BR-4 builds `byPID` once per pass. It still reads argv a second time, but there is a reason for that, and the reason is written at the read. One thing blocks SHIP. The BR-2 fix (`bff93e1e9`) broke a repo-wide guard test, so the suite is red at the pinned head. `go test ./cmd/internal/artifactpath/` fails `TestProductionArtifactReferencesAreExactlyClassified` because the new stderr string `"pair-changelog-open: openlock: %v\n"` (`cmd/internal/opener/run.go:194`) uses unlisted changelog artifact vocabulary. With `run.go` set back to its state before `bff93e1e9`, the same test passes. The rest of the work is sound. The launcher, opener and procutil packages pass.

1. **Strengths**
   - The BR-1 fix keeps one guard in one place. `tagSidecars`/`argvOnlyEmbeds` (`nvim_reap_os.go:230,271`) now pass liveness along as evidence instead of filtering on it. The kill rule for a live tag still sits in `ClassifyNvimReap` (`nvim_reap.go:110-115`): ppid 1, plus full attribution, plus the identity fence. I reverted the two filters and the two new sweep cases went red.
   - The neighbouring test cases were updated so each one tests a single thing. Setting a live parent on the "live tag's draft embed" and "found through a dead tag's record" cases (`nvim_reap_os_test.go:520-567`) stops them from passing by accident under the new rule.
   - BR-2 is fixed where it starts, in the opener, rather than by adding a second ownership check. `TestRunChangelogRefusesAViewerItCannotClaim` checks four things: exit code, that no viewer opens, that no distiller launches, and that the lock is not removed. With the fix reverted it goes red: `code = 0, want 1`.
   - The plan's Revisions entry is precise. It also corrects the earlier claim that the 22 GB orphan was an instance of the closed-pane row.

2. **Critical**
   - `cmd/internal/opener/run.go:194`: the new error message breaks `TestProductionArtifactReferencesAreExactlyClassified` (`artifactpath/coverage_test.go:205`). The fix is to reword the message so it doesn't use the artifact vocabulary, e.g. `"pair-changelog-open: claim viewer: %v\n"`. Don't add it to `known-artifact-violations.txt`; the test itself says not to. Then run `go test ./cmd/internal/...` in full.

3. **Important:** none.

4. **Minor**
   - A live tag's embed at ppid 1 could be killed during the ~31 ms when it is exiting on its own after stdin EOF. This is harmless as far as I can see; at most a record is left behind, and the next pass removes it as `gone`.

5. **Test coverage**
   - The reparented branch is covered by fake-table rows, both through a record and through argv alone. The live test still does not do a real reparenting; the plan records this as a host limitation, which is acceptable.
   - The fix commits seem to have been checked with targeted test runs only. A full run of the affected package would have caught the Critical finding.

6. **Architecture**
   - **ARCH-DRY: pass.** `indexByPID` has a single construction site, and the liveness value is passed through `nvimSidecarsFor`.
   - **ARCH-PURE: pass.** `ClassifyNvimReap` is pure; the IO stays in the `ProcessTable` and `nvimSidecarIO` seams.
   - **ARCH-PURPOSE: pass.** The closed-pane case can now be reached in production for all three tag-keyed kinds.
   - **ARCH-MOCK: pass.** The fake process table sits behind the same seam as production, and there is a gated live test.
   - **ARCH-CONSTRAINTS: pass.** The added cost is about one record read per live tag per kind, on the launch path only.
   - **ARCH-SECURE: pass.** The openlock write is now checked, so its absence is real evidence that the owner is gone.
   - **ARCH-ORDER: pass.** Each pid gets one decision per snapshot, and the identity is re-read before the signal.
   - **ARCH-FUNERAL: pass.** A live tag's record is now removed by the sweep when it names a gone or foreign pid, and kept on `owner live`.

7. **Plan revisions:** none needed beyond a one-line note in the Revisions entry once the stderr string is reworded.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      tagSidecars/argvOnlyEmbeds now pass live[tag]; reverting the two filters turns the two new reparented sweep cases red.
  - id: BR-2
    disposition: addressed
    note: |
      The openlock write is checked (run.go:197) and TestRunChangelogRefusesAViewerItCannotClaim goes red (code = 0, want 1) on revert; the regression it introduced is raised separately below.
  - id: BR-3
    disposition: addressed
    note: |
      Recorded in the plan Non-goals at :551-555; the prose matches ReapNvim, which reads pid records only (osruntime.go:946).
  - id: BR-4
    disposition: addressed
    note: |
      byPID is built once (indexByPID); the second argv read is justified as authorizing evidence read beside the identity fence, with the reason commented at nvim_reap_os.go:336.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: fix-verified-by-targeted-test-only
    title: |
      The BR-2 fix's stderr string breaks TestProductionArtifactReferencesAreExactlyClassified; the suite is red at head
    detail: |
      cmd/internal/opener/run.go:194 adds "pair-changelog-open: openlock: %v\n", which artifactpath/coverage_test.go:205 rejects as unlisted changelog vocabulary; the test passes with run.go reverted to bff93e1e9^. Reword the message without the artifact token, e.g. "claim viewer", and run the full cmd/internal suite. The rule: a fix commit is verified against every package's tests, not only the new targeted test.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"1e65fd4f-c96f-4bbe-8e52-f3d6737efd28","repository":"github.com/xianxu/pair","issue_id":"000583","recipe":"milestone-review","round":2,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/e3c9306e50bcc941","reviewed_head":"42045692d9a2a272484f7ee9af80f882035ab67a","slot":"pair:4","requested_at":"2026-10-16T21:05:16.860739Z"},"phase":"finished","pid":92494,"started_at":"2026-10-16T21:05:16.999741Z","finished_at":"2026-10-16T21:07:23.811937Z","session_id":"bb63606c-29e5-442f-892d-59ec33dd8347","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.695676416s
head: "42045692d9a2a272484f7ee9af80f882035ab67a"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-16T14:36:49-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 583 — Orphaned embedded nvim (22 GB, 13 days) escaped the reaper: untagged 'nvim --embed' children; tag them and sweep orphans |
| repo | pair |
| issue file | workshop/issues/000583-orphaned-embedded-nvim-22-gb-13-days-escaped-the-reaper-untagged-nvim-embed-children-tag-them-and-sweep-orphans.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 758c4b72216ed740e7167f241e8287d2d6db25c6..283e2314a2baae3ace0e1e981c1f034e2fd5d4cc |
| command | sdlc milestone-close --issue 583 --milestone M1 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-16T14:36:49-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

First, BR-5: check the last commit and run the tests.

BR-5 checks out: the coverage test goes red with the fix removed and green with it in place. Next I'm reviewing the reaper code itself against the plan.

Checking where the review pane's pid record gets written, and whether the classifier stays fast on the quit path.

Found one likely gap: a stale review-pane record could kill another tag's live review pane after pid reuse. Running the shell lifecycle test, then writing up.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

BR-5 is fixed. At head, `go test ./cmd/internal/...` passes in every package the diff touches. The one failure is `broadcast/TestNamedTunnelServesFromPrivateSocket`. That package is outside the diff, and the same test fails the same way at base `758c4b72`, so it comes from this host, not this change. The fix is also confirmed by reversal: with only `manifest.go` put back to its `283e2314a^` state, `TestProductionArtifactReferencesAreExactlyClassified` goes red on `"pair-changelog-open: write %s: %v\n"`, and it goes green again at head. The four sidecar configs also pass `tests/nvim-pidfile-lifecycle-test.sh` end to end against real nvim.

The reap design holds up: one closed kind set, a pure classifier and an identity fence just before the signal. One new correctness gap remains. The `review` kind's argv check proves "some pair review pane", not "this tag's review pane". A pid record holds only a pid, with no start stamp. So a stale review record from a dead tag can authorize SIGKILL of another live tag's review pane if the pid is reused. It is cheap to fix and should be closed before crossing the boundary.

1. **Strengths**
   - `ClassifyNvimReap` (`cmd/internal/launcher/nvim_reap.go:72`) is truly pure. Every skip returns a named reason, and the argv checks compare whole fields, not substrings. `TestNvimSidecarsGiveEveryKindArgvEvidence` stops any new kind that arrives without argv evidence.
   - `NvimKinds()` (`artifactpath/paths.go:290`) is now the single source. GC, the rename inventory, the quit path, the reaper and the sweep all derive from it, and the hand-written `{draft, scrollback}` lists are gone.
   - Every name-based kill is gone (`pkill -f`, `kill -9 <pid>` through a subprocess). The only signal left is `syscall.Kill` to a pid whose start stamp is re-read just before the kill (`nvim_reap_os.go:368-371`).
   - The VimLeave cleanup deletes a record only if it still names this process. The shell test covers that rule for all four configs, including "a later embed republished the record".
   - BR-2's refusal (`opener/run.go:192`) makes the openlock's absence trustworthy as evidence that the opener is dead.

2. **Critical findings**: none.

3. **Important findings**
   - `cmd/internal/launcher/nvim_reap_os.go:101` (`ExpectedAssetBase` for review) and `nvim_reap.go:107`:
     - **What's wrong:** the review kind's check is "argv carries `review.lua`". Every tag's review pane has that, so it doesn't tie the process to the record's owner. The record holds a bare pid with no stored identity. The identity fence only catches reuse *between the snapshot and the signal*, not between the record write and the snapshot.
     - **How it fails:** tag A's review embed dies without reaching VimLeave (crash or reboot), so `nvim-pid-A-review` survives. Its pid is later reused by tag B's live review embed. The next sweep treats A as dead, all checks pass, and tag B's live pane is SIGKILLed.
     - **Why the plan doesn't cover it:** the plan's ARCH-ORDER paragraph (plan §State, lines 122-126) says the predicates make a recycled pid safe. That holds for draft (exact path), scrollback (tag parse) and changelog (exact log path), but not for review.
     - **Missing test:** no test covers "dead tag's review record at a pid that is another live tag's review embed".
     - **Fix sketch, either of:**
       - (a) attribute review to its owner by checking that argv carries the file named by the tag's own review state (`ReviewTarget`/`ReviewOpen`);
       - (b) write the start stamp into the record and require it to match the snapshot.

     Add the regression test either way. (ARCH-SECURE, ARCH-ORDER)

4. **Minor findings**
   - ARCH-DRY: the same VimLeave "remove if it still names me" block is pasted into `nvim/init.lua`, `scrollback.lua`, `review.lua` and `changelog.lua`. The VimEnter writer was already duplicated the same way. A small shared Lua helper would replace both.
   - ARCH-DRY: `artifactpath/gc.go:265` hand-lists the changelog suffixes, now including `.nvim-pid`, in parallel with `changelogSiblings`. Deriving the list from the set would stop the two drifting apart.
   - Environment: the `broadcast` named-tunnel test fails on this host at both base and head, so "whole suite green" claims need that caveat.

5. **Test coverage notes**
   - Classifier tests change one field at a time and assert reasons that are written down independently.
   - Sweep tests run the production `reapNvimRows` against an injected `ProcessTable`. The record-backed vs. argv-only dedup and the reparented rule both have cases.
   - The missing case is the review-kind cross-tag reuse above.
   - The live conformance test skips where nvim is not available.

6. **Architectural notes (per principle)**
   - ARCH-DRY: pass for the Go side; Lua duplication is noted as Minor.
   - ARCH-PURE: pass. The decision is pure, and the process table and the record files are injected.
   - ARCH-PURPOSE: pass for M1. Every tag-keyed kind and the changelog viewer now leave a record and get reaped. Untagged orphans are M2, as the plan says.
   - ARCH-MOCK: pass. A fake `ProcessTable`/`procutil.Source` sits behind the same seam production uses, and a live nvim conformance test exists.
   - ARCH-CONSTRAINTS: pass. One snapshot per pass, one argv read per candidate, and argv reads limited to rows named nvim.
   - ARCH-SECURE: **flag**. The review attribution doesn't name the owner (Important finding above).
   - ARCH-ORDER: **flag**, same root cause. The "pid recycled between record write and reap" event is handled for three of the four kinds but not for review.
   - ARCH-FUNERAL: pass. The new `changelog-*.nvim-pid` record is removed by a clean exit, by the reap, and by GC (it was added to the changelog family in `gc.go`).
   - For M2: `UnattributedCandidate` should keep `argvCarriesField`'s exact-field `--embed` rule.

7. **Plan revision recommendations**
   - In §State, ordering and uncertainty, add a `## Revisions` entry stating that a recycled pid is safe only for kinds whose argv names the owner. Then state the review kind's owner check once the fix lands.

```findings
dispose:
  - id: BR-5
    disposition: addressed
    note: |
      manifest.go lists "pair-changelog-open: write %s: %v\n"; reverting manifest.go to 283e2314a^ turns TestProductionArtifactReferencesAreExactlyClassified red, and it is green at head; every touched package passes.
findings:
  - id: new
    severity: Important
    kind: defect
    family: kill-attribution-must-name-owner
    title: |
      The review kind's argv check matches any tag's review pane, so a stale review record can kill another live tag's pane after pid reuse
    detail: |
      nvim_reap_os.go:101 sets ExpectedAssetBase "review.lua", which every pair review embed carries. Pid records store no start stamp, so the identity fence (nvim_reap_os.go:368) only catches reuse after the snapshot. Failure: a dead tag's surviving nvim-pid-A-review names a pid that tag B's live review embed now holds; the sweep reads owner A as dead, every check passes, and B's pane is SIGKILLed. Fix: attribute review to its owner (argv must carry the file in the tag's own review state), or store and compare the start stamp in the record. Add a cross-tag reuse regression test.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"c21e99db-6c8a-4925-be27-23b1ffc2ca14","repository":"github.com/xianxu/pair","issue_id":"000583","recipe":"milestone-review","round":3,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/afeb1ecc6a482a45","reviewed_head":"283e2314a2baae3ace0e1e981c1f034e2fd5d4cc","slot":"pair:4","requested_at":"2026-10-16T21:20:31.703494Z"},"phase":"finished","pid":31489,"started_at":"2026-10-16T21:20:31.889529Z","finished_at":"2026-10-16T21:36:48.883475Z","session_id":"85886e49-ec5a-4130-861f-c0df95e294b7","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.689316625s
head: "283e2314a2baae3ace0e1e981c1f034e2fd5d4cc"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-16T15:21:56-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 583 — Orphaned embedded nvim (22 GB, 13 days) escaped the reaper: untagged 'nvim --embed' children; tag them and sweep orphans |
| repo | pair |
| issue file | workshop/issues/000583-orphaned-embedded-nvim-22-gb-13-days-escaped-the-reaper-untagged-nvim-embed-children-tag-them-and-sweep-orphans.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 758c4b72216ed740e7167f241e8287d2d6db25c6..b2c4283f9abffcd28acd487636e04583c9af7b8b |
| command | sdlc milestone-close --issue 583 --milestone M1 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-16T15:21:56-07:00 |
| verdict | REWORK |

## Review

The main focus: verify BR-6 fix.

Checking whether the BR-6 fix holds up when the review pane is retargeted to another file.

I'm adding a temporary test case to confirm the retarget leak empirically.

```verdict
verdict: REWORK
confidence: high
```

The BR-6 fix stops one tag's stale review record from killing another tag's live pane, as long as the two panes serve different files. It does this by making attribution come from a field that names the owner, at `cmd/internal/launcher/nvim_reap.go:103-121`. The suites for `launcher`, `reviewcmd`, `artifactpath` and `opener` are green at head. The problem is the evidence the fix chose. Review's owner-naming field is read from `review-target-<tag>.json`, and that file changes during the life of a review pane. The pane's argv keeps the file it was spawned with, but the target is rewritten in two places:
- Alt+C activation of a live pane on another document (`nvim/review/restore_client.lua:137`, `write_target(wanted.repo..'/'..wanted.file, …)`).
- `:PairReview` proposing a new file while the pane is alive (`nvim/init.lua` `propose` → `write_target(abs,'proposed')`).

Once that happens, the pane's own orphaned embed is classified `argv serves another path`. That is a decided negative: `reapKeepsRecord` returns false, so the record is deleted and nothing is signalled. This happens both on the quit path (`osruntime.go:946` goes through the same `decideOne` → `reviewAttribution`) and in the sweep. Review embeds have no argv-only fallback, so no later pass finds that pid again. This is the 13-day orphan the issue exists to prevent, and the fix brings it back for review panes. That blocks SHIP.

1. **Strengths**
   - `ClassifyNvimReap` now treats "no owner-naming attribution" as its own skip reason (`attribution unknown`) and keeps the record for it (`nvim_reap_os.go:445`). It also correctly demotes the asset base to corroboration only.
   - Rejecting a start stamp in the record is justified and written down (darwin uses wall clock, linux uses ticks since boot). That is measured evidence, not hand-waving.
   - The sweep table test (`nvim_reap_os_test.go:~655`) pins the cross-tag case with a live parent. Without the fix this test fails: before it, the asset base alone plus a dead owner meant KILL.
   - `TargetFileOf` is a small pure parser with its own tests (`reviewcmd_test.go:39-50`). On bad input it reports missing evidence rather than inventing a value (ARCH-SECURE pass).

2. **Critical findings**
   - **The review pane's own orphan escapes after a retarget, and its record is deleted** (`nvim_reap_os.go:358`, `reviewAttribution`; `nvim_reap_os.go:111`).
     - **Scenario:** a pane is spawned on `/src/post.md` (argv). The user activates it on `/src/second.md`, so the target now names `second.md`. The session quits, or the pane's parent dies and the embed reparents to launchd. The classifier returns `argv serves another path` before it ever checks owner-dead or reparented. The record is removed and the embed stays alive.
     - **Why the tests miss it:** this is the BR-6 test row with ppid 1 instead of 4000. The classifier cannot tell the two apart because the evidence is mutable.
     - **Family:** this is the 2nd finding in family `kill-attribution-must-name-owner`. Earlier rounds fixed instances. The rule covering both: *attribution evidence must be bound to the pid at the moment the record is written, and must not change for the pid's lifetime.* Mutable session state may corroborate, but a mismatch against it must keep the record, never drop it.
     - **Fix that rule:** have `review.lua` write the file it serves next to the pid when it publishes the record (`nvim/review.lua:809-810`). For example, line 2 = the activation's argv file, or `vim.v.argv`'s file. The reaper then compares process argv with the record's own line, not with `review-target`. A record with no line 2 (legacy) is `attribution unknown` and is kept.
     - **Regression rows to add:** a reparented embed after a retarget must be KILLed, and a cross-tag live pane with a different file must still be skipped.
     - **Residual to document in the plan:** a dead tag's record and another tag's live pane at the same recycled pid, serving the same file, still kill. That needs pid reuse plus the same file.
     - **Not verified by running:** I could not run this scenario as a test because file edits were denied in this checkout. The conclusion comes from reading the code path: the classifier's ordering at `nvim_reap.go:103-121` and `reapKeepsRecord`.

3. **Important findings:** none new.

4. **Minor findings**
   - The review-target's `File` comes from `rt.AbsFile` / `gi.abs`, while argv comes from `LogicalDir+base`. These match on the normal open path, but nothing guarantees they are spelled the same. The record-bound fix above removes this dependency too.

5. **Test coverage notes:** the review rows cover a matching target, a target with a different file plus a live parent, and a target with no file. There is no row where the tag's *own* embed has a target that has drifted. That gap hid the regression. Tests pin real logic through the production `decideOne`/`ClassifyNvimReap` path against `fakeProcessTable`; nothing is mock-reasserted.

6. **Architectural notes**
   - **ARCH-DRY:** pass. The four lua VimLeave copies were disposed with a behavioural lifecycle test, which is acceptable.
   - **ARCH-PURE:** pass. The classifier is pure. `reviewAttribution` is a thin IO read.
   - **ARCH-PURPOSE:** flag. The fix narrows over-kill but reopens the leak the issue exists to close (the Critical finding above).
   - **ARCH-MOCK:** pass. There is a stateful process-table fake plus the live conformance test.
   - **ARCH-CONSTRAINTS:** pass. Each target adds one small file read.
   - **ARCH-SECURE:** pass. The target JSON is parsed into a typed value, and failure means skip.
   - **ARCH-ORDER:** flag, folded into the Critical finding. The evidence (target) and the subject (pid record) are written at different times by different actors, and the code assumes they stay in step. That is an unwritten interleaving.
   - **ARCH-FUNERAL:** flag, folded into the Critical finding. Deleting the record on a drifted-evidence negative leaves an embed with no removal path.

7. **Plan revision recommendations**
   - Add a `## Revisions` entry: review attribution is the record's own co-written file line, not `review-target`. A drift between the target and argv is missing evidence, and the record is kept.
   - Record the remaining same-file cross-tag pid-reuse residual and why it is accepted.

```findings
dispose:
  - id: BR-6
    disposition: addressed
    note: |
      Cross-tag kill with distinct files now skips. The sweep row with a live parent fails without the fix (asset base plus dead owner used to mean KILL). The remaining same-file pid-reuse case should be recorded in the plan's Revisions.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: kill-attribution-must-name-owner
    title: |
      Review attribution reads a mutable review-target, so a retargeted pane's own orphan is never killed and its record is deleted
    detail: |
      nvim_reap_os.go:358 (reviewAttribution) sets ExpectedArgPath from review-target-<tag>.json, but restore_client.lua:137 (Alt+C activation of a live pane) and init.lua propose (:PairReview) both rewrite that file while the pane's argv still names its spawn file. A reparented or quit-path embed therefore classifies "argv serves another path", reapKeepsRecord drops the record, and no later pass can find the pid (review has no argv-only fallback). That is the issue's orphan class, reopened. This is the 2nd finding in this family. The rule: attribution evidence must be bound to the pid when the record is written and must not change for its lifetime; a mismatch against mutable state keeps the record and never drops it. Fix: review.lua writes its served file into the pid record (line 2) at publish, the reaper compares process argv with that line, and a legacy one-line record is attribution unknown (kept). Add sweep and quit rows: a reparented embed after a retarget must be KILLed; a cross-tag live pane with a different file must be skipped.
```

{"type":"system","subtype":"permission_denied","tool_name":"Edit","tool_use_id":"toolu_01MCxnewDokD12xEhLZxfMZf","decision_reason_type":"mode","message":"Permission to use Edit has been denied because Claude Code is running in don't ask mode. IMPORTANT: You *may* attempt to accomplish this action using other tools that might naturally be used to accomplish this goal, e.g. using head instead of cat. But you *should not* attempt to work around this denial in malicious ways, e.g. do not use your ability to run tests to execute non-test actions. You should only try to work around this restriction in reasonable ways that do not attempt to bypass the intent behind this denial. If you believe this capability is essential to complete the user's request, STOP and explain to the user what you were trying to do and why you need this permission. Let the user decide how to proceed.","uuid":"6d138210-1a98-463c-9b29-f30fddff1684","session_id":"1f29e64f-c400-4d16-9fee-663a1cb3e426"}

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"621c58ec-0a1a-49db-9776-37749417dd44","repository":"github.com/xianxu/pair","issue_id":"000583","recipe":"milestone-review","round":4,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/4598158c594d57c4","reviewed_head":"b2c4283f9abffcd28acd487636e04583c9af7b8b","slot":"pair:4","requested_at":"2026-10-16T22:18:45.001104Z"},"phase":"finished","pid":50387,"started_at":"2026-10-16T22:18:45.786389Z","finished_at":"2026-10-16T22:21:52.184304Z","session_id":"1f29e64f-c400-4d16-9fee-663a1cb3e426","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 6.295985292s
head: "b2c4283f9abffcd28acd487636e04583c9af7b8b"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-16T16:18:07-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 583 — Orphaned embedded nvim (22 GB, 13 days) escaped the reaper: untagged 'nvim --embed' children; tag them and sweep orphans |
| repo | pair |
| issue file | workshop/issues/000583-orphaned-embedded-nvim-22-gb-13-days-escaped-the-reaper-untagged-nvim-embed-children-tag-them-and-sweep-orphans.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 758c4b72216ed740e7167f241e8287d2d6db25c6..25ae5ed5385381dde0c9cab7e788e147bf5f0b8a |
| command | sdlc milestone-close --issue 583 --milestone M1 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-16T16:18:07-07:00 |
| verdict | SHIP |

## Review

Focus on the BR-7 fix first.

I'm checking whether every production review pane spawn puts the served file last in argv, since the BR-7 fix relies on that.

Go tests for the reaper packages pass; the broadcast failure is outside this diff. Next, I'll run the nvim lifecycle test and revert the fix to confirm its regression tests fail without it.

```verdict
verdict: SHIP
confidence: high
```

The BR-7 fix holds, and nothing new needs to block M1. `review.lua` now writes the file the pane was spawned with into line 2 of its pid record, taking it from its own argv (`nvim/review.lua:814-818`). The reaper reads that line next to the pid (`sidecarRecord`, `cmd/internal/launcher/nvim_reap_os.go:450`) and fills `ExpectedArgPath` from it (`nvim_reap_os.go:330-337`). Nothing reads the mutable review-target state any more: `reviewAttribution` is deleted and so is `reviewcmd.TargetFileOf`. I reverted the fix in the disposable checkout and 9 rows went red across the reap and sweep tables. They include all three retarget rows and the cross-tag "another tag's live pane holds the pid" row; the rest are review attribution rows that need the bound file. The rule BR-7 asked for now holds for every kind: attribution is fixed when the record is written, and a record that names no file is kept, not dropped. The draft path comes from the scope's paths, scrollback is attributed by the scope's argv parser, and changelog is keyed to its artifact. The changelog openlock is read only to decide whether to keep a record, so it can never cause a kill. No sibling in this family still reads mutable state.

1. **Strengths**
   - The publisher takes line 2 from its own `vim.v.argv`, so it cannot drift from the process it names. A mismatch therefore really does mean "this pid is not ours" (`nvim/review.lua:816-817`).
   - A one-line legacy record, or an empty line 2, is treated as unknown attribution and kept. It never falls into the branch that deletes (`nvim_reap_os_test.go:275`, `:728`).
   - The quit-path and sweep rows for retargeted panes write the mutable state as a decoy fixture (`reviewTargetState`, `nvim_reap_os_test.go:42-48`), which proves no decision reads it.
   - The real-nvim lifecycle test pins line 2 byte for byte for review and checks that the other kinds write only one line (`tests/nvim-pidfile-lifecycle-test.sh:125-139`). It passes 21/21.
   - The only production spawn site is `reviewcmd/runtime.go:102`, which runs `nvim -u lua absFile`. nvim puts `--embed` at index 1 of the server's argv, so the served file stays last in the embed's argv as well.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - A review embed started by a `review.lua` from before this change writes a one-line record. The reaper skips it forever, so cleanup is left to artifactpath GC (nvim-pid is collectable). This is a one-time migration leftover and acceptable.
   - The residual the commit names is narrow and documented: two tags serving the same absolute file plus pid reuse. Closing it would need a nonce in the spawn's argv.
   - `sidecarRecord` assumes the served path has no newline. The risk is negligible.

5. **Test coverage notes**
   - Passing: `go test` for `launcher`, `artifactpath`, `reviewcmd` and `opener`; `tests/nvim-pidfile-lifecycle-test.sh` (21/21).
   - Mutation check (drop `target.ExpectedArgPath = served`): 9 rows in `TestReapNvimSidecarsKillsOnlyWhatTheEvidenceNames` and `TestSweepOrphanNvimDecidesEachPidOnItsOwnEvidence` go red. I restored the file afterwards, and the tree is clean.
   - The full `./cmd/internal/...` run has one failure: `broadcast/TestNamedTunnelServesFromPrivateSocket` (the origin is TCP, not a unix socket). It also fails when run alone. No broadcast file is in the window, and the only `procutil` change is two additive methods, so I judge it environmental and outside this boundary's scope.

6. **Architectural notes** (pass on all eight)
   - **ARCH-DRY:** pass. `sidecarRecord` replaces `sidecarPID` at both call sites.
   - **ARCH-PURE:** pass. The decision stays pure (`decideNvimSidecar`); the record read is in the IO shell.
   - **ARCH-PURPOSE:** pass. The class rule is applied to every kind, not just the review instance.
   - **ARCH-MOCK:** pass. The process table is a stateful fake behind `ProcessTable`, and the live conformance test exists.
   - **ARCH-CONSTRAINTS:** pass. There is no new per-pass work: the second line is parsed from a record that is already read.
   - **ARCH-SECURE:** pass. The record is parsed into a typed pid plus served path, a bad pid line is dropped, and missing evidence leads to keeping the record.
   - **ARCH-ORDER:** pass. Attribution is fixed at publish, so a retarget arriving afterwards can't flip a decision.
   - **ARCH-FUNERAL:** pass. A record kept as undecided is owned by storage GC, and a clean exit removes the record only if it still names this pid.

7. **Plan revision recommendations:** none. The plan's BR-7 entry matches the code.

```findings
dispose:
  - id: BR-7
    disposition: addressed
    note: |
      review.lua binds argv's served file into record line 2; reaper reads it via sidecarRecord (nvim_reap_os.go:330-337) and never the review-target state; mutating it out reddens 9 rows including the retarget reap/sweep rows and the cross-tag-live-pane row; lifecycle test pins line 2 (21/21 green).
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"d9f387c4-9145-4676-9d33-83e73cdabd12","repository":"github.com/xianxu/pair","issue_id":"000583","recipe":"milestone-review","round":5,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/0c245756041611c0","reviewed_head":"25ae5ed5385381dde0c9cab7e788e147bf5f0b8a","slot":"pair:4","requested_at":"2026-10-16T23:05:16.659949Z"},"phase":"finished","pid":72053,"started_at":"2026-10-16T23:05:16.931938Z","finished_at":"2026-10-16T23:18:06.411263Z","session_id":"b5dcb61d-5ea2-4c51-9e18-88997eb4276d","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.95203s
head: "25ae5ed5385381dde0c9cab7e788e147bf5f0b8a"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000408
