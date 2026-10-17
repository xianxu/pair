# Boundary Review — pair#583 (whole-issue close)

| field | value |
|-------|-------|
| issue | 583 — Orphaned embedded nvim (22 GB, 13 days) escaped the reaper: untagged 'nvim --embed' children; tag them and sweep orphans |
| repo | pair |
| issue file | workshop/issues/000583-orphaned-embedded-nvim-22-gb-13-days-escaped-the-reaper-untagged-nvim-embed-children-tag-them-and-sweep-orphans.md |
| boundary | whole-issue close |
| milestone | — |
| window | 73e97b0275919c5d6ebf6d16e32b41093276942c..138263a18151cca105a2fdeee0c4aa6fccc0d5f0 |
| command | sdlc close --issue 583 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-16T21:58:15-07:00 |
| verdict | SHIP |

## Review

Reviewing the M2 orphan-sweep code now; next I'll run the test suite and check a few edge cases.

```verdict
verdict: SHIP
confidence: medium
```

The M2 code is ready to close; I found nothing blocking. This round had no open findings to settle: BR-1 through BR-7 were all closed in earlier rounds, so I didn't reopen them and focused on the untagged-orphan sweep added at head. That code matches the Spec and the plan's M2 checklist. The orphan sweep and #587's unattributed report go through one shared predicate (`UnattributedCandidate`). The process age comes from the kernel identity stamp, so the age gate needs no extra syscall. The cwd probe only runs for rows that have already passed the cheaper checks, and it is checked against the process identity. `--confirm` is enforced by the server (`couchmessage.ValidateRequest`), not just by the CLI. A confirmed reap re-classifies from a fresh pass that the sampler's own worker runs, then ends each process tree through `TreeReaper`'s identity check. `go build`, `go vet` and the tests for all seven touched packages pass at head. `GOOS=linux go vet` is clean for `procutil` and `couchcmd`. Three Minor findings remain, none blocking: two places where docs or help text don't match the code, and one gap where symlinked paths are never matched.

1. **Strengths**
   - `couchcore/resource.go:331-371`: `UnattributedKind`, `UnattributedShape` and `UnattributedCandidate` are a real extraction (ARCH-DRY). The sampler's candidate filter (`resources.go` loop) and both classifiers now share one rule, and `ClassifyUnattributed`'s inline name set and `--embed` loop are gone.
   - `couchcore/resource_orphan.go:66-82`: `ClassifyOrphanNvim` is pure. Any probe result the sampler couldn't prove (cwd unknown, age unknown, negative age) means the process is not reported, never reported by default.
   - `couchcmd/resources.go` `run`/`sweep`: the confirmed reap's classification runs inside the sampler's own loop, so it never overlaps a regular pass. A pass that was cut short returns `fresh=false`, and the reap is refused instead of acting on an old cached view.
   - `couchmessage/protocol.go:262-276`: confirmation is required by the protocol itself, and the request must match exactly (`r != Request{Op, Confirmed:true}`).
   - `procutil/identity_age.go`: the parsers are separate from the IO (`ParseIdentityWallClock`, `ParseIdentityTicks`, `ParseBootTime`). It handles the darwin microsecond field correctly, and an unknown age means the process is not reported.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - `cmd/internal/couchcmd/run.go:1417`: the help text says that without `--confirm` the verb "re-classifies and lists them". The code (`refuseOrphanReap`) actually reads the cached `resources` report, as the README and atlas both say. Fix the help line.
   - `cmd/internal/couchcmd/skills/couch/SKILL.md:72` says `--reap-orphans` "holds the console for SIGTERM's grace period". But `messages.go` says the reap runs in the caller's process "and not in the console". One of the two descriptions is wrong.
   - `cmd/internal/couchcore/resource_orphan.go:33`: darwin's `cwdRead` returns a symlink-resolved path, and its own comment says callers must resolve the managed roots the same way. `ManagedRoots` uses `WorktreeRoot` unresolved. A worktree under a symlinked path would never match, and no error is reported. Calling `filepath.EvalSymlinks` on each root (or resolving them at inventory time) would fix it.

5. **Test coverage**
   - `TestClassifyOrphanNvimFalsifiesOneClaimPerRow` disproves each gate independently.
   - `TestWithinManagedRootIsSegmentSafe` pins that a root only matches whole path segments.
   - `TestReapOrphansConfirmationIsAProtocolGate` and `TestReapOrphansSupervisorClassifiesFresh` cover the two safety properties of the confirmed verb.
   - `TestReapOrphanNvimFencesEveryRowOnItsIdentity` covers recycled pids.
   - There is env-gated live conformance for both cwd and the reap.
   - Gap: no test checks that the sampler skips the cwd probe for a young nvim. The cost claim relies on this gate. The live cost re-measurement partly covers it.

6. **Architecture notes**
   - **ARCH-DRY: pass.** The shared predicate is extracted. The `rootWorkspace` map in `AdvanceResources` repeats `ManagedRoots`' de-duplication loop, but that is tolerable.
   - **ARCH-PURE: pass.** Classification is pure. The probes stay in the sampler, and the reap is injected through `launcher.Reaper`.
   - **ARCH-PURPOSE: pass.** Both halves of the issue are delivered: tagged processes are reaped by exact identity, and untagged ones are reported, alerted on and reaped on confirmation. The sweep only covers checkouts of slots that are currently live. That is narrower than the Spec's "pair-managed worktree", but the plan states it as a deliberate trade and the atlas documents it as a known limit.
   - **ARCH-MOCK: pass.** `resourceReader` and `ProcessTable` fakes cover the cwd and reap paths, with env-gated live conformance tests against the real system.
   - **ARCH-CONSTRAINTS: pass.** The cwd probe only runs past the age gate. Classification is capped at `ResourceMaxProcesses`, and the verb has a 2-minute backstop.
   - **ARCH-SECURE: pass.** The socket is owner-only and requests must match exactly. A failed cwd read is recorded as an error, never as "outside the checkouts".
   - **ARCH-ORDER: pass.** Sweep requests are served one at a time in the sampler loop. Every kill is checked against the identity the sweep recorded, and a pid that changed is skipped, never treated as a failure or a success. Ordering between `sweep` and the ticker isn't injectable, but the stale-view outcome is covered by `TestResourcePassReportsWhetherItClassifiedAnything`.
   - **ARCH-FUNERAL: pass.** The `nvim/` episode keys join the process-death cleanup (`processEpisodeKey`). The persisted `ResourceFrame.OrphanNvim` lives inside the existing 24-hour ring. Clean exit removes the pid sidecars, and the GC already covers the changelog `.nvim-pid` files.

7. **Plan revision recommendations:** none. The plan's M2 checklist and Revisions match the code.

```findings
findings:
  - id: new
    severity: Minor
    kind: defect
    family: doc-describes-other-behavior
    title: |
      run.go usage says --reap-orphans without --confirm re-classifies; the code reads the cached report
    detail: |
      cmd/internal/couchcmd/run.go:1417 says "Without --confirm it re-classifies and lists them", but refuseOrphanReap sends Op "resources" (the cached view), as README and atlas/couch.md both state. Fix the help line.
  - id: new
    severity: Minor
    kind: defect
    family: doc-describes-other-behavior
    title: |
      SKILL.md says --reap-orphans holds the console; messages.go says the reap runs in the caller's process
    detail: |
      cmd/internal/couchcmd/skills/couch/SKILL.md cost line says "it holds the console for SIGTERM's grace period", but messages.go runs TreeReaper in the CLI process "and not in the console". Pick the accurate description.
  - id: new
    severity: Minor
    kind: defect
    family: path-containment-unresolved-roots
    title: |
      ManagedRoots compares unresolved WorktreeRoot against the kernel's symlink-resolved cwd
    detail: |
      resource_darwin.go cwdRead says callers must resolve managed roots the same way, but ManagedRoots (resource_orphan.go:33) uses WorktreeRoot as-is. A checkout under a symlinked path is never matched, so the sweep misses it without reporting anything. This narrows the sweep (the safe direction) but is a coverage hole. Fix: EvalSymlinks each root.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"4772d198-5a78-4ea2-938c-fd7a5fd18778","repository":"github.com/xianxu/pair","issue_id":"000583","recipe":"milestone-review","round":6,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/5190fb391b1b0a66","reviewed_head":"138263a18151cca105a2fdeee0c4aa6fccc0d5f0","slot":"pair:4","requested_at":"2026-10-17T04:48:59.450164Z"},"phase":"finished","pid":2917,"started_at":"2026-10-17T04:48:59.729691Z","finished_at":"2026-10-17T04:58:15.42057Z","session_id":"f4b10631-1d8a-4396-bbd5-2ed30058014b","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.075139792s
head: "138263a18151cca105a2fdeee0c4aa6fccc0d5f0"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-17T05:56:02-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 583 — Orphaned embedded nvim (22 GB, 13 days) escaped the reaper: untagged 'nvim --embed' children; tag them and sweep orphans |
| repo | pair |
| issue file | workshop/issues/000583-orphaned-embedded-nvim-22-gb-13-days-escaped-the-reaper-untagged-nvim-embed-children-tag-them-and-sweep-orphans.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1dc8d6c4aec48cf6fc1e89a574202ae6f5b322d4..c1f399ca89b85cf38b1284f5a7966973f4a75cd7 |
| command | sdlc close --issue 583 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-17T05:56:02-07:00 |
| verdict | SHIP |

## Review

Build and the artifactpath/launcher tests pass. Next I'm running the couchcmd tests and checking how the conflicts were resolved.

```verdict
verdict: SHIP
confidence: high
```

This window doesn't change what the close review covered. The base commit (`1dc8d6c4a`, "sdlc: reviewed patch rebased") contained committed conflict markers. Most of the interdiff resolves them in six files. The rest is a bootstrap line added to `tests/nvim-pidfile-lifecycle-test.sh` and some bookkeeping (lessons, close-gate and close-review notes, issue status, and a close Log entry). Each resolution keeps what both sides meant:

- **`message_service.go`:** keeps main's hosted-ingest wiring and the branch's `sweeps` channel. It also keeps both the `refusals` case and the `orphanReapOp` case.
- **`README.md`:** keeps main's `--resources` line and the branch's `--reap-orphans` block.
- **`resources_cli_test.go`:** keeps main's hosted-ingest tests, their imports, and the branch's orphan-sweep render test.
- **`osruntime_test.go`:** keeps main's #617 title-lease behaviour.
- **`Makefile.local`:** adds `test-build-go-atomic` and `test-nvim-pidfile` to the test target.
- **`manifest.go`:** drops `draft`/`nvim-pid`/`title-pid` from the `osruntime.go` classification. The exact-classification guard confirms this matches what `osruntime.go` actually references at head.

None of the six files contains conflict markers any more. `go build ./...` and `go vet` are clean. These all pass at head:

- `go test ./cmd/internal/artifactpath`, which includes `TestProductionArtifactReferencesAreExactlyClassified`
- `go test ./cmd/internal/launcher`
- the `couchcmd` resource, orphan and reap tests
- `tests/nvim-pidfile-lifecycle-test.sh`

`## Spec` and `## Done when` are unchanged. The issue edits only tick M2 and add the close Log entry, so the scope didn't change. The window doesn't touch `run.go`, `SKILL.md` or `resource_orphan.go`, so the three open Minors (BR-8, BR-9, BR-10) are still open. They are Minor and don't block.

1. **Strengths**
   - Each conflict resolution keeps both sides' intent. In `message_service.go`, both the `refusals` and `orphanReapOp` ops are handled, and the sampler keeps both `hostedIngest` and `sweeps`.
   - The manifest resolution is checked by the exhaustive inventory test rather than chosen by hand, and that test passes.
   - The pidfile lifecycle shell test now sources the shared bootstrap, so it runs under the same test-binary environment as the rest of `make test`.
   - The new lesson entries are accurate. The BR-5/6/7 entries were moved to sit next to the other #583 lessons, not lost.
2. **Critical:** none.
3. **Important:** none.
4. **Minor:** BR-8, BR-9 and BR-10 are still open (see the dispositions).
5. **Test coverage:** the conflict resolutions are covered by existing tests that run green at head (artifactpath inventory, launcher, couchcmd resource and orphan tests, and the nvim-pidfile shell test). I only ran targeted packages, not the whole `./...` suite.
6. **Architecture:**
   - ARCH-DRY passes: the resolutions don't duplicate anything, and the `refusals` case appears once.
   - ARCH-PURE passes: the window adds no new logic.
   - ARCH-PURPOSE passes for this window, which only resolves conflicts. BR-10 is the remaining coverage gap against the purpose of sweeping orphans, and it is tracked as Minor.
7. **Plan revisions:** none needed.

```findings
dispose:
  - id: BR-8
    disposition: not-addressed
    note: |
      run.go:1429-1430 still says "Without --confirm it re-classifies and lists them"; run.go is untouched in this window.
  - id: BR-9
    disposition: not-addressed
    note: |
      SKILL.md:72 still says --reap-orphans "holds the console for SIGTERM's grace period"; SKILL.md is untouched in this window.
  - id: BR-10
    disposition: not-addressed
    note: |
      couchcore/resource_orphan.go:37 still uses slot.Workspace.WorktreeRoot unresolved (no EvalSymlinks in the file); untouched in this window.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"d7650925-5257-415a-9a40-cefc8cdbae2e","repository":"github.com/xianxu/pair","issue_id":"000583","recipe":"fix-review","round":7,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/36fe33ff47581ca5","reviewed_head":"c1f399ca89b85cf38b1284f5a7966973f4a75cd7","slot":"pair:4","requested_at":"2026-10-17T12:53:14.127977Z"},"phase":"finished","pid":81549,"started_at":"2026-10-17T12:53:14.342207Z","finished_at":"2026-10-17T12:56:01.913404Z","session_id":"93adf487-ceb2-432c-a117-f68686035fc8","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.32053675s
head: "c1f399ca89b85cf38b1284f5a7966973f4a75cd7"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411
